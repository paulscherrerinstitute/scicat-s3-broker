package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/paulscherrerinstitute/scicat-s3-broker/internal/api"
)

type SciCatCreds = api.S3CredentialsResponse

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
	api := flag.String("api", "http://localhost:8080/datasets/s3-creds", "SciCat S3 creds endpoint")
	operation := flag.String("operation", "read", "operation to request: read or write")
	flag.Parse()

	if *dataset == "" || *token == "" {
		fmt.Fprintln(os.Stderr, "dataset and token required (via --dataset or $SCICAT_TOKEN)")
		os.Exit(1)
	}

	if *operation != "read" && *operation != "write" {
		fmt.Fprintln(os.Stderr, "operation must be 'read' or 'write'")
		os.Exit(1)
	}

	baseURL, err := url.Parse(*api)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	q := baseURL.Query()
	q.Set("pid", *dataset)
	q.Set("operation", *operation)
	baseURL.RawQuery = q.Encode()
	req, err := http.NewRequest("GET", baseURL.String(), nil)
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
		AccessKeyID:     sc.AccessKey,
		SecretAccessKey: sc.SecretAccessKey,
		SessionToken:    sc.SessionToken,
		Expiration:      sc.ExpiryTime.UTC().Format(time.RFC3339),
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(aws); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
