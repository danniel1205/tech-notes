package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	clientset     kubernetes.Interface
	dynamicClient dynamic.Interface
)

func initK8sClient() error {
	config, err := rest.InClusterConfig()
	if err != nil {
		kubeconfig := os.Getenv("KUBECONFIG")
		if kubeconfig == "" {
			kubeconfig = os.Getenv("HOME") + "/.kube/config"
		}
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return fmt.Errorf("could not load kubeconfig: %w", err)
		}
	}

	cs, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("could not create kubernetes clientset: %w", err)
	}
	clientset = cs

	dyn, err := dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("could not create dynamic client: %w", err)
	}
	dynamicClient = dyn

	return nil
}

// =====================================================================
// SCALABLE DYNAMIC INPUT STRUCTS
// =====================================================================

type GetResourceInput struct {
	Group     string `json:"group"`
	Version   string `json:"version"`
	Resource  string `json:"resource"`
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
}

type ListResourcesInput struct {
	Group         string `json:"group"`
	Version       string `json:"version"`
	Resource      string `json:"resource"`
	Namespace     string `json:"namespace,omitempty"`
	LabelSelector string `json:"label_selector,omitempty"`
}

type DescribeResourceInput struct {
	Group     string `json:"group"`
	Version   string `json:"version"`
	Resource  string `json:"resource"`
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
}

type GetPodLogsInput struct {
	PodName       string `json:"pod_name"`
	Namespace     string `json:"namespace,omitempty"`
	ContainerName string `json:"container_name,omitempty"`
	TailLines     int64  `json:"tail_lines,omitempty"`
}

type PatchResourceInput struct {
	Group         string `json:"group"`
	Version       string `json:"version"`
	Resource      string `json:"resource"`
	Name          string `json:"name"`
	Namespace     string `json:"namespace,omitempty"`
	PatchJSON     string `json:"patch_json"`
	ApprovalToken string `json:"approval_token"`
}

func main() {
	log.Println("Starting Scalable Dynamic Kubernetes MCP Server with Official Go SDK...")
	if err := initK8sClient(); err != nil {
		log.Printf("Warning: K8s client init error: %v", err)
	}

	// 1. Initialize MCP Server
	s := mcp.NewServer("k8s-mcp-server-dynamic", "2.1.0", nil)

	// 2. Register Generic Primitive Tools
	s.AddTools(
		mcp.NewServerTool("k8s_get_resource", "Generic tool to fetch ANY Kubernetes resource spec and status.", handleGetResource),
		mcp.NewServerTool("k8s_list_resources", "Generic tool to list ANY Kubernetes resource type in a namespace.", handleListResources),
		mcp.NewServerTool("k8s_describe_resource", "Generic tool providing a comprehensive summary of ANY resource including related events.", handleDescribeResource),
		mcp.NewServerTool("k8s_get_pod_logs", "Specialized tool to retrieve container logs for a pod.", handleGetPodLogs),
		mcp.NewServerTool("k8s_patch_resource", "Generic mutating tool to patch ANY Kubernetes resource. REQUIRES approval_token.", handlePatchResource),
	)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// 3. Create SSE Handler
	handler := mcp.NewSSEHandler(func(r *http.Request) *mcp.Server {
		return s
	})

	log.Printf("Listening on SSE server http://0.0.0.0:%s...", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// =====================================================================
// DYNAMIC GENERIC HANDLERS (Official Go SDK v0.1.0 Signatures)
// =====================================================================

func handleGetResource(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[GetResourceInput]) (*mcp.CallToolResultFor[any], error) {
	in := params.Arguments
	namespace := in.Namespace
	if namespace == "" {
		namespace = "default"
	}

	gvr := schema.GroupVersionResource{
		Group:    in.Group,
		Version:  in.Version,
		Resource: in.Resource,
	}

	res, err := dynamicClient.Resource(gvr).Namespace(namespace).Get(ctx, in.Name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed fetching %s/%s '%s': %w", in.Resource, in.Group, in.Name, err)
	}

	bytes, _ := json.MarshalIndent(res.Object, "", "  ")
	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(bytes)},
		},
	}, nil
}

func handleListResources(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[ListResourcesInput]) (*mcp.CallToolResultFor[any], error) {
	in := params.Arguments
	namespace := in.Namespace
	if namespace == "" {
		namespace = "default"
	}

	gvr := schema.GroupVersionResource{
		Group:    in.Group,
		Version:  in.Version,
		Resource: in.Resource,
	}

	opts := metav1.ListOptions{}
	if in.LabelSelector != "" {
		opts.LabelSelector = in.LabelSelector
	}

	list, err := dynamicClient.Resource(gvr).Namespace(namespace).List(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed listing %s/%s: %w", in.Resource, in.Group, err)
	}

	bytes, _ := json.MarshalIndent(list.Object, "", "  ")
	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(bytes)},
		},
	}, nil
}

func handleDescribeResource(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[DescribeResourceInput]) (*mcp.CallToolResultFor[any], error) {
	in := params.Arguments
	namespace := in.Namespace
	if namespace == "" {
		namespace = "default"
	}

	gvr := schema.GroupVersionResource{
		Group:    in.Group,
		Version:  in.Version,
		Resource: in.Resource,
	}

	res, err := dynamicClient.Resource(gvr).Namespace(namespace).Get(ctx, in.Name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed describing %s/%s '%s': %w", in.Resource, in.Group, in.Name, err)
	}

	events, _ := clientset.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{})
	type EventSummary struct {
		Type    string `json:"type"`
		Reason  string `json:"reason"`
		Message string `json:"message"`
		Count   int32  `json:"count"`
	}
	var relatedEvents []EventSummary
	if events != nil {
		for _, e := range events.Items {
			if e.InvolvedObject.Name == in.Name {
				relatedEvents = append(relatedEvents, EventSummary{
					Type:    e.Type,
					Reason:  e.Reason,
					Message: e.Message,
					Count:   e.Count,
				})
			}
		}
	}

	summary := map[string]interface{}{
		"kind":           res.GetKind(),
		"name":           res.GetName(),
		"namespace":      res.GetNamespace(),
		"labels":         res.GetLabels(),
		"object_details": res.Object,
		"related_events": relatedEvents,
	}

	bytes, _ := json.MarshalIndent(summary, "", "  ")
	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(bytes)},
		},
	}, nil
}

func handleGetPodLogs(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[GetPodLogsInput]) (*mcp.CallToolResultFor[any], error) {
	in := params.Arguments
	namespace := in.Namespace
	if namespace == "" {
		namespace = "default"
	}
	tailLines := in.TailLines
	if tailLines == 0 {
		tailLines = 100
	}

	opts := &corev1.PodLogOptions{TailLines: &tailLines}
	if in.ContainerName != "" {
		opts.Container = in.ContainerName
	}

	reqLog := clientset.CoreV1().Pods(namespace).GetLogs(in.PodName, opts)
	stream, err := reqLog.Stream(ctx)
	if err != nil {
		return nil, fmt.Errorf("error streaming logs for %s: %w", in.PodName, err)
	}
	defer stream.Close()

	buf, err := io.ReadAll(stream)
	if err != nil {
		return nil, fmt.Errorf("error reading log stream: %w", err)
	}

	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(buf)},
		},
	}, nil
}

func handlePatchResource(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[PatchResourceInput]) (*mcp.CallToolResultFor[any], error) {
	in := params.Arguments
	if in.ApprovalToken == "" {
		return nil, fmt.Errorf("permission error: approval_token required to execute k8s_patch_resource")
	}

	namespace := in.Namespace
	if namespace == "" {
		namespace = "default"
	}

	gvr := schema.GroupVersionResource{
		Group:    in.Group,
		Version:  in.Version,
		Resource: in.Resource,
	}

	patched, err := dynamicClient.Resource(gvr).Namespace(namespace).Patch(
		ctx,
		in.Name,
		types.MergePatchType,
		[]byte(in.PatchJSON),
		metav1.PatchOptions{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed patching %s '%s': %w", in.Resource, in.Name, err)
	}

	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("Successfully patched %s/%s '%s'. New resource version: %s", in.Resource, in.Group, in.Name, patched.GetResourceVersion())},
		},
	}, nil
}
