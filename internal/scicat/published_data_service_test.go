package scicat

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/paulscherrerinstitute/scicat-s3-broker/internal/api"
	"github.com/paulscherrerinstitute/scicat-s3-broker/internal/config"
)

type mockDatasetsServiceImpl struct{}

var errMockDatasetsInternal = errors.New("internal error")
var timeA = time.Now()
var timeB = time.Now().AddDate(0, 0, 1)

func (m *mockDatasetsServiceImpl) GetUrls(c context.Context, dataset string) (*api.DatasetsUrlResponse, error) {
	switch dataset {
	case "pid1":
		return &api.DatasetsUrlResponse{Expires: timeA, Urls: []api.UrlInfo{{Url: "http://example.com/pid1"}}}, nil
	case "pid2":
		return &api.DatasetsUrlResponse{Expires: timeB, Urls: []api.UrlInfo{{Url: "http://example.com/pid2"}}}, nil
	case "pid-no-urls":
		return nil, NoUrlsAvailableError{Pid: dataset}
	default:
		return nil, errMockDatasetsInternal
	}
}

func TestPublisheddataServiceGetUrls(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		wantErr        bool
		wantErrIs      error
		wantResult     api.PublishedDataUrlsResponse
	}{
		{
			name: "Success with data",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode([]SciCatPublishedDataItem{
					{DatasetPids: []string{"pid1", "pid2"}},
				})
			},
			wantErr: false,
			wantResult: api.PublishedDataUrlsResponse{
				Expires: timeA,
				Urls: map[string]api.DatasetsUrlResponse{
					"pid1": {Expires: timeA, Urls: []api.UrlInfo{{Url: "http://example.com/pid1"}}},
					"pid2": {Expires: timeB, Urls: []api.UrlInfo{{Url: "http://example.com/pid2"}}},
				},
			},
		},
		{
			name: "Non OK status code from scicat gets publisheddata",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			wantErr: true,
		},
		{
			name: "Not Found (empty array)",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode([]SciCatPublishedDataItem{})
			},
			wantErr:   true,
			wantErrIs: PublishedDataNotFoundError{Id: "test-doi"},
		},
		{
			name: "Invalid JSON response",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("invalid json"))
			},
			wantErr: true,
		},
		{
			name: "Datasets service returns NoUrlsAvailableError",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode([]SciCatPublishedDataItem{
					{DatasetPids: []string{"pid-no-urls"}},
				})
			},
			wantErr:   true,
			wantErrIs: NoUrlsAvailableError{"pid-no-urls"},
		},
		{
			name: "Mix of success and error responses from datasets service",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode([]SciCatPublishedDataItem{
					{DatasetPids: []string{"error-pid", "pid1"}},
				})
			},
			wantErr:   true,
			wantErrIs: errMockDatasetsInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			svc := PublisheddataServiceImpl{
				config:          &config.Config{SciCatURL: server.URL},
				datasetsService: &mockDatasetsServiceImpl{},
			}

			result, err := svc.GetUrls(context.Background(), "test-doi")

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

			if !tt.wantErr && !cmp.Equal(*result, tt.wantResult) {
				t.Errorf("GetUrls() mismatch\ngot:  %+v\nwant: %+v\nDiff: %v", *result, tt.wantResult, cmp.Diff(*result, tt.wantResult))
			}
		})
	}
}
