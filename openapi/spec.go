package openapi

import (
	_ "embed"
	"net/http"

	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

//go:embed openapi.yaml
var spec []byte

func RegisterSpecRoutes(r *gin.Engine) {
	r.GET("/openapi.yaml", func(c *gin.Context) {
		c.Data(http.StatusOK, "application/yaml", spec)
	})
	swaggerHandler := ginSwagger.WrapHandler(swaggerfiles.Handler, ginSwagger.URL("/openapi.yaml"))
	r.GET("/docs/*any", func(c *gin.Context) {
		if c.Param("any") == "/" {
			c.Redirect(http.StatusMovedPermanently, "/docs/index.html")
			return
		}
		swaggerHandler(c)
	})
}
