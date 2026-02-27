package scicat

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/paulscherrerinstitute/scicat-s3-broker/internal/api"
)

func TestGetPublisheddataUrls(t *testing.T) {
	h := &Handler{service: &MockService{}}
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest("GET", "", nil)
	c.Request = req
	h.GetPublisheddataUrls(c, api.GetPublisheddataUrlsParams{Id: "test-doi"})
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}
}

type MockService struct {
}

func (m *MockService) getPublishedDataUrls(ctx context.Context, doi string) (api.PublishedDataUrlsResponse, error) {
	if doi == "no-such-doi" {
		return nil, &DatasetNotAccessibleError{doi}
	}
	return api.PublishedDataUrlsResponse{}, nil
}

func (m *MockService) getDatasetsUrlsObj(c context.Context, dataset string) (api.DatasetsUrlResponse, error) {
	return api.DatasetsUrlResponse{}, nil
}
