package scicat

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/paulscherrerinstitute/scicat-s3-broker/internal/api"
	"github.com/paulscherrerinstitute/scicat-s3-broker/internal/config"
)

type mockDatasetsServiceImpl struct {
	err error
	val map[string]api.DatasetsUrlResponse
}

func (m *mockDatasetsServiceImpl) GetUrls(c context.Context, dataset string) (api.DatasetsUrlResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.val[dataset], nil
}

func TestPublisheddataServiceGetUrls(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		mockVal        map[string]api.DatasetsUrlResponse
		mockErr        error
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
			mockVal: map[string]api.DatasetsUrlResponse{
				"pid1": {{Url: "http://example.com/pid1"}},
				"pid2": {{Url: "http://example.com/pid2"}},
			},
			wantErr: false,
			wantResult: map[string]api.DatasetsUrlResponse{
				"pid1": {{Url: "http://example.com/pid1"}},
				"pid2": {{Url: "http://example.com/pid2"}},
			},
		},
		{
			name: "404 Not Found",
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
					{DatasetPids: []string{"pid1"}},
				})
			},
			mockErr:   NoUrlsAvailableError{"pid1"},
			wantErr:   true,
			wantErrIs: NoUrlsAvailableError{"pid1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			svc := PublisheddataServiceImpl{
				config: &config.Config{SciCatURL: server.URL},
				datasetsService: &mockDatasetsServiceImpl{
					err: tt.mockErr,
					val: tt.mockVal,
				},
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

			if !tt.wantErr && !cmp.Equal(result, tt.wantResult) {
				t.Errorf("GetUrls() mismatch\ngot:  %+v\nwant: %+v", result, tt.wantResult)
			}
		})
	}
}
