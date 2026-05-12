package auth

import "github.com/gin-gonic/gin"

type Operation int

const (
	OperationRead Operation = iota
	OperationWrite
)

type Authorizer interface {
	Authorize(c *gin.Context, pid string, operation Operation) error
}
