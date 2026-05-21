package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSciCatAuthorizer_Authorize(t *testing.T) {
	gin.SetMode(gin.TestMode)

	pid := "test-pid-123"
	ownerGroup := "svc-afs"

	datasetBody, _ := json.Marshal(map[string]string{"ownerGroup": ownerGroup})
	whoamiWithGroup, _ := json.Marshal(map[string]any{"currentGroups": []string{"unx-nogroup", ownerGroup}})
	whoamiWithoutGroup, _ := json.Marshal(map[string]any{"currentGroups": []string{"unx-nogroup"}})

	tests := []struct {
		name              string
		authHeader        string
		operation         Operation
		mockDatasetStatus int
		mockDatasetBody   []byte
		mockWhoamiStatus  int
		mockWhoamiBody    []byte
		wantErr           bool
	}{
		{
			name:       "read: missing Authorization header",
			authHeader: "",
			operation:  OperationRead,
			wantErr:    true,
		},
		{
			name:       "read: malformed Authorization header",
			authHeader: "Basic dGVzdA==",
			operation:  OperationRead,
			wantErr:    true,
		},
		{
			name:              "read: SciCat returns 200",
			authHeader:        "Bearer valid-token",
			operation:         OperationRead,
			mockDatasetStatus: http.StatusOK,
			mockDatasetBody:   datasetBody,
			wantErr:           false,
		},
		{
			name:              "read: SciCat returns 403",
			authHeader:        "Bearer valid-token",
			operation:         OperationRead,
			mockDatasetStatus: http.StatusForbidden,
			wantErr:           true,
		},
		{
			name:              "write: user in ownerGroup: authorized",
			authHeader:        "Bearer valid-token",
			operation:         OperationWrite,
			mockDatasetStatus: http.StatusOK,
			mockDatasetBody:   datasetBody,
			mockWhoamiStatus:  http.StatusOK,
			mockWhoamiBody:    whoamiWithGroup,
			wantErr:           false,
		},
		{
			name:              "write: user not in ownerGroup: denied",
			authHeader:        "Bearer valid-token",
			operation:         OperationWrite,
			mockDatasetStatus: http.StatusOK,
			mockDatasetBody:   datasetBody,
			mockWhoamiStatus:  http.StatusOK,
			mockWhoamiBody:    whoamiWithoutGroup,
			wantErr:           true,
		},
		{
			name:              "write: dataset not accessible: denied before whoami",
			authHeader:        "Bearer valid-token",
			operation:         OperationWrite,
			mockDatasetStatus: http.StatusForbidden,
			wantErr:           true,
		},
		{
			name:              "write: whoami fails: denied",
			authHeader:        "Bearer valid-token",
			operation:         OperationWrite,
			mockDatasetStatus: http.StatusOK,
			mockDatasetBody:   datasetBody,
			mockWhoamiStatus:  http.StatusUnauthorized,
			wantErr:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var datasetReq, whoamiReq *http.Request

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/api/v3/auth/whoami" {
					whoamiReq = r
					w.WriteHeader(tt.mockWhoamiStatus)
					w.Write(tt.mockWhoamiBody)
				} else if strings.HasPrefix(r.URL.Path, "/api/v3/datasets") {
					datasetReq = r
					w.WriteHeader(tt.mockDatasetStatus)
					w.Write(tt.mockDatasetBody)
				}
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

			if tt.mockDatasetStatus != 0 {
				if datasetReq == nil {
					t.Fatal("expected HTTP call to datasets endpoint, none was made")
				}
				if datasetReq.URL.Path != "/api/v3/datasets/"+pid {
					t.Errorf("dataset path = %q, want %q", datasetReq.URL.Path, "/api/v3/datasets/"+pid)
				}
				if got := datasetReq.URL.Query().Get("filter"); got != `{"fields":["ownerGroup"]}` {
					t.Errorf("filter param = %q, want %q", got, `{"fields":["ownerGroup"]}`)
				}
				if got := datasetReq.Header.Get("Authorization"); got != tt.authHeader {
					t.Errorf("dataset Authorization header = %q, want %q", got, tt.authHeader)
				}
			} else if datasetReq != nil {
				t.Error("expected no HTTP call to datasets endpoint, but one was made")
			}

			if tt.mockWhoamiStatus != 0 {
				if whoamiReq == nil {
					t.Fatal("expected HTTP call to whoami endpoint, none was made")
				}
				if got := whoamiReq.Header.Get("Authorization"); got != tt.authHeader {
					t.Errorf("whoami Authorization header = %q, want %q", got, tt.authHeader)
				}
			} else if whoamiReq != nil {
				t.Error("expected no HTTP call to whoami endpoint, but one was made")
			}
		})
	}
}
