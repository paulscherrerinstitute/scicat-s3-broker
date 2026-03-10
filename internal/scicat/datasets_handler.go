package scicat

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/paulscherrerinstitute/scicat-s3-broker/internal/api"
)

type DatasetsHandler struct {
	service DatasetsService
}

func (h *DatasetsHandler) GetDatasetsUrls(c *gin.Context, id api.GetDatasetsUrlsParams) {
	datasetsUrlResp, err := h.service.GetUrls(c.Request.Context(), id.Pid)

	if err != nil {
		var notAccErr DatasetNotAccessibleError
		var notFoundErr DatasetNotFoundError
		switch {
		case errors.As(err, &notAccErr):
			log.Println(err)
			c.JSON(http.StatusForbidden, gin.H{"error": notAccErr.Error()})
		case errors.As(err, &notFoundErr):
			log.Println(err)
			c.JSON(http.StatusNotFound, gin.H{"error": notFoundErr.Error()})
		default:
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		}
		return
	}
	c.PureJSON(http.StatusOK, datasetsUrlResp)
}
