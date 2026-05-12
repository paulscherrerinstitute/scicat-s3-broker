package s3

import (
	"fmt"
	"log"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/gin-gonic/gin"
	"github.com/paulscherrerinstitute/scicat-s3-broker/internal/api"
	"github.com/paulscherrerinstitute/scicat-s3-broker/internal/auth"
)

type Handler struct {
	authorizer auth.Authorizer
}

func NewHandler(authorizer auth.Authorizer) *Handler {
	return &Handler{authorizer: authorizer}
}

// GetDatasetsS3Creds handles the /datasets/s3-creds endpoint
func (h *Handler) GetDatasetsS3Creds(c *gin.Context, params api.GetDatasetsS3CredsParams) {
	dataset := params.Pid

	operation, err := parseOperation(params.Operation)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.authorizer.Authorize(c, dataset, operation); err != nil {
		log.Println("authorization failed", err)
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
		return
	}

	cfg, err := config.LoadDefaultConfig(c.Request.Context(),
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

	policy, err := buildScopedPolicy(dataset, operation)
	if err != nil {
		log.Printf("Failed to build policy: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	stsClient := sts.NewFromConfig(cfg)
	stsOut, err := stsClient.AssumeRole(c.Request.Context(), &sts.AssumeRoleInput{
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

func parseOperation(op *api.GetDatasetsS3CredsParamsOperation) (auth.Operation, error) {
	if op == nil {
		return auth.OperationRead, nil
	}
	switch *op {
	case api.Read:
		return auth.OperationRead, nil
	case api.Write:
		return auth.OperationWrite, nil
	default:
		return 0, fmt.Errorf("invalid operation %q, expected read or write", *op)
	}
}
