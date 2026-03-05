package scicat

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/paulscherrerinstitute/scicat-s3-broker/internal/api"
)

type mockPublisheddataSvc struct{}

func (m *mockPublisheddataSvc) GetUrls(ctx context.Context, doi string) (api.PublishedDataUrlsResponse, error) {
	switch doi {
	case "not-found":
		return nil, PublishedDataNotFoundError{Id: doi}
	case "forbidden":
		return nil, DatasetNotAccessibleError{doi}
	case "no-urls":
		return nil, NoUrlsAvailableError{doi}
	case "internal-error":
		return nil, errors.New("internal error")
	default:
		return api.PublishedDataUrlsResponse{"pid123": api.DatasetsUrlResponse{{Url: "http://example.com/publisheddata1"}}}, nil
	}
}

func TestGetPublisheddataUrls(t *testing.T) {
	tests := []struct {
		name       string
		doi        string
		wantStatus int
	}{
		{
			name:       "Success",
			doi:        "test-doi",
			wantStatus: http.StatusOK,
		},
		{
			name:       "PublishedDataNotFoundError",
			doi:        "not-found",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "DatasetNotAccessibleError",
			doi:        "forbidden",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "NoUrlsAvailableError",
			doi:        "no-urls",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "Internal Server Error",
			doi:        "internal-error",
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &PublisheddataHandler{service: &mockPublisheddataSvc{}}
			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			req, _ := http.NewRequest("GET", "", nil)
			c.Request = req
			h.GetPublisheddataUrls(c, api.GetPublisheddataUrlsParams{Id: tt.doi})

			if w.Code != tt.wantStatus {
				t.Errorf("Expected status code %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}
