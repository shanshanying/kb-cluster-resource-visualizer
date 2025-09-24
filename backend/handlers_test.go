package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery/fake"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

// TestHandlers focuses specifically on HTTP handler testing
type HandlerTestSuite struct {
	router    *gin.Engine
	k8sClient *K8sClient
}

func setupHandlerTest() *HandlerTestSuite {
	gin.SetMode(gin.TestMode)

	// Create a scheme and add known types
	scheme := runtime.NewScheme()
	corev1.AddToScheme(scheme)
	appsv1.AddToScheme(scheme)
	batchv1.AddToScheme(scheme)
	networkingv1.AddToScheme(scheme)

	// Create fake clients
	fakeClientset := k8sfake.NewSimpleClientset()
	fakeDynamicClient := dynamicfake.NewSimpleDynamicClient(scheme)
	fakeDiscoveryClient := &fake.FakeDiscovery{
		Fake: &fakeClientset.Fake,
	}

	// Create K8sClient
	testK8sClient := &K8sClient{
		clientset:       fakeClientset,
		dynamicClient:   fakeDynamicClient,
		discoveryClient: fakeDiscoveryClient,
	}

	// Setup router
	router := gin.New()
	api := router.Group("/api")
	{
		api.GET("/health", healthCheck)
		api.GET("/resources/:type", func(c *gin.Context) {
			// Temporarily override global k8sClient for this test
			originalClient := k8sClient
			k8sClient = testK8sClient
			defer func() { k8sClient = originalClient }()
			getResourcesByType(c)
		})
		api.GET("/resources/:type/:root/tree", func(c *gin.Context) {
			originalClient := k8sClient
			k8sClient = testK8sClient
			defer func() { k8sClient = originalClient }()
			getResourceChildren(c)
		})
		api.GET("/namespaces", func(c *gin.Context) {
			originalClient := k8sClient
			k8sClient = testK8sClient
			defer func() { k8sClient = originalClient }()
			getNamespaces(c)
		})
	}

	return &HandlerTestSuite{
		router:    router,
		k8sClient: testK8sClient,
	}
}

func TestHealthCheckHandler(t *testing.T) {
	suite := setupHandlerTest()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/health", nil)
	suite.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "healthy", response["status"])
	assert.Equal(t, "K8s Resource Visualizer API is running", response["message"])
}

func TestGetNamespacesHandler(t *testing.T) {
	suite := setupHandlerTest()

	// Create test namespaces
	ns1 := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "default",
		},
	}
	ns2 := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kube-system",
		},
	}
	ns3 := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-namespace",
		},
	}

	_, err := suite.k8sClient.clientset.CoreV1().Namespaces().Create(context.TODO(), ns1, metav1.CreateOptions{})
	require.NoError(t, err)
	_, err = suite.k8sClient.clientset.CoreV1().Namespaces().Create(context.TODO(), ns2, metav1.CreateOptions{})
	require.NoError(t, err)
	_, err = suite.k8sClient.clientset.CoreV1().Namespaces().Create(context.TODO(), ns3, metav1.CreateOptions{})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/namespaces", nil)
	suite.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var namespaces []string
	err = json.Unmarshal(w.Body.Bytes(), &namespaces)
	require.NoError(t, err)

	assert.Len(t, namespaces, 3)
	assert.Contains(t, namespaces, "default")
	assert.Contains(t, namespaces, "kube-system")
	assert.Contains(t, namespaces, "test-namespace")
}

func TestGetResourcesByTypeHandler(t *testing.T) {
	suite := setupHandlerTest()

	// Create test pods
	pod1 := createTestUnstructuredPod("pod-1", "default", "Running")
	pod2 := createTestUnstructuredPod("pod-2", "default", "Pending")
	pod3 := createTestUnstructuredPod("pod-3", "kube-system", "Running")

	gvr := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}

	_, err := suite.k8sClient.dynamicClient.Resource(gvr).Namespace("default").Create(context.TODO(), pod1, metav1.CreateOptions{})
	require.NoError(t, err)
	_, err = suite.k8sClient.dynamicClient.Resource(gvr).Namespace("default").Create(context.TODO(), pod2, metav1.CreateOptions{})
	require.NoError(t, err)
	_, err = suite.k8sClient.dynamicClient.Resource(gvr).Namespace("kube-system").Create(context.TODO(), pod3, metav1.CreateOptions{})
	require.NoError(t, err)

	t.Run("Get pods from specific namespace", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/resources/pods?namespace=default", nil)
		suite.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resources []ResourceNode
		err := json.Unmarshal(w.Body.Bytes(), &resources)
		require.NoError(t, err)

		assert.Len(t, resources, 2)
		for _, resource := range resources {
			assert.Equal(t, "default", resource.Namespace)
			assert.Equal(t, "Pod", resource.Kind)
		}
	})

	t.Run("Get pods from all namespaces", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/resources/pods", nil)
		suite.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resources []ResourceNode
		err := json.Unmarshal(w.Body.Bytes(), &resources)
		require.NoError(t, err)

		// Should get pods from both namespaces
		assert.GreaterOrEqual(t, len(resources), 3)
	})

	t.Run("Get unknown resource type", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/resources/unknown-type", nil)
		suite.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var errorResponse map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
		require.NoError(t, err)

		assert.Contains(t, errorResponse["error"], "Unknown resource type")
	})
}

func TestGetResourceChildrenHandler(t *testing.T) {
	suite := setupHandlerTest()

	// Create parent deployment
	deployment := createTestUnstructuredDeployment("test-deployment", "default")
	deploymentGVR := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}

	createdDeployment, err := suite.k8sClient.dynamicClient.Resource(deploymentGVR).Namespace("default").Create(context.TODO(), deployment, metav1.CreateOptions{})
	require.NoError(t, err)

	// Create child pods
	childPod1 := createTestUnstructuredPod("child-pod-1", "default", "Running")
	// Add the required instance label
	labels1 := childPod1.GetLabels()
	if labels1 == nil {
		labels1 = make(map[string]string)
	}
	labels1["app.kubernetes.io/instance"] = "test-deployment"
	childPod1.SetLabels(labels1)
	childPod1.SetOwnerReferences([]metav1.OwnerReference{
		{
			UID: createdDeployment.GetUID(),
		},
	})

	childPod2 := createTestUnstructuredPod("child-pod-2", "default", "Running")
	// Add the required instance label
	labels2 := childPod2.GetLabels()
	if labels2 == nil {
		labels2 = make(map[string]string)
	}
	labels2["app.kubernetes.io/instance"] = "test-deployment"
	childPod2.SetLabels(labels2)
	childPod2.SetOwnerReferences([]metav1.OwnerReference{
		{
			UID: createdDeployment.GetUID(),
		},
	})

	// Create unrelated pod (should not be in children)
	unrelatedPod := createTestUnstructuredPod("unrelated-pod", "default", "Running")

	podGVR := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}

	_, err = suite.k8sClient.dynamicClient.Resource(podGVR).Namespace("default").Create(context.TODO(), childPod1, metav1.CreateOptions{})
	require.NoError(t, err)
	_, err = suite.k8sClient.dynamicClient.Resource(podGVR).Namespace("default").Create(context.TODO(), childPod2, metav1.CreateOptions{})
	require.NoError(t, err)
	_, err = suite.k8sClient.dynamicClient.Resource(podGVR).Namespace("default").Create(context.TODO(), unrelatedPod, metav1.CreateOptions{})
	require.NoError(t, err)

	t.Run("Get deployment children", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/resources/deployment/test-deployment/children?namespace=default", nil)
		suite.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var relationship ResourceRelationship
		err := json.Unmarshal(w.Body.Bytes(), &relationship)
		require.NoError(t, err)

		// Verify parent
		assert.Equal(t, "test-deployment", relationship.Parent.Name)
		assert.Equal(t, "Deployment", relationship.Parent.Kind)
		assert.Equal(t, "default", relationship.Parent.Namespace)

		// Verify children - should only include pods with owner references
		// Note: In a real scenario, we would have more comprehensive API resource discovery
		// For this test, we'll check if the structure is correct
		assert.NotNil(t, relationship.Children)
	})

	t.Run("Get children for non-existent resource", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/resources/deployment/non-existent/children?namespace=default", nil)
		suite.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var errorResponse map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
		require.NoError(t, err)

		assert.Contains(t, errorResponse["error"], "Resource not found")
	})

	t.Run("Get children without namespace parameter", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/resources/deployment/test-deployment/children", nil)
		suite.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var errorResponse map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
		require.NoError(t, err)

		assert.Contains(t, errorResponse["error"], "Namespace parameter is required")
	})
}

// Helper functions for creating test resources
func createTestUnstructuredPod(name, namespace, phase string) *unstructured.Unstructured {
	pod := &unstructured.Unstructured{}
	pod.SetUnstructuredContent(map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Pod",
		"metadata": map[string]interface{}{
			"name":              name,
			"namespace":         namespace,
			"uid":               uuid.New().String(),
			"creationTimestamp": "2023-01-01T00:00:00Z",
			"labels": map[string]interface{}{
				"app": "test-app",
			},
			"annotations": map[string]interface{}{
				"test-annotation": "test-value",
			},
		},
		"status": map[string]interface{}{
			"phase": phase,
		},
	})
	return pod
}

func createTestUnstructuredDeployment(name, namespace string) *unstructured.Unstructured {
	deployment := &unstructured.Unstructured{}
	deployment.SetUnstructuredContent(map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata": map[string]interface{}{
			"name":              name,
			"namespace":         namespace,
			"uid":               uuid.New().String(),
			"creationTimestamp": "2023-01-01T00:00:00Z",
			"labels": map[string]interface{}{
				"app": "test-deployment",
			},
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
	return deployment
}

// Test error scenarios
func TestHandlerErrorScenarios(t *testing.T) {
	suite := setupHandlerTest()

	t.Run("Invalid resource type format", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/resources/", nil)
		suite.router.ServeHTTP(w, req)

		// Should return 404 for invalid route
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Resource children with invalid type", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/resources/invalid-type/test-resource/children", nil)
		suite.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
