package main

import (
	"bytes"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

var (
	agentServiceURL = getEnv("AGENT_SERVICE_URL", "http://k8s-agent-service:8090")
)

const defaultSystemPrompt = `You are an expert Autonomous Kubernetes Troubleshooting Agent.
You have direct access to Kubernetes MCP tools to inspect and diagnose the cluster in real-time.

CRITICAL INSTRUCTIONS:
1. When asked to investigate or diagnose a pod, deployment, or cluster failure, DO NOT ask the user to run kubectl or MCP commands manually.
2. You MUST immediately invoke the available tools (e.g. k8s_describe_resource, k8s_get_pod_logs, k8s_list_resources) using tool calls to retrieve actual live status, conditions, and error logs.
3. After receiving the tool outputs, analyze them to identify the exact root cause (e.g. ImagePullBackOff, CrashLoopBackOff, misconfiguration).
4. If a fix requires modifying or patching a resource (e.g., k8s_patch_resource, k8s_restart_pod), propose the mutating tool call so the user can review and approve it. Never execute mutating actions without permission.`

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

type Message struct {
	Role           string                 `json:"role"`
	Content        string                 `json:"content"`
	ProposedAction map[string]interface{} `json:"proposed_action,omitempty"`
}

type State struct {
	sync.Mutex
	SystemPrompt    string
	Messages        []Message
	PendingApproval map[string]interface{}
}

var globalState = &State{
	SystemPrompt: defaultSystemPrompt,
	Messages:     []Message{},
}

var httpClient = &http.Client{Timeout: 90 * time.Second}

const pageHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Gemma K8s Agent UI (Go)</title>
    <script src="https://unpkg.com/htmx.org@1.9.10"></script>
    <style>
        * { box-sizing: border-box; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; }
        body { display: flex; margin: 0; height: 100vh; background: #0e1117; color: #ffffff; }
        .sidebar { width: 340px; background: #161b22; padding: 20px; border-right: 1px solid #30363d; display: flex; flex-direction: column; }
        .sidebar h2 { font-size: 1.1rem; margin-top: 0; }
        .sidebar textarea { width: 100%; height: 320px; background: #0d1117; border: 1px solid #30363d; color: #e6edf3; padding: 10px; border-radius: 6px; resize: vertical; }
        .main { flex: 1; display: flex; flex-direction: column; height: 100vh; }
        .header { padding: 15px 25px; border-bottom: 1px solid #30363d; background: #161b22; }
        .header h1 { margin: 0; font-size: 1.3rem; }
        .chat-container { flex: 1; overflow-y: auto; padding: 25px; display: flex; flex-direction: column; gap: 15px; }
        .message { max-width: 80%; padding: 12px 18px; border-radius: 8px; line-height: 1.5; }
        .message.user { align-self: flex-end; background: #1f6feb; color: #fff; }
        .message.assistant { align-self: flex-start; background: #21262d; border: 1px solid #30363d; color: #e6edf3; }
        .approval-card { background: #2d1810; border: 1px solid #d29922; border-radius: 6px; padding: 15px; margin-top: 10px; }
        .btn-group { display: flex; gap: 10px; margin-top: 10px; }
        .btn { padding: 8px 16px; border: none; border-radius: 4px; cursor: pointer; font-weight: bold; }
        .btn-approve { background: #238636; color: white; }
        .btn-reject { background: #da3633; color: white; }
        .input-bar { padding: 20px 25px; background: #161b22; border-top: 1px solid #30363d; display: flex; gap: 10px; }
        .input-bar input { flex: 1; padding: 12px; background: #0d1117; border: 1px solid #30363d; color: #fff; border-radius: 6px; }
        .input-bar button { padding: 12px 24px; background: #238636; color: white; border: none; border-radius: 6px; cursor: pointer; font-weight: bold; }
        pre { background: #0d1117; padding: 10px; border-radius: 4px; overflow-x: auto; }
        code { font-family: monospace; }
    </style>
</head>
<body>
    <div class="sidebar">
        <h2>System Instructions</h2>
        <form hx-post="/update-prompt" hx-trigger="change, keyup delay:500ms">
            <textarea name="system_prompt">{{.SystemPrompt}}</textarea>
        </form>
    </div>
    <div class="main">
        <div class="header">
            <h1>🤖 Gemma Autonomous Kubernetes Agent (Go + gVisor)</h1>
        </div>
        <div class="chat-container" id="chat-box">
            {{range .Messages}}
                <div class="message {{.Role}}">
                    <div>{{.Content}}</div>
                    {{if .ProposedAction}}
                        <div class="approval-card">
                            <strong>Proposed Operation:</strong> <code>{{index .ProposedAction "tool_name"}}</code><br>
                            <strong>Target Resource:</strong> <code>{{index .ProposedAction "resource"}}/{{index .ProposedAction "name"}}</code><br>
                            <strong>Namespace:</strong> <code>{{index .ProposedAction "namespace"}}</code><br>
                            <pre><code>{{toJSON (index .ProposedAction "patch")}}</code></pre>
                            <div class="btn-group">
                                <form action="/approve" method="POST" style="display:inline;">
                                    <button class="btn btn-approve" type="submit">Approve & Execute Fix</button>
                                </form>
                                <form action="/reject" method="POST" style="display:inline;">
                                    <button class="btn btn-reject" type="submit">Reject</button>
                                </form>
                            </div>
                        </div>
                    {{end}}
                </div>
            {{end}}
        </div>
        <form class="input-bar" action="/chat" method="POST">
            <input type="text" name="prompt" placeholder="Ask Gemma to diagnose a Kubernetes issue..." required autocomplete="off">
            <button type="submit">Investigate</button>
        </form>
    </div>
</body>
</html>`

func toJSON(v interface{}) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}

func main() {
	tmpl := template.Must(template.New("index").Funcs(template.FuncMap{
		"toJSON": toJSON,
	}).Parse(pageHTML))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		globalState.Lock()
		defer globalState.Unlock()
		tmpl.Execute(w, globalState)
	})

	http.HandleFunc("/update-prompt", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			r.ParseForm()
			globalState.Lock()
			globalState.SystemPrompt = r.FormValue("system_prompt")
			globalState.Unlock()
			w.WriteHeader(http.StatusOK)
		}
	})

	http.HandleFunc("/chat", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			r.ParseForm()
			userPrompt := r.FormValue("prompt")

			globalState.Lock()
			globalState.Messages = append(globalState.Messages, Message{Role: "user", Content: userPrompt})

			// Call Agent Service /api/query REST API
			queryPayload, _ := json.Marshal(map[string]string{
				"query":         userPrompt,
				"system_prompt": globalState.SystemPrompt,
			})

			resp, err := httpClient.Post(agentServiceURL+"/api/query", "application/json", bytes.NewBuffer(queryPayload))
			if err != nil {
				globalState.Messages = append(globalState.Messages, Message{
					Role:    "assistant",
					Content: "❌ **Error communicating with Agent Service**: " + err.Error(),
				})
				globalState.Unlock()
				http.Redirect(w, r, "/", http.StatusSeeOther)
				return
			}

			var agentResp map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&agentResp)
			resp.Body.Close()

			status, _ := agentResp["status"].(string)
			if status == "AWAITING_PERMISSION" {
				diagSummary, _ := agentResp["diagnosis"].(string)
				if diagSummary == "" {
					diagSummary, _ = agentResp["diagnosis_summary"].(string)
				}
				proposedAction, _ := agentResp["proposed_action"].(map[string]interface{})

				globalState.Messages = append(globalState.Messages, Message{
					Role:           "assistant",
					Content:        "### ⚠️ Diagnosis Summary & Proposed Action\n\n" + diagSummary,
					ProposedAction: proposedAction,
				})
				globalState.PendingApproval = proposedAction
			} else {
				finalSummary, _ := agentResp["diagnosis"].(string)
				if finalSummary == "" {
					finalSummary, _ = agentResp["final_summary"].(string)
				}
				if finalSummary == "" {
					finalSummary = "Investigation completed."
				}
				globalState.Messages = append(globalState.Messages, Message{
					Role:    "assistant",
					Content: finalSummary,
				})
			}
			globalState.Unlock()

			http.Redirect(w, r, "/", http.StatusSeeOther)
		}
	})

	http.HandleFunc("/approve", func(w http.ResponseWriter, r *http.Request) {
		globalState.Lock()
		if globalState.PendingApproval != nil {
			approvePayload, _ := json.Marshal(map[string]interface{}{
				"tool_name":      globalState.PendingApproval["tool_name"],
				"arguments":      globalState.PendingApproval["arguments"],
				"approval_token": "USER_APPROVED_TOKEN",
			})

			resp, err := httpClient.Post(agentServiceURL+"/api/approve", "application/json", bytes.NewBuffer(approvePayload))
			if err != nil {
				globalState.Messages = append(globalState.Messages, Message{
					Role:    "assistant",
					Content: "❌ **Error executing approved action via Agent Service**: " + err.Error(),
				})
			} else {
				var approveResp map[string]interface{}
				json.NewDecoder(resp.Body).Decode(&approveResp)
				resp.Body.Close()

				resBytes, _ := json.MarshalIndent(approveResp["result"], "", "  ")
				globalState.Messages = append(globalState.Messages, Message{
					Role:    "assistant",
					Content: "### ✅ Action Executed Successfully\n\n```json\n" + string(resBytes) + "\n```",
				})
			}
			globalState.PendingApproval = nil
		}
		globalState.Unlock()
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	http.HandleFunc("/reject", func(w http.ResponseWriter, r *http.Request) {
		globalState.Lock()
		globalState.Messages = append(globalState.Messages, Message{
			Role:    "assistant",
			Content: "❌ **Action Rejected by User.** No cluster changes were performed.",
		})
		globalState.PendingApproval = nil
		globalState.Unlock()
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	port := getEnv("PORT", "8501")
	log.Printf("Starting Go Web UI on http://0.0.0.0:%s (targeting Agent Service at %s)...", port, agentServiceURL)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
