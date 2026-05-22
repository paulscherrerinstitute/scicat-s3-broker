package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/paulscherrerinstitute/scicat-s3-broker/internal/api"
	"github.com/paulscherrerinstitute/scicat-s3-broker/internal/auth"
	"github.com/paulscherrerinstitute/scicat-s3-broker/internal/config"
	"github.com/paulscherrerinstitute/scicat-s3-broker/internal/s3"
	"github.com/paulscherrerinstitute/scicat-s3-broker/internal/scicat"
	"github.com/paulscherrerinstitute/scicat-s3-broker/openapi"
)

func main() {
	router := gin.Default()
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	openapi.RegisterSpecRoutes(router)

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "healthy",
		})
	})
	opts := api.GinServerOptions{
		ErrorHandler: func(c *gin.Context, err error, statusCode int) {
			c.JSON(statusCode, gin.H{"error": err.Error()})
		},
	}

	type SciCatHandler = scicat.Handler
	type SciCatNotImplHandler = scicat.NotImplHandler
	type S3Handler = s3.Handler

	var authorizer auth.Authorizer
	if cfg.SciCatURL != "" {
		authorizer = auth.NewSciCatAuthorizer(cfg.SciCatURL)
	} else {
		authorizer = auth.NewNoOpAuthorizer()
	}
	s3Handler := s3.NewHandler(authorizer, cfg.BucketConfig)

	if cfg.SciCatURL != "" && cfg.JobManagerPassword != "" {
		var h api.ServerInterface = struct {
			*SciCatHandler
			*S3Handler
		}{
			scicat.NewHandler(cfg),
			s3Handler,
		}
		api.RegisterHandlersWithOptions(router, h, opts)
	} else {
		var h api.ServerInterface = struct {
			*SciCatNotImplHandler
			*S3Handler
		}{
			scicat.NewNoImplHandler(),
			s3Handler,
		}
		api.RegisterHandlersWithOptions(router, h, opts)
	}

	if err := router.Run(); err != nil {
		log.Fatal("Failed to start server: ", err)
	}
}
