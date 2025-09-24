package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
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

// Performance tests for the simplified child resource discovery
func TestFindChildResourcesPerformance(t *testing.T) {
	// Setup test environment
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

	// Override global client for testing
	originalClient := k8sClient
	k8sClient = testK8sClient
	defer func() { k8sClient = originalClient }()

	// Create a test namespace
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "performance-test",
		},
	}
	_, err := k8sClient.clientset.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
	require.NoError(t, err)

	// Create a parent deployment
	parentDeployment := createTestDeploymentForPerf("parent-deployment", "performance-test")
	deploymentGVR := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
	createdParent, err := k8sClient.dynamicClient.Resource(deploymentGVR).
		Namespace("performance-test").Create(context.TODO(), parentDeployment, metav1.CreateOptions{})
	require.NoError(t, err)

	// Create multiple child resources
	podGVR := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
	serviceGVR := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"}
	configMapGVR := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}

	numChildren := 50 // Create 50 child resources of each type

	t.Run("Create test data", func(t *testing.T) {
		start := time.Now()

		// Create child pods
		for i := 0; i < numChildren; i++ {
			pod := createTestPodForPerf(fmt.Sprintf("child-pod-%d", i), "performance-test", "Running")
			// Add the required instance label
			labels := pod.GetLabels()
			if labels == nil {
				labels = make(map[string]string)
			}
			labels["app.kubernetes.io/instance"] = "parent-deployment"
			pod.SetLabels(labels)

			pod.SetOwnerReferences([]metav1.OwnerReference{
				{
					UID:        createdParent.GetUID(),
					Kind:       "Deployment",
					Name:       "parent-deployment",
					APIVersion: "apps/v1",
				},
			})
			_, err := k8sClient.dynamicClient.Resource(podGVR).
				Namespace("performance-test").Create(context.TODO(), pod, metav1.CreateOptions{})
			require.NoError(t, err)
		}

		// Create child services
		for i := 0; i < numChildren; i++ {
			service := createTestServiceForPerf(fmt.Sprintf("child-service-%d", i), "performance-test")
			// Add the required instance label
			labels := service.GetLabels()
			if labels == nil {
				labels = make(map[string]string)
			}
			labels["app.kubernetes.io/instance"] = "parent-deployment"
			service.SetLabels(labels)

			service.SetOwnerReferences([]metav1.OwnerReference{
				{
					UID:        createdParent.GetUID(),
					Kind:       "Deployment",
					Name:       "parent-deployment",
					APIVersion: "apps/v1",
				},
			})
			_, err := k8sClient.dynamicClient.Resource(serviceGVR).
				Namespace("performance-test").Create(context.TODO(), service, metav1.CreateOptions{})
			require.NoError(t, err)
		}

		// Create child configmaps
		for i := 0; i < numChildren; i++ {
			configMap := createTestConfigMapForPerf(fmt.Sprintf("child-configmap-%d", i), "performance-test")
			// Add the required instance label
			labels := configMap.GetLabels()
			if labels == nil {
				labels = make(map[string]string)
			}
			labels["app.kubernetes.io/instance"] = "parent-deployment"
			configMap.SetLabels(labels)

			configMap.SetOwnerReferences([]metav1.OwnerReference{
				{
					UID:        createdParent.GetUID(),
					Kind:       "Deployment",
					Name:       "parent-deployment",
					APIVersion: "apps/v1",
				},
			})
			_, err := k8sClient.dynamicClient.Resource(configMapGVR).
				Namespace("performance-test").Create(context.TODO(), configMap, metav1.CreateOptions{})
			require.NoError(t, err)
		}

		// Create some unrelated resources (should not be found as children)
		for i := 0; i < 20; i++ {
			unrelatedPod := createTestPodForPerf(fmt.Sprintf("unrelated-pod-%d", i), "performance-test", "Running")
			// No owner reference - should not be found as child
			_, err := k8sClient.dynamicClient.Resource(podGVR).
				Namespace("performance-test").Create(context.TODO(), unrelatedPod, metav1.CreateOptions{})
			require.NoError(t, err)
		}

		t.Logf("Test data creation took: %v", time.Since(start))
	})

	t.Run("Performance test - find child resources", func(t *testing.T) {
		// Measure performance of finding child resources
		start := time.Now()

		children, err := findChildResources(createdParent)
		require.NoError(t, err)

		duration := time.Since(start)

		// Verify we found the expected number of children
		expectedChildren := numChildren * 3 // pods + services + configmaps
		require.Equal(t, expectedChildren, len(children), "Should find exactly %d children", expectedChildren)

		t.Logf("Finding %d child resources took: %v", len(children), duration)
		t.Logf("Average time per child: %v", duration/time.Duration(len(children)))

		// Performance assertion - should complete within reasonable time
		// This is a generous limit for the fake client
		require.Less(t, duration, 5*time.Second, "Child resource discovery should complete within 5 seconds")

		// Verify child types
		childTypes := make(map[string]int)
		for _, child := range children {
			childTypes[child.Kind]++
		}

		require.Equal(t, numChildren, childTypes["Pod"], "Should find %d pods", numChildren)
		require.Equal(t, numChildren, childTypes["Service"], "Should find %d services", numChildren)
		require.Equal(t, numChildren, childTypes["ConfigMap"], "Should find %d configmaps", numChildren)
	})
}

// Benchmark the optimized findChildResources function
func BenchmarkFindChildResources(b *testing.B) {
	// Setup test environment (similar to above but for benchmarking)
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

	// Override global client
	originalClient := k8sClient
	k8sClient = testK8sClient
	defer func() { k8sClient = originalClient }()

	// Create test parent
	parentDeployment := createTestDeploymentForPerf("benchmark-deployment", "default")
	deploymentGVR := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
	createdParent, _ := k8sClient.dynamicClient.Resource(deploymentGVR).
		Namespace("default").Create(context.TODO(), parentDeployment, metav1.CreateOptions{})

	// Create some child resources
	podGVR := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
	for i := 0; i < 10; i++ {
		pod := createTestPodForPerf(fmt.Sprintf("benchmark-pod-%d", i), "default", "Running")
		pod.SetOwnerReferences([]metav1.OwnerReference{
			{
				UID:        createdParent.GetUID(),
				Kind:       "Deployment",
				Name:       "benchmark-deployment",
				APIVersion: "apps/v1",
			},
		})
		k8sClient.dynamicClient.Resource(podGVR).
			Namespace("default").Create(context.TODO(), pod, metav1.CreateOptions{})
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := findChildResources(createdParent)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Helper functions for performance tests
func createTestDeploymentForPerf(name, namespace string) *unstructured.Unstructured {
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
	})
	return deployment
}

func createTestPodForPerf(name, namespace, phase string) *unstructured.Unstructured {
	pod := &unstructured.Unstructured{}
	pod.SetUnstructuredContent(map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Pod",
		"metadata": map[string]interface{}{
			"name":              name,
			"namespace":         namespace,
			"uid":               uuid.New().String(),
			"creationTimestamp": time.Now().Format(time.RFC3339),
		},
		"status": map[string]interface{}{
			"phase": phase,
		},
	})
	return pod
}

func createTestServiceForPerf(name, namespace string) *unstructured.Unstructured {
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
	})
	return service
}

func createTestConfigMapForPerf(name, namespace string) *unstructured.Unstructured {
	configMap := &unstructured.Unstructured{}
	configMap.SetUnstructuredContent(map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata": map[string]interface{}{
			"name":              name,
			"namespace":         namespace,
			"uid":               uuid.New().String(),
			"creationTimestamp": time.Now().Format(time.RFC3339),
		},
		"data": map[string]interface{}{
			"key": "value",
		},
	})
	return configMap
}
