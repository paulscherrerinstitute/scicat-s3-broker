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
		var datasetNotAccErr DatasetNotAccessibleError
		var noUrlsErr DatasetNotFoundError
		var pubDataErr PublishedDataNotFoundError
		switch {
		case errors.As(err, &pubDataErr):
			c.JSON(http.StatusNotFound, gin.H{"error": pubDataErr.Error()})
		case errors.As(err, &datasetNotAccErr):
			c.JSON(http.StatusForbidden, gin.H{"error": datasetNotAccErr.Error()})
		case errors.As(err, &noUrlsErr):
			c.JSON(http.StatusNotFound, gin.H{"error": noUrlsErr.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		}
		return
	}
	c.JSON(http.StatusOK, result)
}
