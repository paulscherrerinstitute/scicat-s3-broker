package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type SciCatCreds struct {
	AccessKeyID     string    `json:"access_key"`
	SecretAccessKey string    `json:"secret_key"`
	SessionToken    string    `json:"session_token"`
	Expiry          time.Time `json:"expiry_time"`
}

type AWSCreds struct {
	Version         int    `json:"Version"`
	AccessKeyID     string `json:"AccessKeyId"`
	SecretAccessKey string `json:"SecretAccessKey"`
	SessionToken    string `json:"SessionToken"`
	Expiration      string `json:"Expiration"`
}

func main() {
	dataset := flag.String("dataset", "", "Dataset PID or ID")
	token := flag.String("token", os.Getenv("SCICAT_TOKEN"), "SciCat access token")
	api := flag.String("api", "http://localhost:8085/get-s3-creds", "SciCat S3 creds endpoint")
	flag.Parse()

	if *dataset == "" || *token == "" {
		fmt.Fprintln(os.Stderr, "dataset and token required (via --dataset or $SCICAT_TOKEN)")
		os.Exit(1)
	}

	// Prepare request
	req, err := http.NewRequest("GET", *api+"?dataset="+*dataset, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	req.Header.Set("Authorization", "Bearer "+*token)

	// Call API
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		fmt.Fprintf(os.Stderr, "error from SciCat: %s\n%s\n", resp.Status, body)
		os.Exit(1)
	}

	var sc SciCatCreds
	if err := json.NewDecoder(resp.Body).Decode(&sc); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Convert to AWS credential_process format
	aws := AWSCreds{
		Version:         1,
		AccessKeyID:     sc.AccessKeyID,
		SecretAccessKey: sc.SecretAccessKey,
		SessionToken:    sc.SessionToken,
		Expiration:      sc.Expiry.UTC().Format(time.RFC3339),
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(aws); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
