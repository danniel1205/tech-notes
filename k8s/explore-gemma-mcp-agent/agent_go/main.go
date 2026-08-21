package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var (
	llmAPIBase   = getEnv("LLM_API_BASE", "http://gemma-llm-service:8000/v1")
	llmModel     = getEnv("LLM_MODEL", "Qwen/Qwen2.5-7B-Instruct")
	mcpServerURL = getEnv("MCP_SERVER_URL", "http://k8s-mcp-server:8080/sse")
	port         = getEnv("PORT", "8090")
)

const defaultSystemPrompt = `You are an expert Autonomous Kubernetes Troubleshooting and Operations Agent powered by Qwen.
Your goal is to help users diagnose, analyze, and resolve any kind of failure, error, misconfiguration, performance bottleneck, or operational issue across all Kubernetes cluster components and resources.

INSTRUCTIONS:
1. Use the provided Kubernetes MCP read tools (e.g., k8s_list_resources, k8s_describe_resource, k8s_get_resource, k8s_get_pod_logs, k8s_get_events) to systematically investigate cluster state and gather facts.
2. Inspect resource specifications, status conditions, event streams, metrics/logs, container exit codes, and cross-resource dependencies.
3. Identify the exact root cause of any issue.
4. Formulate a clear diagnostic explanation for the user.
5. If a fix requires modifying the cluster (e.g., k8s_patch_resource, k8s_restart_pod, k8s_scale_deployment), DO NOT run the mutating tool directly. Instead, pause execution, explain the diagnosis, and present the exact remediation action for user permission.`

var mutatingPrefixes = []string{"k8s_patch", "k8s_restart", "k8s_scale", "k8s_delete", "k8s_create", "k8s_apply", "k8s_update"}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

// OpenAI API Data Structures for Tool Calling
type ChatMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type ChatCompletionRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	Tools    []OpenAITool  `json:"tools,omitempty"`
}

type OpenAITool struct {
	Type     string         `json:"type"`
	Function OpenAIFunction `json:"function"`
}

type OpenAIFunction struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

type ChatCompletionResponse struct {
	Choices []struct {
		Message      ChatMessage `json:"message"`
		FinishReason string      `json:"finish_reason"`
	} `json:"choices"`
}

// ProposedAction represents an action awaiting human approval
type ProposedAction struct {
	ToolName    string                 `json:"tool_name"`
	Arguments   map[string]interface{} `json:"arguments"`
	Description string                 `json:"description"`
}

type AgentSession struct {
	mu                  sync.Mutex
	mcpClient           *mcp.ClientSession
	conversationHistory []ChatMessage
	systemPrompt        string
	pendingAction       *ProposedAction
	httpClient          *http.Client
}

func NewAgentSession(mcpSession *mcp.ClientSession, systemPrompt string) *AgentSession {
	if systemPrompt == "" {
		systemPrompt = defaultSystemPrompt
	}
	return &AgentSession{
		mcpClient:  mcpSession,
		httpClient: &http.Client{Timeout: 60 * time.Second},
		conversationHistory: []ChatMessage{
			{Role: "system", Content: systemPrompt},
		},
		systemPrompt: systemPrompt,
	}
}

func isMutatingTool(name string) bool {
	for _, p := range mutatingPrefixes {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}

// ensureMCPClient connects or reconnects to MCP Server over SSE if not already connected
func (a *AgentSession) ensureMCPClient(ctx context.Context) error {
	if a.mcpClient != nil {
		return nil
	}
	transport := mcp.NewSSEClientTransport(mcpServerURL, nil)
	client := mcp.NewClient("k8s-troubleshooter-agent", "1.0.0", nil)
	session, err := client.Connect(ctx, transport)
	if err != nil {
		return fmt.Errorf("failed connecting to MCP server at %s: %w", mcpServerURL, err)
	}
	a.mcpClient = session
	log.Printf("Successfully connected to MCP Server at %s", mcpServerURL)
	return nil
}

// fetchOpenAITools dynamically queries the MCP server over SSE and converts tool definitions to OpenAI format
func (a *AgentSession) fetchOpenAITools(ctx context.Context) ([]OpenAITool, error) {
	if err := a.ensureMCPClient(ctx); err != nil {
		return nil, err
	}

	toolList, err := a.mcpClient.ListTools(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed fetching tools from MCP server: %w", err)
	}

	var openaiTools []OpenAITool
	for _, t := range toolList.Tools {
		openaiTools = append(openaiTools, OpenAITool{
			Type: "function",
			Function: OpenAIFunction{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.InputSchema,
			},
		})
	}
	return openaiTools, nil
}

func (a *AgentSession) RunTroubleshootingLoop(ctx context.Context, userQuery string) (map[string]interface{}, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.conversationHistory = append(a.conversationHistory, ChatMessage{
		Role:    "user",
		Content: userQuery,
	})

	// Fetch dynamic tools exposed by Kubernetes MCP Server
	tools, err := a.fetchOpenAITools(ctx)
	if err != nil {
		log.Printf("Warning: Could not fetch MCP tools: %v", err)
	}

	maxIterations := 8
	for i := 0; i < maxIterations; i++ {
		log.Printf("[Iteration %d] Querying Qwen LLM at %s...", i+1, llmAPIBase)

		reqBody := ChatCompletionRequest{
			Model:    llmModel,
			Messages: a.conversationHistory,
			Tools:    tools,
		}

		payload, err := json.Marshal(reqBody)
		if err != nil {
			return nil, fmt.Errorf("failed marshaling LLM request: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, "POST", llmAPIBase+"/chat/completions", bytes.NewBuffer(payload))
		if err != nil {
			return nil, fmt.Errorf("failed creating LLM request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := a.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("LLM API request error: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("LLM API error (status %d): %s", resp.StatusCode, string(body))
		}

		var compResp ChatCompletionResponse
		if err := json.NewDecoder(resp.Body).Decode(&compResp); err != nil {
			return nil, fmt.Errorf("failed decoding LLM response: %w", err)
		}

		if len(compResp.Choices) == 0 {
			return nil, fmt.Errorf("empty choice array returned by LLM")
		}

		choice := compResp.Choices[0]
		a.conversationHistory = append(a.conversationHistory, choice.Message)

		// 1. Process structured tool calls (Qwen native support)
		if len(choice.Message.ToolCalls) > 0 {
			for _, toolCall := range choice.Message.ToolCalls {
				fnName := toolCall.Function.Name
				argsRaw := toolCall.Function.Arguments

				var args map[string]interface{}
				if err := json.Unmarshal([]byte(argsRaw), &args); err != nil {
					log.Printf("Error unmarshaling tool args for %s: %v", fnName, err)
					args = make(map[string]interface{})
				}

				// Check if tool is mutating (requires Human Permission Gate)
				if isMutatingTool(fnName) {
					log.Printf("⚠️ Mutating tool intercepted: %s. Pausing for human authorization.", fnName)
					a.pendingAction = &ProposedAction{
						ToolName:    fnName,
						Arguments:   args,
						Description: fmt.Sprintf("Execute mutating operation %s with arguments: %s", fnName, argsRaw),
					}
					return map[string]interface{}{
						"status":          "AWAITING_PERMISSION",
						"diagnosis":       choice.Message.Content,
						"proposed_action": a.pendingAction,
					}, nil
				}

				// Execute read-only diagnostics via MCP Server
				log.Printf("Executing read diagnostic tool: %s (%s)", fnName, argsRaw)
				if err := a.ensureMCPClient(ctx); err != nil {
					log.Printf("Failed ensuring MCP client: %v", err)
					a.conversationHistory = append(a.conversationHistory, ChatMessage{
						Role:       "tool",
						ToolCallID: toolCall.ID,
						Content:    fmt.Sprintf("MCP Connection Error: %v", err),
					})
					continue
				}

				callParams := &mcp.CallToolParams{
					Name:      fnName,
					Arguments: args,
				}

				toolResult, err := a.mcpClient.CallTool(ctx, callParams)
				var contentStr string
				if err != nil {
					contentStr = fmt.Sprintf("Tool call error: %v", err)
				} else if len(toolResult.Content) > 0 {
					contentBytes, _ := json.Marshal(toolResult.Content)
					contentStr = string(contentBytes)
				} else {
					contentStr = "{}"
				}

				// Append tool result to conversation history
				a.conversationHistory = append(a.conversationHistory, ChatMessage{
					Role:       "tool",
					ToolCallID: toolCall.ID,
					Content:    contentStr,
				})
			}
			continue
		}

		// 2. No tool calls requested: final diagnostic analysis complete
		return map[string]interface{}{
			"status":    "COMPLETED",
			"diagnosis": choice.Message.Content,
		}, nil
	}

	return map[string]interface{}{
		"status": "MAX_ITERATIONS_REACHED",
		"error":  "Diagnostic troubleshooting exceeded maximum iterations",
	}, nil
}

// ExecuteApprovedAction executes a previously intercepted mutating tool call with approval token
func (a *AgentSession) ExecuteApprovedAction(ctx context.Context, action ProposedAction) (map[string]interface{}, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if err := a.ensureMCPClient(ctx); err != nil {
		return nil, err
	}

	// Inject approval token to satisfy security gate
	action.Arguments["approval_token"] = "USER_APPROVED_TOKEN"

	log.Printf("🚀 Executing authorized remediation action: %s", action.ToolName)
	callParams := &mcp.CallToolParams{
		Name:      action.ToolName,
		Arguments: action.Arguments,
	}

	toolResult, err := a.mcpClient.CallTool(ctx, callParams)
	if err != nil {
		return nil, fmt.Errorf("remediation tool execution failed: %w", err)
	}

	a.pendingAction = nil
	return map[string]interface{}{
		"status": "REMEDIATED",
		"result": toolResult,
	}, nil
}

// REST API Request/Response Types
type QueryRequest struct {
	Query        string `json:"query"`
	SystemPrompt string `json:"system_prompt,omitempty"`
}

type ApproveRequest struct {
	ToolName  string                 `json:"tool_name"`
	Arguments map[string]interface{} `json:"arguments"`
}

func main() {
	ctx := context.Background()
	log.Printf("Starting Sandboxed Kubernetes Troubleshooting Agent...")
	log.Printf("Config: MCP_SERVER_URL=%s, LLM_API_BASE=%s, LLM_MODEL=%s", mcpServerURL, llmAPIBase, llmModel)

	// 1. Establish SSE Client Transport to MCP Server
	transport := mcp.NewSSEClientTransport(mcpServerURL, nil)
	client := mcp.NewClient("k8s-troubleshooter-agent", "1.0.0", nil)

	log.Printf("Connecting to MCP Server over SSE at %s...", mcpServerURL)
	mcpSession, err := client.Connect(ctx, transport)
	if err != nil {
		log.Printf("Warning: Initial connection to MCP Server failed: %v. Agent will retry on queries.", err)
	} else {
		log.Printf("Connected to MCP Server successfully.")
	}

	agentSession := NewAgentSession(mcpSession, defaultSystemPrompt)

	// 2. Register REST API Endpoints for Web UI & External Triggers
	http.HandleFunc("/api/query", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req QueryRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request JSON: "+err.Error(), http.StatusBadRequest)
			return
		}

		if req.SystemPrompt != "" {
			agentSession.systemPrompt = req.SystemPrompt
		}

		resp, err := agentSession.RunTroubleshootingLoop(r.Context(), req.Query)
		if err != nil {
			log.Printf("Troubleshooting error: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	http.HandleFunc("/api/approve", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req ApproveRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request JSON: "+err.Error(), http.StatusBadRequest)
			return
		}

		action := ProposedAction{
			ToolName:  req.ToolName,
			Arguments: req.Arguments,
		}

		resp, err := agentSession.ExecuteApprovedAction(r.Context(), action)
		if err != nil {
			log.Printf("Approval execution error: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// 3. CLI Mode support for direct execution inside pod
	if len(os.Args) > 1 {
		cliQuery := strings.Join(os.Args[1:], " ")
		log.Printf("Running in CLI mode with query: %s", cliQuery)
		res, err := agentSession.RunTroubleshootingLoop(ctx, cliQuery)
		if err != nil {
			log.Fatalf("CLI execution failed: %v", err)
		}
		resBytes, _ := json.MarshalIndent(res, "", "  ")
		fmt.Println(string(resBytes))
		return
	}

	// 4. Start HTTP Server for Web UI integration
	addr := ":" + port
	log.Printf("Agent REST API server listening on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
