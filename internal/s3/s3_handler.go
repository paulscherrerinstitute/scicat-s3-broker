package s3

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/gin-gonic/gin"
	"github.com/paulscherrerinstitute/scicat-s3-broker/internal/api"
	appconfig "github.com/paulscherrerinstitute/scicat-s3-broker/internal/config"
)

type Handler struct {
	cfg *appconfig.Config
}

func NewHandler(cfg *appconfig.Config) *Handler {
	return &Handler{cfg: cfg}
}

// GetDatasetsS3Creds handles the /datasets/s3-creds endpoint
func (h *Handler) GetDatasetsS3Creds(c *gin.Context, params api.GetDatasetsS3CredsParams) {
	dataset := params.Pid

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

	cfg, err := awsconfig.LoadDefaultConfig(context.TODO(),
		awsconfig.WithSharedCredentialsFiles(
			[]string{"env/credentials"},
		),
		awsconfig.WithSharedConfigFiles(
			[]string{"env/config"},
		),
		awsconfig.WithSharedConfigProfile("ceph"))
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

	response := api.S3CredentialsResponse{
		AccessKey:       *stsOut.Credentials.AccessKeyId,
		SecretAccessKey: *stsOut.Credentials.SecretAccessKey,
		SessionToken:    *stsOut.Credentials.SessionToken,
		ExpiryTime:      *stsOut.Credentials.Expiration,
	}

	c.JSON(http.StatusOK, response)
}

type expiryTier struct {
	duration time.Duration
	prefix   string
}

var expiryTiers = map[string]expiryTier{
	"1d": {24 * time.Hour, "d1"},
	"3d": {72 * time.Hour, "d3"},
	"7d": {168 * time.Hour, "d7"},
}

// CreateUploadSession generates presigned PUT and GET URLs for a new random object path.
// Optional query param: expiry=1d|3d|7d (default: 1d)
func (h *Handler) CreateUploadSession(c *gin.Context) {
	filename := c.Query("filename")
	if filename == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "filename query parameter is required"})
		return
	}
	if h.cfg.S3Bucket == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "S3_BUCKET is not configured"})
		return
	}

	expiryKey := c.DefaultQuery("expiry", "1d")
	tier, ok := expiryTiers[expiryKey]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "expiry must be one of: 1d, 3d, 7d"})
		return
	}
	duration := tier.duration

	cfg, err := awsconfig.LoadDefaultConfig(context.TODO(),
		awsconfig.WithSharedCredentialsFiles([]string{"env/credentials"}),
		awsconfig.WithSharedConfigFiles([]string{"env/config"}),
		awsconfig.WithSharedConfigProfile("ceph"))
	if err != nil {
		log.Printf("Failed to load AWS config: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	objectKey := tier.prefix + "/" + randomID() + "/" + filename

	presigner := s3.NewPresignClient(s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	}))

	putReq, err := presigner.PresignPutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(h.cfg.S3Bucket),
		Key:    aws.String(objectKey),
	}, s3.WithPresignExpires(duration))
	if err != nil {
		log.Printf("Failed to presign PUT: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	getReq, err := presigner.PresignGetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(h.cfg.S3Bucket),
		Key:    aws.String(objectKey),
	}, s3.WithPresignExpires(duration))
	if err != nil {
		log.Printf("Failed to presign GET: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.PureJSON(http.StatusOK, gin.H{
		"upload_url":   putReq.URL,
		"download_url": getReq.URL,
		"path":         objectKey,
		"expires":      time.Now().Add(duration),
	})
}

func randomID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
