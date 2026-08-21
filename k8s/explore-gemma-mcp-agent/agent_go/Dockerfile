# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
COPY vendor/ ./vendor/
COPY main.go .

RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -o k8s-agent-go main.go

# Production distroless stage for gVisor sandbox pod
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /
COPY --from=builder /app/k8s-agent-go /k8s-agent-go

USER 65532:65532

ENTRYPOINT ["/k8s-agent-go"]
