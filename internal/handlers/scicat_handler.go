package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type SciCatUrlResponse struct {
	URL     string
	Expires time.Time
}

type JobsResponse struct {
	CreationTime    string `json:"creationTime"`
	JobResultObject struct {
		Result []struct {
			DatasetId string `json:"datasetId"`
			Url       string `json:"url"`
		} `json:"result"`
	} `json:"jobResultObject"`
}

func GetActiveUrls(c *gin.Context) {
	// Get the Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Authorization header is required",
		})
		return
	}

	const bearerPrefix = "Bearer "
	if !strings.HasPrefix(authHeader, bearerPrefix) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid authorization header format. Expected 'Bearer <token>'",
		})
		return
	}

	fieldsQuery, err := json.Marshal(map[string]any{
		"type":             "retrieve",
		"jobParams.option": "URLs",
		"jobStatusMessage": "finishedSuccessful",
		// "datasetList": map[string]any{
		// 	"pid":   "20.500.11935/0e54729b-75c5-42fa-a628-aae5dc3f3dae",
		// 	"files": []any{},
		// },
	})
	if err != nil {
		log.Printf("Failed to marshal json: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Internal server error",
		})
		return
	}

	//{"limit": 10, "skip": 0, "order": "creationTime:desc"}
	limitsQuery, err := json.Marshal(map[string]any{
		"limit": 2,
		"skip":  0,
		"order": "creationTime:desc",
	})
	if err != nil {
		log.Printf("Failed to marshal json: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Internal server error",
		})
		return
	}

	u, err := url.Parse("https://scicat.development.psi.ch/api/v3/jobs/fullquery")
	if err != nil {
		log.Printf("Failed to parse URL: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Internal server error",
		})
		return
	}
	q := u.Query()
	q.Set("fields", string(fieldsQuery))
	q.Set("limits", string(limitsQuery))
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
		log.Println("cannot unmarshall responseBody to jobResp: ", err)
	}
	// log.Println("unmarshalled jobResp: ", jobResp)
	scicatUrlResp, err := toSciCatUrlResponse(jobResp[0])
	if err != nil {
		log.Println("error converting to Url response: ", err)
	}

	c.JSON(http.StatusOK, scicatUrlResp)

}

func toSciCatUrlResponse(resp JobsResponse) (*SciCatUrlResponse, error) {
	if len(resp.JobResultObject.Result) == 0 {
		return nil, errors.New("no URLs")
	}
	creationTime, err := time.Parse(time.RFC3339, resp.CreationTime)
	return &SciCatUrlResponse{resp.JobResultObject.Result[0].Url, creationTime}, err
}
