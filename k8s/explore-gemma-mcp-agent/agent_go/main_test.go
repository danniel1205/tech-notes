package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIsMutatingTool_TableDriven(t *testing.T) {
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
			name:     "mutating patch resource",
			fnName:   "k8s_patch_resource",
			expected: true,
		},
		{
			name:     "mutating restart pod",
			fnName:   "k8s_restart_pod",
			expected: true,
		},
		{
			name:     "mutating scale deployment",
			fnName:   "k8s_scale_deployment",
			expected: true,
		},
		{
			name:     "mutating delete pod",
			fnName:   "k8s_delete_pod",
			expected: true,
		},
		{
			name:     "non-mutating custom tool",
			fnName:   "custom_k8s_query",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isMutatingTool(tt.fnName)
			if got != tt.expected {
				t.Errorf("isMutatingTool(%q) = %v, want %v", tt.fnName, got, tt.expected)
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
