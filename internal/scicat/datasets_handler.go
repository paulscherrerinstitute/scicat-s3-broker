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

func (h *DatasetsHandler) GetDatasetsUrls(c *gin.Context, pid api.GetDatasetsUrlsParams) {
	h.GetUrls(c, api.GetUrlsParams{Id: pid.Pid})
}

func (h *DatasetsHandler) GetUrls(c *gin.Context, id api.GetUrlsParams) {
	datasetsUrlResp, err := h.service.GetUrls(c.Request.Context(), id.Id)

	if err != nil {
		if notAccErr, ok := errors.AsType[DatasetNotAccessibleError](err); ok {
			log.Println(err)
			c.JSON(http.StatusForbidden, gin.H{"error": notAccErr.Error()})
		} else if notFoundErr, ok := errors.AsType[DatasetNotFoundError](err); ok {
			log.Println(err)
			c.JSON(http.StatusNotFound, gin.H{"error": notFoundErr.Error()})
		} else {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		}
		return
	}
	c.PureJSON(http.StatusOK, datasetsUrlResp)
}
