package scicat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/paulscherrerinstitute/scicat-s3-broker/internal/api"
	"github.com/paulscherrerinstitute/scicat-s3-broker/internal/config"
)

func TestDatasetsServiceGetUrls(t *testing.T) {
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
		wantErr        bool
		wantErrIs      error
		wantResult     api.DatasetsUrlResponse
	}{
		{
			name:           "Success",
			datasetPid:     "valid-pid",
			mockPublicCode: http.StatusOK,
			mockLoginCode:  http.StatusCreated,
			mockJobsCode:   http.StatusOK,
			mockJobsBody: fmt.Sprintf(`[{
                "jobResultObject": {
                    "result": [{"datasetId": "valid-pid", "url": "s3://bucket/file?X-Amz-Date=%s&X-Amz-Expires=%v"}]
                }
            }]`, validTimeIso8601Str, expiresSeconds),
			wantErr: false,
			wantResult: api.DatasetsUrlResponse{
				Urls: []api.UrlInfo{
					{
						Url:     fmt.Sprintf("s3://bucket/file?X-Amz-Date=%s&X-Amz-Expires=%v", validTimeIso8601Str, expiresSeconds),
						Expires: now.Add(time.Second * time.Duration(expiresSeconds)),
					},
				},
			},
		},
		{
			name:           "Dataset Not Public",
			datasetPid:     "private-pid",
			mockPublicCode: http.StatusForbidden,
			wantErr:        true,
			wantErrIs:      DatasetNotAccessibleError{"private-pid"},
		},
		{
			name:           "Dataset Not Found",
			datasetPid:     "no-such-pid",
			mockPublicCode: http.StatusNotFound,
			wantErr:        true,
			wantErrIs:      DatasetNotFoundError{"no-such-pid"},
		},
		{
			name:           "Login Failed",
			datasetPid:     "valid-pid",
			mockPublicCode: http.StatusOK,
			mockLoginCode:  http.StatusUnauthorized,
			wantErr:        true,
		},
		{
			name:           "Jobs Fetch Error",
			datasetPid:     "valid-pid",
			mockPublicCode: http.StatusOK,
			mockLoginCode:  http.StatusCreated,
			mockJobsCode:   http.StatusInternalServerError,
			wantErr:        true,
		},
		{
			name:           "No Jobs Found",
			datasetPid:     "valid-pid",
			mockPublicCode: http.StatusOK,
			mockLoginCode:  http.StatusCreated,
			mockJobsCode:   http.StatusOK,
			mockJobsBody:   `[]`,
			wantErr:        false,
			wantResult:     api.DatasetsUrlResponse{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scicatServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/api/v3/auth/login" && r.Method == "POST" {
					if tt.mockLoginCode == http.StatusCreated {
						w.WriteHeader(http.StatusCreated)
						json.NewEncoder(w).Encode(map[string]interface{}{
							"access_token": "dummy-token",
							"expires_in":   3600,
							"created":      now.Format(time.RFC3339),
						})
					} else {
						w.WriteHeader(tt.mockLoginCode)
					}
				} else if strings.Contains(r.URL.Path, "/api/v3/datasets/") && r.Method == "GET" {
					w.WriteHeader(tt.mockPublicCode)
				} else if r.URL.Path == "/api/v4/jobs" && r.Method == "GET" {
					if tt.mockJobsCode == http.StatusOK {
						w.Header().Set("Content-Type", "application/json")
						w.Write([]byte(tt.mockJobsBody))
					} else {
						w.WriteHeader(tt.mockJobsCode)
					}
				} else {
					http.NotFound(w, r)
				}
			}))
			defer scicatServer.Close()

			svc := &DatasetsServiceImpl{
				config: &config.Config{
					SciCatURL:          scicatServer.URL,
					JobManagerUsername: "testuser",
					JobManagerPassword: "testpass",
				},
			}

			result, err := svc.GetUrls(context.Background(), tt.datasetPid)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("GetUrls() expected error, got nil")
				}
				if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
					t.Errorf("GetUrls() error = %v, wantErrIs %v", err, tt.wantErrIs)
				}
			} else {
				if err != nil {
					t.Fatalf("GetUrls() unexpected error: %v", err)
				}
			}

			if !tt.wantErr {
				expectedNoJobsResp := api.DatasetsUrlResponse{Expires: unixEpoch, Urls: []api.UrlInfo{}}
				if len(result.Urls) > 0 && result.Urls[0].Url != tt.wantResult.Urls[0].Url {
					t.Errorf("GetUrls() mismatch\ngot:  %+v\nwant: %+v", result, tt.wantResult)
				} else if tt.name == "No Jobs Found" && !cmp.Equal(*result, expectedNoJobsResp) {
					t.Errorf("GetUrls() mismatch:\ndiff %v", cmp.Diff(*result, expectedNoJobsResp))
				}
			}
		})
	}
}
func TestToDatasetsUrlResponse(t *testing.T) {
	now := time.Now().UTC()
	validTimeIso8601Str := now.Format(iso8601Layout)
	expiresSeconds := 604800
	expiresLater := 691200

	tests := []struct {
		name          string
		pid           string
		inputJSON     string
		wantErr       bool
		expectedCount int
	}{
		{
			name: "Valid Result",
			pid:  "pid-123",
			inputJSON: fmt.Sprintf(`{
				"jobResultObject": {
					"result": [
						{"datasetId": "pid-123", "url": "s3://bucket/file1?X-Amz-Date=%s&X-Amz-Expires=%v"},
						{"datasetId": "pid-123", "url": "s3://bucket/file1?X-Amz-Date=%s&X-Amz-Expires=%v"}
					]
				}
			}`, validTimeIso8601Str, expiresSeconds, validTimeIso8601Str, expiresLater),
			wantErr:       false,
			expectedCount: 2,
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
			got, err := toDatasetsUrlResponse(tt.pid, jobResp)

			if (err != nil) != tt.wantErr {
				t.Errorf("toDatasetsUrlResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(got.Urls) != tt.expectedCount {
					t.Errorf("Expected %d URLs, got %d", tt.expectedCount, len(got.Urls))
				}
				// Verify expiration is 7 days from creation
				expectedExp := now.Add(time.Second * time.Duration(expiresSeconds))
				diff := got.Urls[0].Expires.Sub(expectedExp)
				tolerance := 1 * time.Second
				if diff < -tolerance || diff > tolerance {
					t.Errorf("Expiration date mismatch.\nGot:  %v\nWant: %v\nDiff: %v", got.Urls[0].Expires, expectedExp, diff)
				}

				// verify that earlier expiration is present at the root
				if tt.expectedCount == 2 {
					expectedExp := minTime(got.Urls[0].Expires, got.Urls[1].Expires)
					if !got.Expires.Equal(expectedExp) {
						t.Errorf("Expected earliest expiration at the root of the response\nGot: %v\nWant: %v", got.Expires, expectedExp)
					}
				}
			}
		})
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
