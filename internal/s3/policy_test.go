package s3

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/paulscherrerinstitute/scicat-s3-broker/internal/auth"
	"github.com/paulscherrerinstitute/scicat-s3-broker/internal/config"
)

func TestBuildScopedPolicy(t *testing.T) {
	dataset := "my-dataset"
	retrieveBucket := "test-retrieve"
	uploadBucket := "test-upload"
	h := Handler{bucketConfig: config.BucketConfig{RetrieveBucket: retrieveBucket, UploadBucket: uploadBucket}}

	tests := []struct {
		name        string
		operation   auth.Operation
		wantActions any
		wantBucket  string
	}{
		{
			name:        "read grants restricted actions",
			operation:   auth.OperationRead,
			wantActions: []string{"s3:Get*", "s3:List*"},
			wantBucket:  retrieveBucket,
		},
		{
			name:        "write grants s3:*",
			operation:   auth.OperationWrite,
			wantActions: "s3:*",
			wantBucket:  uploadBucket,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := h.buildScopedPolicy(dataset, tt.operation)
			if err != nil {
				t.Fatalf("buildScopedPolicy() error = %v", err)
			}

			var doc policyDocument
			if err := json.Unmarshal([]byte(got), &doc); err != nil {
				t.Fatalf("output is not valid JSON: %v", err)
			}

			datasetARN := "arn:aws:s3:::" + tt.wantBucket + "/" + dataset
			stmt, ok := findStatementByResource(doc, datasetARN)
			if !ok {
				t.Fatalf("no statement found with resource %q", datasetARN)
			}

			gotActions, _ := json.Marshal(stmt.Action)
			wantActions, _ := json.Marshal(tt.wantActions)
			if string(gotActions) != string(wantActions) {
				t.Errorf("Action = %s, want %s", gotActions, wantActions)
			}
		})
	}
}

func findStatementByResource(doc policyDocument, arn string) (policyStatement, bool) {
	for _, stmt := range doc.Statement {
		resourcesJSON, _ := json.Marshal(stmt.Resource)
		if strings.Contains(string(resourcesJSON), arn) {
			return stmt, true
		}
	}
	return policyStatement{}, false
}
