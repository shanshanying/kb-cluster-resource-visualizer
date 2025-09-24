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
	clientset       *kubernetes.Clientset
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
	// Initialize Kubernetes client
	var err error
	k8sClient, err = initK8sClient()
	if err != nil {
		log.Fatalf("Failed to initialize Kubernetes client: %v", err)
	}

	// Initialize Gin router
	router := gin.Default()

	// Configure CORS
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	router.Use(cors.New(config))

	// API routes
	api := router.Group("/api")
	{
		api.GET("/health", healthCheck)
		api.GET("/resources/:type", getResourcesByType)
		api.GET("/resources/:type/:name/children", getResourceChildren)
		api.GET("/namespaces", getNamespaces)
	}

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}

func initK8sClient() (*K8sClient, error) {
	var config *rest.Config
	var err error

	// Try in-cluster config first
	config, err = rest.InClusterConfig()
	if err != nil {
		// Fallback to kubeconfig
		var kubeconfig string
		if home := homedir.HomeDir(); home != "" {
			kubeconfig = filepath.Join(home, ".kube", "config")
		}
		if envKubeconfig := os.Getenv("KUBECONFIG"); envKubeconfig != "" {
			kubeconfig = envKubeconfig
		}

		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to build config: %v", err)
		}
	}

	// Create clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %v", err)
	}

	// Create dynamic client
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %v", err)
	}

	// Create discovery client
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create discovery client: %v", err)
	}

	return &K8sClient{
		clientset:       clientset,
		dynamicClient:   dynamicClient,
		discoveryClient: discoveryClient,
	}, nil
}

func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"message": "K8s Resource Visualizer API is running",
	})
}

func getNamespaces(c *gin.Context) {
	namespaces, err := k8sClient.clientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var namespaceList []string
	for _, ns := range namespaces.Items {
		namespaceList = append(namespaceList, ns.Name)
	}

	c.JSON(http.StatusOK, namespaceList)
}

func getResourcesByType(c *gin.Context) {
	resourceType := c.Param("type")
	namespace := c.Query("namespace")

	// Get GVR for the resource type
	gvr, err := getGVRForResourceType(resourceType)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Unknown resource type: %s", resourceType)})
		return
	}

	var resources []ResourceNode

	if namespace != "" {
		// Get resources from specific namespace
		resourceList, err := k8sClient.dynamicClient.Resource(gvr).Namespace(namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		resources = convertToResourceNodes(resourceList.Items)
	} else {
		// Try to get cluster-wide resources first
		resourceList, err := k8sClient.dynamicClient.Resource(gvr).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			// If cluster-wide fails, try all namespaces
			namespaces, nsErr := k8sClient.clientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
			if nsErr != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": nsErr.Error()})
				return
			}

			for _, ns := range namespaces.Items {
				nsResourceList, nsErr := k8sClient.dynamicClient.Resource(gvr).Namespace(ns.Name).List(context.TODO(), metav1.ListOptions{})
				if nsErr != nil {
					continue // Skip namespaces where we can't list resources
				}
				resources = append(resources, convertToResourceNodes(nsResourceList.Items)...)
			}
		} else {
			resources = convertToResourceNodes(resourceList.Items)
		}
	}

	c.JSON(http.StatusOK, resources)
}

func getResourceChildren(c *gin.Context) {
	resourceType := c.Param("type")
	resourceName := c.Param("name")
	namespace := c.Query("namespace")

	// Get the parent resource
	gvr, err := getGVRForResourceType(resourceType)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Unknown resource type: %s", resourceType)})
		return
	}

	var parentResource *unstructured.Unstructured
	if namespace != "" {
		parentResource, err = k8sClient.dynamicClient.Resource(gvr).Namespace(namespace).Get(context.TODO(), resourceName, metav1.GetOptions{})
	} else {
		parentResource, err = k8sClient.dynamicClient.Resource(gvr).Get(context.TODO(), resourceName, metav1.GetOptions{})
	}

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Resource not found: %s/%s", resourceType, resourceName)})
		return
	}

	// Find all resources that have this resource as ownerReference
	children, err := findChildResources(parentResource)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	parent := convertToResourceNode(*parentResource)
	relationship := ResourceRelationship{
		Parent:   parent,
		Children: children,
	}

	c.JSON(http.StatusOK, relationship)
}

func getGVRForResourceType(resourceType string) (schema.GroupVersionResource, error) {
	// Common resource mappings
	resourceMappings := map[string]schema.GroupVersionResource{
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
	}

	// Normalize resource type (lowercase)
	normalizedType := strings.ToLower(resourceType)

	if gvr, exists := resourceMappings[normalizedType]; exists {
		return gvr, nil
	}

	return schema.GroupVersionResource{}, fmt.Errorf("unknown resource type: %s", resourceType)
}

func findChildResources(parentResource *unstructured.Unstructured) ([]ResourceNode, error) {
	var children []ResourceNode
	parentUID := parentResource.GetUID()

	// Get all API resources
	apiResourceLists, err := k8sClient.discoveryClient.ServerPreferredResources()
	if err != nil {
		return nil, fmt.Errorf("failed to get API resources: %v", err)
	}

	// Search through all resource types
	for _, apiResourceList := range apiResourceLists {
		gv, err := schema.ParseGroupVersion(apiResourceList.GroupVersion)
		if err != nil {
			continue
		}

		for _, apiResource := range apiResourceList.APIResources {
			if !contains(apiResource.Verbs, "list") {
				continue
			}

			gvr := schema.GroupVersionResource{
				Group:    gv.Group,
				Version:  gv.Version,
				Resource: apiResource.Name,
			}

			// Try to list resources
			var resourceList *unstructured.UnstructuredList
			if apiResource.Namespaced {
				// For namespaced resources, search in the same namespace as parent or all namespaces
				if parentResource.GetNamespace() != "" {
					resourceList, err = k8sClient.dynamicClient.Resource(gvr).Namespace(parentResource.GetNamespace()).List(context.TODO(), metav1.ListOptions{})
				} else {
					// Search all namespaces
					namespaces, nsErr := k8sClient.clientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
					if nsErr != nil {
						continue
					}

					var allResources []unstructured.Unstructured
					for _, ns := range namespaces.Items {
						nsList, nsErr := k8sClient.dynamicClient.Resource(gvr).Namespace(ns.Name).List(context.TODO(), metav1.ListOptions{})
						if nsErr != nil {
							continue
						}
						allResources = append(allResources, nsList.Items...)
					}
					resourceList = &unstructured.UnstructuredList{Items: allResources}
				}
			} else {
				// For cluster-scoped resources
				resourceList, err = k8sClient.dynamicClient.Resource(gvr).List(context.TODO(), metav1.ListOptions{})
			}

			if err != nil {
				continue // Skip resources we can't list
			}

			// Check each resource for ownerReference
			for _, resource := range resourceList.Items {
				ownerRefs := resource.GetOwnerReferences()
				for _, ownerRef := range ownerRefs {
					if ownerRef.UID == parentUID {
						children = append(children, convertToResourceNode(resource))
						break
					}
				}
			}
		}
	}

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
