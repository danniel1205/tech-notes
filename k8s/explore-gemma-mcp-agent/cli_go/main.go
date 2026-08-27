package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// ANSI terminal color codes
const (
	colorReset  = "\033[0m"
	colorBold   = "\033[1m"
	colorDim    = "\033[2m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
)

type Config struct {
	Kubeconfig  string
	Namespace   string
	ServiceName string
	ServicePort int
}

func (cfg *Config) Validate() error {
	if strings.TrimSpace(cfg.Namespace) == "" {
		return fmt.Errorf("mandatory configuration 'namespace' is missing (specify via --namespace / -n or AGENT_NAMESPACE)")
	}
	if strings.TrimSpace(cfg.ServiceName) == "" {
		return fmt.Errorf("mandatory configuration 'service' is missing (specify via --service / -s or AGENT_SERVICE_NAME)")
	}
	if cfg.ServicePort <= 0 || cfg.ServicePort > 65535 {
		return fmt.Errorf("mandatory configuration 'port' must be a valid port between 1 and 65535 (specify via --port / -p or AGENT_SERVICE_PORT)")
	}
	return nil
}

type ProposedAction struct {
	ToolName    string                 `json:"tool_name"`
	Arguments   map[string]interface{} `json:"arguments"`
	Description string                 `json:"description"`
}

type QueryRequest struct {
	Query        string `json:"query"`
	SystemPrompt string `json:"system_prompt,omitempty"`
}

type QueryResponse struct {
	Status         string          `json:"status"`
	Diagnosis      string          `json:"diagnosis,omitempty"`
	ProposedAction *ProposedAction `json:"proposed_action,omitempty"`
	Message        string          `json:"message,omitempty"`
	Result         interface{}     `json:"result,omitempty"`
	Error          string          `json:"error,omitempty"`
}

type ApproveRequest struct {
	ToolName  string                 `json:"tool_name"`
	Arguments map[string]interface{} `json:"arguments"`
}

type ApproveResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type AgentClient struct {
	restClient  rest.Interface
	namespace   string
	serviceName string
	port        int
}

func NewAgentClient(cfg Config) (*AgentClient, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if cfg.Kubeconfig != "" {
		loadingRules.ExplicitPath = cfg.Kubeconfig
	}
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules, &clientcmd.ConfigOverrides{},
	).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed loading kubeconfig: %w", err)
	}

	config.GroupVersion = &schema.GroupVersion{Group: "", Version: "v1"}
	config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()
	restClient, err := rest.RESTClientFor(config)
	if err != nil {
		return nil, fmt.Errorf("failed creating K8s REST client: %w", err)
	}

	return &AgentClient{
		restClient:  restClient,
		namespace:   cfg.Namespace,
		serviceName: cfg.ServiceName,
		port:        cfg.ServicePort,
	}, nil
}

func (c *AgentClient) post(ctx context.Context, endpoint string, reqData interface{}) ([]byte, error) {
	bodyBytes, err := json.Marshal(reqData)
	if err != nil {
		return nil, fmt.Errorf("failed encoding request body: %w", err)
	}

	proxyPath := fmt.Sprintf("/api/v1/namespaces/%s/services/%s:%d/proxy/%s",
		c.namespace, c.serviceName, c.port, strings.TrimPrefix(endpoint, "/"))

	result := c.restClient.Post().
		AbsPath(proxyPath).
		SetHeader("Content-Type", "application/json").
		Body(bodyBytes).
		Do(ctx)

	raw, _ := result.Raw()
	if err := result.Error(); err != nil {
		if len(raw) > 0 {
			var errResp map[string]interface{}
			if jsonErr := json.Unmarshal(raw, &errResp); jsonErr == nil {
				if msg, ok := errResp["error"].(string); ok && msg != "" {
					return nil, fmt.Errorf("%s", msg)
				}
			}
			return nil, fmt.Errorf("%s", string(raw))
		}
		return nil, fmt.Errorf("request to %s failed: %w", proxyPath, err)
	}

	return raw, nil
}

func (c *AgentClient) SendQuery(ctx context.Context, query, systemPrompt string) (*QueryResponse, error) {
	raw, err := c.post(ctx, "api/query", QueryRequest{
		Query:        query,
		SystemPrompt: systemPrompt,
	})
	if err != nil {
		return nil, err
	}

	var res QueryResponse
	if err := json.Unmarshal(raw, &res); err != nil {
		return nil, fmt.Errorf("failed parsing agent response JSON: %w", err)
	}
	return &res, nil
}

func (c *AgentClient) SendApprove(ctx context.Context, toolName string, args map[string]interface{}) (*ApproveResponse, error) {
	raw, err := c.post(ctx, "api/approve", ApproveRequest{
		ToolName:  toolName,
		Arguments: args,
	})
	if err != nil {
		return nil, err
	}

	var aRes ApproveResponse
	if err := json.Unmarshal(raw, &aRes); err != nil {
		return nil, fmt.Errorf("failed parsing approval response JSON: %w", err)
	}
	return &aRes, nil
}

func getKubectlContext() string {
	cmd := exec.Command("kubectl", "config", "current-context")
	out, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

func printBanner(namespace, serviceName string, port int, k8sContext string) {
	targetInfo := fmt.Sprintf("%s/%s:%d", namespace, serviceName, port)
	fmt.Printf("%s%s", colorCyan, colorBold)
	fmt.Println("╔══════════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║  ☸️  Autonomous Kubernetes Troubleshooting Agent (kubectl-agent)                  ║")
	fmt.Printf("║  Target Service: %-26s | Context: %-26s ║\n", targetInfo, k8sContext)
	fmt.Println("╚══════════════════════════════════════════════════════════════════════════════════╝")
	fmt.Printf("%s\n", colorReset)
}

func printHelp() {
	fmt.Println("Available commands:")
	fmt.Printf("  %shelp%s           Show this help message\n", colorBold, colorReset)
	fmt.Printf("  %scontext%s        Show the active kubectl context\n", colorBold, colorReset)
	fmt.Printf("  %sclear%s          Clear the terminal screen\n", colorBold, colorReset)
	fmt.Printf("  %sexit%s / %squit%s   Exit the interactive agent session\n\n", colorBold, colorReset, colorBold, colorReset)
	fmt.Println("You can enter any natural language troubleshooting or operational request.")
	fmt.Println("Example: 'Check for crashlooping pods in default and explain why they are failing'")
}

func formatJSON(data interface{}) string {
	b, err := json.MarshalIndent(data, "  ", "  ")
	if err != nil {
		return fmt.Sprintf("%v", data)
	}
	return string(b)
}

func executeQueryWorkflow(ctx context.Context, client *AgentClient, query, systemPrompt string, autoApprove bool, inReader *bufio.Reader) error {
	fmt.Printf("\n%s🔍 Investigating cluster state via MCP Server...%s\n", colorCyan, colorReset)

	resp, err := client.SendQuery(ctx, query, systemPrompt)
	if err != nil {
		fmt.Printf("%s❌ Error communicating with Agent:%s %v\n", colorRed, colorReset, err)
		return err
	}

	if resp.Error != "" {
		fmt.Printf("%s❌ Agent Error:%s %s\n", colorRed, colorReset, resp.Error)
		return fmt.Errorf("%s", resp.Error)
	}

	if resp.Diagnosis != "" {
		fmt.Printf("\n%s📋 Diagnosis & Analysis:%s\n%s\n", colorBold, colorReset, resp.Diagnosis)
	}

	if resp.Status == "AWAITING_PERMISSION" && resp.ProposedAction != nil {
		action := resp.ProposedAction
		fmt.Printf("\n%s%s──────────────────────────────────────────────────────────────────────────────%s\n", colorYellow, colorBold, colorReset)
		fmt.Printf("%s⚠️  PROPOSED REMEDIATION ACTION (Human Authorization Gate)%s\n", colorYellow, colorReset)
		fmt.Printf("  %sTool:%s      %s\n", colorBold, colorReset, action.ToolName)
		if action.Description != "" {
			fmt.Printf("  %sSummary:%s   %s\n", colorBold, colorReset, action.Description)
		}

		if patchJSON, ok := action.Arguments["patch_json"].(string); ok && patchJSON != "" {
			var parsedPatch interface{}
			if err := json.Unmarshal([]byte(patchJSON), &parsedPatch); err == nil {
				fmt.Printf("  %sPatch JSON:%s\n  %s\n", colorBold, colorReset, formatJSON(parsedPatch))
			} else {
				fmt.Printf("  %sPatch JSON:%s %s\n", colorBold, colorReset, patchJSON)
			}
		} else {
			fmt.Printf("  %sArguments:%s\n  %s\n", colorBold, colorReset, formatJSON(action.Arguments))
		}
		fmt.Printf("%s%s──────────────────────────────────────────────────────────────────────────────%s\n", colorYellow, colorBold, colorReset)

		approved := autoApprove
		if !autoApprove {
			fmt.Printf("\n%s%sApprove and execute this fix on the cluster? [y/N]: %s", colorBold, colorGreen, colorReset)
			input, _ := inReader.ReadString('\n')
			input = strings.TrimSpace(strings.ToLower(input))
			if input == "y" || input == "yes" {
				approved = true
			}
		}

		if approved {
			fmt.Printf("\n%s🚀 Applying approved fix to cluster...%s\n", colorCyan, colorReset)
			aRes, err := client.SendApprove(ctx, action.ToolName, action.Arguments)
			if err != nil {
				fmt.Printf("%s❌ Execution failed:%s %v\n", colorRed, colorReset, err)
				return err
			}
			if aRes.Status == "ERROR" {
				fmt.Printf("%s❌ Remediation Error:%s %s\n", colorRed, colorReset, aRes.Error)
			} else {
				msg := aRes.Message
				if msg == "" && aRes.Result != nil {
					msg = fmt.Sprintf("%v", aRes.Result)
				}
				if msg == "" {
					msg = "Remediation executed successfully."
				}
				fmt.Printf("%s✅ Remediation Successful!%s\n%s\n", colorGreen, colorReset, msg)
			}
		} else {
			fmt.Printf("%s❌ Remediation cancelled by user.%s\n", colorYellow, colorReset)
		}
	} else if resp.Status == "REMEDIATED" {
		fmt.Printf("\n%s✅ Action successfully completed!%s\n", colorGreen, colorReset)
	}

	return nil
}

func runREPL(ctx context.Context, client *AgentClient, systemPrompt string, autoApprove bool) {
	k8sCtx := getKubectlContext()
	printBanner(client.namespace, client.serviceName, client.port, k8sCtx)

	fmt.Println("Type your question or operational goal. Type 'help' for options, 'exit' to quit.")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("%s%sk8s-agent❯%s ", colorBold, colorCyan, colorReset)
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		switch strings.ToLower(line) {
		case "exit", "quit", "q":
			fmt.Println("Goodbye! 👋")
			return
		case "help", "h", "?":
			printHelp()
			continue
		case "context":
			fmt.Printf("Active kubectl context: %s%s%s\n", colorBold, getKubectlContext(), colorReset)
			continue
		case "clear":
			fmt.Print("\033[H\033[2J")
			printBanner(client.namespace, client.serviceName, client.port, getKubectlContext())
			continue
		}

		_ = executeQueryWorkflow(ctx, client, line, systemPrompt, autoApprove, reader)
		fmt.Println()
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("%s❌ Input reading error:%s %v\n", colorRed, colorReset, err)
	}
}

func main() {
	var cfg Config

	// 1. Read flags with environment variable defaults
	envNamespace := os.Getenv("AGENT_NAMESPACE")
	envService := os.Getenv("AGENT_SERVICE_NAME")
	envPort := 0
	if p, err := strconv.Atoi(os.Getenv("AGENT_SERVICE_PORT")); err == nil {
		envPort = p
	}

	flag.StringVar(&cfg.Namespace, "n", envNamespace, "Target Kubernetes namespace (mandatory, or AGENT_NAMESPACE)")
	flag.StringVar(&cfg.Namespace, "namespace", envNamespace, "Target Kubernetes namespace (mandatory, or AGENT_NAMESPACE)")
	flag.StringVar(&cfg.ServiceName, "s", envService, "Target Agent service name (mandatory, or AGENT_SERVICE_NAME)")
	flag.StringVar(&cfg.ServiceName, "service", envService, "Target Agent service name (mandatory, or AGENT_SERVICE_NAME)")
	flag.IntVar(&cfg.ServicePort, "p", envPort, "Target Agent service port (mandatory, or AGENT_SERVICE_PORT)")
	flag.IntVar(&cfg.ServicePort, "port", envPort, "Target Agent service port (mandatory, or AGENT_SERVICE_PORT)")
	flag.StringVar(&cfg.Kubeconfig, "kubeconfig", os.Getenv("KUBECONFIG"), "Path to kubeconfig file (optional)")

	interactive := flag.Bool("i", false, "Launch interactive multi-turn REPL mode")
	interactiveLong := flag.Bool("interactive", false, "Launch interactive multi-turn REPL mode")
	autoApprove := flag.Bool("y", false, "Automatically approve proposed mutating actions without prompting")
	autoApproveLong := flag.Bool("yes", false, "Automatically approve proposed mutating actions without prompting")
	systemPrompt := flag.String("system-prompt", "", "Custom system prompt override for the troubleshooting session")

	flag.Usage = func() {
		fmt.Printf("%sAutonomous Kubernetes Troubleshooting Agent CLI & kubectl plugin%s\n\n", colorBold, colorReset)
		fmt.Println("Usage:")
		fmt.Println("  kubectl agent -n <namespace> -s <service> -p <port> [flags] [query]")
		fmt.Println("  k8s-agent-cli -n <namespace> -s <service> -p <port> [flags] [query]")
		fmt.Println("\nMandatory Connection Parameters:")
		fmt.Println("  -n, --namespace string   Target Kubernetes namespace (or AGENT_NAMESPACE)")
		fmt.Println("  -s, --service string     Target Agent service name (or AGENT_SERVICE_NAME)")
		fmt.Println("  -p, --port int           Target Agent service port (or AGENT_SERVICE_PORT)")
		fmt.Println("\nOptional Flags:")
		flag.PrintDefaults()
	}

	flag.Parse()

	// 2. Validate mandatory connection configuration
	if err := cfg.Validate(); err != nil {
		fmt.Printf("%s❌ Configuration Error:%s %v\n\n", colorRed, colorReset, err)
		flag.Usage()
		os.Exit(1)
	}

	// 3. Initialize AgentClient with Kubernetes API Proxy
	client, err := NewAgentClient(cfg)
	if err != nil {
		fmt.Printf("%s❌ Connection Initialization Error:%s %v\n", colorRed, colorReset, err)
		os.Exit(1)
	}

	ctx := context.Background()
	isInteractive := *interactive || *interactiveLong
	isAutoApprove := *autoApprove || *autoApproveLong

	// 4. If positional arguments were provided, execute one-shot query
	args := flag.Args()
	if len(args) > 0 {
		query := strings.Join(args, " ")
		reader := bufio.NewReader(os.Stdin)
		if err := executeQueryWorkflow(ctx, client, query, *systemPrompt, isAutoApprove, reader); err != nil {
			os.Exit(1)
		}
		return
	}

	// 5. Default to interactive REPL mode
	_ = isInteractive
	runREPL(ctx, client, *systemPrompt, isAutoApprove)
}
