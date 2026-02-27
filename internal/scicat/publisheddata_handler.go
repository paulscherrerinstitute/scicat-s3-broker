package scicat

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/paulscherrerinstitute/scicat-s3-broker/internal/api"
)

type SciCatPublishedDataItem struct {
	DatasetPids []string `json:"datasetPids"`
}

func makePublishedDataQuery(doi string) ([]byte, error) {
	filterQuery, err := json.Marshal(gin.H{
		"where": gin.H{
			"_id": doi,
		},
		"fields": []string{"datasetPids"},
	})
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal json: %w", err)
	}
	return filterQuery, nil
}

func (h *Handler) GetPublisheddataUrls(c *gin.Context, params api.GetPublisheddataUrlsParams) {
	filterQuery, _ := makePublishedDataQuery(params.Id)
	u, err := url.Parse(fmt.Sprintf("%s/api/v4/publisheddata", h.config.SciCatURL))
	if err != nil {
		log.Printf("failed to parse publisheddata URL: %v", err)
		c.JSON(http.StatusInternalServerError, "")
		return
	}
	q := u.Query()
	q.Set("filter", string(filterQuery))
	u.RawQuery = q.Encode()

	resp, err := http.Get(u.String())
	if err != nil {
		log.Printf("failed to check dataset public status: %v", err)
		c.JSON(http.StatusInternalServerError, "")
		return
	}
	defer resp.Body.Close()
	var publishedDataResp []SciCatPublishedDataItem
	err = json.NewDecoder(resp.Body).Decode(&publishedDataResp)
	if err != nil {
		log.Printf("failed to decode publisheddata response: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	if len(publishedDataResp) == 0 {
		log.Printf("No datasets associated with published data id %s", params.Id)
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("No datasets associated with published data id %s", params.Id)})
		return
	}
	result := api.PublishedDataUrlsResponse{}
	for _, pid := range publishedDataResp[0].DatasetPids {
		urls, err := h.getDatasetsUrlsObj(c, pid)
		if err != nil {
			switch {
			case errors.Is(err, ErrDatasetNotAccessible):
				log.Printf("Dataset %s is not accessible", pid)
				c.JSON(http.StatusForbidden, gin.H{"error": fmt.Sprintf("Dataset %s is not accessible", pid)})
			case errors.Is(err, ErrNoUrlsAvailable):
				log.Printf("No URLs available for dataset %s", pid)
				c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("No URLs available for dataset %s. Submit a URL retrive job in SciCat", pid)})
			default:
				log.Printf("Failed to get URLs for dataset %s: %v", pid, err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			}
			return
		}
		result[pid] = urls
	}
	c.JSON(http.StatusOK, result)
}
