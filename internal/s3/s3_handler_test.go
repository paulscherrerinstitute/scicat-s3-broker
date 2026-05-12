package s3

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/paulscherrerinstitute/scicat-s3-broker/internal/api"
	"github.com/paulscherrerinstitute/scicat-s3-broker/internal/auth"
)

type mockAuthorizer struct{ err error }

func (m *mockAuthorizer) Authorize(_ *gin.Context, _ string, _ auth.Operation) error {
	return m.err
}

func TestS3Handler_GetDatasetsS3Creds(t *testing.T) {
	gin.SetMode(gin.TestMode)

	invalidOp := api.GetDatasetsS3CredsParamsOperation("delete")

	tests := []struct {
		name       string
		params     api.GetDatasetsS3CredsParams
		authorizer auth.Authorizer
		wantStatus int
	}{
		{
			name:       "invalid operation returns 400",
			params:     api.GetDatasetsS3CredsParams{Pid: "some-dataset", Operation: &invalidOp},
			authorizer: &mockAuthorizer{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "authorizer error returns 403",
			params:     api.GetDatasetsS3CredsParams{Pid: "some-dataset"},
			authorizer: &mockAuthorizer{err: fmt.Errorf("internal auth error")},
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHandler(tt.authorizer)

			req := httptest.NewRequest(http.MethodGet, "/datasets/s3-creds", nil)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			handler.GetDatasetsS3Creds(c, tt.params)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}
