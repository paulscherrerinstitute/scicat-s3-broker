package scicat

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/paulscherrerinstitute/scicat-s3-broker/internal/api"
)

type NotImplHandler struct{}

func NewNoImplHandler() *NotImplHandler {
	return &NotImplHandler{}
}

func (*NotImplHandler) GetUrls(c *gin.Context, _ api.GetUrlsParams) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "This endpoint is disabled",
	})
}

func (*NotImplHandler) GetDatasetsUrls(c *gin.Context, _ api.GetDatasetsUrlsParams) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "This endpoint is disabled",
	})
}

func (*NotImplHandler) GetPublisheddataUrls(c *gin.Context, _ api.GetPublisheddataUrlsParams) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "This endpoint is disabled",
	})
}
