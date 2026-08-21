# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod ./
COPY main.go .

RUN CGO_ENABLED=0 GOOS=linux go build -o k8s-agent-ui-go main.go

# Production distroless stage
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /
COPY --from=builder /app/k8s-agent-ui-go /k8s-agent-ui-go

EXPOSE 8501

USER 65532:65532

ENTRYPOINT ["/k8s-agent-ui-go"]
