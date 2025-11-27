package handlers

import (
	"net/http"

	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sts"

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
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithSharedCredentialsFiles(
			[]string{"env/credentials"},
		),
		config.WithSharedConfigFiles(
			[]string{"env/config"},
		),
		config.WithSharedConfigProfile("ceph"))
	if err != nil {
		log.Printf("Failed to load AWS config: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Internal server error",
		})
		return
	}

	s3Client := s3.NewFromConfig(cfg)
	out, err := s3Client.ListBuckets(context.TODO(), &s3.ListBucketsInput{})
	if err != nil {
		log.Printf("Failed to list S3 buckets: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Internal server error",
		})
		return
	}
	for _, bucket := range out.Buckets {
		log.Printf("Bucket: %s, Created on: %s", *bucket.Name, bucket.CreationDate)
	}

	stsClient := sts.NewFromConfig(cfg)
	policy := `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Effect": "Allow",
				"Action": "s3:*",
				"Resource": [
					"arn:aws:s3:::datasets/` + dataset + `",
					"arn:aws:s3:::datasets/` + dataset + `/*"
				]
			},
			{
				"Effect": "Allow",
				"Action": "s3:ListBucket",
				"Resource": "arn:aws:s3:::datasets",
				"Condition": {
					"StringLike": {
						"s3:prefix": [
							"` + dataset + `/"
						]
					}
				}
			}
		]
	}`
	stsOut, err := stsClient.AssumeRole(context.TODO(), &sts.AssumeRoleInput{
		RoleArn:         aws.String("arn:aws:iam:::role/PsiLimitedAccessRole"),
		RoleSessionName: aws.String("scicat-session"),
		Policy:          aws.String(policy),
	})
	if err != nil {
		log.Printf("Failed to assume role: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Internal server error",
		})
		return
	}

	response := models.S3CredentialsResponse{
		AccessKey:    *stsOut.Credentials.AccessKeyId,
		SecretKey:    *stsOut.Credentials.SecretAccessKey,
		SessionToken: *stsOut.Credentials.SessionToken,
		ExpiryTime:   *stsOut.Credentials.Expiration,
	}

	c.JSON(http.StatusOK, response)
}
