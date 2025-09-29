package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

// ResourceTreeNode represents a node in the resource tree
type ResourceTreeNode struct {
	Resource *unstructured.Unstructured `json:"resource"`
	Children []*ResourceTreeNode        `json:"children"`
}

// ResourcePool manages a pool of resources for efficient tree building
type ResourcePool struct {
	resources map[types.UID]*unstructured.Unstructured
	byOwner   map[types.UID][]*unstructured.Unstructured
}

// ResourceTreeBuilder builds resource trees based on ownerReference relationships
type ResourceTreeBuilder struct {
	client      *K8sClient
	namespace   string
	visited     map[types.UID]bool // To prevent cycles
	listOptions metav1.ListOptions
	pool        *ResourcePool // Resource pool for efficient lookups
}

// NewResourceTreeBuilder creates a new ResourceTreeBuilder
func NewResourceTreeBuilder(client *K8sClient, namespace string, listOptions metav1.ListOptions) *ResourceTreeBuilder {
	return &ResourceTreeBuilder{
		client:      client,
		namespace:   namespace,
		visited:     make(map[types.UID]bool),
		listOptions: listOptions,
		pool:        nil, // Will be built when needed
	}
}

// NewResourcePool creates a new ResourcePool
func NewResourcePool() *ResourcePool {
	return &ResourcePool{
		resources: make(map[types.UID]*unstructured.Unstructured),
		byOwner:   make(map[types.UID][]*unstructured.Unstructured),
	}
}

// AddResource adds a resource to the pool and indexes it by owner references
func (rp *ResourcePool) AddResource(resource *unstructured.Unstructured) {
	uid := resource.GetUID()
	rp.resources[uid] = resource

	// Index by owner references
	ownerReferences := resource.GetOwnerReferences()
	for _, ownerRef := range ownerReferences {
		if rp.byOwner[ownerRef.UID] == nil {
			rp.byOwner[ownerRef.UID] = make([]*unstructured.Unstructured, 0)
		}
		rp.byOwner[ownerRef.UID] = append(rp.byOwner[ownerRef.UID], resource)
	}
}

// GetChildrenByOwner returns all resources that have the specified owner UID
func (rp *ResourcePool) GetChildrenByOwner(ownerUID types.UID) []*unstructured.Unstructured {
	return rp.byOwner[ownerUID]
}

// GetResource returns a resource by its UID
func (rp *ResourcePool) GetResource(uid types.UID) *unstructured.Unstructured {
	return rp.resources[uid]
}

// Size returns the number of resources in the pool
func (rp *ResourcePool) Size() int {
	return len(rp.resources)
}

// GetRootResources returns all resources that have no owner references
func (rp *ResourcePool) GetRootResources() []*unstructured.Unstructured {
	var roots []*unstructured.Unstructured
	for _, resource := range rp.resources {
		ownerReferences := resource.GetOwnerReferences()
		if len(ownerReferences) == 0 {
			roots = append(roots, resource)
		}
	}
	return roots
}

// GetAllResources returns all resources in the pool
func (rp *ResourcePool) GetAllResources() []*unstructured.Unstructured {
	resources := make([]*unstructured.Unstructured, 0, len(rp.resources))
	for _, resource := range rp.resources {
		resources = append(resources, resource)
	}
	return resources
}

// PrintResourcePool prints a detailed view of the resource pool for debugging
func (rp *ResourcePool) PrintResourcePool() {
	fmt.Printf("📦 Resource Pool Summary\n")
	fmt.Printf("========================\n")
	fmt.Printf("Total resources: %d\n", len(rp.resources))
	fmt.Printf("Owner relationships: %d\n", len(rp.byOwner))
	fmt.Printf("\n")

	if len(rp.resources) == 0 {
		fmt.Printf("🔍 Pool is empty\n")
		return
	}

	// Group resources by kind
	resourcesByKind := make(map[string][]*unstructured.Unstructured)
	rootResources := make([]*unstructured.Unstructured, 0)

	for _, resource := range rp.resources {
		kind := resource.GetKind()
		if resourcesByKind[kind] == nil {
			resourcesByKind[kind] = make([]*unstructured.Unstructured, 0)
		}
		resourcesByKind[kind] = append(resourcesByKind[kind], resource)

		// Check if it's a root resource (no owner references)
		if len(resource.GetOwnerReferences()) == 0 {
			rootResources = append(rootResources, resource)
		}
	}

	// Print resources grouped by kind
	fmt.Printf("📋 Resources by Kind:\n")
	for kind, resources := range resourcesByKind {
		fmt.Printf("  %s: %d\n", kind, len(resources))
		for _, resource := range resources {
			ownerCount := len(resource.GetOwnerReferences())
			childrenCount := len(rp.byOwner[resource.GetUID()])
			fmt.Printf("    - %s (UID: %s, owners: %d, children: %d)\n",
				resource.GetName(), resource.GetUID(), ownerCount, childrenCount)
		}
	}

	// Print root resources
	fmt.Printf("\n🌱 Root Resources (no owners): %d\n", len(rootResources))
	for _, root := range rootResources {
		childrenCount := len(rp.byOwner[root.GetUID()])
		fmt.Printf("  - %s/%s (children: %d)\n", root.GetKind(), root.GetName(), childrenCount)
	}

	// Print ownership relationships
	fmt.Printf("\n🔗 Ownership Relationships:\n")
	for ownerUID, children := range rp.byOwner {
		var ownerName string = "Unknown"
		var ownerKind string = "Unknown"

		if owner := rp.resources[ownerUID]; owner != nil {
			ownerName = owner.GetName()
			ownerKind = owner.GetKind()
		}

		fmt.Printf("  %s/%s (UID: %s) -> %d children:\n", ownerKind, ownerName, ownerUID, len(children))
		for _, child := range children {
			fmt.Printf("    - %s/%s\n", child.GetKind(), child.GetName())
		}
	}

	fmt.Printf("\n")
}

// PrintResourcePoolSummary prints a compact summary of the resource pool
func (rp *ResourcePool) PrintResourcePoolSummary() {
	fmt.Printf("📦 Pool: %d resources, %d ownership relationships\n",
		len(rp.resources), len(rp.byOwner))

	// Count by kind
	kindCounts := make(map[string]int)
	rootCount := 0

	for _, resource := range rp.resources {
		kind := resource.GetKind()
		kindCounts[kind]++

		if len(resource.GetOwnerReferences()) == 0 {
			rootCount++
		}
	}

	fmt.Printf("📊 Kinds: ")
	first := true
	for kind, count := range kindCounts {
		if !first {
			fmt.Printf(", ")
		}
		fmt.Printf("%s(%d)", kind, count)
		first = false
	}
	fmt.Printf("\n🌱 Roots: %d\n", rootCount)
}

// buildResourcePool builds a pool of all resources matching the ListOptions
func (rtb *ResourceTreeBuilder) buildResourcePool() error {
	log.Printf("🏗️  Building resource pool...")

	rtb.pool = NewResourcePool()
	resourceTypes := rtb.getSupportedResourceTypes()

	totalResources := 0
	for _, gvr := range resourceTypes {
		log.Printf("  📦 Loading resource type: %s", gvr.Resource)

		var resourceList *unstructured.UnstructuredList
		var err error

		// Search in the specified namespace or cluster-wide
		if rtb.namespace != "" {
			resourceList, err = rtb.client.dynamicClient.Resource(gvr).Namespace(rtb.namespace).List(context.TODO(), rtb.listOptions)
		} else {
			resourceList, err = rtb.client.dynamicClient.Resource(gvr).List(context.TODO(), rtb.listOptions)
		}

		if err != nil {
			log.Printf("    ⚠️  Skipping resource type %s due to error: %v", gvr.Resource, err)
			continue
		}

		// Add all resources to the pool
		resourceCount := 0
		for i := range resourceList.Items {
			resource := &resourceList.Items[i]
			rtb.pool.AddResource(resource)
			resourceCount++
		}

		if resourceCount > 0 {
			log.Printf("    ✅ Added %d resources of type %s", resourceCount, gvr.Resource)
			totalResources += resourceCount
		}
	}

	log.Printf("🎯 Resource pool built successfully with %d total resources", totalResources)

	// Print resource pool summary for debugging
	log.Printf("📊 Resource Pool Summary:")
	rtb.pool.PrintResourcePoolSummary()
	rtb.pool.PrintResourcePool()

	return nil
}

// GetResourceTree builds a complete resource tree with the given resource as root
// This function implements pure ownerReference-based tree building using resource pool
func (rtb *ResourceTreeBuilder) GetResourceTree(rootResource *unstructured.Unstructured) (*ResourceTreeNode, error) {
	if rootResource == nil {
		return nil, fmt.Errorf("root resource cannot be nil")
	}

	// Build resource pool if not already built
	if rtb.pool == nil {
		if err := rtb.buildResourcePool(); err != nil {
			return nil, fmt.Errorf("failed to build resource pool: %v", err)
		}
	}

	return rtb.buildTreeFromPool(rootResource)
}

// buildTreeFromPool builds a tree using the pre-built resource pool
func (rtb *ResourceTreeBuilder) buildTreeFromPool(rootResource *unstructured.Unstructured) (*ResourceTreeNode, error) {
	rootUID := rootResource.GetUID()
	if rtb.visited[rootUID] {
		log.Printf("⚠️  Cycle detected for resource %s/%s (UID: %s)", rootResource.GetKind(), rootResource.GetName(), rootUID)
		return &ResourceTreeNode{
			Resource: rootResource,
			Children: []*ResourceTreeNode{},
		}, nil
	}

	// Mark this resource as visited to prevent cycles
	rtb.visited[rootUID] = true
	defer func() {
		rtb.visited[rootUID] = false // Reset for other branches
	}()

	log.Printf("🌳 Building tree node for %s/%s (UID: %s)",
		rootResource.GetKind(), rootResource.GetName(), rootUID)

	node := &ResourceTreeNode{
		Resource: rootResource,
		Children: []*ResourceTreeNode{},
	}

	// Find all child resources that have this resource as owner from the pool
	children := rtb.pool.GetChildrenByOwner(rootUID)
	log.Printf("📊 Found %d direct children for %s/%s from resource pool",
		len(children), rootResource.GetKind(), rootResource.GetName())

	// Recursively build subtrees for each child
	for _, child := range children {
		// Remove the child from pool since it's now being used
		log.Printf("🔍 Removing child %s/%s (UID: %s) from resource pool (remaining: %d)",
			child.GetKind(), child.GetName(), child.GetUID(), rtb.pool.Size()-1)
		// rtb.pool.RemoveResource(child.GetUID())

		childNode, err := rtb.buildTreeFromPool(child)
		if err != nil {
			log.Printf("⚠️  Error building subtree for %s/%s: %v",
				child.GetKind(), child.GetName(), err)
			// Create a leaf node for this child
			leafNode := &ResourceTreeNode{
				Resource: child,
				Children: []*ResourceTreeNode{},
			}
			node.Children = append(node.Children, leafNode)
			continue
		}
		node.Children = append(node.Children, childNode)
	}

	log.Printf("✅ Successfully built tree node for %s/%s with %d children",
		rootResource.GetKind(), rootResource.GetName(), len(node.Children))

	return node, nil
}

// GetAllResourceTrees builds trees for all root resources (resources without owners)
func (rtb *ResourceTreeBuilder) GetAllResourceTrees() ([]*ResourceTreeNode, error) {
	// Build resource pool if not already built
	if rtb.pool == nil {
		if err := rtb.buildResourcePool(); err != nil {
			return nil, fmt.Errorf("failed to build resource pool: %v", err)
		}
	}

	roots := rtb.pool.GetRootResources()
	log.Printf("🌲 Found %d root resources to build trees from", len(roots))

	var trees []*ResourceTreeNode
	for _, root := range roots {
		// Reset visited map for each tree
		rtb.visited = make(map[types.UID]bool)

		tree, err := rtb.buildTreeFromPool(root)
		if err != nil {
			log.Printf("⚠️  Error building tree for root %s/%s: %v",
				root.GetKind(), root.GetName(), err)
			continue
		}
		trees = append(trees, tree)
	}

	log.Printf("🎯 Successfully built %d resource trees", len(trees))

	// Print final resource pool state
	log.Printf("📊 Final Resource Pool State:")
	rtb.pool.PrintResourcePoolSummary()
	rtb.pool.PrintResourcePool()
	if rtb.pool.Size() > 0 {
		log.Printf("⚠️  Warning: %d resources remain in pool (orphaned resources)", rtb.pool.Size())
	}

	return trees, nil
}

// hasOwnerReference checks if a resource has the specified UID as an owner
func (rtb *ResourceTreeBuilder) hasOwnerReference(resource *unstructured.Unstructured, ownerUID types.UID) bool {
	ownerReferences := resource.GetOwnerReferences()
	for _, ownerRef := range ownerReferences {
		if ownerRef.UID == ownerUID {
			return true
		}
	}
	return false
}

// getSupportedResourceTypes returns all resource types that should be searched for children
func (rtb *ResourceTreeBuilder) getSupportedResourceTypes() []schema.GroupVersionResource {
	return []schema.GroupVersionResource{
		// Core resources
		{Group: "", Version: "v1", Resource: "pods"},
		{Group: "", Version: "v1", Resource: "services"},
		{Group: "", Version: "v1", Resource: "configmaps"},
		// {Group: "", Version: "v1", Resource: "secrets"},
		{Group: "", Version: "v1", Resource: "persistentvolumeclaims"},
		// {Group: "", Version: "v1", Resource: "serviceaccounts"},
		// {Group: "", Version: "v1", Resource: "endpoints"},

		// Apps resources
		{Group: "apps", Version: "v1", Resource: "deployments"},
		{Group: "apps", Version: "v1", Resource: "replicasets"},
		{Group: "apps", Version: "v1", Resource: "statefulsets"},
		{Group: "apps", Version: "v1", Resource: "daemonsets"},

		// Batch resources
		{Group: "batch", Version: "v1", Resource: "jobs"},
		{Group: "batch", Version: "v1", Resource: "cronjobs"},

		// // Networking resources
		// {Group: "networking.k8s.io", Version: "v1", Resource: "ingresses"},
		// {Group: "networking.k8s.io", Version: "v1", Resource: "networkpolicies"},

		// // RBAC resources
		// {Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "roles"},
		// {Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "rolebindings"},
		// {Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "clusterroles"},
		// {Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "clusterrolebindings"},

		// KubeBlocks resources
		{Group: "apps.kubeblocks.io", Version: "v1", Resource: "clusters"},
		{Group: "apps.kubeblocks.io", Version: "v1", Resource: "components"},
		{Group: "dataprotection.kubeblocks.io", Version: "v1alpha1", Resource: "backuppolicies"},
		{Group: "dataprotection.kubeblocks.io", Version: "v1alpha1", Resource: "backups"},
		{Group: "dataprotection.kubeblocks.io", Version: "v1alpha1", Resource: "backupschedules"},
		{Group: "dataprotection.kubeblocks.io", Version: "v1alpha1", Resource: "restores"},
		{Group: "operations.kubeblocks.io", Version: "v1alpha1", Resource: "opsrequests"},
		{Group: "parameters.kubeblocks.io", Version: "v1alpha1", Resource: "componentparameters"},
		{Group: "parameters.kubeblocks.io", Version: "v1alpha1", Resource: "parameters"},
		{Group: "workloads.kubeblocks.io", Version: "v1", Resource: "instances"},
		{Group: "workloads.kubeblocks.io", Version: "v1", Resource: "instancesets"},
	}
}

// PrintTree prints the tree structure for debugging (optional utility function)
func (rtb *ResourceTreeBuilder) PrintTree(node *ResourceTreeNode, indent string) {
	if node == nil {
		return
	}

	resource := node.Resource
	fmt.Printf("%s%s/%s (UID: %s)\n",
		indent, resource.GetKind(), resource.GetName(), resource.GetUID())

	for i, child := range node.Children {
		childIndent := indent + "  "
		if i == len(node.Children)-1 {
			childIndent = indent + "  "
		}
		rtb.PrintTree(child, childIndent)
	}
}

// ValidateTree validates the tree structure for consistency
func (rtb *ResourceTreeBuilder) ValidateTree(node *ResourceTreeNode) error {
	if node == nil {
		return fmt.Errorf("tree node is nil")
	}

	if node.Resource == nil {
		return fmt.Errorf("resource in tree node is nil")
	}

	// Validate each child
	for i, child := range node.Children {
		if err := rtb.ValidateTree(child); err != nil {
			return fmt.Errorf("invalid child at index %d: %v", i, err)
		}

		// Verify parent-child relationship
		if !rtb.hasOwnerReference(child.Resource, node.Resource.GetUID()) {
			log.Printf("⚠️  Warning: Child %s/%s does not have ownerReference to parent %s/%s",
				child.Resource.GetKind(), child.Resource.GetName(),
				node.Resource.GetKind(), node.Resource.GetName())
		}
	}

	return nil
}

// CountNodes counts the total number of nodes in the tree
func (rtb *ResourceTreeBuilder) CountNodes(node *ResourceTreeNode) int {
	if node == nil {
		return 0
	}

	count := 1
	for _, child := range node.Children {
		count += rtb.CountNodes(child)
	}
	return count
}

// GetAllResources returns a flat list of all resources in the tree
func (rtb *ResourceTreeBuilder) GetAllResources(node *ResourceTreeNode) []*unstructured.Unstructured {
	if node == nil {
		return nil
	}

	resources := []*unstructured.Unstructured{node.Resource}
	for _, child := range node.Children {
		resources = append(resources, rtb.GetAllResources(child)...)
	}
	return resources
}

// FindResourceByUID finds a resource in the tree by its UID
func (rtb *ResourceTreeBuilder) FindResourceByUID(node *ResourceTreeNode, uid types.UID) *unstructured.Unstructured {
	if node == nil {
		return nil
	}

	if node.Resource.GetUID() == uid {
		return node.Resource
	}

	for _, child := range node.Children {
		if result := rtb.FindResourceByUID(child, uid); result != nil {
			return result
		}
	}

	return nil
}

// GetDepth returns the maximum depth of the tree
func (rtb *ResourceTreeBuilder) GetDepth(node *ResourceTreeNode) int {
	if node == nil || len(node.Children) == 0 {
		return 1
	}

	maxChildDepth := 0
	for _, child := range node.Children {
		childDepth := rtb.GetDepth(child)
		if childDepth > maxChildDepth {
			maxChildDepth = childDepth
		}
	}

	return 1 + maxChildDepth
}

// GetResourcesByKind returns all resources of a specific kind from the tree
func (rtb *ResourceTreeBuilder) GetResourcesByKind(node *ResourceTreeNode, kind string) []*unstructured.Unstructured {
	if node == nil {
		return nil
	}

	var resources []*unstructured.Unstructured

	if strings.EqualFold(node.Resource.GetKind(), kind) {
		resources = append(resources, node.Resource)
	}

	for _, child := range node.Children {
		resources = append(resources, rtb.GetResourcesByKind(child, kind)...)
	}

	return resources
}
