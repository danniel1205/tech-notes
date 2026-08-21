package main

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
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
			if !tt.wantError && len(result.Content) == 0 {
				t.Errorf("expected non-empty content in result")
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
