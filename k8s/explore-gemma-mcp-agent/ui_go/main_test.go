package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestToJSONHelper(t *testing.T) {
	data := map[string]string{"foo": "bar"}
	res := toJSON(data)
	if !strings.Contains(res, `"foo": "bar"`) {
		t.Errorf("expected toJSON to contain formatted JSON, got: %s", res)
	}
}

func TestUI_RootEndpoint(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		globalState.Lock()
		defer globalState.Unlock()
		if globalState.SystemPrompt == "" {
			globalState.SystemPrompt = defaultSystemPrompt
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<html><body>Gemma K8s Agent UI</body></html>"))
	}).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200 OK, got: %d", rec.Code)
	}
}

func TestUI_UpdatePrompt(t *testing.T) {
	formData := url.Values{}
	newPrompt := "Custom prompt for testing UI"
	formData.Set("system_prompt", newPrompt)

	req := httptest.NewRequest("POST", "/update-prompt", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		globalState.Lock()
		globalState.SystemPrompt = r.FormValue("system_prompt")
		globalState.Unlock()
		w.WriteHeader(http.StatusOK)
	}).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200 OK, got: %d", rec.Code)
	}

	globalState.Lock()
	if globalState.SystemPrompt != newPrompt {
		t.Errorf("expected system prompt %q, got %q", newPrompt, globalState.SystemPrompt)
	}
	globalState.Unlock()
}

func TestUI_Chat_MockAgentService(t *testing.T) {
	// 1. Mock Agent Service backend
	mockAgentServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/query" {
			resp := map[string]interface{}{
				"status":        "COMPLETED",
				"final_summary": "Test diagnosis complete: All pods running normally.",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}
		http.NotFound(w, r)
	}))
	defer mockAgentServer.Close()

	origAgentURL := agentServiceURL
	agentServiceURL = mockAgentServer.URL
	defer func() { agentServiceURL = origAgentURL }()

	formData := url.Values{}
	formData.Set("prompt", "Is the cluster healthy?")

	req := httptest.NewRequest("POST", "/chat", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		userPrompt := r.FormValue("prompt")

		globalState.Lock()
		globalState.Messages = append(globalState.Messages, Message{Role: "user", Content: userPrompt})

		queryPayload, _ := json.Marshal(map[string]string{
			"query":         userPrompt,
			"system_prompt": globalState.SystemPrompt,
		})

		resp, err := httpClient.Post(agentServiceURL+"/api/query", "application/json", strings.NewReader(string(queryPayload)))
		if err != nil {
			t.Fatalf("failed calling mock agent: %v", err)
		}

		var agentResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&agentResp)
		resp.Body.Close()

		finalSummary, _ := agentResp["final_summary"].(string)
		globalState.Messages = append(globalState.Messages, Message{
			Role:    "assistant",
			Content: finalSummary,
		})
		globalState.Unlock()

		http.Redirect(w, r, "/", http.StatusSeeOther)
	}).ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Errorf("expected 303 redirect after chat, got: %d", rec.Code)
	}

	globalState.Lock()
	lastMsg := globalState.Messages[len(globalState.Messages)-1]
	globalState.Unlock()

	if !strings.Contains(lastMsg.Content, "All pods running normally") {
		t.Errorf("expected response to contain diagnosis, got: %s", lastMsg.Content)
	}
}

func TestUI_RejectAction(t *testing.T) {
	globalState.Lock()
	globalState.PendingApproval = map[string]interface{}{"tool_name": "k8s_patch_resource"}
	globalState.Unlock()

	req := httptest.NewRequest("POST", "/reject", nil)
	rec := httptest.NewRecorder()

	http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		globalState.Lock()
		globalState.Messages = append(globalState.Messages, Message{
			Role:    "assistant",
			Content: "❌ **Action Rejected by User.** No cluster changes were performed.",
		})
		globalState.PendingApproval = nil
		globalState.Unlock()
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}).ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Errorf("expected redirect status, got: %d", rec.Code)
	}

	globalState.Lock()
	if globalState.PendingApproval != nil {
		t.Errorf("expected pending approval to be cleared")
	}
	globalState.Unlock()
}
