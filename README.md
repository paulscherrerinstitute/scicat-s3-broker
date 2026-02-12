# SciCat S3 Broker

A lightweight service that brokers short-term S3 credentials for SciCat datasets.  
It delegates authorization to SciCat, then issues temporary, scoped S3 credentials
(e.g. via Ceph STS) that clients can consume through the AWS SDK/CLI
using the [`credential_process`](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-sourcing-external.html) mechanism.

Also included is a simple CLI client that can be used as an AWS CLI credential process. This might be migrated to its own repo in the future.

---

## Features

- 🔑 **Credential broker**: returns temporary S3 credentials for a given dataset.
- 📥 **Download URLs** for public datasets: get presigned URLs for retrieved, public datasets.
- 🛡 **Authorization via SciCat**: forwards the end-user’s SciCat token for access checks. //TO-DO

---

## Quickstart

### Prerequisites

- Go 1.21+
- SciCat backend instance
- Ceph or AWS-compatible S3 backend with STS enabled

### Configuration

The following environement variables are available for configuration:

| env var              | required | default    | description                                                 | example                            |
| -------------------- | -------- | ---------- | ----------------------------------------------------------- | ---------------------------------- |
| SCICAT_URL           | yes      |            | SciCat backend base url                                     | https://scicat.development.psi.ch/ |
| JOB_MANAGER_USERNAME | no       | jobManager | credentials for functional account to query /jobs in SciCat |                                    |
| JOB_MANAGER_PASSWORD | no\*     | ""         |                                                             |                                    |
| PORT                 | no       | 8080       | The port to serve from. This is a Gin configuration         |                                    |
| GIN_MODE             | no       | debug      | Set to `release` for production                             |                                    |

\* JOB_MANAGER_PASSWORD is _required_ for the `/get-urls` endpoint. If not set, the server returns `HTTP 501 Not Implemented`.
It is not required for the `/get-s3-creds` endpoint.

### Run locally

#### Server

```bash
git clone https://github.com/paulscherrerinstitute/scicat-s3-broker.git
cd scicat-s3-broker
go run ./cmd/server
```

The server will start on port `8080` by default, or `${PORT}` env variable if specified.

##### Example requests

###### get-s3-creds

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

###### get-urls

```bash
curl http://localhost:8080/get-urls?dataset=20.500.11935/0e54729b-75c5-42fa-a628-aae5dc3f3dae
```

Response:

```json
{
  "urls": [
    "https://rgw.cscs.ch/firecrest_hpc%3Anoderedd/datastblockfile.tar?Signature=xyz123"
  ],
  "expires": "2026-02-18T09:34:41.038Z"
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
    config/         # Server configuration
    handlers/       # API handlers
    models/         # API request/response models, etc.
```

---

## License

[MIT](LICENSE) Copyright (c) 2025 Paul Scherrer Institute
