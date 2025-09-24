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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery/fake"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

// TreeAPITestSuite for testing the new tree API
type TreeAPITestSuite struct {
	router    *gin.Engine
	k8sClient *K8sClient
}

func setupTreeAPITest() *TreeAPITestSuite {
	gin.SetMode(gin.TestMode)

	// Create a scheme and add known types
	scheme := runtime.NewScheme()
	corev1.AddToScheme(scheme)
	appsv1.AddToScheme(scheme)
	batchv1.AddToScheme(scheme)

	// Create fake clients
	fakeClientset := k8sfake.NewSimpleClientset()
	fakeDynamicClient := dynamicfake.NewSimpleDynamicClient(scheme)
	fakeDiscoveryClient := &fake.FakeDiscovery{
		Fake: &fakeClientset.Fake,
	}

	// Create K8sClient
	k8sClient := &K8sClient{
		clientset:       fakeClientset,
		dynamicClient:   fakeDynamicClient,
		discoveryClient: fakeDiscoveryClient,
	}

	// Set up router with new tree API endpoint
	router := gin.New()
	api := router.Group("/api")
	{
		api.GET("/health", healthCheck)
		api.GET("/resources/:type", getResourcesByType)
		api.GET("/resources/:type/:root/tree", getResourceChildren)
		api.GET("/namespaces", getNamespaces)
	}

	return &TreeAPITestSuite{
		router:    router,
		k8sClient: k8sClient,
	}
}

func TestTreeAPI_BasicTreeStructure(t *testing.T) {
	suite := setupTreeAPITest()

	// Set the global k8sClient for the handlers
	originalClient := k8sClient
	k8sClient = suite.k8sClient
	defer func() { k8sClient = originalClient }()

	// Create test resources with proper labels and owner references
	namespace := "test-namespace"
	rootResourceName := "nginx-app"

	// Create root deployment
	deployment := createTestDeployment(rootResourceName, namespace)

	// Create child resources
	replicaSet := createTestReplicaSet(rootResourceName+"-rs", namespace, rootResourceName, string(deployment.UID))
	pod1 := createTestPod(rootResourceName+"-pod1", namespace, rootResourceName, string(replicaSet.UID))
	pod2 := createTestPod(rootResourceName+"-pod2", namespace, rootResourceName, string(replicaSet.UID))
	service := createTestService(rootResourceName+"-svc", namespace, rootResourceName, string(deployment.UID))
	configMap := createTestConfigMap(rootResourceName+"-config", namespace, rootResourceName, string(deployment.UID))

	// Add resources to fake client
	resources := []runtime.Object{deployment, replicaSet, pod1, pod2, service, configMap}
	for _, resource := range resources {
		unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(resource)
		require.NoError(t, err)

		unstructuredResource := &unstructured.Unstructured{Object: unstructuredObj}
		gvr := getGVRFromObject(resource)

		_, err = suite.k8sClient.dynamicClient.Resource(gvr).Namespace(namespace).Create(
			context.TODO(), unstructuredResource, metav1.CreateOptions{})
		require.NoError(t, err)
	}

	// Test the tree API
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/resources/deployment/%s/tree?namespace=%s", rootResourceName, namespace), nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response []*TreeNode
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Verify response structure
	assert.Len(t, response, 1, "Should return exactly one root node")

	rootNode := response[0]
	assert.Equal(t, "Deployment", rootNode.Resource.Kind)
	assert.Equal(t, rootResourceName, rootNode.Resource.Name)
	assert.Equal(t, namespace, rootNode.Resource.Namespace)

	// Verify children are found
	assert.Greater(t, len(rootNode.Children), 0, "Root node should have children")

	// Count total nodes in tree
	totalNodes := countNodesInTree(rootNode)
	assert.Equal(t, 6, totalNodes, "Tree should contain all 6 resources")
}

func TestTreeAPI_RecursiveTreeBuilding(t *testing.T) {
	suite := setupTreeAPITest()

	// Set the global k8sClient for the handlers
	originalClient := k8sClient
	k8sClient = suite.k8sClient
	defer func() { k8sClient = originalClient }()

	namespace := "test-namespace"
	rootResourceName := "recursive-app"

	// Create a 3-level hierarchy: Deployment -> ReplicaSet -> Pod
	deployment := createTestDeployment(rootResourceName, namespace)
	replicaSet := createTestReplicaSet(rootResourceName+"-rs", namespace, rootResourceName, string(deployment.UID))
	pod := createTestPod(rootResourceName+"-pod", namespace, rootResourceName, string(replicaSet.UID))

	// Add resources to fake client
	resources := []runtime.Object{deployment, replicaSet, pod}
	for _, resource := range resources {
		unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(resource)
		require.NoError(t, err)

		unstructuredResource := &unstructured.Unstructured{Object: unstructuredObj}
		gvr := getGVRFromObject(resource)

		_, err = suite.k8sClient.dynamicClient.Resource(gvr).Namespace(namespace).Create(
			context.TODO(), unstructuredResource, metav1.CreateOptions{})
		require.NoError(t, err)
	}

	// Test the tree API
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/resources/deployment/%s/tree?namespace=%s", rootResourceName, namespace), nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response []*TreeNode
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Verify 3-level hierarchy
	rootNode := response[0]
	assert.Equal(t, "Deployment", rootNode.Resource.Kind)

	// Find ReplicaSet child
	var replicaSetNode *TreeNode
	for _, child := range rootNode.Children {
		if child.Resource.Kind == "ReplicaSet" {
			replicaSetNode = child
			break
		}
	}
	require.NotNil(t, replicaSetNode, "Should find ReplicaSet child")

	// Find Pod grandchild
	var podNode *TreeNode
	for _, grandchild := range replicaSetNode.Children {
		if grandchild.Resource.Kind == "Pod" {
			podNode = grandchild
			break
		}
	}
	require.NotNil(t, podNode, "Should find Pod grandchild")

	// Verify the complete 3-level hierarchy
	assert.Equal(t, rootResourceName+"-rs", replicaSetNode.Resource.Name)
	assert.Equal(t, rootResourceName+"-pod", podNode.Resource.Name)
}

func TestTreeAPI_KubeBlocksResources(t *testing.T) {
	suite := setupTreeAPITest()

	// Set the global k8sClient for the handlers
	originalClient := k8sClient
	k8sClient = suite.k8sClient
	defer func() { k8sClient = originalClient }()

	namespace := "test-namespace"
	rootResourceName := "kubeblocks-app"

	// Create a KubeBlocks Cluster as root
	cluster := createTestKubeBlocksCluster(rootResourceName, namespace)

	// Create KubeBlocks Component as child
	component := createTestKubeBlocksComponent(rootResourceName+"-comp", namespace, rootResourceName, string(cluster.GetUID()))

	// Create standard Kubernetes resources as children of the component
	pod := createTestPod(rootResourceName+"-comp-pod", namespace, rootResourceName, string(component.GetUID()))

	// Add resources to fake client
	resources := []runtime.Object{cluster, component, pod}
	for _, resource := range resources {
		unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(resource)
		require.NoError(t, err)

		unstructuredResource := &unstructured.Unstructured{Object: unstructuredObj}
		gvr := getGVRFromObject(resource)

		_, err = suite.k8sClient.dynamicClient.Resource(gvr).Namespace(namespace).Create(
			context.TODO(), unstructuredResource, metav1.CreateOptions{})
		require.NoError(t, err)
	}

	// Test the tree API with KubeBlocks cluster as root
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/resources/cluster/%s/tree?namespace=%s", rootResourceName, namespace), nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response []*TreeNode
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Verify KubeBlocks hierarchy
	rootNode := response[0]
	assert.Equal(t, "Cluster", rootNode.Resource.Kind)
	assert.Equal(t, rootResourceName, rootNode.Resource.Name)

	// Should find mixed KubeBlocks and standard Kubernetes resources
	totalNodes := countNodesInTree(rootNode)
	assert.Equal(t, 3, totalNodes, "Tree should contain Cluster, Component, and Pod")
}

func TestTreeAPI_ErrorHandling(t *testing.T) {
	suite := setupTreeAPITest()

	// Set the global k8sClient for the handlers
	originalClient := k8sClient
	k8sClient = suite.k8sClient
	defer func() { k8sClient = originalClient }()

	t.Run("Resource not found", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/resources/deployment/nonexistent/tree?namespace=default", nil)
		w := httptest.NewRecorder()
		suite.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Missing namespace", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/resources/deployment/test/tree", nil)
		w := httptest.NewRecorder()
		suite.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Invalid resource type", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/resources/invalidtype/test/tree?namespace=default", nil)
		w := httptest.NewRecorder()
		suite.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestTreeAPI_EmptyTree(t *testing.T) {
	suite := setupTreeAPITest()

	// Set the global k8sClient for the handlers
	originalClient := k8sClient
	k8sClient = suite.k8sClient
	defer func() { k8sClient = originalClient }()

	namespace := "test-namespace"
	rootResourceName := "lonely-app"

	// Create only root deployment with no children
	deployment := createTestDeployment(rootResourceName, namespace)

	unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(deployment)
	require.NoError(t, err)

	unstructuredResource := &unstructured.Unstructured{Object: unstructuredObj}
	gvr := getGVRFromObject(deployment)

	_, err = suite.k8sClient.dynamicClient.Resource(gvr).Namespace(namespace).Create(
		context.TODO(), unstructuredResource, metav1.CreateOptions{})
	require.NoError(t, err)

	// Test the tree API
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/resources/deployment/%s/tree?namespace=%s", rootResourceName, namespace), nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response []*TreeNode
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Verify single node tree
	assert.Len(t, response, 1)
	rootNode := response[0]
	assert.Equal(t, "Deployment", rootNode.Resource.Kind)
	assert.Equal(t, rootResourceName, rootNode.Resource.Name)
	assert.Len(t, rootNode.Children, 0, "Root node should have no children")
}

// Helper functions for creating test resources

func createTestDeployment(name, namespace string) *appsv1.Deployment {
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			UID:       generateUIDType(),
			Labels: map[string]string{
				"app.kubernetes.io/name":     name,
				"app.kubernetes.io/instance": name,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":                        name,
						"app.kubernetes.io/instance": name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "nginx",
							Image: "nginx:latest",
						},
					},
				},
			},
		},
	}
}

func createTestReplicaSet(name, namespace, instanceLabel string, ownerUID string) *appsv1.ReplicaSet {
	return &appsv1.ReplicaSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ReplicaSet",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			UID:       generateUIDType(),
			Labels: map[string]string{
				"app.kubernetes.io/name":     instanceLabel,
				"app.kubernetes.io/instance": instanceLabel,
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					UID: types.UID(ownerUID),
				},
			},
		},
		Spec: appsv1.ReplicaSetSpec{
			Replicas: int32Ptr(2),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": instanceLabel,
				},
			},
		},
	}
}

func createTestPod(name, namespace, instanceLabel string, ownerUID string) *corev1.Pod {
	return &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			UID:       generateUIDType(),
			Labels: map[string]string{
				"app.kubernetes.io/name":     instanceLabel,
				"app.kubernetes.io/instance": instanceLabel,
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					UID: types.UID(ownerUID),
				},
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "nginx",
					Image: "nginx:latest",
				},
			},
		},
	}
}

func createTestService(name, namespace, instanceLabel string, ownerUID string) *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			UID:       generateUIDType(),
			Labels: map[string]string{
				"app.kubernetes.io/name":     instanceLabel,
				"app.kubernetes.io/instance": instanceLabel,
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					UID: types.UID(ownerUID),
				},
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": instanceLabel,
			},
			Ports: []corev1.ServicePort{
				{
					Port: 80,
				},
			},
		},
	}
}

func createTestConfigMap(name, namespace, instanceLabel string, ownerUID string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			UID:       generateUIDType(),
			Labels: map[string]string{
				"app.kubernetes.io/name":     instanceLabel,
				"app.kubernetes.io/instance": instanceLabel,
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					UID: types.UID(ownerUID),
				},
			},
		},
		Data: map[string]string{
			"config.yaml": "test: value",
		},
	}
}

func createTestKubeBlocksCluster(name, namespace string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apps.kubeblocks.io/v1",
			"kind":       "Cluster",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
				"uid":       string(generateUID()),
				"labels": map[string]interface{}{
					"app.kubernetes.io/name":     name,
					"app.kubernetes.io/instance": name,
				},
			},
			"spec": map[string]interface{}{
				"clusterDefinitionRef": "postgresql",
			},
		},
	}
}

func createTestKubeBlocksComponent(name, namespace, instanceLabel string, ownerUID string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apps.kubeblocks.io/v1",
			"kind":       "Component",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
				"uid":       string(generateUID()),
				"labels": map[string]interface{}{
					"app.kubernetes.io/name":     instanceLabel,
					"app.kubernetes.io/instance": instanceLabel,
				},
				"ownerReferences": []interface{}{
					map[string]interface{}{
						"uid": ownerUID,
					},
				},
			},
			"spec": map[string]interface{}{
				"compDef": "postgresql",
			},
		},
	}
}

// Helper functions

func generateUID() string {
	return uuid.New().String()
}

func generateUIDType() types.UID {
	return types.UID(uuid.New().String())
}

func generateUIDFromString(uid string) string {
	if uid == "" {
		return uuid.New().String()
	}
	return uid
}

func int32Ptr(i int32) *int32 {
	return &i
}

func getGVRFromObject(obj runtime.Object) schema.GroupVersionResource {
	switch obj.(type) {
	case *appsv1.Deployment:
		return schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
	case *appsv1.ReplicaSet:
		return schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "replicasets"}
	case *corev1.Pod:
		return schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
	case *corev1.Service:
		return schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"}
	case *corev1.ConfigMap:
		return schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}
	case *unstructured.Unstructured:
		u := obj.(*unstructured.Unstructured)
		gvk := u.GroupVersionKind()
		if gvk.Group == "apps.kubeblocks.io" && gvk.Kind == "Cluster" {
			return schema.GroupVersionResource{Group: "apps.kubeblocks.io", Version: "v1", Resource: "clusters"}
		}
		if gvk.Group == "apps.kubeblocks.io" && gvk.Kind == "Component" {
			return schema.GroupVersionResource{Group: "apps.kubeblocks.io", Version: "v1", Resource: "components"}
		}
	}
	return schema.GroupVersionResource{}
}

func countNodesInTree(node *TreeNode) int {
	count := 1 // Count current node
	for _, child := range node.Children {
		count += countNodesInTree(child)
	}
	return count
}
