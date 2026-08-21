.PHONY: all unit-test test test-mcp test-agent test-ui kind-create kind-delete integ-test integration-test build docker-build docker-push docker-all deploy-all clean vendor help

SHELL := /bin/bash
KIND_CLUSTER_NAME ?= k8s-agent-test
PROJECT_ID ?= $(shell gcloud config get-value project 2>/dev/null || echo "your-project-id")
REGISTRY ?= gcr.io/$(PROJECT_ID)
TAG ?= latest

# Default target
all: unit-test

## unit-test: Run unit tests across all Go subprojects (mcp_server_go, agent_go, ui_go)
unit-test:
	@echo "🧪 Running unit tests for all Go subprojects..."
	@echo "--------------------------------------------------------"
	@go test -v -cover ./mcp_server_go/... ./agent_go/... ./ui_go/...
	@echo "--------------------------------------------------------"
	@echo "✅ All unit tests passed!"

## test: Alias for unit-test
test: unit-test

## test-mcp: Run unit tests only for mcp_server_go
test-mcp:
	@echo "🧪 Testing mcp_server_go..."
	@cd mcp_server_go && go test -v -cover .

## test-agent: Run unit tests only for agent_go
test-agent:
	@echo "🧪 Testing agent_go..."
	@cd agent_go && go test -v -cover .

## test-ui: Run unit tests only for ui_go
test-ui:
	@echo "🧪 Testing ui_go..."
	@cd ui_go && go test -v -cover .

## kind-create: Create local Kind cluster for testing
kind-create:
	@echo "🚀 Creating Kind cluster '$(KIND_CLUSTER_NAME)'..."
	@kind get clusters | grep -q "^$(KIND_CLUSTER_NAME)$$" || kind create cluster --name $(KIND_CLUSTER_NAME)

## kind-delete: Delete local Kind cluster
kind-delete:
	@echo "🧹 Deleting Kind cluster '$(KIND_CLUSTER_NAME)'..."
	@kind delete cluster --name $(KIND_CLUSTER_NAME) 2>/dev/null || true

## integ-test: Automatically provision Kind cluster, start MCP server, run E2E tests, and tear down
integ-test:
	@./scripts/run_integ_test.sh

## integration-test: Alias for integ-test
integration-test: integ-test

## build: Build local binaries for all Go subprojects
build:
	@echo "🔨 Building binaries for all subprojects..."
	@cd mcp_server_go && go build -o bin/k8s-mcp-server-go main.go
	@cd agent_go && go build -o bin/k8s-agent-go main.go
	@cd ui_go && go build -o bin/k8s-agent-ui-go main.go
	@echo "✅ Build complete! Binaries located in subproject bin/ directories."

## docker-build: Build Docker images for all subprojects (k8s-mcp-server, k8s-agent, k8s-agent-ui)
docker-build:
	@echo "🐳 Building Docker images for $(REGISTRY)..."
	@docker build -t $(REGISTRY)/k8s-mcp-server:$(TAG) ./mcp_server_go
	@docker build -t $(REGISTRY)/k8s-agent:$(TAG) ./agent_go
	@docker build -t $(REGISTRY)/k8s-agent-ui:$(TAG) ./ui_go
	@echo "✅ All Docker images built successfully!"

## docker-push: Push Docker images for all subprojects to container registry
docker-push:
	@echo "🚀 Pushing Docker images to $(REGISTRY)..."
	@docker push $(REGISTRY)/k8s-mcp-server:$(TAG)
	@docker push $(REGISTRY)/k8s-agent:$(TAG)
	@docker push $(REGISTRY)/k8s-agent-ui:$(TAG)
	@echo "✅ All Docker images pushed successfully!"

## docker-all: Build and push all Docker images in one step
docker-all: docker-build docker-push

## deploy-all: Deploy RBAC, MCP Server, Sandboxed Agent, and Web UI to GKE
deploy-all:
	@echo "☸️ Deploying RBAC, MCP Server, Agent (gVisor), and Web UI to cluster..."
	@kubectl apply -f deploy/02-mcp-rbac.yaml
	@kubectl apply -f deploy/03-mcp-server.yaml
	@kubectl apply -f deploy/04-agent-gvisor.yaml
	@kubectl apply -f deploy/05-ui.yaml
	@echo "✅ Deployment complete!"

## vendor: Synchronize and vendor dependencies for all subprojects
vendor:
	@echo "📦 Vendoring dependencies..."
	@cd mcp_server_go && GOWORK=off go mod tidy && GOWORK=off go mod vendor
	@cd agent_go && GOWORK=off go mod tidy && GOWORK=off go mod vendor
	@cd ui_go && GOWORK=off go mod tidy && GOWORK=off go mod vendor
	@cd tests/integration && GOWORK=off go mod tidy && GOWORK=off go mod vendor
	@echo "✅ Vendoring complete!"

## clean: Remove built binaries and test caches
clean:
	@echo "🧹 Cleaning binaries and test cache..."
	@rm -rf mcp_server_go/bin agent_go/bin ui_go/bin
	@go clean -testcache
	@echo "✅ Cleanup complete!"

## help: Display this help message
help:
	@echo "Usage: make [target] [PROJECT_ID=my-project] [TAG=latest]"
	@echo ""
	@echo "Targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed -e 's/## //g' | awk 'BEGIN {FS = ":"}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'
