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
func buildScopedPolicy(dataset string, operation auth.Operation) (string, error) {
	datasetArn := "arn:aws:s3:::datasets/" + dataset

	var objectActions any
	if operation == auth.OperationWrite {
		objectActions = "s3:*"
	} else {
		objectActions = []string{"s3:Get*", "s3:List*"}
	}

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
				Resource: "arn:aws:s3:::datasets",
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
