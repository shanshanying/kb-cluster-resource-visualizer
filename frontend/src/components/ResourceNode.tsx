import React from 'react';
import { Handle, Position } from 'reactflow';
import { Card, Tag, Tooltip, Badge } from 'antd';
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

// Get node styling based on level and type
const getNodeStyling = (level: number = 0, isRoot: boolean = false, isParent: boolean = false): React.CSSProperties => {
  const baseStyle: React.CSSProperties = {
    minWidth: 220,
    maxWidth: 280,
  };

  if (isRoot) {
    return {
      ...baseStyle,
      border: '3px solid #1890ff',
      borderRadius: 12,
      backgroundColor: '#f0f8ff',
      boxShadow: '0 4px 16px rgba(24, 144, 255, 0.3)',
      transform: 'scale(1.05)',
    };
  }

  if (isParent) {
    return {
      ...baseStyle,
      border: '2px solid #52c41a',
      borderRadius: 10,
      backgroundColor: '#f6ffed',
      boxShadow: '0 3px 12px rgba(82, 196, 26, 0.2)',
    };
  }

  // Level-based styling
  const levelColors = [
    { border: '#d9d9d9', bg: '#ffffff', shadow: 'rgba(0, 0, 0, 0.1)' },
    { border: '#b7eb8f', bg: '#f6ffed', shadow: 'rgba(82, 196, 26, 0.1)' },
    { border: '#91d5ff', bg: '#e6f7ff', shadow: 'rgba(24, 144, 255, 0.1)' },
    { border: '#ffd6cc', bg: '#fff2e8', shadow: 'rgba(250, 84, 28, 0.1)' },
    { border: '#efdbff', bg: '#f9f0ff', shadow: 'rgba(114, 46, 209, 0.1)' },
  ];

  const colorIndex = Math.min(level, levelColors.length - 1);
  const colors = levelColors[colorIndex];

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
  const nodeStyle = getNodeStyling(level, isRoot, isParent);

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
          <div style={{ marginRight: 10, fontSize: 18 }}>
            {getResourceIcon(resource.kind)}
          </div>
          <div style={{ flex: 1 }}>
            <div style={{
              fontWeight: isRoot ? 'bold' : 'normal',
              fontSize: isRoot ? 15 : 14,
              color: isRoot ? '#1890ff' : '#262626',
              marginBottom: 2,
            }}>
              {resource.name}
            </div>
            <div style={{ fontSize: 12, color: '#8c8c8c' }}>
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
          {resource.namespace && (
            <Tag
              color="blue"
              style={{ fontSize: '11px', marginBottom: 4 }}
            >
              📁 {resource.namespace}
            </Tag>
          )}
          {resource.status && resource.kind.toLowerCase() !== 'configmap' && (
            <Tag
              color={getStatusColor(resource.status)}
              style={{ fontSize: '11px', marginBottom: 4 }}
            >
              {resource.status}
            </Tag>
          )}
          {isRoot && (
            <Tag
              color="gold"
              style={{ fontSize: '11px', marginBottom: 4 }}
            >
              🌱 Root
            </Tag>
          )}
          {/* Display KubeBlocks role for Pods */}
          {resource.kind.toLowerCase() === 'pod' && resource.labels && resource.labels['kubeblocks.io/role'] && (
            <Tag
              color="purple"
              style={{ fontSize: '11px', marginBottom: 4 }}
            >
              🎭 {resource.labels['kubeblocks.io/role']}
            </Tag>
          )}
        </div>

        {/* Detailed tooltip */}
        <Tooltip
          title={
            <div style={{ maxWidth: 300 }}>
              <div style={{ marginBottom: 8, fontWeight: 'bold', fontSize: '13px' }}>
                {resource.kind}/{resource.name}
              </div>
              <div><strong>API Version:</strong> {resource.apiVersion}</div>
              <div><strong>UID:</strong> <code style={{ fontSize: '11px' }}>{resource.uid}</code></div>
              <div><strong>Created:</strong> {resource.creationTime}</div>
              {resource.namespace && (
                <div><strong>Namespace:</strong> {resource.namespace}</div>
              )}
              {resource.status && resource.kind.toLowerCase() !== 'configmap' && (
                <div><strong>Status:</strong>
                  <Tag color={getStatusColor(resource.status)} style={{ marginLeft: 4 }}>
                    {resource.status}
                  </Tag>
                </div>
              )}
              {resource.labels && Object.keys(resource.labels).length > 0 && (
                <div style={{ marginTop: 8 }}>
                  <strong>Labels:</strong>
                  <div style={{ marginLeft: 8, fontSize: '11px', fontFamily: 'monospace' }}>
                    {Object.entries(resource.labels).slice(0, 5).map(([key, value]) => (
                      <div key={key} style={{ marginBottom: 2 }}>
                        <span style={{ color: '#1890ff' }}>{key}:</span> {value}
                      </div>
                    ))}
                    {Object.keys(resource.labels).length > 5 && (
                      <div style={{ color: '#8c8c8c' }}>
                        ... and {Object.keys(resource.labels).length - 5} more
                      </div>
                    )}
                  </div>
                </div>
              )}
              <div style={{ marginTop: 8, fontSize: '11px', color: '#8c8c8c' }}>
                💡 Click to expand details
              </div>
            </div>
          }
          placement="right"
          overlayStyle={{ maxWidth: 350 }}
        >
          <div style={{
            fontSize: 11,
            color: '#bfbfbf',
            cursor: 'pointer',
            textAlign: 'center',
            padding: '2px 0',
            borderTop: '1px solid #f0f0f0',
            marginTop: 4,
          }}>
            ℹ️ Details
          </div>
        </Tooltip>
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
