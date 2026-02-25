package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/paulscherrerinstitute/scicat-s3-broker/internal/api"
	"github.com/paulscherrerinstitute/scicat-s3-broker/internal/config"
	"github.com/paulscherrerinstitute/scicat-s3-broker/internal/handlers"
)

func main() {
	router := gin.Default()
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	var h api.ServerInterface = handlers.NewSciCatHandler(cfg)

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "healthy",
		})
	})

	router.GET("/get-s3-creds", handlers.GetS3Credentials)

	if cfg.JobManagerPassword != "" {
		api.RegisterHandlers(router, h)
	} else {
		router.GET("/get-urls", func(c *gin.Context) {
			c.JSON(http.StatusNotImplemented, gin.H{
				"error": "This endpoint is disabled",
			})
		})
	}

	if err := router.Run(); err != nil {
		log.Fatal("Failed to start server: ", err)
	}
}
