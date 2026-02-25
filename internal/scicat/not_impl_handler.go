package scicat

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/paulscherrerinstitute/scicat-s3-broker/internal/api"
)

type SciCatNotImplHandler struct{}

func NewNoImplHandler() *SciCatNotImplHandler {
	return &SciCatNotImplHandler{}
}
func (*SciCatNotImplHandler) GetDatasetsUrls(c *gin.Context, _ api.GetDatasetsUrlsParams) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "This endpoint is disabled",
	})
}
