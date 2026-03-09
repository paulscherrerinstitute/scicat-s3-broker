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
	"golang.org/x/sync/errgroup"
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
	type concurrentResult struct {
		pid            string
		datasetUrlResp api.DatasetsUrlResponse
	}
	resultSlice := make([]concurrentResult, len(publishedDataResp[0].DatasetPids))
	g, ctx := errgroup.WithContext(ctx)
	for i, pid := range publishedDataResp[0].DatasetPids {
		g.Go(func() error {
			urls, err := s.datasetsService.GetUrls(ctx, pid)
			if err == nil {
				resultSlice[i] = concurrentResult{pid, urls}
				return nil
			}
			return fmt.Errorf("failed to get URLs for dataset %s: %w", pid, err)
		})
	}
	if err = g.Wait(); err != nil {
		return nil, fmt.Errorf("error from a goroutine executing datasetsService.GetUrls: %w", err)
	}
	result := make(api.PublishedDataUrlsResponse)
	for _, r := range resultSlice {
		result[r.pid] = r.datasetUrlResp
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
