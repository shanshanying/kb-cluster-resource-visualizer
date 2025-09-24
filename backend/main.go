package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type K8sClient struct {
	clientset       kubernetes.Interface
	dynamicClient   dynamic.Interface
	discoveryClient discovery.DiscoveryInterface
}

type ResourceNode struct {
	Name         string            `json:"name"`
	Kind         string            `json:"kind"`
	APIVersion   string            `json:"apiVersion"`
	Namespace    string            `json:"namespace,omitempty"`
	UID          string            `json:"uid"`
	Labels       map[string]string `json:"labels,omitempty"`
	Annotations  map[string]string `json:"annotations,omitempty"`
	CreationTime string            `json:"creationTime"`
	Status       string            `json:"status,omitempty"`
}

type ResourceRelationship struct {
	Parent   ResourceNode   `json:"parent"`
	Children []ResourceNode `json:"children"`
}

type TreeNode struct {
	Resource ResourceNode `json:"resource"`
	Children []*TreeNode  `json:"children"`
}

var k8sClient *K8sClient

func main() {
	log.Println("Starting K8s Resource Visualizer backend...")

	// Initialize Kubernetes client
	log.Println("Initializing Kubernetes client...")
	var err error
	k8sClient, err = initK8sClient()
	if err != nil {
		log.Fatalf("Failed to initialize Kubernetes client: %v", err)
	}
	log.Println("✓ Kubernetes client initialized successfully")

	// Initialize Gin router
	log.Println("Setting up HTTP router and middleware...")
	router := gin.Default()

	// Configure CORS
	log.Println("Configuring CORS middleware...")
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	router.Use(cors.New(config))
	log.Println("✓ CORS middleware configured")

	// API routes
	log.Println("Registering API routes...")
	api := router.Group("/api")
	{
		api.GET("/health", healthCheck)
		api.GET("/resources/:type", getResourcesByType)
		api.GET("/resources/:type/:root/tree", getResourceChildren)
		api.GET("/namespaces", getNamespaces)
	}
	log.Println("✓ API routes registered:")
	log.Println("  - GET /api/health")
	log.Println("  - GET /api/resources/:type")
	log.Println("  - GET /api/resources/:type/:root/tree")
	log.Println("  - GET /api/namespaces")

	log.Println("🚀 Server starting on :8080")
	log.Println("Ready to accept requests...")
	log.Fatal(http.ListenAndServe(":8080", router))
}

func initK8sClient() (*K8sClient, error) {
	var config *rest.Config
	var err error

	// Try in-cluster config first
	log.Println("Attempting to use in-cluster Kubernetes configuration...")
	config, err = rest.InClusterConfig()
	if err != nil {
		log.Printf("In-cluster config not available: %v", err)
		// Fallback to kubeconfig
		log.Println("Falling back to kubeconfig file...")
		var kubeconfig string
		if home := homedir.HomeDir(); home != "" {
			kubeconfig = filepath.Join(home, ".kube", "config")
			log.Printf("Using default kubeconfig path: %s", kubeconfig)
		}
		if envKubeconfig := os.Getenv("KUBECONFIG"); envKubeconfig != "" {
			kubeconfig = envKubeconfig
			log.Printf("Using KUBECONFIG environment variable: %s", kubeconfig)
		}

		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to build config from kubeconfig %s: %v", kubeconfig, err)
		}
		log.Printf("✓ Successfully loaded kubeconfig from: %s", kubeconfig)
	} else {
		log.Println("✓ Using in-cluster Kubernetes configuration")
	}

	// Create clientset
	log.Println("Creating Kubernetes clientset...")
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %v", err)
	}
	log.Println("✓ Kubernetes clientset created successfully")

	// Create dynamic client
	log.Println("Creating dynamic client...")
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %v", err)
	}
	log.Println("✓ Dynamic client created successfully")

	// Create discovery client
	log.Println("Creating discovery client...")
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create discovery client: %v", err)
	}
	log.Println("✓ Discovery client created successfully")

	return &K8sClient{
		clientset:       clientset,
		dynamicClient:   dynamicClient,
		discoveryClient: discoveryClient,
	}, nil
}

func healthCheck(c *gin.Context) {
	log.Printf("Health check requested from %s", c.ClientIP())
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"message": "K8s Resource Visualizer API is running",
	})
}

func getNamespaces(c *gin.Context) {
	log.Printf("Fetching namespaces requested from %s", c.ClientIP())
	namespaces, err := k8sClient.clientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Printf("Error fetching namespaces: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var namespaceList []string
	for _, ns := range namespaces.Items {
		namespaceList = append(namespaceList, ns.Name)
	}
	log.Printf("Found %d namespaces: %v", len(namespaceList), namespaceList)

	c.JSON(http.StatusOK, namespaceList)
}

func getResourcesByType(c *gin.Context) {
	resourceType := c.Param("type")
	namespace := c.Query("namespace")
	log.Printf("Fetching resources of type '%s' from namespace '%s' requested from %s", resourceType, namespace, c.ClientIP())

	// Get GVR for the resource type
	log.Printf("Resolving GVR for resource type: %s", resourceType)
	gvr, err := getGVRForResourceType(resourceType)
	if err != nil {
		log.Printf("Unknown resource type '%s': %v", resourceType, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Unknown resource type: %s", resourceType)})
		return
	}
	log.Printf("Resolved GVR: %+v", gvr)

	var resources []ResourceNode

	if namespace != "" {
		// Get resources from specific namespace
		log.Printf("Fetching resources from namespace: %s", namespace)
		resourceList, err := k8sClient.dynamicClient.Resource(gvr).Namespace(namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			log.Printf("Error fetching resources from namespace %s: %v", namespace, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		log.Printf("Found %d resources in namespace %s", len(resourceList.Items), namespace)
		resources = convertToResourceNodes(resourceList.Items)
	} else {
		// Try to get cluster-wide resources first
		log.Println("Attempting to fetch cluster-wide resources...")
		resourceList, err := k8sClient.dynamicClient.Resource(gvr).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			log.Printf("Cluster-wide resource fetch failed: %v. Trying all namespaces...", err)
			// If cluster-wide fails, try all namespaces
			namespaces, nsErr := k8sClient.clientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
			if nsErr != nil {
				log.Printf("Error fetching namespaces: %v", nsErr)
				c.JSON(http.StatusInternalServerError, gin.H{"error": nsErr.Error()})
				return
			}
			log.Printf("Searching across %d namespaces...", len(namespaces.Items))

			for _, ns := range namespaces.Items {
				nsResourceList, nsErr := k8sClient.dynamicClient.Resource(gvr).Namespace(ns.Name).List(context.TODO(), metav1.ListOptions{})
				if nsErr != nil {
					log.Printf("Skipping namespace %s due to error: %v", ns.Name, nsErr)
					continue // Skip namespaces where we can't list resources
				}
				if len(nsResourceList.Items) > 0 {
					log.Printf("Found %d resources in namespace %s", len(nsResourceList.Items), ns.Name)
				}
				resources = append(resources, convertToResourceNodes(nsResourceList.Items)...)
			}
		} else {
			log.Printf("Found %d cluster-wide resources", len(resourceList.Items))
			resources = convertToResourceNodes(resourceList.Items)
		}
	}

	log.Printf("Returning %d resources of type %s", len(resources), resourceType)
	c.JSON(http.StatusOK, resources)
}

func getResourceChildren(c *gin.Context) {
	resourceType := c.Param("type")
	rootResourceName := c.Param("root")
	namespace := c.Query("namespace")
	log.Printf("Building resource tree with %s/%s as root node in namespace '%s' requested from %s", resourceType, rootResourceName, namespace, c.ClientIP())

	// Get the root resource that will serve as the tree's root node
	log.Printf("Resolving GVR for root resource type: %s", resourceType)
	gvr, err := getGVRForResourceType(resourceType)
	if err != nil {
		log.Printf("Unknown resource type '%s': %v", resourceType, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Unknown resource type: %s", resourceType)})
		return
	}

	// For tree structure building, we require a namespace to be specified
	if namespace == "" {
		log.Printf("Namespace is required for building resource tree")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Namespace parameter is required for building resource tree"})
		return
	}

	var rootResource *unstructured.Unstructured
	log.Printf("Fetching root resource: %s/%s in namespace %s", resourceType, rootResourceName, namespace)
	rootResource, err = k8sClient.dynamicClient.Resource(gvr).Namespace(namespace).Get(context.TODO(), rootResourceName, metav1.GetOptions{})

	if err != nil {
		log.Printf("Root resource not found: %s/%s in namespace %s: %v", resourceType, rootResourceName, namespace, err)
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Root resource not found: %s/%s in namespace %s", resourceType, rootResourceName, namespace)})
		return
	}
	log.Printf("Found root resource: %s (UID: %s)", rootResource.GetName(), rootResource.GetUID())

	// Build tree structure starting from the root resource
	log.Printf("Building tree structure with root node: %s/%s...", rootResource.GetKind(), rootResource.GetName())
	rootTreeNode, err := buildResourceTree(rootResource, rootResourceName)
	if err != nil {
		log.Printf("Error building resource tree: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Return tree structure as an array with the root node
	treeArray := []*TreeNode{rootTreeNode}
	totalNodes := countTreeNodes(treeArray)
	log.Printf("Successfully built resource tree with root %s/%s containing %d total nodes", rootResource.GetKind(), rootResource.GetName(), totalNodes)

	c.JSON(http.StatusOK, treeArray)
}

func getGVRForResourceType(resourceType string) (schema.GroupVersionResource, error) {
	// Common resource mappings (including KubeBlocks custom resources)
	resourceMappings := map[string]schema.GroupVersionResource{
		// Standard Kubernetes resources
		"pod":                    {Group: "", Version: "v1", Resource: "pods"},
		"pods":                   {Group: "", Version: "v1", Resource: "pods"},
		"service":                {Group: "", Version: "v1", Resource: "services"},
		"services":               {Group: "", Version: "v1", Resource: "services"},
		"deployment":             {Group: "apps", Version: "v1", Resource: "deployments"},
		"deployments":            {Group: "apps", Version: "v1", Resource: "deployments"},
		"replicaset":             {Group: "apps", Version: "v1", Resource: "replicasets"},
		"replicasets":            {Group: "apps", Version: "v1", Resource: "replicasets"},
		"statefulset":            {Group: "apps", Version: "v1", Resource: "statefulsets"},
		"statefulsets":           {Group: "apps", Version: "v1", Resource: "statefulsets"},
		"daemonset":              {Group: "apps", Version: "v1", Resource: "daemonsets"},
		"daemonsets":             {Group: "apps", Version: "v1", Resource: "daemonsets"},
		"configmap":              {Group: "", Version: "v1", Resource: "configmaps"},
		"configmaps":             {Group: "", Version: "v1", Resource: "configmaps"},
		"secret":                 {Group: "", Version: "v1", Resource: "secrets"},
		"secrets":                {Group: "", Version: "v1", Resource: "secrets"},
		"ingress":                {Group: "networking.k8s.io", Version: "v1", Resource: "ingresses"},
		"ingresses":              {Group: "networking.k8s.io", Version: "v1", Resource: "ingresses"},
		"job":                    {Group: "batch", Version: "v1", Resource: "jobs"},
		"jobs":                   {Group: "batch", Version: "v1", Resource: "jobs"},
		"cronjob":                {Group: "batch", Version: "v1", Resource: "cronjobs"},
		"cronjobs":               {Group: "batch", Version: "v1", Resource: "cronjobs"},
		"persistentvolumeclaim":  {Group: "", Version: "v1", Resource: "persistentvolumeclaims"},
		"persistentvolumeclaims": {Group: "", Version: "v1", Resource: "persistentvolumeclaims"},
		"pvc":                    {Group: "", Version: "v1", Resource: "persistentvolumeclaims"},

		// KubeBlocks custom resources
		"cluster":             {Group: "apps.kubeblocks.io", Version: "v1", Resource: "clusters"},
		"clusters":            {Group: "apps.kubeblocks.io", Version: "v1", Resource: "clusters"},
		"component":           {Group: "apps.kubeblocks.io", Version: "v1", Resource: "components"},
		"components":          {Group: "apps.kubeblocks.io", Version: "v1", Resource: "components"},
		"cmp":                 {Group: "apps.kubeblocks.io", Version: "v1", Resource: "components"},
		"backuppolicy":        {Group: "dataprotection.kubeblocks.io", Version: "v1alpha1", Resource: "backuppolicies"},
		"backuppolicies":      {Group: "dataprotection.kubeblocks.io", Version: "v1alpha1", Resource: "backuppolicies"},
		"bp":                  {Group: "dataprotection.kubeblocks.io", Version: "v1alpha1", Resource: "backuppolicies"},
		"backup":              {Group: "dataprotection.kubeblocks.io", Version: "v1alpha1", Resource: "backups"},
		"backups":             {Group: "dataprotection.kubeblocks.io", Version: "v1alpha1", Resource: "backups"},
		"backupschedule":      {Group: "dataprotection.kubeblocks.io", Version: "v1alpha1", Resource: "backupschedules"},
		"backupschedules":     {Group: "dataprotection.kubeblocks.io", Version: "v1alpha1", Resource: "backupschedules"},
		"bs":                  {Group: "dataprotection.kubeblocks.io", Version: "v1alpha1", Resource: "backupschedules"},
		"restore":             {Group: "dataprotection.kubeblocks.io", Version: "v1alpha1", Resource: "restores"},
		"restores":            {Group: "dataprotection.kubeblocks.io", Version: "v1alpha1", Resource: "restores"},
		"opsrequest":          {Group: "operations.kubeblocks.io", Version: "v1alpha1", Resource: "opsrequests"},
		"opsrequests":         {Group: "operations.kubeblocks.io", Version: "v1alpha1", Resource: "opsrequests"},
		"ops":                 {Group: "operations.kubeblocks.io", Version: "v1alpha1", Resource: "opsrequests"},
		"componentparameter":  {Group: "parameters.kubeblocks.io", Version: "v1alpha1", Resource: "componentparameters"},
		"componentparameters": {Group: "parameters.kubeblocks.io", Version: "v1alpha1", Resource: "componentparameters"},
		"parameter":           {Group: "parameters.kubeblocks.io", Version: "v1alpha1", Resource: "parameters"},
		"parameters":          {Group: "parameters.kubeblocks.io", Version: "v1alpha1", Resource: "parameters"},
		"instance":            {Group: "workloads.kubeblocks.io", Version: "v1", Resource: "instances"},
		"instances":           {Group: "workloads.kubeblocks.io", Version: "v1", Resource: "instances"},
		"inst":                {Group: "workloads.kubeblocks.io", Version: "v1", Resource: "instances"},
		"instanceset":         {Group: "workloads.kubeblocks.io", Version: "v1", Resource: "instancesets"},
		"instancesets":        {Group: "workloads.kubeblocks.io", Version: "v1", Resource: "instancesets"},
		"its":                 {Group: "workloads.kubeblocks.io", Version: "v1", Resource: "instancesets"},
	}

	// Normalize resource type (lowercase)
	normalizedType := strings.ToLower(resourceType)

	if gvr, exists := resourceMappings[normalizedType]; exists {
		return gvr, nil
	}

	return schema.GroupVersionResource{}, fmt.Errorf("unknown resource type: %s", resourceType)
}

// isResourceTypeMatch checks if a GVR matches a Kubernetes resource kind
func isResourceTypeMatch(gvr schema.GroupVersionResource, kind string) bool {
	// Map of resource names to their corresponding kinds (including KubeBlocks resources)
	resourceToKindMap := map[string]string{
		// Standard Kubernetes resources
		"pods":                   "Pod",
		"services":               "Service",
		"configmaps":             "ConfigMap",
		"secrets":                "Secret",
		"persistentvolumeclaims": "PersistentVolumeClaim",
		"replicasets":            "ReplicaSet",
		"deployments":            "Deployment",
		"statefulsets":           "StatefulSet",
		"daemonsets":             "DaemonSet",
		"jobs":                   "Job",
		"cronjobs":               "CronJob",
		"ingresses":              "Ingress",

		// KubeBlocks custom resources
		"clusters":            "Cluster",
		"components":          "Component",
		"backuppolicies":      "BackupPolicy",
		"backups":             "Backup",
		"backupschedules":     "BackupSchedule",
		"restores":            "Restore",
		"opsrequests":         "OpsRequest",
		"componentparameters": "ComponentParameter",
		"parameters":          "Parameter",
		"instances":           "Instance",
		"instancesets":        "InstanceSet",
	}

	expectedKind, exists := resourceToKindMap[gvr.Resource]
	if !exists {
		return false
	}

	return expectedKind == kind
}

func findChildResources(parentResource *unstructured.Unstructured, rootResourceName string) ([]ResourceNode, error) {
	var children []ResourceNode
	parentUID := parentResource.GetUID()
	parentNamespace := parentResource.GetNamespace()
	parentKind := parentResource.GetKind()
	parentName := parentResource.GetName()

	log.Printf("🔍 Searching for children of %s/%s (UID: %s) in namespace: %s", parentKind, parentName, parentUID, parentNamespace)
	log.Printf("🏷️  Using label selector: app.kubernetes.io/instance=%s", rootResourceName)
	// Define child resource types to check (including KubeBlocks custom resources)
	commonChildTypes := []schema.GroupVersionResource{
		// Standard Kubernetes resources
		{Group: "", Version: "v1", Resource: "pods"},
		{Group: "", Version: "v1", Resource: "services"},
		{Group: "", Version: "v1", Resource: "configmaps"},
		{Group: "", Version: "v1", Resource: "persistentvolumeclaims"},
		{Group: "apps", Version: "v1", Resource: "deployments"},
		{Group: "apps", Version: "v1", Resource: "statefulsets"},
		{Group: "batch", Version: "v1", Resource: "jobs"},

		// KubeBlocks custom resources
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

	// Filter out resource types that are the same as parent type
	var filteredChildTypes []schema.GroupVersionResource
	for _, gvr := range commonChildTypes {
		// Skip if the child type is the same as parent type
		if isResourceTypeMatch(gvr, parentKind) {
			log.Printf("Skipping resource type %s as it matches parent type %s", gvr.Resource, parentKind)
			continue
		}
		filteredChildTypes = append(filteredChildTypes, gvr)
	}

	// Search through filtered resource types only
	for _, gvr := range filteredChildTypes {
		log.Printf("Checking for children of type: %s", gvr.Resource)

		// Use label selector to optimize search - look for resources with app.kubernetes.io/instance label
		labelSelector := fmt.Sprintf("app.kubernetes.io/instance=%s", rootResourceName)
		listOptions := metav1.ListOptions{
			LabelSelector: labelSelector,
		}

		var resourceList *unstructured.UnstructuredList
		var err error

		// Only search in the same namespace as parent (or cluster-wide if parent is cluster-scoped)
		if parentNamespace != "" {
			// Parent is namespaced, search only in parent's namespace
			resourceList, err = k8sClient.dynamicClient.Resource(gvr).Namespace(parentNamespace).List(context.TODO(), listOptions)
		} else {
			// Parent is cluster-scoped, search cluster-wide
			resourceList, err = k8sClient.dynamicClient.Resource(gvr).List(context.TODO(), listOptions)
		}

		if err != nil {
			log.Printf("Skipping resource type %s due to error: %v", gvr.Resource, err)
			continue // Skip resources we can't list
		}

		// Check each resource for ownerReference matching parent UID and verify label
		foundChildren := 0
		for _, resource := range resourceList.Items {
			// Double-check the label (in case label selector didn't work perfectly)
			labels := resource.GetLabels()
			if labels == nil {
				continue
			}

			instanceLabel, hasInstanceLabel := labels["app.kubernetes.io/instance"]
			if !hasInstanceLabel || instanceLabel != rootResourceName {
				log.Printf("Resource %s/%s doesn't have correct app.kubernetes.io/instance label", resource.GetKind(), resource.GetName())
				continue
			}

			// Check ownerReference for additional validation
			ownerRefs := resource.GetOwnerReferences()
			hasOwnerRef := false
			for _, ownerRef := range ownerRefs {
				if ownerRef.UID == parentUID {
					hasOwnerRef = true
					break
				}
			}

			// Accept resource if it has either ownerReference OR the instance label
			// (some resources might not have ownerReference but still be managed by the parent)
			if hasOwnerRef || hasInstanceLabel {
				children = append(children, convertToResourceNode(resource))
				foundChildren++
				log.Printf("Found child: %s/%s (has ownerRef: %t, has instance label: %t)",
					resource.GetKind(), resource.GetName(), hasOwnerRef, hasInstanceLabel)
			}
		}

		if foundChildren > 0 {
			log.Printf("Found %d children of type %s", foundChildren, gvr.Resource)
		}
	}

	log.Printf("Total children found: %d", len(children))
	return children, nil
}

func convertToResourceNodes(resources []unstructured.Unstructured) []ResourceNode {
	var nodes []ResourceNode
	for _, resource := range resources {
		nodes = append(nodes, convertToResourceNode(resource))
	}
	return nodes
}

func convertToResourceNode(resource unstructured.Unstructured) ResourceNode {
	status := "Unknown"
	if statusObj, found, err := unstructured.NestedFieldNoCopy(resource.Object, "status"); found && err == nil {
		if statusMap, ok := statusObj.(map[string]interface{}); ok {
			if phase, found, err := unstructured.NestedString(statusMap, "phase"); found && err == nil {
				status = phase
			} else if conditions, found, err := unstructured.NestedSlice(statusMap, "conditions"); found && err == nil {
				for _, condition := range conditions {
					if condMap, ok := condition.(map[string]interface{}); ok {
						if condType, found, err := unstructured.NestedString(condMap, "type"); found && err == nil {
							if condStatus, found, err := unstructured.NestedString(condMap, "status"); found && err == nil {
								if condType == "Ready" && condStatus == "True" {
									status = "Ready"
									break
								}
							}
						}
					}
				}
			}
		}
	}

	return ResourceNode{
		Name:         resource.GetName(),
		Kind:         resource.GetKind(),
		APIVersion:   resource.GetAPIVersion(),
		Namespace:    resource.GetNamespace(),
		UID:          string(resource.GetUID()),
		Labels:       resource.GetLabels(),
		Annotations:  resource.GetAnnotations(),
		CreationTime: resource.GetCreationTimestamp().Time.Format("2006-01-02 15:04:05"),
		Status:       status,
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// buildResourceTree recursively builds a tree structure starting from a root resource
func buildResourceTree(resource *unstructured.Unstructured, rootResourceName string) (*TreeNode, error) {
	resourceKind := resource.GetKind()
	resourceName := resource.GetName()

	log.Printf("🌳 Building subtree for %s/%s (searching for children with root label: %s)", resourceKind, resourceName, rootResourceName)

	node := &TreeNode{
		Resource: convertToResourceNode(*resource),
		Children: []*TreeNode{},
	}

	// Find direct children of this resource using the original root resource name for label matching
	children, err := findChildResources(resource, rootResourceName)
	if err != nil {
		return nil, fmt.Errorf("error finding children for %s/%s: %v", resourceKind, resourceName, err)
	}

	log.Printf("📊 Found %d direct children for %s/%s", len(children), resourceKind, resourceName)

	// For each child, recursively build its subtree
	for i, childResourceNode := range children {
		log.Printf("🔄 Processing child %d/%d: %s/%s", i+1, len(children), childResourceNode.Kind, childResourceNode.Name)

		// Convert ResourceNode back to unstructured.Unstructured to continue recursion
		childResource, err := getResourceByUID(childResourceNode.UID, resource.GetNamespace())
		if err != nil {
			log.Printf("⚠️  Could not fetch child resource %s/%s (UID: %s): %v", childResourceNode.Kind, childResourceNode.Name, childResourceNode.UID, err)
			// Create a leaf node if we can't fetch the full resource
			childNode := &TreeNode{
				Resource: childResourceNode,
				Children: []*TreeNode{},
			}
			node.Children = append(node.Children, childNode)
			continue
		}

		// Recursively build subtree for this child, passing the original root resource name
		log.Printf("↳ Recursively building subtree for %s/%s...", childResource.GetKind(), childResource.GetName())
		childNode, err := buildResourceTree(childResource, rootResourceName)
		if err != nil {
			log.Printf("⚠️  Error building subtree for %s/%s: %v", childResource.GetKind(), childResource.GetName(), err)
			// Create a leaf node if we can't build the subtree
			leafNode := &TreeNode{
				Resource: childResourceNode,
				Children: []*TreeNode{},
			}
			node.Children = append(node.Children, leafNode)
			continue
		}

		node.Children = append(node.Children, childNode)
		log.Printf("✅ Successfully built subtree for %s/%s", childResource.GetKind(), childResource.GetName())
	}

	return node, nil
}

// flattenTree converts a tree structure to a flat list of nodes
func flattenTree(root *TreeNode) []*TreeNode {
	var result []*TreeNode

	// Add root node
	result = append(result, root)

	// Recursively add all children
	for _, child := range root.Children {
		result = append(result, flattenTree(child)...)
	}

	return result
}

// getResourceByUID fetches a resource by its UID within a namespace
func getResourceByUID(uid string, namespace string) (*unstructured.Unstructured, error) {
	// Search through all supported resource types including KubeBlocks custom resources

	allResourceTypes := []schema.GroupVersionResource{
		// Standard Kubernetes resources
		{Group: "", Version: "v1", Resource: "pods"},
		{Group: "", Version: "v1", Resource: "services"},
		{Group: "", Version: "v1", Resource: "configmaps"},
		{Group: "", Version: "v1", Resource: "secrets"},
		{Group: "", Version: "v1", Resource: "persistentvolumeclaims"},
		{Group: "apps", Version: "v1", Resource: "replicasets"},
		{Group: "apps", Version: "v1", Resource: "deployments"},
		{Group: "apps", Version: "v1", Resource: "statefulsets"},
		{Group: "apps", Version: "v1", Resource: "daemonsets"},
		{Group: "batch", Version: "v1", Resource: "jobs"},
		{Group: "networking.k8s.io", Version: "v1", Resource: "ingresses"},

		// KubeBlocks custom resources
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

	for _, gvr := range allResourceTypes {
		var resourceList *unstructured.UnstructuredList
		var err error

		if namespace != "" {
			resourceList, err = k8sClient.dynamicClient.Resource(gvr).Namespace(namespace).List(context.TODO(), metav1.ListOptions{})
		} else {
			resourceList, err = k8sClient.dynamicClient.Resource(gvr).List(context.TODO(), metav1.ListOptions{})
		}

		if err != nil {
			continue // Skip resource types we can't list
		}

		for _, resource := range resourceList.Items {
			if string(resource.GetUID()) == uid {
				return &resource, nil
			}
		}
	}

	return nil, fmt.Errorf("resource with UID %s not found", uid)
}

// countTreeNodes counts the total number of nodes in a tree structure
func countTreeNodes(nodes []*TreeNode) int {
	count := 0
	for _, node := range nodes {
		count += 1 + countTreeNodes(node.Children)
	}
	return count
}
