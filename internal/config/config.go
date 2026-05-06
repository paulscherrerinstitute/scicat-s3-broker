package config

import (
	"os"
	"strings"
)

type Config struct {
	SciCatURL          string
	JobManagerUsername string
	JobManagerPassword string
	S3Bucket           string
}

// Load reads environment variables and validates them.
func Load() (*Config, error) {
	username := os.Getenv("JOB_MANAGER_USERNAME")
	if username == "" {
		username = "jobManager"
	}

	return &Config{
		SciCatURL:          strings.TrimRight(os.Getenv("SCICAT_URL"), "/"),
		JobManagerUsername: username,
		JobManagerPassword: os.Getenv("JOB_MANAGER_PASSWORD"),
		S3Bucket:           os.Getenv("S3_BUCKET"),
	}, nil
}
