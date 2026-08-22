package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	clientset     kubernetes.Interface
	dynamicClient dynamic.Interface
	restMapper    meta.RESTMapper
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

	// Initialize Dynamic Discovery Client & RESTMapper for 100% generic resource & CRD resolution
	dc, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return fmt.Errorf("could not create discovery client: %w", err)
	}
	cachedDC := memory.NewMemCacheClient(dc)
	restMapper = restmapper.NewDeferredDiscoveryRESTMapper(cachedDC)

	return nil
}

// =====================================================================
// SCALABLE DYNAMIC INPUT STRUCTS
// =====================================================================

type GetResourceInput struct {
	Group     string `json:"group,omitempty"`
	Version   string `json:"version,omitempty"`
	Resource  string `json:"resource"`
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
}

type ListResourcesInput struct {
	Group         string `json:"group,omitempty"`
	Version       string `json:"version,omitempty"`
	Resource      string `json:"resource"`
	Namespace     string `json:"namespace,omitempty"`
	LabelSelector string `json:"label_selector,omitempty"`
}

type DescribeResourceInput struct {
	Group     string `json:"group,omitempty"`
	Version   string `json:"version,omitempty"`
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
	Group         string `json:"group,omitempty"`
	Version       string `json:"version,omitempty"`
	Resource      string `json:"resource"`
	Name          string `json:"name"`
	Namespace     string `json:"namespace,omitempty"`
	PatchJSON     string `json:"patch_json"`
	ApprovalToken string `json:"approval_token,omitempty"`
}

// resolveGVR dynamically resolves any Kubernetes resource, CRD, singular/plural, or shortname to its GroupVersionResource.
// It supports multiple calling conventions used by LLMs and human operators:
// 1. Standard separate fields: group="apps", version="v1", resource="deployments"
// 2. Combined apiVersion string: version="apps/v1", resource="deployments"
// 3. Simple resource name: resource="deployments" (automatically discovered via RESTMapper)
func resolveGVR(group, version, resource string) schema.GroupVersionResource {
	resName := strings.TrimSpace(resource)

	// Normalize apiVersion format: if the model/caller supplied a combined "group/version" string in the
	// version field (e.g. "apps/v1" or "networking.k8s.io/v1"), split it into distinct Group and Version components.
	if strings.Contains(version, "/") {
		parts := strings.SplitN(version, "/", 2)
		if group == "" {
			group = parts[0]
		}
		version = parts[1]
	}

	// Fast-path: if group, version, and resource are all explicitly provided, use them directly
	if group != "" && version != "" && resName != "" {
		return schema.GroupVersionResource{
			Group:    group,
			Version:  version,
			Resource: resName,
		}
	}

	// Dynamic Discovery via client-go RESTMapper:
	// Queries live cluster discovery endpoints to resolve built-in resources, custom CRDs,
	// shortnames (e.g. "po", "deploy", "svc"), singular names ("deployment"), and plural forms.
	if restMapper != nil {
		gvr, _ := schema.ParseResourceArg(resName)
		if gvr == nil {
			gvr = &schema.GroupVersionResource{Group: group, Version: version, Resource: resName}
		} else {
			if group != "" {
				gvr.Group = group
			}
			if version != "" {
				gvr.Version = version
			}
		}

		// 1. Resolve partial resource arg or shortname across all groups (e.g. "deployments" or "deploy" -> apps/v1/deployments)
		if gvrs, err := restMapper.ResourcesFor(*gvr); err == nil && len(gvrs) > 0 {
			return gvrs[0]
		}

		if fullyQualifiedGVR, err := restMapper.ResourceFor(*gvr); err == nil {
			return fullyQualifiedGVR
		}

		// 2. Resolve by Kind (e.g. "Deployment" -> apps/v1/deployments)
		if gvks, err := restMapper.KindsFor(*gvr); err == nil && len(gvks) > 0 {
			if mapping, err := restMapper.RESTMapping(gvks[0].GroupKind(), gvks[0].Version); err == nil && mapping != nil {
				return mapping.Resource
			}
		}
		if gvk, err := restMapper.KindFor(*gvr); err == nil {
			if mapping, err := restMapper.RESTMapping(gvk.GroupKind(), gvk.Version); err == nil && mapping != nil {
				return mapping.Resource
			}
		}

		// 3. Resolve by GroupKind mapping
		gk := schema.GroupKind{Group: group, Kind: resName}
		if mapping, err := restMapper.RESTMapping(gk, version); err == nil && mapping != nil {
			return mapping.Resource
		}
	}

	// Default version to "v1" if omitted and unresolved
	if version == "" {
		version = "v1"
	}

	return schema.GroupVersionResource{
		Group:    group,
		Version:  version,
		Resource: resName,
	}
}

// NewGatedServerTool creates an MCP Server Tool marked with requires_approval: true
// so that AI agents automatically gate it behind human-in-the-loop permission.
func NewGatedServerTool[In, Out any](name, description string, handler mcp.ToolHandlerFor[In, Out]) *mcp.ServerTool {
	t := mcp.NewServerTool(name, description, handler)
	t.Tool.Meta = mcp.Meta{"requires_approval": true}
	return t
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
		NewGatedServerTool("k8s_patch_resource", "Generic mutating tool to patch ANY Kubernetes resource.", handlePatchResource),
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

// cleanUnstructured removes verbose internal Kubernetes metadata fields (like managedFields
// and kubectl.kubernetes.io/last-applied-configuration) that bloat token consumption
// and exceed LLM context window limits without providing diagnostic value.
//
// 💡 FUTURE UPGRADE NOTE:
// Even with giant context models (e.g. 128k+ tokens), stripping managedFields is recommended
// because SSA fieldsets are internal Kubernetes accounting noise that increases inference latency.
// However, if you ever need full unstripped raw manifests for specialized auditing tools, you can
// bypass this function by simply removing calls to cleanUnstructured(res) or cleanUnstructured(&list.Items[i]).
func cleanUnstructured(u *unstructured.Unstructured) {
	if u == nil || u.Object == nil {
		return
	}
	unstructured.RemoveNestedField(u.Object, "metadata", "managedFields")
	if annotations := u.GetAnnotations(); annotations != nil {
		delete(annotations, "kubectl.kubernetes.io/last-applied-configuration")
		u.SetAnnotations(annotations)
	}
}

func handleGetResource(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[GetResourceInput]) (*mcp.CallToolResultFor[any], error) {
	in := params.Arguments
	namespace := in.Namespace
	if namespace == "" {
		namespace = "default"
	}

	gvr := resolveGVR(in.Group, in.Version, in.Resource)
	log.Printf("handleGetResource: in={Group:%q, Version:%q, Resource:%q, Name:%q, Namespace:%q} -> GVR={Group:%q, Version:%q, Resource:%q}",
		in.Group, in.Version, in.Resource, in.Name, namespace, gvr.Group, gvr.Version, gvr.Resource)

	res, err := dynamicClient.Resource(gvr).Namespace(namespace).Get(ctx, in.Name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed fetching %s/%s '%s': %w", gvr.Resource, gvr.Group, in.Name, err)
	}

	cleanUnstructured(res)
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

	gvr := resolveGVR(in.Group, in.Version, in.Resource)
	log.Printf("handleListResources: in={Group:%q, Version:%q, Resource:%q, Namespace:%q} -> GVR={Group:%q, Version:%q, Resource:%q}",
		in.Group, in.Version, in.Resource, namespace, gvr.Group, gvr.Version, gvr.Resource)

	opts := metav1.ListOptions{}
	if in.LabelSelector != "" {
		opts.LabelSelector = in.LabelSelector
	}

	list, err := dynamicClient.Resource(gvr).Namespace(namespace).List(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed listing %s/%s: %w", gvr.Resource, gvr.Group, err)
	}

	for i := range list.Items {
		cleanUnstructured(&list.Items[i])
	}

	bytes, _ := json.MarshalIndent(list, "", "  ")
	log.Printf("handleListResources: retrieved %d bytes for %s/%s in %s", len(bytes), gvr.Resource, gvr.Group, namespace)
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

	gvr := resolveGVR(in.Group, in.Version, in.Resource)

	res, err := dynamicClient.Resource(gvr).Namespace(namespace).Get(ctx, in.Name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed describing %s/%s '%s': %w", gvr.Resource, gvr.Group, in.Name, err)
	}
	cleanUnstructured(res)

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
			if e.InvolvedObject.Name == in.Name || strings.HasPrefix(e.InvolvedObject.Name, in.Name) {
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

// normalizePatchData ensures container arrays in workload patches include the container "name".
//
// WHY THIS FUNCTION IS NEEDED:
//
//  1. LLM Partial JSON Generation: When LLMs (e.g. Qwen, Gemma) propose remediation patches (e.g. updating an image tag),
//     they frequently omit the container "name" field, producing partial patches like:
//     {"spec": {"template": {"spec": {"containers": [{"image": "nginx:latest"}]}}}}
//
//  2. Kubernetes StrategicMergePatch Requirement: Kubernetes list-merges container arrays using the
//     patchMergeKey "name". If "name" is missing, the K8s API server rejects the patch with:
//     "ValidationError: missing required field 'name' in spec.template.spec.containers[0]".
//
//  3. Fallback MergePatch Protection: Without "name", a standard JSON Merge Patch would overwrite
//     the entire container list, unintentionally wiping out existing environment variables, ports, probes, and volume mounts.
//
// HOW IT WORKS:
// If any container entry in the patch lacks a "name", this function queries the live cluster resource
// via dynamicClient.Get(), matches the target container by index, and automatically injects the
// container's existing name before submitting the patch to the Kubernetes API.
func normalizePatchData(ctx context.Context, gvr schema.GroupVersionResource, namespace, name string, rawPatch []byte) []byte {
	var patchMap map[string]interface{}
	if err := json.Unmarshal(rawPatch, &patchMap); err != nil {
		return rawPatch
	}

	containerPaths := [][]string{
		{"spec", "template", "spec", "containers"},
		{"spec", "containers"},
	}

	for _, path := range containerPaths {
		containers, found, _ := unstructured.NestedSlice(patchMap, path...)
		if !found || len(containers) == 0 {
			continue
		}

		needsName := false
		for _, c := range containers {
			if cMap, ok := c.(map[string]interface{}); ok {
				if _, hasName := cMap["name"]; !hasName {
					needsName = true
					break
				}
			}
		}

		if needsName && dynamicClient != nil {
			if current, err := dynamicClient.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{}); err == nil {
				if currContainers, currFound, _ := unstructured.NestedSlice(current.Object, path...); currFound {
					for idx, c := range containers {
						if cMap, ok := c.(map[string]interface{}); ok {
							if _, hasName := cMap["name"]; !hasName && idx < len(currContainers) {
								if currCMap, currOk := currContainers[idx].(map[string]interface{}); currOk {
									if currName, ok := currCMap["name"].(string); ok && currName != "" {
										cMap["name"] = currName
									}
								}
							}
						}
					}
					_ = unstructured.SetNestedSlice(patchMap, containers, path...)
					if updatedBytes, err := json.Marshal(patchMap); err == nil {
						return updatedBytes
					}
				}
			}
		}
	}

	return rawPatch
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

	gvr := resolveGVR(in.Group, in.Version, in.Resource)
	patchData := normalizePatchData(ctx, gvr, namespace, in.Name, []byte(in.PatchJSON))

	// Try StrategicMergePatchType first (standard for K8s built-ins), fallback to MergePatchType (CRDs)
	patched, err := dynamicClient.Resource(gvr).Namespace(namespace).Patch(
		ctx,
		in.Name,
		types.StrategicMergePatchType,
		patchData,
		metav1.PatchOptions{},
	)
	if err != nil {
		patched, err = dynamicClient.Resource(gvr).Namespace(namespace).Patch(
			ctx,
			in.Name,
			types.MergePatchType,
			patchData,
			metav1.PatchOptions{},
		)
	}
	if err != nil {
		return nil, fmt.Errorf("failed patching %s '%s': %w", gvr.Resource, in.Name, err)
	}

	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("Successfully patched %s/%s '%s'. New resource version: %s", gvr.Resource, gvr.Group, in.Name, patched.GetResourceVersion())},
		},
	}, nil
}
