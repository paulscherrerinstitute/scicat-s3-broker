package scicat

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/paulscherrerinstitute/scicat-s3-broker/internal/api"
)

type mockService struct{}

func (m *mockService) GetUrls(c context.Context, dataset string) (*api.DatasetsUrlResponse, error) {
	switch dataset {
	case "not-found":
		return nil, DatasetNotAccessibleError{dataset}
	case "forbidden":
		return nil, DatasetNotAccessibleError{dataset}
	case "no-urls":
		return nil, NoUrlsAvailableError{dataset}
	case "internal-error":
		return nil, fmt.Errorf("internal error")
	default:
		return &api.DatasetsUrlResponse{
			Urls: []api.UrlInfo{
				{Url: "http://example.com/dataset1"},
				{Url: "http://example.com/dataset2"},
			},
		}, nil
	}
}

func TestDatasetsHandler_GetDatasetsUrls(t *testing.T) {
	handler := &DatasetsHandler{service: &mockService{}}

	tests := []struct {
		name       string
		pid        string
		wantStatus int
	}{
		{
			name:       "Success",
			pid:        "valid-dataset",
			wantStatus: http.StatusOK,
		},
		{
			name:       "Dataset Not Accessible",
			pid:        "not-found",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "Dataset Not Accessible (forbidden)",
			pid:        "forbidden",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "No URLs Available",
			pid:        "no-urls",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "Internal Server Error",
			pid:        "internal-error",
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/datasets/"+tt.pid+"/urls", nil)
			w := httptest.NewRecorder()

			c, _ := gin.CreateTestContext(w)
			c.Request = req

			handler.GetDatasetsUrls(c, api.GetDatasetsUrlsParams{Pid: tt.pid})

			if w.Code != tt.wantStatus {
				t.Errorf("Expected status code %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}
