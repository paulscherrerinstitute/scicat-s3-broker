package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/paulscherrerinstitute/scicat-s3-broker/internal/models"
)

// GetS3Credentials handles the /get-s3-creds endpoint
func GetS3Credentials(c *gin.Context) {
	// Get the dataset parameter from query string
	dataset := c.Query("dataset")
	if dataset == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "dataset parameter is required",
		})
		return
	}

	// Get the Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Authorization header is required",
		})
		return
	}

	// TODO: In a real implementation, validate the SciCat token here
	// For now, we'll just check if it starts with "Bearer "
	if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid authorization header format. Expected 'Bearer <token>'",
		})
		return
	}

	// Return dummy S3 credentials, TODO: Replace with real logic to fetch credentials
	response := models.S3CredentialsResponse{
		AccessKey:    "ASIA...",
		SecretKey:    "wJalrXUtnFEMI/K7MDENG/bPxRfiCY...",
		SessionToken: "FQoGZXIvYXdzE...",
		ExpiryTime:   time.Now().Add(time.Hour), // Expires in 1 hour
	}

	c.JSON(http.StatusOK, response)
}
