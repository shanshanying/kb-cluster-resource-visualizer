# K8s Resource Visualizer

一个受 ArgoCD UI 启发的 Kubernetes 资源可视化工具，使用 React TypeScript 前端和 Go 后端构建。

## ✨ 功能特性

- 🔍 输入资源类型，可视化所有具有 ownerReference 关系的资源
- 🎨 现代化 Web 界面，支持交互式资源关系图
- ⚡ 实时连接 K8s 集群
- 📊 支持所有 Kubernetes 资源类型
- 🔄 自动布局算法（垂直/水平）
- 🏷️ 丰富的资源信息展示（状态、标签、命名空间等）

## 🏗️ 技术架构

- **前端**: React + TypeScript + Vite + Ant Design + React Flow
- **后端**: Go + Gin + client-go
- **可视化**: React Flow 交互式图表
- **容器化**: Docker + Docker Compose

## 🚀 快速开始

### 📋 先决条件

- Go 1.21+
- Node.js 18+
- 可访问的 Kubernetes 集群
- kubectl 已配置

### 🎯 一键启动

```bash
# 克隆项目
git clone <your-repo-url>
cd k8s-resource-visualizer

# 使用启动脚本
./scripts/start.sh
```

### 🔧 手动启动

#### 后端设置

```bash
cd backend
go mod tidy
go run main.go
```

#### 前端设置

```bash
cd frontend
npm install
npm run dev
```

### 🐳 Docker 启动

```bash
# 确保 ~/.kube/config 存在并可访问
docker-compose up -d
```

## 📖 使用说明

1. **启动服务**: 运行前后端服务
2. **打开浏览器**: 访问 http://localhost:5173
3. **选择资源类型**: 从下拉菜单选择资源类型（如 Deployment、Pod 等）
4. **选择命名空间**: 可选择特定命名空间或查看所有命名空间
5. **选择资源**: 点击左侧列表中的资源
6. **查看关系图**: 右侧将显示该资源的 ownerReference 关系图

## 🔌 API 接口

- `GET /api/health` - 健康检查
- `GET /api/namespaces` - 获取所有命名空间
- `GET /api/resources/:type` - 获取指定类型的所有资源
- `GET /api/resources/:type/:name/children` - 获取指定资源拥有的所有子资源

### 请求示例

```bash
# 获取所有 Deployment
curl "http://localhost:8080/api/resources/deployment"

# 获取特定命名空间的 Pod
curl "http://localhost:8080/api/resources/pod?namespace=default"

# 获取 Deployment 的子资源
curl "http://localhost:8080/api/resources/deployment/my-app/children?namespace=default"
```

## 🎨 界面预览

### 主界面
- 左侧：资源选择器，支持资源类型和命名空间筛选
- 右侧：交互式资源关系图，支持拖拽和缩放

### 资源节点
- 显示资源名称、类型、状态
- 支持命名空间标签和状态标签
- 鼠标悬停显示详细信息

### 布局控制
- 垂直布局：适合层级关系展示
- 水平布局：适合并行关系展示

## 📁 项目结构

```
k8s-resource-visualizer/
├── backend/                 # Go 后端
│   ├── main.go             # 主程序
│   ├── go.mod              # Go 模块文件
│   └── Dockerfile          # 后端 Docker 配置
├── frontend/               # React 前端
│   ├── src/
│   │   ├── components/     # React 组件
│   │   ├── services/       # API 服务
│   │   ├── types/          # TypeScript 类型定义
│   │   └── App.tsx         # 主应用组件
│   ├── package.json        # NPM 配置
│   └── Dockerfile          # 前端 Docker 配置
├── scripts/
│   └── start.sh            # 启动脚本
├── docker-compose.yml      # Docker Compose 配置
└── README.md               # 项目文档
```

## 🔍 支持的资源类型

- **工作负载**: Deployment, ReplicaSet, StatefulSet, DaemonSet, Pod, Job, CronJob
- **服务**: Service, Ingress
- **配置**: ConfigMap, Secret
- **存储**: PersistentVolumeClaim
- **其他**: 所有标准 Kubernetes 资源

## 🛠️ 开发说明

### 后端开发

```bash
cd backend
go mod tidy
go run main.go
```

### 前端开发

```bash
cd frontend
npm install
npm run dev
```

### 构建生产版本

```bash
# 后端
cd backend
go build -o k8s-visualizer main.go

# 前端
cd frontend
npm run build
```

## 🔧 配置说明

### 环境变量

- `KUBECONFIG`: Kubernetes 配置文件路径（默认: `~/.kube/config`）
- `PORT`: 后端服务端口（默认: 8080）

### Kubernetes 权限

确保您的 Kubernetes 用户/服务账户具有以下权限：

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: k8s-resource-visualizer
rules:
- apiGroups: ["*"]
  resources: ["*"]
  verbs: ["get", "list"]
```

## 🚨 故障排除

### 常见问题

1. **无法连接到 Kubernetes 集群**
   - 检查 `kubectl cluster-info` 是否正常
   - 确认 kubeconfig 文件路径正确

2. **后端 API 连接失败**
   - 确认后端服务在 8080 端口运行
   - 检查防火墙设置

3. **前端显示空白页面**
   - 检查浏览器控制台错误信息
   - 确认后端 API 可访问

### 日志查看

```bash
# 后端日志
cd backend && go run main.go

# 前端日志
cd frontend && npm run dev

# Docker 日志
docker-compose logs -f
```

## 🤝 贡献指南

1. Fork 本项目
2. 创建功能分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 创建 Pull Request

## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 🙏 致谢

- 感谢 [ArgoCD](https://argoproj.github.io/argo-cd/) 项目的设计灵感
- 感谢 [React Flow](https://reactflow.dev/) 提供的优秀可视化组件
- 感谢 [client-go](https://github.com/kubernetes/client-go) 提供的 Kubernetes 客户端库
