package scicat

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/paulscherrerinstitute/scicat-s3-broker/internal/api"
	"github.com/paulscherrerinstitute/scicat-s3-broker/internal/config"
)

type PublisheddataService interface {
	GetUrls(ctx context.Context, doi string) (api.PublishedDataUrlsResponse, error)
}

type PublisheddataServiceImpl struct {
	config          *config.Config
	datasetsService DatasetsService
}

type SciCatPublishedDataItem struct {
	DatasetPids []string `json:"datasetPids"`
}

func (s *PublisheddataServiceImpl) GetUrls(ctx context.Context, doi string) (api.PublishedDataUrlsResponse, error) {
	filterQuery, _ := makePublishedDataQuery(doi)
	u, err := url.Parse(fmt.Sprintf("%s/api/v4/publisheddata", s.config.SciCatURL))
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
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get publisheddata: %v", resp.Status)
	}
	var publishedDataResp []SciCatPublishedDataItem
	if err := json.NewDecoder(resp.Body).Decode(&publishedDataResp); err != nil {
		return nil, fmt.Errorf("failed to decode publisheddata response: %w", err)
	}
	if len(publishedDataResp) == 0 {
		return nil, PublishedDataNotFoundError{Id: doi}
	}
	result := make(api.PublishedDataUrlsResponse)
	for _, pid := range publishedDataResp[0].DatasetPids {
		urls, err := s.datasetsService.GetUrls(ctx, pid)
		if err != nil {
			return nil, fmt.Errorf("failed to get URLs for dataset %s: %w", pid, err)
		}
		result[pid] = urls
	}
	return result, nil
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
