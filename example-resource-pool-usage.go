package main

import (
	"fmt"
	"log"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
)

// Example demonstrating how to use the ResourcePool printing functions
func main() {
	fmt.Println("🧪 Resource Pool Printing Example")
	fmt.Println("==================================")

	// Create a sample resource pool
	pool := NewResourcePool()

	// Add some mock resources to demonstrate the functionality
	addMockResources(pool)

	fmt.Println("1️⃣  Resource Pool Summary:")
	pool.PrintResourcePoolSummary()

	fmt.Println("\n2️⃣  Detailed Resource Pool:")
	pool.PrintResourcePool()

	fmt.Println("3️⃣  Simulating Resource Removal:")
	// Get all resources and remove them one by one to show the pool shrinking
	allResources := pool.GetAllResources()
	for i, resource := range allResources {
		if i >= 3 { // Only remove first 3 for demo
			break
		}
		fmt.Printf("Removing %s/%s...\n", resource.GetKind(), resource.GetName())
		pool.RemoveResource(resource.GetUID())
		fmt.Printf("Pool size now: %d\n", pool.Size())
	}

	fmt.Println("\n4️⃣  Final Pool State:")
	pool.PrintResourcePoolSummary()
}

// addMockResources adds some sample resources to the pool for demonstration
func addMockResources(pool *ResourcePool) {
	// Create a mock cluster (root resource)
	cluster := &unstructured.Unstructured{}
	cluster.SetKind("Cluster")
	cluster.SetName("test-cluster")
	cluster.SetUID(types.UID("cluster-uid-123"))
	cluster.SetNamespace("default")
	pool.AddResource(cluster)

	// Create a mock component (child of cluster)
	component := &unstructured.Unstructured{}
	component.SetKind("Component")
	component.SetName("test-component")
	component.SetUID(types.UID("component-uid-456"))
	component.SetNamespace("default")
	component.SetOwnerReferences([]metav1.OwnerReference{
		{
			UID:  cluster.GetUID(),
			Kind: cluster.GetKind(),
			Name: cluster.GetName(),
		},
	})
	pool.AddResource(component)

	// Create a mock pod (child of component)
	pod := &unstructured.Unstructured{}
	pod.SetKind("Pod")
	pod.SetName("test-pod")
	pod.SetUID(types.UID("pod-uid-789"))
	pod.SetNamespace("default")
	pod.SetOwnerReferences([]metav1.OwnerReference{
		{
			UID:  component.GetUID(),
			Kind: component.GetKind(),
			Name: component.GetName(),
		},
	})
	pool.AddResource(pod)

	// Create another mock pod
	pod2 := &unstructured.Unstructured{}
	pod2.SetKind("Pod")
	pod2.SetName("test-pod-2")
	pod2.SetUID(types.UID("pod-uid-abc"))
	pod2.SetNamespace("default")
	pod2.SetOwnerReferences([]metav1.OwnerReference{
		{
			UID:  component.GetUID(),
			Kind: component.GetKind(),
			Name: component.GetName(),
		},
	})
	pool.AddResource(pod2)

	// Create a service (another root resource)
	service := &unstructured.Unstructured{}
	service.SetKind("Service")
	service.SetName("test-service")
	service.SetUID(types.UID("service-uid-def"))
	service.SetNamespace("default")
	pool.AddResource(service)

	log.Printf("Added %d mock resources to pool", pool.Size())
}
