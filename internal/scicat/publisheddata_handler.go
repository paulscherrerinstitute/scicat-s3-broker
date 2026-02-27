package scicat

import (
	"context"
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
	result, err := h.getPublishedDataUrls(c.Request.Context(), params.Id)
	if err != nil {
		log.Println(err)
		var datasetNotAccErr DatasetNotAccessibleError
		var noUrlsErr NoUrlsAvailableError
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

func (h *Handler) getPublishedDataUrls(ctx context.Context, doi string) (api.PublishedDataUrlsResponse, error) {
	filterQuery, _ := makePublishedDataQuery(doi)
	u, err := url.Parse(fmt.Sprintf("%s/api/v4/publisheddata", h.config.SciCatURL))
	if err != nil {
		return nil, fmt.Errorf("failed to parse publisheddata URL: %w", err)
	}
	q := u.Query()
	q.Set("filter", string(filterQuery))
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for publisheddata: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch publisheddata: %w", err)
	}
	defer resp.Body.Close()
	var publishedDataResp []SciCatPublishedDataItem
	if err := json.NewDecoder(resp.Body).Decode(&publishedDataResp); err != nil {
		return nil, fmt.Errorf("failed to decode publisheddata response: %w", err)
	}
	if len(publishedDataResp) == 0 {
		return nil, PublishedDataNotFoundError{Id: doi}
	}
	result := make(api.PublishedDataUrlsResponse)
	for _, pid := range publishedDataResp[0].DatasetPids {
		urls, err := h.getDatasetsUrlsObj(ctx, pid)
		if err != nil {
			return nil, fmt.Errorf("failed to get URLs for dataset %s: %w", pid, err)
		}
		result[pid] = urls
	}
	return result, nil
}
