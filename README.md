# SciCat S3 Broker

A lightweight service that brokers short-term S3 credentials for SciCat datasets.  
It delegates authorization to SciCat, then issues temporary, scoped S3 credentials 
(e.g. via Ceph STS) that clients can consume through the AWS SDK/CLI 
using the [`credential_process`](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-sourcing-external.html) mechanism.

Also included is a simple CLI client that can be used as an AWS CLI credential process. This might be migrated to its own repo in the future.

---

## Features

- 🔑 **Credential broker**: returns temporary S3 credentials for a given dataset.
- 🛡 **Authorization via SciCat**: forwards the end-user’s SciCat token for access checks.

---

## Quickstart

### Prerequisites
- Go 1.21+
- SciCat running (for authorization calls)
- Ceph or AWS-compatible S3 backend with STS enabled

### Run locally
#### Server

```bash
git clone https://github.com/paulscherrerinstitute/scicat-s3-broker.git
cd scicat-s3-broker
go run ./cmd/server
```

The server will start on port `8085`.

##### Example request

```bash
curl -H "Authorization: Bearer <scicat-token>" \
  "http://localhost:8080/get-s3-creds?dataset=PID12345"
```

Response:

```json
{
  "access_key": "ASIA...",
  "secret_key": "wJalrXUtnFEMI/K7MDENG/bPxRfiCY...",
  "session_token": "FQoGZXIvYXdzE...",
  "expiry_time": "2025-09-09T16:20:00Z"
}
```

#### Client

```bash
git clone https://github.com/paulscherrerinstitute/scicat-s3-broker.git
cd scicat-s3-broker
go run ./cmd/client/credential_process.go --dataset PID12345 --token <scicat-token> --api http://localhost:8085/get-s3-creds
```
For use with AWS CLI and SDKs, build the client binary and configure your AWS profile to use it as a `credential_process`:

```bash
go build ./cmd/client/credential_process.go
./credential_process --dataset PID12345 --token <scicat-token> --api http://localhost:8085/get-s3-creds
```
Output:

```json
{
  "Version": 1,
  "AccessKeyId": "ASIA...",
  "SecretAccessKey": "wJalrXUtnFEMI/K7MDENG/bPxRfiCY...",
  "SessionToken": "FQoGZXIvYXdzE...",
  "Expiration": "2025-09-09T16:20:00Z"
}
```
---

## Development

Project layout follows [golang-standards/project-layout](https://github.com/golang-standards/project-layout):

```
cmd/            # main entrypoints
    server/         # API server
    client/         # CLI client for credential_process
internal/
    handlers/       # API handlers, auth, STS integration
    models/         # API request/response models, etc.
```

---

## License

[MIT](LICENSE) Copyright (c) 2025 Paul Scherrer Institute
