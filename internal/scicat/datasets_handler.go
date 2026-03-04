package scicat

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/paulscherrerinstitute/scicat-s3-broker/internal/api"
	"github.com/paulscherrerinstitute/scicat-s3-broker/internal/config"
)

type DatasetsHandler struct {
	service DatasetsService
}

func NewDatasetsHandler(cfg *config.Config) *DatasetsHandler {
	return &DatasetsHandler{
		service: &DatasetsServiceImpl{config: cfg},
	}
}

func (h *DatasetsHandler) GetDatasetsUrls(c *gin.Context, id api.GetDatasetsUrlsParams) {
	datasetsUrlResp, err := h.service.GetUrls(c.Request.Context(), id.Pid)

	if err != nil {
		var datasetErr DatasetNotAccessibleError
		var noUrlsErr NoUrlsAvailableError
		switch {
		case errors.As(err, &datasetErr):
			log.Println(err)
			c.JSON(http.StatusForbidden, gin.H{"error": datasetErr.Error()})
		case errors.As(err, &noUrlsErr):
			log.Println(err)
			c.JSON(http.StatusNotFound, gin.H{"error": noUrlsErr.Error()})
		default:
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		}
		return
	}
	c.PureJSON(http.StatusOK, datasetsUrlResp)
}
