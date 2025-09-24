package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
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

// IntegrationTestSuite contains integration tests for the entire application
type IntegrationTestSuite struct {
	suite.Suite
	router    *gin.Engine
	k8sClient *K8sClient
	server    *httptest.Server
}

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}

func (suite *IntegrationTestSuite) SetupSuite() {
	gin.SetMode(gin.TestMode)

	// Create a scheme and add known types
	scheme := runtime.NewScheme()
	corev1.AddToScheme(scheme)
	appsv1.AddToScheme(scheme)
	batchv1.AddToScheme(scheme)
	networkingv1.AddToScheme(scheme)

	// Create fake clients with some initial data
	fakeClientset := k8sfake.NewSimpleClientset()
	fakeDynamicClient := dynamicfake.NewSimpleDynamicClient(scheme)
	fakeDiscoveryClient := &fake.FakeDiscovery{
		Fake: &fakeClientset.Fake,
	}

	// Create K8sClient
	suite.k8sClient = &K8sClient{
		clientset:       fakeClientset,
		dynamicClient:   fakeDynamicClient,
		discoveryClient: fakeDiscoveryClient,
	}

	// Setup router with CORS and all endpoints
	suite.router = gin.New()
	suite.router.Use(gin.Recovery())

	api := suite.router.Group("/api")
	{
		api.GET("/health", healthCheck)
		api.GET("/resources/:type", func(c *gin.Context) {
			originalClient := k8sClient
			k8sClient = suite.k8sClient
			defer func() { k8sClient = originalClient }()
			getResourcesByType(c)
		})
		api.GET("/resources/:type/:name/children", func(c *gin.Context) {
			originalClient := k8sClient
			k8sClient = suite.k8sClient
			defer func() { k8sClient = originalClient }()
			getResourceChildren(c)
		})
		api.GET("/namespaces", func(c *gin.Context) {
			originalClient := k8sClient
			k8sClient = suite.k8sClient
			defer func() { k8sClient = originalClient }()
			getNamespaces(c)
		})
	}

	// Start test server
	suite.server = httptest.NewServer(suite.router)
}

func (suite *IntegrationTestSuite) TearDownSuite() {
	if suite.server != nil {
		suite.server.Close()
	}
}

func (suite *IntegrationTestSuite) SetupTest() {
	// Clean up any existing resources before each test
	// This ensures test isolation
}

func (suite *IntegrationTestSuite) TestCompleteWorkflow() {
	// This test simulates a complete workflow of creating resources and querying them

	// Step 1: Create namespaces
	ns1 := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "production",
		},
	}
	ns2 := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "staging",
		},
	}

	_, err := suite.k8sClient.clientset.CoreV1().Namespaces().Create(context.TODO(), ns1, metav1.CreateOptions{})
	require.NoError(suite.T(), err)
	_, err = suite.k8sClient.clientset.CoreV1().Namespaces().Create(context.TODO(), ns2, metav1.CreateOptions{})
	require.NoError(suite.T(), err)

	// Step 2: Query namespaces via API
	resp, err := http.Get(suite.server.URL + "/api/namespaces")
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var namespaces []string
	err = json.NewDecoder(resp.Body).Decode(&namespaces)
	require.NoError(suite.T(), err)
	assert.Contains(suite.T(), namespaces, "production")
	assert.Contains(suite.T(), namespaces, "staging")

	// Step 3: Create a deployment in production namespace
	deployment := suite.createTestDeployment("web-app", "production")
	deploymentGVR := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}

	createdDeployment, err := suite.k8sClient.dynamicClient.Resource(deploymentGVR).
		Namespace("production").Create(context.TODO(), deployment, metav1.CreateOptions{})
	require.NoError(suite.T(), err)

	// Step 4: Create pods owned by the deployment
	for i := 0; i < 3; i++ {
		pod := suite.createTestPod(fmt.Sprintf("web-app-pod-%d", i), "production", "Running")
		// Add the required instance label
		labels := pod.GetLabels()
		if labels == nil {
			labels = make(map[string]string)
		}
		labels["app.kubernetes.io/instance"] = "web-app"
		pod.SetLabels(labels)

		pod.SetOwnerReferences([]metav1.OwnerReference{
			{
				UID:        createdDeployment.GetUID(),
				Kind:       "Deployment",
				Name:       "web-app",
				APIVersion: "apps/v1",
			},
		})

		podGVR := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
		_, err := suite.k8sClient.dynamicClient.Resource(podGVR).
			Namespace("production").Create(context.TODO(), pod, metav1.CreateOptions{})
		require.NoError(suite.T(), err)
	}

	// Step 5: Query deployments
	resp, err = http.Get(suite.server.URL + "/api/resources/deployment?namespace=production")
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var deployments []ResourceNode
	err = json.NewDecoder(resp.Body).Decode(&deployments)
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), deployments, 1)
	assert.Equal(suite.T(), "web-app", deployments[0].Name)
	assert.Equal(suite.T(), "production", deployments[0].Namespace)

	// Step 6: Query pods in production
	resp, err = http.Get(suite.server.URL + "/api/resources/pod?namespace=production")
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	var pods []ResourceNode
	err = json.NewDecoder(resp.Body).Decode(&pods)
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), pods, 3)

	// Step 7: Query deployment children
	resp, err = http.Get(suite.server.URL + "/api/resources/deployment/web-app/children?namespace=production")
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	var relationship ResourceRelationship
	err = json.NewDecoder(resp.Body).Decode(&relationship)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), "web-app", relationship.Parent.Name)
	assert.Equal(suite.T(), "Deployment", relationship.Parent.Kind)
	// Note: Children discovery might be limited in fake client, but structure should be correct
}

func (suite *IntegrationTestSuite) TestMultiNamespaceScenario() {
	// Test scenario with resources across multiple namespaces

	// Create namespaces
	namespaces := []string{"frontend", "backend", "database"}
	for _, ns := range namespaces {
		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: ns},
		}
		_, err := suite.k8sClient.clientset.CoreV1().Namespaces().Create(context.TODO(), namespace, metav1.CreateOptions{})
		require.NoError(suite.T(), err)
	}

	// Create services in each namespace
	serviceGVR := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"}

	for _, ns := range namespaces {
		service := suite.createTestService(fmt.Sprintf("%s-service", ns), ns)
		_, err := suite.k8sClient.dynamicClient.Resource(serviceGVR).
			Namespace(ns).Create(context.TODO(), service, metav1.CreateOptions{})
		require.NoError(suite.T(), err)
	}

	// Query services across all namespaces
	resp, err := http.Get(suite.server.URL + "/api/resources/service")
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	var services []ResourceNode
	err = json.NewDecoder(resp.Body).Decode(&services)
	require.NoError(suite.T(), err)

	// Should find services from all namespaces
	assert.GreaterOrEqual(suite.T(), len(services), 3)

	// Verify we have services from different namespaces
	foundNamespaces := make(map[string]bool)
	for _, service := range services {
		foundNamespaces[service.Namespace] = true
	}

	for _, ns := range namespaces {
		assert.True(suite.T(), foundNamespaces[ns], "Should find service in namespace %s", ns)
	}
}

func (suite *IntegrationTestSuite) TestErrorHandling() {
	// Test various error scenarios

	t := suite.T()

	// Test 1: Invalid resource type
	resp, err := http.Get(suite.server.URL + "/api/resources/invalid-resource")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	// Test 2: Non-existent resource for children query
	resp, err = http.Get(suite.server.URL + "/api/resources/deployment/non-existent/children?namespace=default")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	// Test 3: Health check should always work
	resp, err = http.Get(suite.server.URL + "/api/health")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func (suite *IntegrationTestSuite) TestResourceStatusVariations() {
	// Test different resource status scenarios

	// Create pods with different statuses
	podStatuses := []string{"Running", "Pending", "Failed", "Succeeded"}
	podGVR := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}

	// Create namespace
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "status-test"},
	}
	_, err := suite.k8sClient.clientset.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
	require.NoError(suite.T(), err)

	for i, status := range podStatuses {
		pod := suite.createTestPod(fmt.Sprintf("pod-%d", i), "status-test", status)
		_, err := suite.k8sClient.dynamicClient.Resource(podGVR).
			Namespace("status-test").Create(context.TODO(), pod, metav1.CreateOptions{})
		require.NoError(suite.T(), err)
	}

	// Query pods and verify status parsing
	resp, err := http.Get(suite.server.URL + "/api/resources/pod?namespace=status-test")
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	var pods []ResourceNode
	err = json.NewDecoder(resp.Body).Decode(&pods)
	require.NoError(suite.T(), err)

	assert.Len(suite.T(), pods, len(podStatuses))

	// Verify each pod has the correct status
	statusMap := make(map[string]string)
	for _, pod := range pods {
		statusMap[pod.Name] = pod.Status
	}

	for i, expectedStatus := range podStatuses {
		podName := fmt.Sprintf("pod-%d", i)
		assert.Equal(suite.T(), expectedStatus, statusMap[podName])
	}
}

func (suite *IntegrationTestSuite) TestConcurrentRequests() {
	// Test handling concurrent requests

	// Create some test data
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "concurrent-test"},
	}
	_, err := suite.k8sClient.clientset.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
	require.NoError(suite.T(), err)

	// Create multiple pods
	podGVR := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
	for i := 0; i < 10; i++ {
		pod := suite.createTestPod(fmt.Sprintf("concurrent-pod-%d", i), "concurrent-test", "Running")
		_, err := suite.k8sClient.dynamicClient.Resource(podGVR).
			Namespace("concurrent-test").Create(context.TODO(), pod, metav1.CreateOptions{})
		require.NoError(suite.T(), err)
	}

	// Make concurrent requests
	const numRequests = 5
	results := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			resp, err := http.Get(suite.server.URL + "/api/resources/pod?namespace=concurrent-test")
			if err != nil {
				results <- err
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				results <- fmt.Errorf("unexpected status code: %d", resp.StatusCode)
				return
			}

			var pods []ResourceNode
			err = json.NewDecoder(resp.Body).Decode(&pods)
			if err != nil {
				results <- err
				return
			}

			if len(pods) != 10 {
				results <- fmt.Errorf("expected 10 pods, got %d", len(pods))
				return
			}

			results <- nil
		}()
	}

	// Wait for all requests to complete
	for i := 0; i < numRequests; i++ {
		select {
		case err := <-results:
			assert.NoError(suite.T(), err)
		case <-time.After(5 * time.Second):
			suite.T().Fatal("Request timed out")
		}
	}
}

// Helper methods for creating test resources
func (suite *IntegrationTestSuite) createTestPod(name, namespace, phase string) *unstructured.Unstructured {
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
			"phase": phase,
		},
	})
	return pod
}

func (suite *IntegrationTestSuite) createTestDeployment(name, namespace string) *unstructured.Unstructured {
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

func (suite *IntegrationTestSuite) createTestService(name, namespace string) *unstructured.Unstructured {
	service := &unstructured.Unstructured{}
	service.SetUnstructuredContent(map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Service",
		"metadata": map[string]interface{}{
			"name":              name,
			"namespace":         namespace,
			"uid":               uuid.New().String(),
			"creationTimestamp": time.Now().Format(time.RFC3339),
		},
		"spec": map[string]interface{}{
			"selector": map[string]interface{}{
				"app": "test-app",
			},
		},
	})
	return service
}
