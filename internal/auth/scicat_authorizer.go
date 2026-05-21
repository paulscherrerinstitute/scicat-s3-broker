package auth

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
)

type SciCatAuthorizer struct {
	scicatURL string
}

func NewSciCatAuthorizer(scicatURL string) *SciCatAuthorizer {
	return &SciCatAuthorizer{scicatURL: scicatURL}
}

func (a *SciCatAuthorizer) Authorize(c *gin.Context, pid string, operation Operation) error {
	if operation == OperationWrite {
		return fmt.Errorf("write operation not implemented yet")
	}

	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return fmt.Errorf("Authorization header is required")
	}
	if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
		return fmt.Errorf("invalid authorization header format, expected 'Bearer <token>'")
	}
	token := authHeader[7:]

	return a.scicatGetDataset(c.Request.Context(), pid, token)
}

func (a *SciCatAuthorizer) scicatGetDataset(ctx context.Context, pid string, token string) error {
	u, err := url.Parse(fmt.Sprintf("%s/api/v3/datasets/%s", a.scicatURL, url.PathEscape(pid)))
	if err != nil {
		return fmt.Errorf("failed to build dataset URL: %w", err)
	}
	q := u.Query()
	q.Set("filter", `{"fields":["_id"]}`)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to contact SciCat: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("SciCat returned %d for dataset %q", resp.StatusCode, pid)
	}
	return nil
}
