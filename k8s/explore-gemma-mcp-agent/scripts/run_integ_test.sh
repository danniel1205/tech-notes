#!/usr/bin/env bash

set -euo pipefail

# Configuration
KIND_CLUSTER_NAME="${KIND_CLUSTER_NAME:-k8s-agent-test}"
MCP_PORT="${MCP_PORT:-8089}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

TMP_BIN_DIR="$(mktemp -d)"
MCP_PID=""

cleanup() {
    echo "🧹 Cleaning up test resources..."
    if [ -n "${MCP_PID}" ]; then
        echo "Stopping MCP Server (PID ${MCP_PID})..."
        kill "${MCP_PID}" 2>/dev/null || true
    fi
    rm -rf "${TMP_BIN_DIR}"
    echo "Tearing down Kind cluster '${KIND_CLUSTER_NAME}'..."
    kind delete cluster --name "${KIND_CLUSTER_NAME}" 2>/dev/null || true
    echo "✅ Teardown complete."
}

trap cleanup EXIT INT TERM

# 1. Provision Kind cluster if not running
if ! kind get clusters 2>/dev/null | grep -q "^${KIND_CLUSTER_NAME}$"; then
    echo "🚀 Creating Kind cluster '${KIND_CLUSTER_NAME}'..."
    kind create cluster --name "${KIND_CLUSTER_NAME}"
fi

# 2. Export Kubeconfig
KUBECONFIG_FILE="${TMP_BIN_DIR}/kubeconfig.yaml"
kind get kubeconfig --name "${KIND_CLUSTER_NAME}" > "${KUBECONFIG_FILE}"
export KUBECONFIG="${KUBECONFIG_FILE}"

# 3. Compile local Go MCP Server
echo "🔨 Compiling local Go MCP Server..."
go build -o "${TMP_BIN_DIR}/k8s-mcp-server-go" "${ROOT_DIR}/mcp_server_go/main.go"

# 4. Start Go MCP Server in background
echo "🚀 Starting Go MCP Server on port ${MCP_PORT}..."
PORT="${MCP_PORT}" "${TMP_BIN_DIR}/k8s-mcp-server-go" &
MCP_PID=$!

echo "Waiting for MCP server to start..."
sleep 2

# 5. Run Integration Tests
echo "🧪 Running integration test suite against '${KIND_CLUSTER_NAME}'..."
cd "${ROOT_DIR}/tests/integration"
MCP_SERVER_URL="http://127.0.0.1:${MCP_PORT}/sse" go test -v -tags=integration -timeout=180s .

echo "✅ All integration tests passed successfully!"
