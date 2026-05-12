package auth

import "github.com/gin-gonic/gin"

type NoOpAuthorizer struct{}

func NewNoOpAuthorizer() *NoOpAuthorizer {
	return &NoOpAuthorizer{}
}

func (*NoOpAuthorizer) Authorize(_ *gin.Context, _ string, _ Operation) error {
	return nil
}
