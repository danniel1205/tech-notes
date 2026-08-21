package main

import (
	"context"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
)

func setupFakeK8sEnvironment() {
	scheme := runtime.NewScheme()
	corev1.AddToScheme(scheme)

	fakePod := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata": map[string]interface{}{
				"name":      "test-pod",
				"namespace": "default",
				"labels": map[string]interface{}{
					"app": "test",
				},
			},
			"status": map[string]interface{}{
				"phase": "Running",
			},
		},
	}

	dynamicClient = dynamicfake.NewSimpleDynamicClient(scheme, fakePod)
	clientset = fake.NewSimpleClientset(&corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-event",
			Namespace: "default",
		},
		InvolvedObject: corev1.ObjectReference{
			Name: "test-pod",
			Kind: "Pod",
		},
		Type:    "Warning",
		Reason:  "FailedScheduling",
		Message: "Nodes are unavailable",
		Count:   1,
	})

	mapper := meta.NewDefaultRESTMapper([]schema.GroupVersion{
		corev1.SchemeGroupVersion,
		{Group: "apps", Version: "v1"},
		{Group: "batch", Version: "v1"},
		{Group: "networking.k8s.io", Version: "v1"},
		{Group: "custom.acme.com", Version: "v1alpha1"},
	})
	mapper.Add(schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Pod"}, meta.RESTScopeNamespace)
	mapper.Add(schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"}, meta.RESTScopeNamespace)
	mapper.Add(schema.GroupVersionKind{Group: "batch", Version: "v1", Kind: "Job"}, meta.RESTScopeNamespace)
	mapper.Add(schema.GroupVersionKind{Group: "networking.k8s.io", Version: "v1", Kind: "Ingress"}, meta.RESTScopeNamespace)
	mapper.Add(schema.GroupVersionKind{Group: "custom.acme.com", Version: "v1alpha1", Kind: "Widget"}, meta.RESTScopeNamespace)
	restMapper = mapper
}

func TestHandleGetResource(t *testing.T) {
	setupFakeK8sEnvironment()
	ctx := context.Background()

	params := &mcp.CallToolParamsFor[GetResourceInput]{
		Arguments: GetResourceInput{
			Group:     "",
			Version:   "v1",
			Resource:  "pods",
			Name:      "test-pod",
			Namespace: "default",
		},
	}

	result, err := handleGetResource(ctx, nil, params)
	if err != nil {
		t.Fatalf("handleGetResource returned error: %v", err)
	}

	if len(result.Content) == 0 {
		t.Fatalf("expected result content, got empty")
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok || textContent.Text == "" {
		t.Errorf("expected non-empty text content, got: %v", result.Content[0])
	}
}

func TestHandleListResources(t *testing.T) {
	setupFakeK8sEnvironment()
	ctx := context.Background()

	tests := []struct {
		name      string
		input     ListResourcesInput
		wantError bool
	}{
		{
			name: "list pods in default namespace",
			input: ListResourcesInput{
				Group:     "",
				Version:   "v1",
				Resource:  "pods",
				Namespace: "default",
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := &mcp.CallToolParamsFor[ListResourcesInput]{
				Arguments: tt.input,
			}
			result, err := handleListResources(ctx, nil, params)
			if (err != nil) != tt.wantError {
				t.Fatalf("handleListResources error = %v, wantError = %v", err, tt.wantError)
			}
			if !tt.wantError {
				if len(result.Content) == 0 {
					t.Fatalf("expected non-empty content in result")
				}
				tc, ok := result.Content[0].(*mcp.TextContent)
				if !ok || !strings.Contains(tc.Text, "test-pod") {
					t.Errorf("expected items with test-pod in list result, got: %v", tc)
				}
			}
		})
	}
}

func TestHandleDescribeResource(t *testing.T) {
	setupFakeK8sEnvironment()
	ctx := context.Background()

	params := &mcp.CallToolParamsFor[DescribeResourceInput]{
		Arguments: DescribeResourceInput{
			Group:     "",
			Version:   "v1",
			Resource:  "pods",
			Name:      "test-pod",
			Namespace: "default",
		},
	}

	result, err := handleDescribeResource(ctx, nil, params)
	if err != nil {
		t.Fatalf("handleDescribeResource error: %v", err)
	}

	if len(result.Content) == 0 {
		t.Fatalf("expected non-empty describe output")
	}
}

func TestHandlePatchResource_PermissionCheck(t *testing.T) {
	setupFakeK8sEnvironment()
	ctx := context.Background()

	// 1. Missing approval token -> Must Fail
	noTokenParams := &mcp.CallToolParamsFor[PatchResourceInput]{
		Arguments: PatchResourceInput{
			Group:         "",
			Version:       "v1",
			Resource:      "pods",
			Name:          "test-pod",
			Namespace:     "default",
			PatchJSON:     `{"metadata":{"labels":{"env":"prod"}}}`,
			ApprovalToken: "", // Missing token
		},
	}

	_, err := handlePatchResource(ctx, nil, noTokenParams)
	if err == nil {
		t.Errorf("expected permission error when approval_token is empty, got nil")
	}

	// 2. With approval token -> Must Succeed
	withTokenParams := &mcp.CallToolParamsFor[PatchResourceInput]{
		Arguments: PatchResourceInput{
			Group:         "",
			Version:       "v1",
			Resource:      "pods",
			Name:          "test-pod",
			Namespace:     "default",
			PatchJSON:     `{"metadata":{"labels":{"env":"prod"}}}`,
			ApprovalToken: "USER_APPROVED_TOKEN",
		},
	}

	res, err := handlePatchResource(ctx, nil, withTokenParams)
	if err != nil {
		t.Fatalf("expected successful patch with approval token, got error: %v", err)
	}

	if len(res.Content) == 0 {
		t.Errorf("expected success message in content")
	}
}

func TestResolveGVR(t *testing.T) {
	setupFakeK8sEnvironment()
	tests := []struct {
		name        string
		group       string
		version     string
		resource    string
		wantGroup   string
		wantVersion string
		wantRes     string
	}{
		{
			name:        "deployments without group defaults to apps/v1",
			group:       "",
			version:     "",
			resource:    "deployments",
			wantGroup:   "apps",
			wantVersion: "v1",
			wantRes:     "deployments",
		},
		{
			name:        "singular deployment normalized to plural",
			group:       "",
			version:     "",
			resource:    "deployment",
			wantGroup:   "apps",
			wantVersion: "v1",
			wantRes:     "deployments",
		},
		{
			name:        "pods without group defaults to core v1",
			group:       "",
			version:     "",
			resource:    "pods",
			wantGroup:   "",
			wantVersion: "v1",
			wantRes:     "pods",
		},
		{
			name:        "jobs without group defaults to batch/v1",
			group:       "",
			version:     "",
			resource:    "jobs",
			wantGroup:   "batch",
			wantVersion: "v1",
			wantRes:     "jobs",
		},
		{
			name:        "custom explicit group and version preserved",
			group:       "custom.acme.com",
			version:     "v1alpha1",
			resource:    "widgets",
			wantGroup:   "custom.acme.com",
			wantVersion: "v1alpha1",
			wantRes:     "widgets",
		},
		{
			name:        "combined apiVersion in version field split into apps/v1",
			group:       "",
			version:     "apps/v1",
			resource:    "deployments",
			wantGroup:   "apps",
			wantVersion: "v1",
			wantRes:     "deployments",
		},
		{
			name:        "combined apiVersion for networking split properly",
			group:       "",
			version:     "networking.k8s.io/v1",
			resource:    "ingresses",
			wantGroup:   "networking.k8s.io",
			wantVersion: "v1",
			wantRes:     "ingresses",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveGVR(tt.group, tt.version, tt.resource)
			if got.Group != tt.wantGroup || got.Version != tt.wantVersion || got.Resource != tt.wantRes {
				t.Errorf("resolveGVR(%q, %q, %q) = %v, want {Group: %q, Version: %q, Resource: %q}",
					tt.group, tt.version, tt.resource, got, tt.wantGroup, tt.wantVersion, tt.wantRes)
			}
		})
	}
}
