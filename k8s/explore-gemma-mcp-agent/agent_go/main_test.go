package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestIsMutatingTool_TableDriven(t *testing.T) {
	session := NewAgentSession(nil, "")
	session.availableTools = map[string]*mcp.Tool{
		"k8s_get_resource": {
			Name: "k8s_get_resource",
			// Read-only tool (no requires_approval metadata)
		},
		"k8s_list_resources": {
			Name: "k8s_list_resources",
		},
		"k8s_describe_resource": {
			Name: "k8s_describe_resource",
		},
		"k8s_get_pod_logs": {
			Name: "k8s_get_pod_logs",
		},
		"k8s_patch_resource": {
			Name: "k8s_patch_resource",
			Meta: mcp.Meta{"requires_approval": true},
		},
		"k8s_restart_pod": {
			Name: "k8s_restart_pod",
			Meta: mcp.Meta{"requires_approval": true},
		},
		"k8s_scale_deployment": {
			Name: "k8s_scale_deployment",
			Meta: mcp.Meta{"requires_approval": true},
		},
		"custom_gated_action": {
			Name: "custom_gated_action",
			Meta: mcp.Meta{"requires_approval": true},
		},
		"custom_read_action": {
			Name: "custom_read_action",
			Meta: mcp.Meta{"requires_approval": false},
		},
	}

	tests := []struct {
		name     string
		fnName   string
		expected bool
	}{
		{
			name:     "read-only get resource",
			fnName:   "k8s_get_resource",
			expected: false,
		},
		{
			name:     "read-only list resources",
			fnName:   "k8s_list_resources",
			expected: false,
		},
		{
			name:     "read-only describe resource",
			fnName:   "k8s_describe_resource",
			expected: false,
		},
		{
			name:     "read-only pod logs",
			fnName:   "k8s_get_pod_logs",
			expected: false,
		},
		{
			name:     "mutating patch resource with requires_approval true",
			fnName:   "k8s_patch_resource",
			expected: true,
		},
		{
			name:     "mutating restart pod with requires_approval true",
			fnName:   "k8s_restart_pod",
			expected: true,
		},
		{
			name:     "mutating scale deployment with requires_approval true",
			fnName:   "k8s_scale_deployment",
			expected: true,
		},
		{
			name:     "custom gated action with requires_approval true",
			fnName:   "custom_gated_action",
			expected: true,
		},
		{
			name:     "custom read action with requires_approval false",
			fnName:   "custom_read_action",
			expected: false,
		},
		{
			name:     "unknown tool not found in cache",
			fnName:   "unknown_tool",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := session.isMutatingTool(tt.fnName)
			if got != tt.expected {
				t.Errorf("session.isMutatingTool(%q) = %v, want %v", tt.fnName, got, tt.expected)
			}
		})
	}
}

func TestNewAgentSession_PromptInitialization(t *testing.T) {
	t.Run("default prompt fallback", func(t *testing.T) {
		session := NewAgentSession(nil, "")
		if session.systemPrompt != defaultSystemPrompt {
			t.Errorf("expected default system prompt, got: %s", session.systemPrompt)
		}
	})

	t.Run("custom prompt override", func(t *testing.T) {
		customPrompt := "Custom prompt for network troubleshooting"
		session := NewAgentSession(nil, customPrompt)
		if session.systemPrompt != customPrompt {
			t.Errorf("expected custom prompt, got: %s", session.systemPrompt)
		}
	})
}

func TestRunTroubleshootingLoop_MockLLM(t *testing.T) {
	// 1. Mock Gemma vLLM Server
	mockLLMServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/chat/completions") {
			http.NotFound(w, r)
			return
		}

		resp := ChatCompletionResponse{
			Choices: []struct {
				Message      ChatMessage `json:"message"`
				FinishReason string      `json:"finish_reason"`
			}{
				{
					Message: ChatMessage{
						Role:    "assistant",
						Content: "Identified issue: Port misconfiguration.",
					},
					FinishReason: "stop",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer mockLLMServer.Close()

	origLLMBase := llmAPIBase
	llmAPIBase = mockLLMServer.URL
	defer func() { llmAPIBase = origLLMBase }()

	agent := NewAgentSession(nil, "")
	ctx := context.Background()

	result, err := agent.RunTroubleshootingLoop(ctx, "Diagnose crashlooping app")
	if err != nil {
		t.Fatalf("unexpected error running troubleshooting loop: %v", err)
	}

	if result["status"] != "COMPLETED" {
		t.Errorf("expected status COMPLETED, got: %v", result["status"])
	}
	if result["diagnosis"] != "Identified issue: Port misconfiguration." {
		t.Errorf("expected diagnosis content, got: %v", result["diagnosis"])
	}
}

func TestAgentAPIs_QueryAndApprove(t *testing.T) {
	mockLLMServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ChatCompletionResponse{
			Choices: []struct {
				Message      ChatMessage `json:"message"`
				FinishReason string      `json:"finish_reason"`
			}{
				{
					Message: ChatMessage{
						Role:    "assistant",
						Content: "Identified issue: Port misconfiguration.",
					},
					FinishReason: "stop",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer mockLLMServer.Close()

	origLLMBase := llmAPIBase
	llmAPIBase = mockLLMServer.URL
	defer func() { llmAPIBase = origLLMBase }()

	agent := NewAgentSession(nil, "")

	// Test POST /api/query handler
	queryHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var qReq QueryRequest
		json.NewDecoder(r.Body).Decode(&qReq)
		res, err := agent.RunTroubleshootingLoop(r.Context(), qReq.Query)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(res)
	})

	reqBody, _ := json.Marshal(QueryRequest{Query: "Check service health"})
	req := httptest.NewRequest("POST", "/api/query", bytes.NewBuffer(reqBody))
	rec := httptest.NewRecorder()

	queryHandler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200 OK, got: %d", rec.Code)
	}
}

func TestGetMandatoryEnv(t *testing.T) {
	t.Run("returns value when set", func(t *testing.T) {
		t.Setenv("TEST_KEY", "test_value")
		val, err := getMandatoryEnv("TEST_KEY")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if val != "test_value" {
			t.Errorf("expected 'test_value', got: %q", val)
		}
	})

	t.Run("errors when unset", func(t *testing.T) {
		_, err := getMandatoryEnv("NON_EXISTENT_VAR_123")
		if err == nil {
			t.Fatal("expected error for unset environment variable, got nil")
		}
	})

	t.Run("errors when empty or whitespace", func(t *testing.T) {
		t.Setenv("EMPTY_VAR", "   ")
		_, err := getMandatoryEnv("EMPTY_VAR")
		if err == nil {
			t.Fatal("expected error for whitespace-only environment variable, got nil")
		}
	})
}

func TestLoadConfig(t *testing.T) {
	t.Run("success when all mandatory env vars are present", func(t *testing.T) {
		t.Setenv("LLM_API_BASE", "http://llm:8000/v1")
		t.Setenv("LLM_MODEL", "custom-model")
		t.Setenv("MCP_SERVER_URL", "http://mcp:8080/sse")
		t.Setenv("PORT", "8090")

		if err := loadConfig(); err != nil {
			t.Fatalf("expected successful config load, got: %v", err)
		}
		if llmAPIBase != "http://llm:8000/v1" || llmModel != "custom-model" || mcpServerURL != "http://mcp:8080/sse" || port != "8090" {
			t.Errorf("unexpected loaded values: LLM_API_BASE=%q, LLM_MODEL=%q, MCP_SERVER_URL=%q, PORT=%q", llmAPIBase, llmModel, mcpServerURL, port)
		}
	})

	t.Run("fails when LLM_API_BASE is missing", func(t *testing.T) {
		t.Setenv("LLM_MODEL", "custom-model")
		t.Setenv("MCP_SERVER_URL", "http://mcp:8080/sse")
		t.Setenv("PORT", "8090")
		// Explicitly ensure LLM_API_BASE is unset in subtest
		t.Setenv("LLM_API_BASE", "")

		if err := loadConfig(); err == nil {
			t.Fatal("expected error when LLM_API_BASE is missing, got nil")
		}
	})
}
