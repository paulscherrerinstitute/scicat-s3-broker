package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type SciCatUrlResponse struct {
	URL     string    `json:"url"`
	Expires time.Time `json:"expires"`
}

type JobsResponse struct {
	CreationTime    string `json:"createdAt"`
	JobResultObject struct {
		Result []struct {
			DatasetId string `json:"datasetId"`
			Url       string `json:"url"`
		} `json:"result"`
	} `json:"jobResultObject"`
}

type SciCatLoginResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	CreatedAt   string `json:"created"`
}

const urlExpireDays = 7

var (
	tokenMutex       = &sync.RWMutex{}
	currentLoginResp SciCatLoginResponse
)

// logIn returns a valid jobManager token or an error
func logIn() (SciCatLoginResponse, error) {
	var loginResp SciCatLoginResponse

	creds, err := json.Marshal(gin.H{
		"username": "jobManager",
		"password": os.Getenv("JOB_MANAGER_PASSWORD"),
	})
	if err != nil {
		return loginResp, fmt.Errorf("failed to marshal credentials: %w", err)
	}

	resp, err := http.Post(fmt.Sprintf("%s/api/v3/auth/login", os.Getenv("SCICAT_URL")), "application/json", bytes.NewReader(creds))
	if err != nil || resp.StatusCode != http.StatusCreated {
		return loginResp, fmt.Errorf("failed to login: %v %v", err, resp.StatusCode)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return loginResp, fmt.Errorf("failed to read response body: %w", err)
	}

	err = json.Unmarshal(respBody, &loginResp)
	if err != nil {
		return loginResp, fmt.Errorf("failed to unmarshal login response: %w", err)
	}
	return loginResp, nil
}

func isExpired(resp SciCatLoginResponse) bool {
	tokenMutex.RLock()
	defer tokenMutex.RUnlock()
	if currentLoginResp.AccessToken == "" {
		return true
	}
	createdAt, err := time.Parse(time.RFC3339, resp.CreatedAt)
	if err != nil {
		log.Printf("failed to parse token creation time: %v", err)
		return true // treat as expired if we can't parse the time
	}
	expirationTime := createdAt.Add(time.Second * time.Duration(resp.ExpiresIn))
	return time.Now().Add(time.Hour).After(expirationTime)
}

func isPublic(datasetPid string) bool {
	filterQuery, err := json.Marshal(gin.H{
		"fields": []string{"_id"},
	})
	if err != nil {
		log.Printf("failed to marshal filter query: %v", err)
		return false
	}

	u, err := url.Parse(fmt.Sprintf("%s/api/v3/datasets/%s", os.Getenv("SCICAT_URL"), url.PathEscape(datasetPid)))
	if err != nil {
		log.Printf("failed to parse dataset URL: %v", err)
		return false
	}
	q := u.Query()
	q.Set("filter", string(filterQuery))
	u.RawQuery = q.Encode()

	resp, err := http.Get(u.String())
	if err != nil {
		log.Printf("failed to check dataset public status: %v", err)
		return false
	}
	defer resp.Body.Close()

	log.Printf("dataset public check status: %s", resp.Status)
	return resp.StatusCode == http.StatusOK
}

func GetActiveUrls(c *gin.Context) {
	dataset := c.Query("dataset")

	if !isPublic(dataset) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "dataset not public"})
		return
	}

	if isExpired(currentLoginResp) {
		log.Println("refreshing expired token")
		loginResp, err := logIn()
		if err != nil {
			log.Printf("failed to login: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to authenticate"})
			return
		}
		tokenMutex.Lock()
		currentLoginResp = loginResp
		tokenMutex.Unlock()
	}

	tokenMutex.RLock()
	authHeader := fmt.Sprintf("Bearer %s", currentLoginResp.AccessToken)
	tokenMutex.RUnlock()

	filterQuery, err := json.Marshal(gin.H{
		"where": gin.H{
			"type":                      gin.H{"$in": []string{"retrieve", "public"}},
			"jobParams.option":          "URLs",
			"statusCode":                "finishedSuccessful",
			"jobParams.datasetList.pid": dataset,
		},
		"sort": gin.H{"createdAt": -1},
		"limits": gin.H{
			"limit": 1,
			"skip":  0,
		}})
	if err != nil {
		log.Printf("Failed to marshal json: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Internal server error",
		})
		return
	}

	u, err := url.Parse(fmt.Sprintf("%s/api/v4/jobs", os.Getenv("SCICAT_URL")))
	if err != nil {
		log.Printf("Failed to parse URL: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Internal server error",
		})
		return
	}
	q := u.Query()
	q.Set("filter", string(filterQuery))
	u.RawQuery = q.Encode()
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		log.Printf("Failed to create request: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	req.Header.Set("Authorization", authHeader)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("Failed to execute request: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read response body: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	var jobResp []JobsResponse
	err = json.Unmarshal(responseBody, &jobResp)
	if err != nil {
		log.Printf("failed to unmarshal jobs response: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	if len(jobResp) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "no URLs available"})
		return
	}

	scicatUrlResp, err := toSciCatUrlResponse(jobResp[0])
	if err != nil {
		log.Printf("failed to convert to URL response: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	c.PureJSON(http.StatusOK, scicatUrlResp)
}

func toSciCatUrlResponse(resp JobsResponse) (*SciCatUrlResponse, error) {
	if len(resp.JobResultObject.Result) == 0 {
		return nil, errors.New("no URLs available in job response")
	}

	creationTime, err := time.Parse(time.RFC3339, resp.CreationTime)
	if err != nil {
		return nil, fmt.Errorf("failed to parse creation time: %w", err)
	}

	expirationTime := creationTime.AddDate(0, 0, urlExpireDays)
	return &SciCatUrlResponse{resp.JobResultObject.Result[0].Url, expirationTime}, nil
}
