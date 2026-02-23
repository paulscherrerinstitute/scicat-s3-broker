package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/paulscherrerinstitute/scicat-s3-broker/internal/config"
)

type SciCatUrlResponse struct {
	URL     string    `json:"url"`
	Expires time.Time `json:"expires"`
}

type JobsResponse struct {
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

type SciCatHandler struct {
	config     *config.Config
	tokenMutex sync.RWMutex
	token      SciCatLoginResponse
}

func NewSciCatHandler(cfg *config.Config) *SciCatHandler {
	return &SciCatHandler{
		config: cfg,
	}
}

const iso8601Layout = "20060102T150405Z"

func (h *SciCatHandler) logIn() (SciCatLoginResponse, error) {
	var loginResp SciCatLoginResponse

	creds, err := json.Marshal(gin.H{
		"username": h.config.JobManagerUsername,
		"password": h.config.JobManagerPassword,
	})
	if err != nil {
		return loginResp, fmt.Errorf("failed to marshal credentials: %w", err)
	}

	resp, err := http.Post(fmt.Sprintf("%s/api/v3/auth/login", h.config.SciCatURL), "application/json", bytes.NewReader(creds))
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

	// Refreshes 10 mins before actual expiration
	return time.Now().Add(10 * time.Minute).After(expirationTime)
}

func (h *SciCatHandler) isPublic(datasetPid string) bool {
	filterQuery, err := json.Marshal(gin.H{"fields": []string{"_id"}})
	if err != nil {
		return false
	}

	u, err := url.Parse(fmt.Sprintf("%s/api/v3/datasets/%s", h.config.SciCatURL, url.PathEscape(datasetPid)))
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

func makeJobsFilter(pid string) ([]byte, error) {
	filterQuery, err := json.Marshal(gin.H{
		"where": gin.H{
			"type":                      gin.H{"$in": []string{"retrieve", "public"}},
			"jobParams.option":          "URLs",
			"statusCode":                "finishedSuccessful",
			"jobParams.datasetList.pid": pid,
		},
		"sort": gin.H{"createdAt": -1},
		"limits": gin.H{
			"limit": 1,
			"skip":  0,
		}})
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal json: %w", err)
	}
	return filterQuery, nil

}

func (h *SciCatHandler) GetActiveUrls(c *gin.Context) {
	dataset := c.Query("dataset")

	if dataset == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "dataset parameter is required",
		})
		return
	}

	if !h.isPublic(dataset) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Dataset not accessible"})
		return
	}

	accessToken, err := h.getToken()
	if err != nil {
		log.Printf("Error in getToken: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	authHeader := fmt.Sprintf("Bearer %s", accessToken)

	filterQuery, err := makeJobsFilter(dataset)
	if err != nil {
		log.Printf("Error creating filter: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	u, err := url.Parse(fmt.Sprintf("%s/api/v4/jobs", h.config.SciCatURL))
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

	if resp.StatusCode != http.StatusOK {
		log.Printf("Invalid response code from /jobs: %v", resp.StatusCode)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	var jobResp []JobsResponse
	err = json.NewDecoder(resp.Body).Decode(&jobResp)
	if err != nil {
		log.Printf("failed to unmarshal jobs response: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	if len(jobResp) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No URLs available. Submit a URL retrive job in SciCat"})
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

// parseExpirationTime computes the expiration time from
// X-Amz-Date and X-Amz-Expires query params. See
// https://docs.aws.amazon.com/AmazonS3/latest/API/sigv4-query-string-auth.html
func parseExpirationTime(urlstr string) (time.Time, error) {
	var result time.Time
	u, err := url.Parse(urlstr)
	if err != nil {
		return result, fmt.Errorf("failed to parse url: %v", err)
	}
	date, exp := u.Query().Get("X-Amz-Date"), u.Query().Get("X-Amz-Expires")
	if date == "" || exp == "" {
		return result, fmt.Errorf("required params X-Amz-Date and X-Amz-Expires not present in %v", urlstr)
	}
	result, err = time.Parse(iso8601Layout, date)
	if err != nil {
		return result, fmt.Errorf("failed to parse date according to iso8601, %v", date)
	}
	expint, err := strconv.Atoi(exp)
	if err != nil {
		return result, fmt.Errorf("failed to parse expriry to int %v", exp)
	}
	return result.Add(time.Second * time.Duration(expint)), nil
}

func toSciCatUrlResponse(pid string, resp JobsResponse) ([]SciCatUrlResponse, error) {
	if len(resp.JobResultObject.Result) == 0 {
		return nil, errors.New("no URLs available in job response")
	}

	result := []SciCatUrlResponse{}
	for _, x := range resp.JobResultObject.Result {
		if x.DatasetId == pid {
			expirationTime, err := parseExpirationTime(x.Url)
			if err != nil {
				return result, err
			}
			result = append(result, SciCatUrlResponse{x.Url, expirationTime})
		}
	}

	return result, nil
}
