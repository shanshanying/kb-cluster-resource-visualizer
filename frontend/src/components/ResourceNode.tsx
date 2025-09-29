import React from 'react';
import { Handle, Position } from 'reactflow';
import { Card, Tag, Badge } from 'antd';
import {
  DatabaseOutlined,
  CloudServerOutlined,
  AppstoreOutlined,
  SettingOutlined,
  KeyOutlined,
  GlobalOutlined,
  ScheduleOutlined,
  HddOutlined,
  ClusterOutlined,
  NodeIndexOutlined,
  ApiOutlined,
  FolderOutlined,
  SafetyOutlined,
  BankOutlined,
  ToolOutlined,
  RocketOutlined,
  BranchesOutlined
} from '@ant-design/icons';
import { ResourceNode as ResourceNodeType } from '../types';

interface ResourceNodeProps {
  data: {
    resource: ResourceNodeType;
    isParent?: boolean;
    level?: number;
    isRoot?: boolean;
    layoutDirection?: 'TB' | 'LR';
  };
}

const getResourceIcon = (kind: string) => {
  switch (kind.toLowerCase()) {
    // Core Kubernetes resources
    case 'pod':
      return <DatabaseOutlined style={{ color: '#52c41a' }} />;
    case 'deployment':
      return <AppstoreOutlined style={{ color: '#1890ff' }} />;
    case 'service':
      return <GlobalOutlined style={{ color: '#722ed1' }} />;
    case 'replicaset':
      return <CloudServerOutlined style={{ color: '#13c2c2' }} />;
    case 'statefulset':
      return <NodeIndexOutlined style={{ color: '#eb2f96' }} />;
    case 'daemonset':
      return <BranchesOutlined style={{ color: '#fa8c16' }} />;
    case 'configmap':
      return <SettingOutlined style={{ color: '#faad14' }} />;
    case 'secret':
      return <KeyOutlined style={{ color: '#f5222d' }} />;
    case 'job':
      return <ScheduleOutlined style={{ color: '#2f54eb' }} />;
    case 'cronjob':
      return <ScheduleOutlined style={{ color: '#722ed1' }} />;
    case 'persistentvolumeclaim':
    case 'pvc':
      return <HddOutlined style={{ color: '#fa541c' }} />;

    // KubeBlocks resources
    case 'cluster':
      return <ClusterOutlined style={{ color: '#1890ff' }} />;
    case 'component':
      return <ApiOutlined style={{ color: '#52c41a' }} />;
    case 'instance':
      return <DatabaseOutlined style={{ color: '#13c2c2' }} />;
    case 'instanceset':
      return <NodeIndexOutlined style={{ color: '#eb2f96' }} />;
    case 'backup':
      return <BankOutlined style={{ color: '#fa8c16' }} />;
    case 'backuppolicy':
      return <SafetyOutlined style={{ color: '#faad14' }} />;
    case 'backupschedule':
      return <ScheduleOutlined style={{ color: '#fa541c' }} />;
    case 'restore':
      return <RocketOutlined style={{ color: '#52c41a' }} />;
    case 'opsrequest':
      return <ToolOutlined style={{ color: '#722ed1' }} />;

    // Default
    default:
      return <FolderOutlined style={{ color: '#8c8c8c' }} />;
  }
};

const getStatusColor = (status?: string) => {
  switch (status?.toLowerCase()) {
    case 'running':
    case 'ready':
    case 'active':
    case 'succeeded':
    case 'available':
      return 'green';
    case 'pending':
    case 'creating':
    case 'updating':
    case 'scaling':
      return 'orange';
    case 'failed':
    case 'error':
    case 'crashloopbackoff':
    case 'imagepullbackoff':
      return 'red';
    case 'terminating':
    case 'deleting':
      return 'volcano';
    case 'unknown':
    case 'unavailable':
      return 'default';
    default:
      return 'blue';
  }
};

// Get resource type color theme
const getResourceTypeColors = (resourceKind: string) => {
  const kind = resourceKind.toLowerCase();

  // Core Kubernetes workload resources - Blue theme
  if (['pod', 'deployment', 'replicaset', 'statefulset', 'daemonset', 'job', 'cronjob'].includes(kind)) {
    return {
      border: '#1890ff',
      bg: '#e6f7ff',
      shadow: 'rgba(24, 144, 255, 0.15)',
      category: 'workload'
    };
  }

  // Network resources - Purple theme
  if (['service', 'ingress', 'networkpolicy'].includes(kind)) {
    return {
      border: '#722ed1',
      bg: '#f9f0ff',
      shadow: 'rgba(114, 46, 209, 0.15)',
      category: 'network'
    };
  }

  // Configuration resources - Yellow theme
  if (['configmap', 'secret'].includes(kind)) {
    return {
      border: '#fadb14',
      bg: '#feffe6',
      shadow: 'rgba(250, 219, 20, 0.15)',
      category: 'config'
    };
  }

  // Storage resources - Green theme
  if (['persistentvolumeclaim', 'pvc', 'persistentvolume', 'pv', 'storageclass'].includes(kind)) {
    return {
      border: '#52c41a',
      bg: '#f6ffed',
      shadow: 'rgba(82, 196, 26, 0.15)',
      category: 'storage'
    };
  }

  // KubeBlocks cluster resources - Cyan theme
  if (['cluster', 'component', 'instance', 'instanceset'].includes(kind)) {
    return {
      border: '#13c2c2',
      bg: '#e6fffb',
      shadow: 'rgba(19, 194, 194, 0.15)',
      category: 'kubeblocks-cluster'
    };
  }

  // KubeBlocks backup resources - Brown theme
  if (['backup', 'backuppolicy', 'backupschedule', 'restore'].includes(kind)) {
    return {
      border: '#a0845c',
      bg: '#f6f2ed',
      shadow: 'rgba(160, 132, 92, 0.15)',
      category: 'kubeblocks-backup'
    };
  }

  // KubeBlocks operations - Magenta theme
  if (['opsrequest'].includes(kind)) {
    return {
      border: '#eb2f96',
      bg: '#fff0f6',
      shadow: 'rgba(235, 47, 150, 0.15)',
      category: 'kubeblocks-ops'
    };
  }

  // Default - Gray theme
  return {
    border: '#d9d9d9',
    bg: '#fafafa',
    shadow: 'rgba(0, 0, 0, 0.1)',
    category: 'default'
  };
};

// Get node styling based on resource type
const getNodeStyling = (_level: number = 0, isRoot: boolean = false, isParent: boolean = false, resourceKind?: string): React.CSSProperties => {
  const baseStyle: React.CSSProperties = {
    minWidth: 220,
    maxWidth: 280,
  };

  // Get colors based on resource type
  const colors = resourceKind ? getResourceTypeColors(resourceKind) : getResourceTypeColors('default');

  if (isRoot) {
    return {
      ...baseStyle,
      border: `3px solid ${colors.border}`,
      borderRadius: 12,
      backgroundColor: colors.bg,
      boxShadow: `0 4px 16px ${colors.shadow}`,
      transform: 'scale(1.05)',
    };
  }

  if (isParent) {
    return {
      ...baseStyle,
      border: `2px solid ${colors.border}`,
      borderRadius: 10,
      backgroundColor: colors.bg,
      boxShadow: `0 3px 12px ${colors.shadow}`,
    };
  }

  // Regular node styling with resource type colors
  return {
    ...baseStyle,
    border: `1px solid ${colors.border}`,
    borderRadius: 8,
    backgroundColor: colors.bg,
    boxShadow: `0 2px 8px ${colors.shadow}`,
  };
};

const ResourceNodeComponent: React.FC<ResourceNodeProps> = ({ data }) => {
  const { resource, isParent = false, level = 0, isRoot = false, layoutDirection = 'TB' } = data;
  const nodeStyle = getNodeStyling(level, isRoot, isParent, resource.kind);

  // Determine handle positions based on layout direction
  const isHorizontal = layoutDirection === 'LR';
  const targetPosition = isHorizontal ? Position.Left : Position.Top;
  const sourcePosition = isHorizontal ? Position.Right : Position.Bottom;


  return (
    <div style={nodeStyle}>
      <Handle
        type="target"
        position={targetPosition}
        style={{
          background: isRoot ? '#1890ff' : '#52c41a',
          border: '2px solid white',
          width: 12,
          height: 12,
        }}
      />

      <Card
        size="small"
        style={nodeStyle}
        bodyStyle={{ padding: '12px' }}
      >
        {/* Header with icon and basic info */}
        <div style={{ display: 'flex', alignItems: 'center', marginBottom: 10 }}>
          <div style={{ marginRight: 10, fontSize: 20 }}>
            {getResourceIcon(resource.kind)}
          </div>
          <div style={{ flex: 1 }}>
            <div style={{
              fontWeight: isRoot ? 'bold' : 'normal',
              fontSize: isRoot ? 17 : 16,
              color: isRoot ? '#1890ff' : '#262626',
              marginBottom: 2,
            }}>
              {resource.name}
            </div>
            <div style={{ fontSize: 13, color: '#8c8c8c' }}>
              {resource.kind}
              {level > 0 && (
                <span style={{ marginLeft: 8, color: '#bfbfbf' }}>
                  L{level}
                </span>
              )}
            </div>
          </div>

          {/* Status Badge */}
          {resource.status && (
            <Badge
              status={getStatusColor(resource.status) === 'green' ? 'success' :
                     getStatusColor(resource.status) === 'red' ? 'error' :
                     getStatusColor(resource.status) === 'orange' ? 'warning' : 'processing'}
              text=""
              style={{ marginLeft: 8 }}
            />
          )}
        </div>

        {/* Tags section */}
        <div style={{ marginBottom: 8, minHeight: 24 }}>
          {resource.namespace && isRoot && (
            <Tag
              color="blue"
              style={{ fontSize: '12px', marginBottom: 4 }}
            >
              📁 {resource.namespace}
            </Tag>
          )}
          {resource.status && resource.kind.toLowerCase() !== 'configmap' && (
            <Tag
              color={getStatusColor(resource.status)}
              style={{ fontSize: '12px', marginBottom: 4 }}
            >
              {resource.status}
            </Tag>
          )}
          {isRoot && (
            <Tag
              color="gold"
              style={{ fontSize: '12px', marginBottom: 4 }}
            >
              🌱 Root
            </Tag>
          )}
          {/* Display KubeBlocks role for Pods */}
          {resource.kind.toLowerCase() === 'pod' && resource.labels && resource.labels['kubeblocks.io/role'] && (
            <Tag
              color="purple"
              style={{ fontSize: '12px', marginBottom: 4 }}
            >
              🎭 {resource.labels['kubeblocks.io/role']}
            </Tag>
          )}
        </div>
      </Card>

      <Handle
        type="source"
        position={sourcePosition}
        style={{
          background: isRoot ? '#1890ff' : '#52c41a',
          border: '2px solid white',
          width: 12,
          height: 12,
        }}
      />
    </div>
  );
};

export default ResourceNodeComponent;
