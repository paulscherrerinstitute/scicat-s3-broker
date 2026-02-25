package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/paulscherrerinstitute/scicat-s3-broker/internal/api"
	"github.com/paulscherrerinstitute/scicat-s3-broker/internal/config"
)

func getTestConfig(serverURL string) *config.Config {
	return &config.Config{
		SciCatURL:          serverURL,
		JobManagerUsername: "testuser",
		JobManagerPassword: "testpass",
	}
}

func makeJobResponse(t *testing.T, jsonStr string) JobsResponse {
	var jr JobsResponse
	err := json.Unmarshal([]byte(jsonStr), &jr)
	if err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}
	return jr
}

func TestToSciCatUrlResponse(t *testing.T) {
	now := time.Now().UTC()
	validTimeIso8601Str := now.Format(iso8601Layout)
	expiresSeconds := 604800

	tests := []struct {
		name          string
		pid           string
		inputJSON     string
		wantErr       bool
		expectedCount int
	}{
		{
			name: "Valid Single Result",
			pid:  "pid-123",
			inputJSON: fmt.Sprintf(`{
				"jobResultObject": {
					"result": [
						{"datasetId": "pid-123", "url": "s3://bucket/file1?X-Amz-Date=%s&X-Amz-Expires=%v"}
					]
				}
			}`, validTimeIso8601Str, expiresSeconds),
			wantErr:       false,
			expectedCount: 1,
		},
		{
			name: "Filter Irrelevant PIDs",
			pid:  "pid-123",
			inputJSON: fmt.Sprintf(`{
				"jobResultObject": {
					"result": [
						{"datasetId": "pid-123", "url": "s3://bucket/match?X-Amz-Date=%s&X-Amz-Expires=%v"},
						{"datasetId": "pid-456", "url": "s3://bucket/ignore"}
					]
				}
			}`, validTimeIso8601Str, expiresSeconds),
			wantErr:       false,
			expectedCount: 1,
		},
		{
			name: "Empty Result List",
			pid:  "pid-123",
			inputJSON: `{
				"jobResultObject": {
					"result": []
				}
			}`,
			wantErr: true,
		},
		{
			name: "Invalid Time Format",
			pid:  "pid-123",
			inputJSON: `{
				"jobResultObject": {
					"result": [
						{"datasetId": "pid-123", "url": "s3://bucket/file1?X-Amz-Date=not-a-timestamp"}
					]
				}
			}`,
			wantErr: true,
		},
		{
			name: "Missing required url param X-Amz-Expires",
			pid:  "pid-123",
			inputJSON: fmt.Sprintf(`{
				"jobResultObject": {
					"result": [
						{"datasetId": "pid-123", "url": "s3://bucket/file1?X-Amz-Date=%s"}
					]
				}
			}`, validTimeIso8601Str),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jobResp := makeJobResponse(t, tt.inputJSON)
			got, err := toSciCatUrlResponse(tt.pid, jobResp)

			if (err != nil) != tt.wantErr {
				t.Errorf("toSciCatUrlResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(got) != tt.expectedCount {
					t.Errorf("Expected %d URLs, got %d", tt.expectedCount, len(got))
				}
				// Verify expiration is 7 days from creation
				expectedExp := now.Add(time.Second * time.Duration(expiresSeconds))
				diff := got[0].Expires.Sub(expectedExp)
				tolerance := 1 * time.Second
				if diff < -tolerance || diff > tolerance {
					t.Errorf("Expiration date mismatch.\nGot:  %v\nWant: %v\nDiff: %v", got[0].Expires, expectedExp, diff)
				}
			}
		})
	}
}

func TestGetDatasetsUrls(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Now().UTC()
	validTimeIso8601Str := now.Format(iso8601Layout)
	expiresSeconds := 604800

	tests := []struct {
		name           string
		datasetPid     string
		mockPublicCode int
		mockLoginCode  int
		mockJobsCode   int
		mockJobsBody   string
		wantStatusCode int
	}{
		{
			name:           "Success",
			datasetPid:     "valid-pid",
			mockPublicCode: http.StatusOK,
			mockLoginCode:  http.StatusCreated,
			mockJobsCode:   http.StatusOK,
			mockJobsBody: fmt.Sprintf(`[{
				"jobResultObject": {
					"result": [{"datasetId": "valid-pid", "url": "http://result?X-Amz-Date=%s&X-Amz-Expires=%v"}]
				}
			}]`, validTimeIso8601Str, expiresSeconds),
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "No dataset",
			datasetPid:     "",
			mockPublicCode: http.StatusNotFound,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "Dataset Not Public or Not Found",
			datasetPid:     "private-pid-or-no-such-pid",
			mockPublicCode: http.StatusNotFound,
			wantStatusCode: http.StatusUnauthorized,
		},
		{
			name:           "Login Failed",
			datasetPid:     "valid-pid",
			mockPublicCode: http.StatusOK,
			mockLoginCode:  http.StatusUnauthorized,
			wantStatusCode: http.StatusInternalServerError,
		},
		{
			name:           "Jobs Fetch Error",
			datasetPid:     "valid-pid",
			mockPublicCode: http.StatusOK,
			mockLoginCode:  http.StatusCreated,
			mockJobsCode:   http.StatusInternalServerError,
			wantStatusCode: http.StatusInternalServerError,
		},
		{
			name:           "No Jobs Found",
			datasetPid:     "valid-pid",
			mockPublicCode: http.StatusOK,
			mockLoginCode:  http.StatusCreated,
			mockJobsCode:   http.StatusOK,
			mockJobsBody:   `[]`,
			wantStatusCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scicatServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Mock /login
				if r.URL.Path == "/api/v3/auth/login" {
					if tt.mockLoginCode != http.StatusCreated {
						w.WriteHeader(tt.mockLoginCode)
						return
					}
					w.WriteHeader(http.StatusCreated)
					// Return a valid dummy token
					json.NewEncoder(w).Encode(SciCatLoginResponse{
						AccessToken: "test-token-123",
						ExpiresIn:   3600,
						CreatedAt:   time.Now().Format(time.RFC3339),
					})
					return
				}

				// Mock /datasets/{pid}
				if strings.Contains(r.URL.Path, "/api/v3/datasets/") {
					w.WriteHeader(tt.mockPublicCode)
					return
				}

				// Mock /jobs
				if r.URL.Path == "/api/v4/jobs" {
					if tt.mockJobsCode != http.StatusOK {
						w.WriteHeader(tt.mockJobsCode)
						return
					}
					w.Header().Set("Content-Type", "application/json")
					w.Write([]byte(tt.mockJobsBody))
					return
				}

				http.NotFound(w, r)
			}))
			defer scicatServer.Close()

			h := NewSciCatHandler(getTestConfig(scicatServer.URL))

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			req, _ := http.NewRequest("GET", "datasets/urls/?id="+tt.datasetPid, nil)
			c.Request = req

			h.GetDatasetsUrls(c, api.GetDatasetsUrlsParams{Id: tt.datasetPid})

			if w.Code != tt.wantStatusCode {
				t.Errorf("GetDatasetsUrls() status = %v, want %v", w.Code, tt.wantStatusCode)
			}

			if tt.wantStatusCode == http.StatusOK {
				var resp []SciCatUrlResponse
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Errorf("Failed to unmarshal success response: %v", err)
				}
				if len(resp) == 0 {
					t.Error("Expected URLs in success response, got empty list")
				}
			}
		})
	}
}
