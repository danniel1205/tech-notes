package main

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		cfg         Config
		expectError bool
		errContains string
	}{
		{
			name: "valid configuration",
			cfg: Config{
				Namespace:   "default",
				ServiceName: "k8s-agent-service",
				ServicePort: 8090,
			},
			expectError: false,
		},
		{
			name: "missing namespace",
			cfg: Config{
				Namespace:   "",
				ServiceName: "k8s-agent-service",
				ServicePort: 8090,
			},
			expectError: true,
			errContains: "namespace",
		},
		{
			name: "whitespace namespace",
			cfg: Config{
				Namespace:   "   ",
				ServiceName: "k8s-agent-service",
				ServicePort: 8090,
			},
			expectError: true,
			errContains: "namespace",
		},
		{
			name: "missing service name",
			cfg: Config{
				Namespace:   "default",
				ServiceName: "",
				ServicePort: 8090,
			},
			expectError: true,
			errContains: "service",
		},
		{
			name: "invalid port zero",
			cfg: Config{
				Namespace:   "default",
				ServiceName: "k8s-agent-service",
				ServicePort: 0,
			},
			expectError: true,
			errContains: "port",
		},
		{
			name: "invalid port negative",
			cfg: Config{
				Namespace:   "default",
				ServiceName: "k8s-agent-service",
				ServicePort: -1,
			},
			expectError: true,
			errContains: "port",
		},
		{
			name: "invalid port out of range",
			cfg: Config{
				Namespace:   "default",
				ServiceName: "k8s-agent-service",
				ServicePort: 70000,
			},
			expectError: true,
			errContains: "port",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.errContains)
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("expected error to contain %q, got %q", tt.errContains, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

func newTestAgentClient(t *testing.T, handler http.HandlerFunc) (*AgentClient, *httptest.Server) {
	server := httptest.NewServer(handler)
	serverURL, _ := url.Parse(server.URL)

	config := &rest.Config{
		Host: serverURL.String(),
		ContentConfig: rest.ContentConfig{
			GroupVersion:         &schema.GroupVersion{Group: "", Version: "v1"},
			NegotiatedSerializer: scheme.Codecs.WithoutConversion(),
		},
	}
	restClient, err := rest.RESTClientFor(config)
	if err != nil {
		t.Fatalf("failed creating test restClient: %v", err)
	}

	client := &AgentClient{
		restClient:  restClient,
		namespace:   "default",
		serviceName: "k8s-agent-service",
		port:        8090,
	}

	return client, server
}

func TestAgentClient_SendQuery(t *testing.T) {
	client, server := newTestAgentClient(t, func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/api/v1/namespaces/default/services/k8s-agent-service:8090/proxy/api/query"
		if r.URL.Path != expectedPath {
			t.Errorf("unexpected proxy request path: %s (expected %s)", r.URL.Path, expectedPath)
			http.NotFound(w, r)
			return
		}

		var req QueryRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		resp := QueryResponse{
			Status:    "COMPLETED",
			Diagnosis: "All pods healthy in namespace default.",
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	ctx := context.Background()
	resp, err := client.SendQuery(ctx, "check pods", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Status != "COMPLETED" {
		t.Errorf("expected status COMPLETED, got: %s", resp.Status)
	}
	if resp.Diagnosis != "All pods healthy in namespace default." {
		t.Errorf("expected diagnosis, got: %s", resp.Diagnosis)
	}
}

func TestAgentClient_SendApprove(t *testing.T) {
	client, server := newTestAgentClient(t, func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/api/v1/namespaces/default/services/k8s-agent-service:8090/proxy/api/approve"
		if r.URL.Path != expectedPath {
			t.Errorf("unexpected proxy path: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}

		var req ApproveRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		resp := map[string]interface{}{
			"status":  "REMEDIATED",
			"message": "Successfully patched deployment.",
			"result": map[string]interface{}{
				"content": []map[string]string{
					{"type": "text", "text": "Patch applied"},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	ctx := context.Background()
	args := map[string]interface{}{"name": "test-app", "patch_json": "{}"}
	resp, err := client.SendApprove(ctx, "k8s_patch_resource", args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Status != "REMEDIATED" {
		t.Errorf("expected status REMEDIATED, got: %s", resp.Status)
	}
	if resp.Message != "Successfully patched deployment." {
		t.Errorf("expected message string, got: %s", resp.Message)
	}
}

func TestExecuteQueryWorkflow_ApprovalFlow(t *testing.T) {
	approvedCalled := false
	client, server := newTestAgentClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/namespaces/default/services/k8s-agent-service:8090/proxy/api/query":
			resp := QueryResponse{
				Status:    "AWAITING_PERMISSION",
				Diagnosis: "Found broken image tag.",
				ProposedAction: &ProposedAction{
					ToolName: "k8s_patch_resource",
					Arguments: map[string]interface{}{
						"name":       "test-broken-app",
						"namespace":  "default",
						"patch_json": "{\"spec\":{\"template\":{\"spec\":{\"containers\":[{\"name\":\"web\",\"image\":\"nginx:latest\"}]}}}}",
					},
					Description: "Update image tag to latest",
				},
			}
			json.NewEncoder(w).Encode(resp)
		case "/api/v1/namespaces/default/services/k8s-agent-service:8090/proxy/api/approve":
			approvedCalled = true
			resp := map[string]interface{}{
				"status":  "REMEDIATED",
				"message": "Successfully patched deployment.",
				"result": map[string]interface{}{
					"content": []map[string]string{
						{"type": "text", "text": "patched"},
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
		default:
			http.NotFound(w, r)
		}
	})
	defer server.Close()

	ctx := context.Background()

	t.Run("User approves with 'y'", func(t *testing.T) {
		approvedCalled = false
		inReader := bufio.NewReader(strings.NewReader("y\n"))
		err := executeQueryWorkflow(ctx, client, "fix broken app", "", false, inReader)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !approvedCalled {
			t.Errorf("expected /api/approve to be called when user inputs 'y'")
		}
	})

	t.Run("User declines with 'n'", func(t *testing.T) {
		approvedCalled = false
		inReader := bufio.NewReader(strings.NewReader("n\n"))
		err := executeQueryWorkflow(ctx, client, "fix broken app", "", false, inReader)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if approvedCalled {
			t.Errorf("did not expect /api/approve to be called when user inputs 'n'")
		}
	})

	t.Run("Auto-approve flag bypasses prompt", func(t *testing.T) {
		approvedCalled = false
		inReader := bufio.NewReader(strings.NewReader(""))
		err := executeQueryWorkflow(ctx, client, "fix broken app", "", true, inReader)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !approvedCalled {
			t.Errorf("expected /api/approve to be called automatically with autoApprove=true")
		}
	})
}

func TestAgentClient_ErrorUnwrapping(t *testing.T) {
	client, server := newTestAgentClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "LLM API error (status 400): context length exceeded",
		})
	})
	defer server.Close()

	ctx := context.Background()
	_, err := client.SendQuery(ctx, "test query", "")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	expectedSubstr := "LLM API error (status 400): context length exceeded"
	if !strings.Contains(err.Error(), expectedSubstr) {
		t.Errorf("expected error to contain %q, got %q", expectedSubstr, err.Error())
	}
}
