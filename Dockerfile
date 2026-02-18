FROM golang:1.25 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -v -o scicat-s3-broker ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /

COPY --from=builder /app/scicat-s3-broker /scicat-s3-broker

EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/scicat-s3-broker"]
