//go:build integration

package integration_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const brokenAppManifest = "../../deploy/06-test-broken-app.yaml"

func TestKindE2E_FullTroubleshootingAndRemediation(t *testing.T) {
	// 1. Verify connection to Kind cluster
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		kubeconfig = filepath.Join(os.Getenv("HOME"), ".kube", "config")
	}
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		t.Fatalf("Could not load kubeconfig from %s: %v", kubeconfig, err)
	}

	cs, err := kubernetes.NewForConfig(config)
	if err != nil {
		t.Fatalf("Failed creating Kubernetes clientset: %v", err)
	}

	ctx := context.Background()

	// 2. Deploy test-broken-app with intentional image typo (06-test-broken-app.yaml)
	t.Logf("Deploying broken test app (%s)...", brokenAppManifest)
	cmd := exec.Command("kubectl", "apply", "-f", brokenAppManifest)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("kubectl apply failed: %s", string(out))
	}
	defer func() {
		t.Logf("Cleaning up test-broken-app (%s)...", brokenAppManifest)
		exec.Command("kubectl", "delete", "-f", brokenAppManifest, "--ignore-not-found").Run()
	}()

	mcpURL := os.Getenv("MCP_SERVER_URL")
	if mcpURL == "" {
		mcpURL = "http://127.0.0.1:8089/sse"
	}

	// 3. Connect to MCP Server with retry loop
	t.Logf("Connecting MCP client to %s...", mcpURL)
	transport := mcp.NewSSEClientTransport(mcpURL, nil)
	client := mcp.NewClient("integration-tester", "1.0.0", nil)

	var session *mcp.ClientSession
	var connErr error
	for attempt := 1; attempt <= 15; attempt++ {
		session, connErr = client.Connect(ctx, transport)
		if connErr == nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if connErr != nil {
		t.Fatalf("Failed connecting to MCP server at %s after retries: %v", mcpURL, connErr)
	}
	defer session.Close()

	// 4. Test MCP Tool Discovery
	t.Log("Verifying tool discovery...")
	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("Error listing tools: %v", err)
	}
	if len(tools.Tools) < 5 {
		t.Fatalf("Expected at least 5 registered tools, got %d", len(tools.Tools))
	}
	t.Logf("Discovered %d MCP tools successfully", len(tools.Tools))

	// 5. Test Resource Description via MCP
	t.Log("Executing k8s_describe_resource for test-broken-app...")
	describeRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "k8s_describe_resource",
		Arguments: map[string]interface{}{
			"group":     "apps",
			"version":   "v1",
			"resource":  "deployments",
			"name":      "test-broken-app",
			"namespace": "default",
		},
	})
	if err != nil {
		t.Fatalf("Failed calling k8s_describe_resource: %v", err)
	}
	if len(describeRes.Content) == 0 {
		t.Fatalf("Expected non-empty describe result")
	}

	// 6. Test Security Gate: Patching WITHOUT approval token must fail
	t.Log("Verifying security gate: Patching without approval token must fail...")
	_, err = session.CallTool(ctx, &mcp.CallToolParams{
		Name: "k8s_patch_resource",
		Arguments: map[string]interface{}{
			"group":          "apps",
			"version":        "v1",
			"resource":       "deployments",
			"name":           "test-broken-app",
			"namespace":      "default",
			"patch_json":     `{"spec":{"template":{"spec":{"containers":[{"name":"web","image":"nginx:latest"}]}}}}`,
			"approval_token": "", // missing token
		},
	})
	if err == nil {
		t.Fatalf("Expected permission failure when approval_token is empty, got nil")
	}

	// 7. Test Remediation: Patching WITH approval token
	t.Log("Executing approved remediation via k8s_patch_resource...")
	patchRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "k8s_patch_resource",
		Arguments: map[string]interface{}{
			"group":          "apps",
			"version":        "v1",
			"resource":       "deployments",
			"name":           "test-broken-app",
			"namespace":      "default",
			"patch_json":     `{"spec":{"template":{"spec":{"containers":[{"name":"web","image":"nginx:latest"}]}}}}`,
			"approval_token": "USER_APPROVED_TOKEN",
		},
	})
	if err != nil {
		t.Fatalf("Failed executing approved patch: %v", err)
	}
	t.Logf("Remediation result: %v", patchRes.Content[0])

	// 8. Verify Cluster Convergence (Pod transitions to Running & Deployment is Ready)
	t.Log("Waiting for test-broken-app deployment to achieve ready state (timeout: 90s)...")
	ready := false
	maxAttempts := 90
	for i := 0; i < maxAttempts; i++ {
		dep, err := cs.AppsV1().Deployments("default").Get(ctx, "test-broken-app", metav1.GetOptions{})
		if err == nil {
			if dep.Status.ReadyReplicas >= 1 || dep.Status.AvailableReplicas >= 1 {
				ready = true
				t.Logf("✅ Deployment test-broken-app successfully recovered to Running status! (ReadyReplicas: %d, AvailableReplicas: %d)",
					dep.Status.ReadyReplicas, dep.Status.AvailableReplicas)
				break
			}
			if i%5 == 0 || i == maxAttempts-1 {
				t.Logf("[Attempt %d/%d] Deployment status: Replicas=%d, ReadyReplicas=%d, AvailableReplicas=%d (pulling/starting image)...",
					i+1, maxAttempts, dep.Status.Replicas, dep.Status.ReadyReplicas, dep.Status.AvailableReplicas)
			}
		}
		time.Sleep(1 * time.Second)
	}

	if !ready {
		t.Fatalf("Deployment did not become ready within %ds timeout", maxAttempts)
	}
}
