package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/paulscherrerinstitute/scicat-s3-broker/internal/api"
	"github.com/paulscherrerinstitute/scicat-s3-broker/internal/config"
	"github.com/paulscherrerinstitute/scicat-s3-broker/internal/handlers"
	"github.com/paulscherrerinstitute/scicat-s3-broker/internal/scicat"
)

func main() {
	router := gin.Default()
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	s3Handler := handlers.NewS3Handler()

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "healthy",
		})
	})

	if cfg.JobManagerPassword != "" {
		var h api.ServerInterface = struct {
			*scicat.SciCatHandler
			*handlers.S3Handler
		}{
			scicat.NewSciCatHandler(cfg),
			s3Handler,
		}
		api.RegisterHandlers(router, h)
	} else {
		var h api.ServerInterface = struct {
			*scicat.SciCatNotImplHandler
			*handlers.S3Handler
		}{
			scicat.NewSciCatNotImplementedHandler(),
			s3Handler,
		}

		api.RegisterHandlers(router, h)
	}

	if err := router.Run(); err != nil {
		log.Fatal("Failed to start server: ", err)
	}
}
