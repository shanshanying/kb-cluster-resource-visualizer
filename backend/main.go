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
		api.GET("/resources/:type/:root/tree", getResourceTree)
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
	// make sure namespace is not empty
	if namespace == "" {
		log.Printf("Namespace is required for fetching resources")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Namespace is required for fetching resources"})
		return
	}

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

	log.Printf("Returning %d resources of type %s", len(resources), resourceType)
	c.JSON(http.StatusOK, resources)
}

func getResourceTree(c *gin.Context) {
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

	// Build tree structure using the new ResourceTreeBuilder
	log.Printf("Building tree structure with root node: %s/%s...", rootResource.GetKind(), rootResource.GetName())
	// add a list option, each resource has a label: app.kubernetes.io/instance=rootResourceName
	listOptions := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app.kubernetes.io/instance=%s", rootResourceName),
	}
	// Create tree builder
	treeBuilder := NewResourceTreeBuilder(k8sClient, namespace, listOptions)

	// Build the tree using new format
	rootTreeNode, err := treeBuilder.GetResourceTree(rootResource)
	if err != nil {
		log.Printf("Error building resource tree: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Return tree structure as an array with the root node
	treeArray := []*ResourceTreeNode{rootTreeNode}
	totalNodes := treeBuilder.CountNodes(rootTreeNode)
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
