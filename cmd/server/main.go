package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/paulscherrerinstitute/scicat-s3-broker/internal/api"
	"github.com/paulscherrerinstitute/scicat-s3-broker/internal/config"
	"github.com/paulscherrerinstitute/scicat-s3-broker/internal/s3"
	"github.com/paulscherrerinstitute/scicat-s3-broker/internal/scicat"
)

func main() {
	router := gin.Default()
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	s3Handler := s3.NewHandler()

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "healthy",
		})
	})
	type SciCatHandler = scicat.Handler
	type SciCatNotImplHandler = scicat.NotImplHandler
	type S3Handler = s3.Handler

	if cfg.JobManagerPassword != "" {
		var h api.ServerInterface = struct {
			*SciCatHandler
			*S3Handler
		}{
			scicat.NewHandler(cfg),
			s3Handler,
		}
		api.RegisterHandlers(router, h)
	} else {
		var h api.ServerInterface = struct {
			*SciCatNotImplHandler
			*S3Handler
		}{
			scicat.NewNoImplHandler(),
			s3Handler,
		}
		api.RegisterHandlers(router, h)
	}

	if err := router.Run(); err != nil {
		log.Fatal("Failed to start server: ", err)
	}
}
