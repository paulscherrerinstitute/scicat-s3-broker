package config

import (
	"os"
	"strings"
)

type Config struct {
	SciCatURL          string
	JobManagerUsername string
	JobManagerPassword string
}

// Load reads environment variables and validates them.
func Load() (*Config, error) {
	scicatURL := os.Getenv("SCICAT_URL")

	password := os.Getenv("JOB_MANAGER_PASSWORD")

	username := os.Getenv("JOB_MANAGER_USERNAME")
	if username == "" {
		username = "jobManager"
	}

	return &Config{
		SciCatURL:          strings.TrimRight(scicatURL, "/"),
		JobManagerUsername: username,
		JobManagerPassword: password,
	}, nil
}
