# KB Cluster Resource Visualizer

A Kubernetes resource visualization tool inspired by ArgoCD UI, built with React TypeScript frontend and Go backend.

## ✨ Features

- 🔍 **Resource Tree Visualization**: Visualize all resources with ownerReference relationships
- 🎨 **Modern Web Interface**: Interactive resource relationship graphs with drag & drop
- ⚡ **Real-time K8s Connection**: Live connection to Kubernetes clusters
- 📊 **Comprehensive Resource Support**: Support for all Kubernetes and KubeBlocks resource types
- 🎯 **Advanced Layout Algorithms**: Multiple tree layout algorithms (Hierarchical, Reingold-Tilford)
- 🏷️ **Rich Resource Information**: Status, labels, namespaces, and detailed metadata
- 🌈 **Resource Type Coloring**: Color-coded nodes based on resource categories
- 📱 **Responsive Design**: Optimized for different screen sizes and resolutions
- 🔧 **Layout Controls**: Vertical/horizontal layouts with customizable spacing

## 🏗️ Tech Stack

- **Frontend**: React + TypeScript + Vite + Ant Design + React Flow
- **Backend**: Go + Gin + client-go
- **Visualization**: React Flow interactive diagrams
- **Containerization**: Docker + Docker Compose
- **Layout Algorithms**: Custom Reingold-Tilford implementation + Hierarchical layout

## 🚀 Quick Start

### 📋 Prerequisites

- Go 1.21+
- Node.js 18+
- Accessible Kubernetes cluster
- kubectl configured

### 🎯 One-Click Start

```bash
# Clone the project
git clone <your-repo-url>
cd kb-cluster-resource-visualizer

# Use the startup script
./scripts/start.sh
```

### 🔧 Manual Setup

#### Backend Setup

```bash
cd backend
go mod tidy
go run main.go
```

#### Frontend Setup

```bash
cd frontend
npm install
npm run dev
```

### 🐳 Docker Setup

```bash
# Ensure ~/.kube/config exists and is accessible
docker-compose up -d
```

## 📖 Usage Guide

1. **Start Services**: Run both frontend and backend services
2. **Open Browser**: Navigate to http://localhost:5173
3. **Select Resource Type**: Choose resource type from dropdown (e.g., Deployment, Pod, etc.)
4. **Select Namespace**: Choose specific namespace or view all namespaces
5. **Select Resource**: Click on resources in the left panel
6. **View Relationship Graph**: The right panel will display the ownerReference relationship tree
7. **Customize Layout**: Choose between different layout algorithms and orientations
8. **Explore Nodes**: Hover over nodes to see detailed resource information

## 🔌 API Endpoints

- `GET /api/health` - Health check
- `GET /api/namespaces` - Get all namespaces
- `GET /api/resources/:type` - Get all resources of specified type
- `GET /api/tree` - Get resource tree with ownerReference relationships

### Request Examples

```bash
# Get all Deployments
curl "http://localhost:8080/api/resources/deployment"

# Get Pods in specific namespace
curl "http://localhost:8080/api/resources/pod?namespace=default"

# Get resource tree starting from a specific resource
curl "http://localhost:8080/api/tree?resource=deployment&name=my-app&namespace=default"
```

## 🎨 UI Overview

### Main Interface
- **Left Panel**: Resource selector with resource type and namespace filtering
- **Right Panel**: Interactive resource relationship graph with drag & zoom support
- **Control Panel**: Layout algorithm selector and orientation controls

### Resource Nodes
- **Color-coded by Type**: Different colors for workload, network, config, storage resources
- **Rich Information**: Resource name, type, status, and metadata
- **Interactive**: Hover for detailed information tooltips
- **Status Indicators**: Visual status badges and KubeBlocks role labels

### Layout Controls
- **Multiple Algorithms**: Hierarchical and Reingold-Tilford tree layouts
- **Orientation**: Vertical layout for hierarchy, horizontal for parallel relationships
- **Customizable Spacing**: Adjustable node and layer spacing for optimal viewing

## 📁 Project Structure

```
kb-cluster-resource-visualizer/
├── backend/                 # Go backend
│   ├── main.go             # Main application
│   ├── resource_tree.go    # Resource tree building logic
│   ├── go.mod              # Go module file
│   └── Dockerfile          # Backend Docker config
├── frontend/               # React frontend
│   ├── src/
│   │   ├── components/     # React components
│   │   │   ├── ResourceFlow.tsx     # Main visualization component
│   │   │   ├── ResourceNode.tsx     # Individual resource node
│   │   │   └── ResourceSelector.tsx # Resource selection panel
│   │   ├── services/       # API services
│   │   ├── types/          # TypeScript type definitions
│   │   ├── utils/          # Layout algorithms
│   │   └── App.tsx         # Main application component
│   ├── package.json        # NPM configuration
│   └── Dockerfile          # Frontend Docker config
├── scripts/
│   └── start.sh            # Startup script
├── docker-compose.yml      # Docker Compose configuration
└── README.md               # Project documentation
```

## 🔍 Supported Resource Types

### Kubernetes Resources
- 🔵 **Workload**: Deployment, ReplicaSet, StatefulSet, DaemonSet, Pod, Job, CronJob
- 🟣 **Network**: Service, Ingress, NetworkPolicy
- 🟡 **Configuration**: ConfigMap, Secret
- 🟢 **Storage**: PersistentVolumeClaim, PersistentVolume, StorageClass

### KubeBlocks Resources
- 🩵 **Cluster Management**: Cluster, Component, Instance, InstanceSet
- 🟤 **Backup & Restore**: Backup, BackupPolicy, BackupSchedule, Restore
- 🩷 **Operations**: OpsRequest

### Color Coding
Each resource type is color-coded for easy identification:
- Blue for workload resources
- Purple for network resources
- Yellow for configuration resources
- Green for storage resources
- Cyan for KubeBlocks cluster resources
- Brown for backup resources
- Pink for operation resources

## 🛠️ Development Guide

### Backend Development

```bash
cd backend
go mod tidy
go run main.go
```

The backend features:
- **Resource Pool Optimization**: Efficient tree building with O(n) complexity
- **Dynamic Resource Discovery**: Support for custom resource types
- **RESTful API**: Clean API design with proper error handling

### Frontend Development

```bash
cd frontend
npm install
npm run dev
```

The frontend features:
- **Advanced Layout Algorithms**: Reingold-Tilford and Hierarchical layouts
- **Resource Type Coloring**: Automatic color coding based on resource categories
- **Interactive Controls**: Layout algorithm selection and orientation controls
- **Responsive Design**: Optimized for different screen sizes

### Build Production Version

```bash
# Backend
cd backend
go build -o kb-cluster-resource-visualizer main.go

# Frontend
cd frontend
npm run build
```

## 🔧 Configuration

### Environment Variables

- `KUBECONFIG`: Kubernetes config file path (default: `~/.kube/config`)
- `PORT`: Backend service port (default: 8080)

### Kubernetes Permissions

Ensure your Kubernetes user/service account has the following permissions:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: kb-cluster-resource-visualizer
rules:
- apiGroups: ["*"]
  resources: ["*"]
  verbs: ["get", "list"]
```

### Layout Algorithm Configuration

The application supports multiple layout algorithms:
- **Hierarchical**: Simple layered layout, good for small trees
- **Reingold-Tilford**: Optimal tree layout with no edge crossings, perfect for complex trees
- **Customizable Spacing**: Adjustable node and layer spacing (currently 280px layer spacing, 140px node spacing)

### Logging

```bash
# Backend logs
cd backend && go run main.go

# Frontend logs
cd frontend && npm run dev

# Docker logs
docker-compose logs -f
```
