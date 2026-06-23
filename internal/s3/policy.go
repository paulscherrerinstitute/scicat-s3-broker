package s3

import (
	"encoding/json"
	"fmt"

	"github.com/paulscherrerinstitute/scicat-s3-broker/internal/auth"
)

type policyDocument struct {
	Version   string            `json:"Version"`
	Statement []policyStatement `json:"Statement"`
}

type policyStatement struct {
	Effect    string           `json:"Effect"`
	Action    any              `json:"Action"`
	Resource  any              `json:"Resource"`
	Condition *policyCondition `json:"Condition,omitempty"`
}

type policyCondition struct {
	StringLike map[string][]string `json:"StringLike,omitempty"`
}

// buildScopedPolicy returns an IAM policy JSON string that restricts access to
// the given dataset and operation
func (h *Handler) buildScopedPolicy(dataset string, operation auth.Operation) (string, error) {
	var objectActions any
	var bucket string
	switch operation {
	case auth.OperationWrite:
		objectActions = "s3:*"
		bucket = h.bucketConfig.UploadBucket
	case auth.OperationRead:
		objectActions = []string{"s3:Get*", "s3:List*"}
		bucket = h.bucketConfig.RetrieveBucket
	default:
		return "", fmt.Errorf("invalid operation %v", operation)
	}
	if bucket == "" {
		return "", fmt.Errorf("bucket is not configured")
	}

	datasetArn := "arn:aws:s3:::" + bucket + "/" + dataset

	doc := policyDocument{
		Version: "2012-10-17",
		Statement: []policyStatement{
			{
				Effect:   "Allow",
				Action:   objectActions,
				Resource: []string{datasetArn, datasetArn + "/*"},
			},
			{
				Effect:   "Allow",
				Action:   "s3:ListBucket",
				Resource: "arn:aws:s3:::" + bucket,
				Condition: &policyCondition{
					StringLike: map[string][]string{
						"s3:prefix": {dataset + "/"},
					},
				},
			},
		},
	}

	b, err := json.Marshal(doc)
	if err != nil {
		return "", fmt.Errorf("error marshalling policy document: %w", err)
	}
	return string(b), nil
}
