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

- Go 1.25+
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

\* JOB_MANAGER_PASSWORD is _required_ for the `/datasets/urls` endpoint. If not set, the server returns `HTTP 501 Not Implemented`.
It is not required for the `/datasets/s3-creds` endpoint.

#### AWS Config
The AWS shared config and credentials files are in [env/](./env) directory. Copy `credentials.example` to `credentials` and replace with your secret / access key.

### Run locally

#### Server

```bash
git clone https://github.com/paulscherrerinstitute/scicat-s3-broker.git
cd scicat-s3-broker
go run ./cmd/server
```

The server will start on port `8080` by default, or `${PORT}` env variable if specified.

##### Example requests

###### /datasets/s3-creds

```bash
curl -H "Authorization: Bearer <scicat-token>" \
  "http://localhost:8080/datasets/s3-creds?dataset=PID12345"
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

###### /datasets/urls

```bash
curl http://localhost:8080/datasets/urls?pid=20.500.11935/0e54729b-75c5-42fa-a628-aae5dc3f3dae
```

Response:

```json
[
  {
    "url": "https://rgw.cscs.ch/firecrest_hpc%3Anoderedd/8414927a-55cb-4b03-8ed5-3af195fe0524/0e54729b-75c5-42fa-a628-aae5dc3f3dae_0_2022-09-08-14-52-32.tar?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=S82RBBK66XUCNDL3NGXD%2F20260211%2Fcscs-zonegroup%2Fs3%2Faws4_request&X-Amz-Date=20260211T093601Z&X-Amz-Expires=604800&X-Amz-SignedHeaders=host&X-Amz-Signature=422a4f7e759cf51c99459fa32596baec37c1064fd6c5c9900cc488c80ece097a",
    "expires": "2026-02-18T09:36:01Z"
  }
]
```

#### Client

```bash
git clone https://github.com/paulscherrerinstitute/scicat-s3-broker.git
cd scicat-s3-broker
go run ./cmd/client/credential_process.go --dataset PID12345 --token <scicat-token> --api http://localhost:8085/datasets/s3-creds
```

For use with AWS CLI and SDKs, build the client binary and configure your AWS profile to use it as a `credential_process`:

```bash
go build ./cmd/client/credential_process.go
./credential_process --dataset PID12345 --token <scicat-token> --api http://localhost:8080/datasets/s3-creds
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

Project layout follows [golang-standards/project-layout](https://github.com/golang-standards/project-layout).
Within `/internal`, we use group packages by features.

```
cmd/            # main entrypoints
    server/         # API server
    client/         # CLI client for credential_process
internal/
    config/         # Server configuration
    api/            # Generated server interface
    s3/             # S3 handlers and related functionality 
    scicat/         # SciCat handlers and realted functionality
```

---

## License

[MIT](LICENSE) Copyright (c) 2025 Paul Scherrer Institute
