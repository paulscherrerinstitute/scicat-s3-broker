package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type SciCatUrlResponse struct {
	URLs    []string  `json:"urls"`
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

type Config struct {
	sciCatURL string
	username  string
	password  string
}

type SciCatHandler struct {
	config     Config
	tokenMutex sync.RWMutex
	token      SciCatLoginResponse
}

func NewSciCatHandler() *SciCatHandler {
	scicatURL := os.Getenv("SCICAT_URL")
	if scicatURL == "" {
		log.Fatal("SCICAT_URL environment variable is required")
	}

	return &SciCatHandler{
		config: Config{
			sciCatURL: strings.TrimRight(scicatURL, "/"),
			username:  "jobManager",
			password:  os.Getenv("JOB_MANAGER_PASSWORD"),
		},
	}
}

func (h *SciCatHandler) logIn() (SciCatLoginResponse, error) {
	var loginResp SciCatLoginResponse

	creds, err := json.Marshal(gin.H{
		"username": h.config.username,
		"password": h.config.password,
	})
	if err != nil {
		return loginResp, fmt.Errorf("failed to marshal credentials: %w", err)
	}

	resp, err := http.Post(fmt.Sprintf("%s/api/v3/auth/login", h.config.sciCatURL), "application/json", bytes.NewReader(creds))
	if err != nil {
		return loginResp, fmt.Errorf("POST /login failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return loginResp, fmt.Errorf("Invalid status code from /login: %v", resp.StatusCode)
	}

	err = json.NewDecoder(resp.Body).Decode(&loginResp)
	if err != nil {
		return loginResp, fmt.Errorf("failed to unmarshal login response: %w", err)
	}
	return loginResp, nil
}

func (h *SciCatHandler) isTokenExpired() bool {
	h.tokenMutex.RLock()
	defer h.tokenMutex.RUnlock()

	if h.token.AccessToken == "" {
		return true
	}
	createdAt, err := time.Parse(time.RFC3339, h.token.CreatedAt)
	if err != nil {
		log.Printf("failed to parse token creation time: %v", err)
		return true
	}
	expirationTime := createdAt.Add(time.Second * time.Duration(h.token.ExpiresIn))

	// Refreshes 1 hour before actual expiration
	return time.Now().Add(time.Hour).After(expirationTime)
}

func (h *SciCatHandler) isPublic(datasetPid string) bool {
	filterQuery, err := json.Marshal(gin.H{"fields": []string{"_id"}})
	if err != nil {
		return false
	}

	u, err := url.Parse(fmt.Sprintf("%s/api/v3/datasets/%s", h.config.sciCatURL, url.PathEscape(datasetPid)))
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

	return resp.StatusCode == http.StatusOK
}

func (h *SciCatHandler) getToken() (string, error) {
	if h.isTokenExpired() {
		log.Println("refreshing expired token")
		loginResp, err := h.logIn()
		if err != nil {
			return "", err
		}
		h.tokenMutex.Lock()
		h.token = loginResp
		h.tokenMutex.Unlock()
	}

	h.tokenMutex.RLock()
	defer h.tokenMutex.RUnlock()
	return h.token.AccessToken, nil
}

func (h *SciCatHandler) GetActiveUrls(c *gin.Context) {
	dataset := c.Query("dataset")

	if !h.isPublic(dataset) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "dataset not public"})
		return
	}

	accessToken, err := h.getToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	authHeader := fmt.Sprintf("Bearer %s", accessToken)

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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	u, err := url.Parse(fmt.Sprintf("%s/api/v4/jobs", h.config.sciCatURL))
	if err != nil {
		log.Printf("Failed to parse URL: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
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
	req = req.WithContext(c.Request.Context())
	req.Header.Set("Authorization", authHeader)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("Failed to execute request: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	defer resp.Body.Close()

	var jobResp []JobsResponse
	err = json.NewDecoder(resp.Body).Decode(&jobResp)
	if err != nil {
		log.Printf("failed to unmarshal jobs response: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	if len(jobResp) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "no URLs available"})
		return
	}

	scicatUrlResp, err := toSciCatUrlResponse(dataset, jobResp[0])
	if err != nil {
		log.Printf("failed to convert to URL response: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	c.PureJSON(http.StatusOK, scicatUrlResp)
}

func toSciCatUrlResponse(pid string, resp JobsResponse) (*SciCatUrlResponse, error) {
	if len(resp.JobResultObject.Result) == 0 {
		return nil, errors.New("no URLs available in job response")
	}

	result := []string{}
	for _, x := range resp.JobResultObject.Result {
		if x.DatasetId == pid {
			result = append(result, x.Url)
		}
	}

	creationTime, err := time.Parse(time.RFC3339, resp.CreationTime)
	if err != nil {
		return nil, fmt.Errorf("failed to parse creation time: %w", err)
	}
	const urlExpireDays = 7
	expirationTime := creationTime.AddDate(0, 0, urlExpireDays)
	return &SciCatUrlResponse{result, expirationTime}, nil
}
