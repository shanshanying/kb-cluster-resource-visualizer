package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// TestUtilityFunctions focuses on testing utility and helper functions
func TestGetGVRForResourceTypeComprehensive(t *testing.T) {
	testCases := []struct {
		name         string
		resourceType string
		expectedGVR  schema.GroupVersionResource
		expectError  bool
	}{
		// Core resources
		{
			name:         "pod singular",
			resourceType: "pod",
			expectedGVR:  schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"},
			expectError:  false,
		},
		{
			name:         "pods plural",
			resourceType: "pods",
			expectedGVR:  schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"},
			expectError:  false,
		},
		{
			name:         "service singular",
			resourceType: "service",
			expectedGVR:  schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"},
			expectError:  false,
		},
		{
			name:         "services plural",
			resourceType: "services",
			expectedGVR:  schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"},
			expectError:  false,
		},
		{
			name:         "configmap",
			resourceType: "configmap",
			expectedGVR:  schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"},
			expectError:  false,
		},
		{
			name:         "secret",
			resourceType: "secret",
			expectedGVR:  schema.GroupVersionResource{Group: "", Version: "v1", Resource: "secrets"},
			expectError:  false,
		},
		{
			name:         "pvc abbreviation",
			resourceType: "pvc",
			expectedGVR:  schema.GroupVersionResource{Group: "", Version: "v1", Resource: "persistentvolumeclaims"},
			expectError:  false,
		},
		{
			name:         "KubeBlocks Cluster",
			resourceType: "cluster",
			expectedGVR:  schema.GroupVersionResource{Group: "apps.kubeblocks.io", Version: "v1", Resource: "clusters"},
			expectError:  false,
		},
		{
			name:         "KubeBlocks Component (cmp abbreviation)",
			resourceType: "cmp",
			expectedGVR:  schema.GroupVersionResource{Group: "apps.kubeblocks.io", Version: "v1", Resource: "components"},
			expectError:  false,
		},
		{
			name:         "KubeBlocks BackupPolicy (bp abbreviation)",
			resourceType: "bp",
			expectedGVR:  schema.GroupVersionResource{Group: "dataprotection.kubeblocks.io", Version: "v1alpha1", Resource: "backuppolicies"},
			expectError:  false,
		},
		{
			name:         "KubeBlocks OpsRequest (ops abbreviation)",
			resourceType: "ops",
			expectedGVR:  schema.GroupVersionResource{Group: "operations.kubeblocks.io", Version: "v1alpha1", Resource: "opsrequests"},
			expectError:  false,
		},
		{
			name:         "KubeBlocks Instance (inst abbreviation)",
			resourceType: "inst",
			expectedGVR:  schema.GroupVersionResource{Group: "workloads.kubeblocks.io", Version: "v1", Resource: "instances"},
			expectError:  false,
		},
		{
			name:         "KubeBlocks InstanceSet (its abbreviation)",
			resourceType: "its",
			expectedGVR:  schema.GroupVersionResource{Group: "workloads.kubeblocks.io", Version: "v1", Resource: "instancesets"},
			expectError:  false,
		},

		// Apps resources
		{
			name:         "deployment",
			resourceType: "deployment",
			expectedGVR:  schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"},
			expectError:  false,
		},
		{
			name:         "replicaset",
			resourceType: "replicaset",
			expectedGVR:  schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "replicasets"},
			expectError:  false,
		},
		{
			name:         "statefulset",
			resourceType: "statefulset",
			expectedGVR:  schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "statefulsets"},
			expectError:  false,
		},
		{
			name:         "daemonset",
			resourceType: "daemonset",
			expectedGVR:  schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "daemonsets"},
			expectError:  false,
		},

		// Networking resources
		{
			name:         "ingress",
			resourceType: "ingress",
			expectedGVR:  schema.GroupVersionResource{Group: "networking.k8s.io", Version: "v1", Resource: "ingresses"},
			expectError:  false,
		},

		// Batch resources
		{
			name:         "job",
			resourceType: "job",
			expectedGVR:  schema.GroupVersionResource{Group: "batch", Version: "v1", Resource: "jobs"},
			expectError:  false,
		},
		{
			name:         "cronjob",
			resourceType: "cronjob",
			expectedGVR:  schema.GroupVersionResource{Group: "batch", Version: "v1", Resource: "cronjobs"},
			expectError:  false,
		},

		// Case sensitivity tests
		{
			name:         "POD uppercase",
			resourceType: "POD",
			expectedGVR:  schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"},
			expectError:  false,
		},
		{
			name:         "Deployment mixed case",
			resourceType: "Deployment",
			expectedGVR:  schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"},
			expectError:  false,
		},

		// Error cases
		{
			name:         "unknown resource",
			resourceType: "unknown",
			expectedGVR:  schema.GroupVersionResource{},
			expectError:  true,
		},
		{
			name:         "empty resource type",
			resourceType: "",
			expectedGVR:  schema.GroupVersionResource{},
			expectError:  true,
		},
		{
			name:         "invalid resource type",
			resourceType: "not-a-resource",
			expectedGVR:  schema.GroupVersionResource{},
			expectError:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gvr, err := getGVRForResourceType(tc.resourceType)

			if tc.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "unknown resource type")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedGVR.Group, gvr.Group)
				assert.Equal(t, tc.expectedGVR.Version, gvr.Version)
				assert.Equal(t, tc.expectedGVR.Resource, gvr.Resource)
			}
		})
	}
}

func TestContainsFunction(t *testing.T) {
	testCases := []struct {
		name     string
		slice    []string
		item     string
		expected bool
	}{
		{
			name:     "item exists at beginning",
			slice:    []string{"apple", "banana", "cherry"},
			item:     "apple",
			expected: true,
		},
		{
			name:     "item exists in middle",
			slice:    []string{"apple", "banana", "cherry"},
			item:     "banana",
			expected: true,
		},
		{
			name:     "item exists at end",
			slice:    []string{"apple", "banana", "cherry"},
			item:     "cherry",
			expected: true,
		},
		{
			name:     "item does not exist",
			slice:    []string{"apple", "banana", "cherry"},
			item:     "grape",
			expected: false,
		},
		{
			name:     "empty slice",
			slice:    []string{},
			item:     "apple",
			expected: false,
		},
		{
			name:     "empty item",
			slice:    []string{"apple", "banana", "cherry"},
			item:     "",
			expected: false,
		},
		{
			name:     "slice with empty string",
			slice:    []string{"apple", "", "cherry"},
			item:     "",
			expected: true,
		},
		{
			name:     "single item slice - match",
			slice:    []string{"apple"},
			item:     "apple",
			expected: true,
		},
		{
			name:     "single item slice - no match",
			slice:    []string{"apple"},
			item:     "banana",
			expected: false,
		},
		{
			name:     "case sensitive - no match",
			slice:    []string{"Apple", "Banana", "Cherry"},
			item:     "apple",
			expected: false,
		},
		{
			name:     "duplicate items",
			slice:    []string{"apple", "banana", "apple", "cherry"},
			item:     "apple",
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := contains(tc.slice, tc.item)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestConvertToResourceNodeDetailed(t *testing.T) {
	t.Run("Pod with complete metadata", func(t *testing.T) {
		pod := &unstructured.Unstructured{}
		pod.SetUnstructuredContent(map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata": map[string]interface{}{
				"name":              "test-pod",
				"namespace":         "default",
				"uid":               "test-uid-123",
				"creationTimestamp": "2023-01-01T12:00:00.000Z",
				"labels": map[string]interface{}{
					"app":     "test-app",
					"version": "v1.0.0",
					"tier":    "frontend",
				},
				"annotations": map[string]interface{}{
					"deployment.kubernetes.io/revision":                "1",
					"kubectl.kubernetes.io/last-applied-configuration": "{}",
				},
			},
			"status": map[string]interface{}{
				"phase": "Running",
			},
		})

		node := convertToResourceNode(*pod)

		assert.Equal(t, "test-pod", node.Name)
		assert.Equal(t, "Pod", node.Kind)
		assert.Equal(t, "v1", node.APIVersion)
		assert.Equal(t, "default", node.Namespace)
		assert.Equal(t, "test-uid-123", node.UID)
		assert.Equal(t, "Running", node.Status)
		assert.Contains(t, node.CreationTime, "2023-01-01")

		// Check labels
		assert.Len(t, node.Labels, 3)
		assert.Equal(t, "test-app", node.Labels["app"])
		assert.Equal(t, "v1.0.0", node.Labels["version"])
		assert.Equal(t, "frontend", node.Labels["tier"])

		// Check annotations
		assert.Len(t, node.Annotations, 2)
		assert.Equal(t, "1", node.Annotations["deployment.kubernetes.io/revision"])
	})

	t.Run("Deployment with Ready condition", func(t *testing.T) {
		deployment := &unstructured.Unstructured{}
		deployment.SetUnstructuredContent(map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]interface{}{
				"name":              "test-deployment",
				"namespace":         "production",
				"uid":               "deployment-uid-456",
				"creationTimestamp": "2023-06-15T08:30:00.000Z",
			},
			"status": map[string]interface{}{
				"conditions": []interface{}{
					map[string]interface{}{
						"type":   "Available",
						"status": "True",
					},
					map[string]interface{}{
						"type":   "Progressing",
						"status": "True",
					},
					map[string]interface{}{
						"type":   "Ready",
						"status": "True",
					},
				},
			},
		})

		node := convertToResourceNode(*deployment)

		assert.Equal(t, "test-deployment", node.Name)
		assert.Equal(t, "Deployment", node.Kind)
		assert.Equal(t, "apps/v1", node.APIVersion)
		assert.Equal(t, "production", node.Namespace)
		assert.Equal(t, "deployment-uid-456", node.UID)
		assert.Equal(t, "Ready", node.Status)
		assert.Contains(t, node.CreationTime, "2023-06-15")
	})

	t.Run("Service with no status", func(t *testing.T) {
		service := &unstructured.Unstructured{}
		service.SetUnstructuredContent(map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Service",
			"metadata": map[string]interface{}{
				"name":              "test-service",
				"namespace":         "default",
				"uid":               "service-uid-789",
				"creationTimestamp": "2023-03-20T14:45:30.000Z",
			},
		})

		node := convertToResourceNode(*service)

		assert.Equal(t, "test-service", node.Name)
		assert.Equal(t, "Service", node.Kind)
		assert.Equal(t, "v1", node.APIVersion)
		assert.Equal(t, "default", node.Namespace)
		assert.Equal(t, "service-uid-789", node.UID)
		assert.Equal(t, "Unknown", node.Status)
		assert.Contains(t, node.CreationTime, "2023-03-20")
	})

	t.Run("Resource with Ready condition false", func(t *testing.T) {
		resource := &unstructured.Unstructured{}
		resource.SetUnstructuredContent(map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "StatefulSet",
			"metadata": map[string]interface{}{
				"name":              "test-statefulset",
				"namespace":         "default",
				"uid":               "statefulset-uid-999",
				"creationTimestamp": "2023-04-10T10:15:45Z",
			},
			"status": map[string]interface{}{
				"conditions": []interface{}{
					map[string]interface{}{
						"type":   "Ready",
						"status": "False",
						"reason": "PodNotReady",
					},
				},
			},
		})

		node := convertToResourceNode(*resource)

		assert.Equal(t, "test-statefulset", node.Name)
		assert.Equal(t, "StatefulSet", node.Kind)
		assert.Equal(t, "Unknown", node.Status) // Ready is False, so status should be Unknown
	})

	t.Run("Cluster-scoped resource (no namespace)", func(t *testing.T) {
		clusterRole := &unstructured.Unstructured{}
		clusterRole.SetUnstructuredContent(map[string]interface{}{
			"apiVersion": "rbac.authorization.k8s.io/v1",
			"kind":       "ClusterRole",
			"metadata": map[string]interface{}{
				"name":              "test-cluster-role",
				"uid":               "cluster-role-uid-111",
				"creationTimestamp": "2023-05-05T16:20:10Z",
			},
		})

		node := convertToResourceNode(*clusterRole)

		assert.Equal(t, "test-cluster-role", node.Name)
		assert.Equal(t, "ClusterRole", node.Kind)
		assert.Equal(t, "rbac.authorization.k8s.io/v1", node.APIVersion)
		assert.Empty(t, node.Namespace) // Cluster-scoped resource
		assert.Equal(t, "cluster-role-uid-111", node.UID)
		assert.Equal(t, "Unknown", node.Status)
	})
}

func TestConvertToResourceNodesFunction(t *testing.T) {
	// Create test resources
	pod1 := &unstructured.Unstructured{}
	pod1.SetUnstructuredContent(map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Pod",
		"metadata": map[string]interface{}{
			"name":              "pod-1",
			"namespace":         "default",
			"uid":               "pod-uid-1",
			"creationTimestamp": "2023-01-01T00:00:00Z",
		},
		"status": map[string]interface{}{
			"phase": "Running",
		},
	})

	pod2 := &unstructured.Unstructured{}
	pod2.SetUnstructuredContent(map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Pod",
		"metadata": map[string]interface{}{
			"name":              "pod-2",
			"namespace":         "kube-system",
			"uid":               "pod-uid-2",
			"creationTimestamp": "2023-01-02T00:00:00Z",
		},
		"status": map[string]interface{}{
			"phase": "Pending",
		},
	})

	service := &unstructured.Unstructured{}
	service.SetUnstructuredContent(map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Service",
		"metadata": map[string]interface{}{
			"name":              "test-service",
			"namespace":         "default",
			"uid":               "service-uid-1",
			"creationTimestamp": "2023-01-03T00:00:00Z",
		},
	})

	t.Run("Convert multiple resources", func(t *testing.T) {
		resources := []unstructured.Unstructured{*pod1, *pod2, *service}
		nodes := convertToResourceNodes(resources)

		assert.Len(t, nodes, 3)

		// Check first pod
		assert.Equal(t, "pod-1", nodes[0].Name)
		assert.Equal(t, "Pod", nodes[0].Kind)
		assert.Equal(t, "default", nodes[0].Namespace)
		assert.Equal(t, "Running", nodes[0].Status)

		// Check second pod
		assert.Equal(t, "pod-2", nodes[1].Name)
		assert.Equal(t, "Pod", nodes[1].Kind)
		assert.Equal(t, "kube-system", nodes[1].Namespace)
		assert.Equal(t, "Pending", nodes[1].Status)

		// Check service
		assert.Equal(t, "test-service", nodes[2].Name)
		assert.Equal(t, "Service", nodes[2].Kind)
		assert.Equal(t, "default", nodes[2].Namespace)
		assert.Equal(t, "Unknown", nodes[2].Status)
	})

	t.Run("Convert empty slice", func(t *testing.T) {
		resources := []unstructured.Unstructured{}
		nodes := convertToResourceNodes(resources)

		assert.Len(t, nodes, 0)
		// Empty slice is valid, don't check for nil
	})
}

func TestIsResourceTypeMatch(t *testing.T) {
	testCases := []struct {
		name     string
		gvr      schema.GroupVersionResource
		kind     string
		expected bool
	}{
		{
			name:     "Pod match",
			gvr:      schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"},
			kind:     "Pod",
			expected: true,
		},
		{
			name:     "Service match",
			gvr:      schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"},
			kind:     "Service",
			expected: true,
		},
		{
			name:     "Deployment match",
			gvr:      schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"},
			kind:     "Deployment",
			expected: true,
		},
		{
			name:     "Pod vs Service - no match",
			gvr:      schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"},
			kind:     "Service",
			expected: false,
		},
		{
			name:     "Unknown resource",
			gvr:      schema.GroupVersionResource{Group: "", Version: "v1", Resource: "unknowns"},
			kind:     "Unknown",
			expected: false,
		},
		{
			name:     "ConfigMap match",
			gvr:      schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"},
			kind:     "ConfigMap",
			expected: true,
		},
		{
			name:     "StatefulSet match",
			gvr:      schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "statefulsets"},
			kind:     "StatefulSet",
			expected: true,
		},
		{
			name:     "Job match",
			gvr:      schema.GroupVersionResource{Group: "batch", Version: "v1", Resource: "jobs"},
			kind:     "Job",
			expected: true,
		},
		{
			name:     "Ingress match",
			gvr:      schema.GroupVersionResource{Group: "networking.k8s.io", Version: "v1", Resource: "ingresses"},
			kind:     "Ingress",
			expected: true,
		},
		{
			name:     "KubeBlocks Cluster match",
			gvr:      schema.GroupVersionResource{Group: "apps.kubeblocks.io", Version: "v1", Resource: "clusters"},
			kind:     "Cluster",
			expected: true,
		},
		{
			name:     "KubeBlocks Component match",
			gvr:      schema.GroupVersionResource{Group: "apps.kubeblocks.io", Version: "v1", Resource: "components"},
			kind:     "Component",
			expected: true,
		},
		{
			name:     "KubeBlocks BackupPolicy match",
			gvr:      schema.GroupVersionResource{Group: "dataprotection.kubeblocks.io", Version: "v1alpha1", Resource: "backuppolicies"},
			kind:     "BackupPolicy",
			expected: true,
		},
		{
			name:     "KubeBlocks Instance match",
			gvr:      schema.GroupVersionResource{Group: "workloads.kubeblocks.io", Version: "v1", Resource: "instances"},
			kind:     "Instance",
			expected: true,
		},
		{
			name:     "KubeBlocks InstanceSet match",
			gvr:      schema.GroupVersionResource{Group: "workloads.kubeblocks.io", Version: "v1", Resource: "instancesets"},
			kind:     "InstanceSet",
			expected: true,
		},
		{
			name:     "Cluster vs Component - no match",
			gvr:      schema.GroupVersionResource{Group: "apps.kubeblocks.io", Version: "v1", Resource: "clusters"},
			kind:     "Component",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isResourceTypeMatch(tc.gvr, tc.kind)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// Benchmark tests for performance
func BenchmarkUtilsGetGVRForResourceType(b *testing.B) {
	resourceTypes := []string{"pod", "deployment", "service", "ingress", "job"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resourceType := resourceTypes[i%len(resourceTypes)]
		_, _ = getGVRForResourceType(resourceType)
	}
}

func BenchmarkContains(b *testing.B) {
	slice := []string{"apple", "banana", "cherry", "date", "elderberry", "fig", "grape"}
	items := []string{"apple", "fig", "nonexistent"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		item := items[i%len(items)]
		contains(slice, item)
	}
}

func BenchmarkUtilsConvertToResourceNode(b *testing.B) {
	pod := &unstructured.Unstructured{}
	pod.SetUnstructuredContent(map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Pod",
		"metadata": map[string]interface{}{
			"name":              "benchmark-pod",
			"namespace":         "default",
			"uid":               "benchmark-uid",
			"creationTimestamp": time.Now().Format(time.RFC3339),
			"labels": map[string]interface{}{
				"app":     "benchmark",
				"version": "v1.0.0",
			},
			"annotations": map[string]interface{}{
				"test": "benchmark",
			},
		},
		"status": map[string]interface{}{
			"phase": "Running",
			"conditions": []interface{}{
				map[string]interface{}{
					"type":   "Ready",
					"status": "True",
				},
			},
		},
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		convertToResourceNode(*pod)
	}
}
