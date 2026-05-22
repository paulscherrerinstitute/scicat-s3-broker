package config

import (
	"os"
	"strings"
)

type BucketConfig struct {
	RetrieveBucket string
	UploadBucket   string
}

type Config struct {
	SciCatURL          string
	JobManagerUsername string
	JobManagerPassword string
	BucketConfig       BucketConfig
}

// Load reads environment variables and validates them.
func Load() (*Config, error) {
	scicatURL := os.Getenv("SCICAT_URL")

	password := os.Getenv("JOB_MANAGER_PASSWORD")

	username := os.Getenv("JOB_MANAGER_USERNAME")
	if username == "" {
		username = "jobManager"
	}

	retrieveBucket := os.Getenv("RETRIEVE_BUCKET")
	if retrieveBucket == "" {
		retrieveBucket = "datasets"
	}

	uploadBucket := os.Getenv("UPLOAD_BUCKET")
	if uploadBucket == "" {
		uploadBucket = "datasets"
	}

	return &Config{
		SciCatURL:          strings.TrimRight(scicatURL, "/"),
		JobManagerUsername: username,
		JobManagerPassword: password,
		BucketConfig:       BucketConfig{RetrieveBucket: retrieveBucket, UploadBucket: uploadBucket},
	}, nil
}
