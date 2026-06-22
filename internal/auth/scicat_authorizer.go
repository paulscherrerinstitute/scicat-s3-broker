package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"

	"github.com/gin-gonic/gin"
)

type SciCatAuthorizer struct {
	scicatURL string
}

func NewSciCatAuthorizer(scicatURL string) *SciCatAuthorizer {
	return &SciCatAuthorizer{scicatURL: scicatURL}
}

func (a *SciCatAuthorizer) Authorize(c *gin.Context, pid string, operation Operation) error {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return fmt.Errorf("Authorization header is required")
	}
	if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
		return fmt.Errorf("invalid authorization header format, expected 'Bearer <token>'")
	}
	token := authHeader[7:]

	// All operations require read
	ownerGroup, err := a.scicatGetDataset(c.Request.Context(), pid, token)
	if err != nil {
		return err
	} else if operation == OperationRead {
		return nil
	}

	if operation == OperationWrite {
		return a.authorizeWrite(c.Request.Context(), ownerGroup, token)
	}

	// No delete allowed
	return fmt.Errorf("unauthorized for operation %v", operation)
}

func (a *SciCatAuthorizer) authorizeWrite(ctx context.Context, ownerGroup string, token string) error {
	groups, err := a.scicatWhoami(ctx, token)
	if err != nil {
		return err
	}

	if !slices.Contains(groups, ownerGroup) {
		return fmt.Errorf("user is not a member of the dataset's owner group %q", ownerGroup)
	}
	return nil
}

// scicatWhoami calls /api/v3/auth/whoami and returns user's currentGroups from the response
func (a *SciCatAuthorizer) scicatWhoami(ctx context.Context, token string) ([]string, error) {
	u := fmt.Sprintf("%s/api/v3/auth/whoami", a.scicatURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create whoami request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to contact SciCat whoami: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("SciCat whoami returned %d", resp.StatusCode)
	}

	var body struct {
		CurrentGroups []string `json:"currentGroups"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("failed to decode whoami response: %w", err)
	}
	return body.CurrentGroups, nil
}

// scicatGetDataset checks that the user's token grants read access to the dataset
// by calling /api/v3/datasets/{pid} and returns the datasets's ownerGroup.
func (a *SciCatAuthorizer) scicatGetDataset(ctx context.Context, pid string, token string) (string, error) {
	u, err := url.Parse(fmt.Sprintf("%s/api/v3/datasets/%s", a.scicatURL, url.PathEscape(pid)))
	if err != nil {
		return "", fmt.Errorf("failed to build dataset URL: %w", err)
	}
	q := u.Query()
	q.Set("filter", `{"fields":["ownerGroup"]}`)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to contact SciCat: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("SciCat returned %d for dataset %q", resp.StatusCode, pid)
	}

	var body struct {
		OwnerGroup string `json:"ownerGroup"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", fmt.Errorf("failed to decode dataset response: %w", err)
	}
	return body.OwnerGroup, nil
}
