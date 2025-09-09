package models

import "time"

// S3CredentialsResponse represents the structure of the S3 credentials response
type S3CredentialsResponse struct {
	AccessKey    string    `json:"access_key"`
	SecretKey    string    `json:"secret_key"`
	SessionToken string    `json:"session_token"`
	ExpiryTime   time.Time `json:"expiry_time"`
}
