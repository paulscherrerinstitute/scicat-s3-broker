package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/paulscherrerinstitute/scicat-s3-broker/internal/config"
	"github.com/paulscherrerinstitute/scicat-s3-broker/internal/handlers"
)

func main() {
	router := gin.Default()
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	h := handlers.NewSciCatHandler(cfg)

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "healthy",
		})
	})

	router.GET("/get-s3-creds", handlers.GetS3Credentials)

	router.GET("/get-urls", h.GetActiveUrls)

	log.Println("Starting SciCat S3 Broker server on port 8085...")
	if err := router.Run(":8085"); err != nil {
		log.Fatal("Failed to start server: ", err)
	}
}
