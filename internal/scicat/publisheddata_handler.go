package scicat

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/paulscherrerinstitute/scicat-s3-broker/internal/api"
)

type PublisheddataHandler struct {
	service PublisheddataService
}

func (h *PublisheddataHandler) GetPublisheddataUrls(c *gin.Context, params api.GetPublisheddataUrlsParams) {
	result, err := h.service.GetUrls(c.Request.Context(), params.Id)
	if err != nil {
		log.Println(err)
		if pubDataErr, ok := errors.AsType[PublishedDataNotFoundError](err); ok {
			c.JSON(http.StatusNotFound, gin.H{"error": pubDataErr.Error()})
		} else if datasetNotAccErr, ok := errors.AsType[DatasetNotAccessibleError](err); ok {
			c.JSON(http.StatusForbidden, gin.H{"error": datasetNotAccErr.Error()})
		} else if noUrlsErr, ok := errors.AsType[DatasetNotFoundError](err); ok {
			c.JSON(http.StatusNotFound, gin.H{"error": noUrlsErr.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		}
		return
	}
	c.JSON(http.StatusOK, result)
}
