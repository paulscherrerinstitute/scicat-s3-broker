package scicat

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/paulscherrerinstitute/scicat-s3-broker/internal/api"
)

func TestGetDatasetsUrls_ReturnsNotImplemented(t *testing.T) {
	handler := NewSciCatNotImplementedHandler()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	handler.GetDatasetsUrls(c, api.GetDatasetsUrlsParams{})

	if w.Code != http.StatusNotImplemented {
		t.Errorf("expected status code %d, got %d", http.StatusNotImplemented, w.Code)
	}
}
