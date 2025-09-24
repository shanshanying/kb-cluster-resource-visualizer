package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	discoveryfake "k8s.io/client-go/discovery/fake"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
)

// MockK8sClient for testing - removed as unused

// Test setup helper functions
func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	api := router.Group("/api")
	{
		api.GET("/health", healthCheck)
		api.GET("/resources/:type", getResourcesByType)
		api.GET("/resources/:type/:root/tree", getResourceChildren)
		api.GET("/namespaces", getNamespaces)
	}

	return router
}

func setupMockK8sClient() *K8sClient {
	scheme := runtime.NewScheme()
	corev1.AddToScheme(scheme)
	appsv1.AddToScheme(scheme)
	batchv1.AddToScheme(scheme)
	networkingv1.AddToScheme(scheme)

	// Create fake clients
	fakeClientset := fake.NewSimpleClientset()
	fakeDynamicClient := dynamicfake.NewSimpleDynamicClient(scheme)
	fakeDiscoveryClient := &discoveryfake.FakeDiscovery{
		Fake: &fakeClientset.Fake,
	}

	return &K8sClient{
		clientset:       fakeClientset,
		dynamicClient:   fakeDynamicClient,
		discoveryClient: fakeDiscoveryClient,
	}
}

func createTestNamespace(name string) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}

func createTestPod(name, namespace string) *unstructured.Unstructured {
	pod := &unstructured.Unstructured{}
	pod.SetUnstructuredContent(map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Pod",
		"metadata": map[string]interface{}{
			"name":              name,
			"namespace":         namespace,
			"uid":               uuid.New().String(),
			"creationTimestamp": time.Now().Format(time.RFC3339),
			"labels": map[string]interface{}{
				"app": "test-app",
			},
		},
		"status": map[string]interface{}{
			"phase": "Running",
		},
	})
	return pod
}

func createTestDeployment(name, namespace string) *unstructured.Unstructured {
	deployment := &unstructured.Unstructured{}
	deployment.SetUnstructuredContent(map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata": map[string]interface{}{
			"name":              name,
			"namespace":         namespace,
			"uid":               uuid.New().String(),
			"creationTimestamp": time.Now().Format(time.RFC3339),
		},
		"status": map[string]interface{}{
			"conditions": []interface{}{
				map[string]interface{}{
					"type":   "Ready",
					"status": "True",
				},
			},
		},
	})
	return deployment
}

// Test HTTP handlers
func TestHealthCheck(t *testing.T) {
	router := setupTestRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/health", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "healthy", response["status"])
	assert.Equal(t, "K8s Resource Visualizer API is running", response["message"])
}

func TestGetNamespaces(t *testing.T) {
	// Setup mock client
	originalClient := k8sClient
	k8sClient = setupMockK8sClient()
	defer func() { k8sClient = originalClient }()

	// Create test namespaces
	ns1 := createTestNamespace("default")
	ns2 := createTestNamespace("kube-system")

	_, err := k8sClient.clientset.CoreV1().Namespaces().Create(context.TODO(), ns1, metav1.CreateOptions{})
	assert.NoError(t, err)
	_, err = k8sClient.clientset.CoreV1().Namespaces().Create(context.TODO(), ns2, metav1.CreateOptions{})
	assert.NoError(t, err)

	router := setupTestRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/namespaces", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var namespaces []string
	err = json.Unmarshal(w.Body.Bytes(), &namespaces)
	assert.NoError(t, err)
	assert.Contains(t, namespaces, "default")
	assert.Contains(t, namespaces, "kube-system")
}

func TestGetResourcesByType(t *testing.T) {
	// Setup mock client
	originalClient := k8sClient
	k8sClient = setupMockK8sClient()
	defer func() { k8sClient = originalClient }()

	// Create test pod
	testPod := createTestPod("test-pod", "default")
	gvr := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
	_, err := k8sClient.dynamicClient.Resource(gvr).Namespace("default").Create(
		context.TODO(), testPod, metav1.CreateOptions{})
	assert.NoError(t, err)

	router := setupTestRouter()

	t.Run("Get pods with namespace", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/resources/pod?namespace=default", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resources []ResourceNode
		err := json.Unmarshal(w.Body.Bytes(), &resources)
		assert.NoError(t, err)
		assert.Len(t, resources, 1)
		assert.Equal(t, "test-pod", resources[0].Name)
		assert.Equal(t, "Pod", resources[0].Kind)
		assert.Equal(t, "default", resources[0].Namespace)
	})

	t.Run("Get unknown resource type", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/resources/unknown", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "Unknown resource type")
	})
}

func TestGetResourceChildren(t *testing.T) {
	// Setup mock client
	originalClient := k8sClient
	k8sClient = setupMockK8sClient()
	defer func() { k8sClient = originalClient }()

	// Create test deployment
	testDeployment := createTestDeployment("test-deployment", "default")
	deploymentGVR := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
	createdDeployment, err := k8sClient.dynamicClient.Resource(deploymentGVR).Namespace("default").Create(
		context.TODO(), testDeployment, metav1.CreateOptions{})
	assert.NoError(t, err)

	// Create child pod with owner reference
	childPod := createTestPod("child-pod", "default")
	// Add the required instance label
	labels := childPod.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	labels["app.kubernetes.io/instance"] = "test-deployment"
	childPod.SetLabels(labels)

	childPod.SetOwnerReferences([]metav1.OwnerReference{
		{
			UID: createdDeployment.GetUID(),
		},
	})

	podGVR := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
	_, err = k8sClient.dynamicClient.Resource(podGVR).Namespace("default").Create(
		context.TODO(), childPod, metav1.CreateOptions{})
	assert.NoError(t, err)

	router := setupTestRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/resources/deployment/test-deployment/children?namespace=default", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var relationship ResourceRelationship
	err = json.Unmarshal(w.Body.Bytes(), &relationship)
	assert.NoError(t, err)
	assert.Equal(t, "test-deployment", relationship.Parent.Name)
	assert.Equal(t, "Deployment", relationship.Parent.Kind)
}

// Test utility functions
func TestGetGVRForResourceType(t *testing.T) {
	tests := []struct {
		resourceType string
		expectedGVR  schema.GroupVersionResource
		expectError  bool
	}{
		{
			resourceType: "pod",
			expectedGVR:  schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"},
			expectError:  false,
		},
		{
			resourceType: "pods",
			expectedGVR:  schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"},
			expectError:  false,
		},
		{
			resourceType: "deployment",
			expectedGVR:  schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"},
			expectError:  false,
		},
		{
			resourceType: "service",
			expectedGVR:  schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"},
			expectError:  false,
		},
		{
			resourceType: "unknown",
			expectedGVR:  schema.GroupVersionResource{},
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.resourceType, func(t *testing.T) {
			gvr, err := getGVRForResourceType(tt.resourceType)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedGVR, gvr)
			}
		})
	}
}

func TestConvertToResourceNode(t *testing.T) {
	testPod := createTestPod("test-pod", "default")

	node := convertToResourceNode(*testPod)

	assert.Equal(t, "test-pod", node.Name)
	assert.Equal(t, "Pod", node.Kind)
	assert.Equal(t, "v1", node.APIVersion)
	assert.Equal(t, "default", node.Namespace)
	assert.Equal(t, "Running", node.Status)
	assert.Contains(t, node.Labels, "app")
	assert.Equal(t, "test-app", node.Labels["app"])
}

func TestConvertToResourceNodes(t *testing.T) {
	testPod1 := createTestPod("test-pod-1", "default")
	testPod2 := createTestPod("test-pod-2", "default")

	resources := []unstructured.Unstructured{*testPod1, *testPod2}
	nodes := convertToResourceNodes(resources)

	assert.Len(t, nodes, 2)
	assert.Equal(t, "test-pod-1", nodes[0].Name)
	assert.Equal(t, "test-pod-2", nodes[1].Name)
}

func TestContains(t *testing.T) {
	slice := []string{"apple", "banana", "cherry"}

	assert.True(t, contains(slice, "apple"))
	assert.True(t, contains(slice, "banana"))
	assert.False(t, contains(slice, "grape"))
	assert.False(t, contains([]string{}, "apple"))
}

// Test resource status parsing
func TestResourceStatusParsing(t *testing.T) {
	t.Run("Pod with phase status", func(t *testing.T) {
		pod := &unstructured.Unstructured{}
		pod.SetUnstructuredContent(map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata": map[string]interface{}{
				"name":              "test-pod",
				"namespace":         "default",
				"uid":               "test-uid",
				"creationTimestamp": time.Now().Format(time.RFC3339),
			},
			"status": map[string]interface{}{
				"phase": "Running",
			},
		})

		node := convertToResourceNode(*pod)
		assert.Equal(t, "Running", node.Status)
	})

	t.Run("Resource with Ready condition", func(t *testing.T) {
		resource := &unstructured.Unstructured{}
		resource.SetUnstructuredContent(map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]interface{}{
				"name":              "test-deployment",
				"namespace":         "default",
				"uid":               "test-uid",
				"creationTimestamp": time.Now().Format(time.RFC3339),
			},
			"status": map[string]interface{}{
				"conditions": []interface{}{
					map[string]interface{}{
						"type":   "Available",
						"status": "True",
					},
					map[string]interface{}{
						"type":   "Ready",
						"status": "True",
					},
				},
			},
		})

		node := convertToResourceNode(*resource)
		assert.Equal(t, "Ready", node.Status)
	})

	t.Run("Resource with no status", func(t *testing.T) {
		resource := &unstructured.Unstructured{}
		resource.SetUnstructuredContent(map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]interface{}{
				"name":              "test-configmap",
				"namespace":         "default",
				"uid":               "test-uid",
				"creationTimestamp": time.Now().Format(time.RFC3339),
			},
		})

		node := convertToResourceNode(*resource)
		assert.Equal(t, "Unknown", node.Status)
	})
}

// Benchmark tests
func BenchmarkMainConvertToResourceNode(b *testing.B) {
	testPod := createTestPod("test-pod", "default")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		convertToResourceNode(*testPod)
	}
}

func BenchmarkMainGetGVRForResourceType(b *testing.B) {
	resourceType := "pod"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		getGVRForResourceType(resourceType)
	}
}
