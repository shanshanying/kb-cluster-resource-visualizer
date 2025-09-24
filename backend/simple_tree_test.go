package main

import (
	"context"
	"encoding/json"
	"fmt"
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
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery/fake"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

func TestTreeAPI_SimpleHierarchy(t *testing.T) {
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

	// Set up router
	router := gin.New()
	api := router.Group("/api")
	{
		api.GET("/resources/:type/:root/tree", func(c *gin.Context) {
			originalClient := k8sClient
			k8sClient = testK8sClient
			defer func() { k8sClient = originalClient }()
			getResourceChildren(c)
		})
	}

	// Test data
	namespace := "test-namespace"
	rootResourceName := "nginx-app"

	// Create test resources
	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      rootResourceName,
			Namespace: namespace,
			UID:       types.UID(uuid.New().String()),
			Labels: map[string]string{
				"app.kubernetes.io/name":     rootResourceName,
				"app.kubernetes.io/instance": rootResourceName,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": rootResourceName,
				},
			},
		},
	}

	service := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      rootResourceName + "-svc",
			Namespace: namespace,
			UID:       types.UID(uuid.New().String()),
			Labels: map[string]string{
				"app.kubernetes.io/name":     rootResourceName,
				"app.kubernetes.io/instance": rootResourceName,
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					UID: deployment.UID,
				},
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": rootResourceName,
			},
			Ports: []corev1.ServicePort{
				{
					Port: 80,
				},
			},
		},
	}

	// Add resources to fake client
	resources := []runtime.Object{deployment, service}
	for _, resource := range resources {
		unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(resource)
		require.NoError(t, err)

		unstructuredResource := &unstructured.Unstructured{Object: unstructuredObj}
		gvr := getGVRFromResource(resource)

		_, err = testK8sClient.dynamicClient.Resource(gvr).Namespace(namespace).Create(
			context.TODO(), unstructuredResource, metav1.CreateOptions{})
		require.NoError(t, err)
	}

	// Test the tree API
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/resources/deployment/%s/tree?namespace=%s", rootResourceName, namespace), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)

	var response []*TreeNode
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Basic assertions
	assert.Len(t, response, 1, "Should return exactly one root node")

	rootNode := response[0]
	assert.Equal(t, "Deployment", rootNode.Resource.Kind)
	assert.Equal(t, rootResourceName, rootNode.Resource.Name)
	assert.Equal(t, namespace, rootNode.Resource.Namespace)

	// Verify we found the service as a child
	assert.Greater(t, len(rootNode.Children), 0, "Root node should have at least one child")

	// Look for the service child
	var foundService bool
	for _, child := range rootNode.Children {
		if child.Resource.Kind == "Service" && child.Resource.Name == rootResourceName+"-svc" {
			foundService = true
			break
		}
	}
	assert.True(t, foundService, "Should find the service as a child")
}

func TestTreeAPI_ErrorHandling_Simple(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create minimal test setup
	scheme := runtime.NewScheme()
	corev1.AddToScheme(scheme)
	appsv1.AddToScheme(scheme)
	batchv1.AddToScheme(scheme)
	networkingv1.AddToScheme(scheme)

	fakeClientset := k8sfake.NewSimpleClientset()
	fakeDynamicClient := dynamicfake.NewSimpleDynamicClient(scheme)
	fakeDiscoveryClient := &fake.FakeDiscovery{
		Fake: &fakeClientset.Fake,
	}

	testK8sClient := &K8sClient{
		clientset:       fakeClientset,
		dynamicClient:   fakeDynamicClient,
		discoveryClient: fakeDiscoveryClient,
	}

	// Set up router
	router := gin.New()
	api := router.Group("/api")
	{
		api.GET("/resources/:type/:root/tree", func(c *gin.Context) {
			originalClient := k8sClient
			k8sClient = testK8sClient
			defer func() { k8sClient = originalClient }()
			getResourceChildren(c)
		})
	}

	t.Run("Resource not found", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/resources/deployment/nonexistent/tree?namespace=default", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Missing namespace", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/resources/deployment/test/tree", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Invalid resource type", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/resources/invalidtype/test/tree?namespace=default", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// Helper functions
func getGVRFromResource(obj runtime.Object) schema.GroupVersionResource {
	switch obj.(type) {
	case *appsv1.Deployment:
		return schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
	case *corev1.Service:
		return schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"}
	case *corev1.Pod:
		return schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
	case *corev1.ConfigMap:
		return schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}
	}
	return schema.GroupVersionResource{}
}

func int32Ptr(i int32) *int32 {
	return &i
}
