package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSciCatAuthorizer_Authorize(t *testing.T) {
	gin.SetMode(gin.TestMode)

	pid := "test-pid-123"

	tests := []struct {
		name           string
		authHeader     string
		operation      Operation
		mockStatusCode int
		wantErr        bool
		wantHTTPCall   bool
	}{
		{
			name:         "temporary/to-do: not implemented Operation write",
			authHeader:   "Bearer valid-token",
			operation:    OperationWrite,
			wantErr:      true,
			wantHTTPCall: false,
		},
		{
			name:         "missing Authorization header",
			authHeader:   "",
			operation:    OperationRead,
			wantErr:      true,
			wantHTTPCall: false,
		},
		{
			name:         "malformed Authorization header",
			authHeader:   "Basic dGVzdA==",
			operation:    OperationRead,
			wantErr:      true,
			wantHTTPCall: false,
		},
		{
			name:           "SciCat returns 200",
			authHeader:     "Bearer valid-token",
			operation:      OperationRead,
			mockStatusCode: http.StatusOK,
			wantErr:        false,
			wantHTTPCall:   true,
		},
		{
			name:           "SciCat returns non-200",
			authHeader:     "Bearer valid-token",
			operation:      OperationRead,
			mockStatusCode: http.StatusForbidden,
			wantErr:        true,
			wantHTTPCall:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedReq *http.Request
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedReq = r
				w.WriteHeader(tt.mockStatusCode)
			}))
			defer server.Close()

			authorizer := NewSciCatAuthorizer(server.URL)

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			err := authorizer.Authorize(c, pid, tt.operation)

			if (err != nil) != tt.wantErr {
				t.Errorf("Authorize() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantHTTPCall {
				if capturedReq == nil {
					t.Fatal("expected HTTP call to SciCat, none was made")
				}
				if capturedReq.URL.Path != "/api/v3/datasets/"+pid {
					t.Errorf("request path = %q, want %q", capturedReq.URL.Path, "/api/v3/datasets/"+pid)
				}
				if got := capturedReq.URL.Query().Get("filter"); got != `{"fields":["_id"]}` {
					t.Errorf("filter param = %q, want %q", got, `{"fields":["_id"]}`)
				}
				if got := capturedReq.Header.Get("Authorization"); got != tt.authHeader {
					t.Errorf("Authorization header = %q, want %q", got, tt.authHeader)
				}
			} else if capturedReq != nil {
				t.Error("expected no HTTP call to SciCat, but one was made")
			}
		})
	}
}
