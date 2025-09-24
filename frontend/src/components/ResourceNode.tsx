import React from 'react';
import { Handle, Position } from 'reactflow';
import { Card, Tag, Tooltip } from 'antd';
import {
  DatabaseOutlined,
  CloudServerOutlined,
  AppstoreOutlined,
  SettingOutlined,
  KeyOutlined,
  GlobalOutlined,
  ScheduleOutlined,
  HddOutlined
} from '@ant-design/icons';
import { ResourceNode as ResourceNodeType } from '../types';

interface ResourceNodeProps {
  data: {
    resource: ResourceNodeType;
    isParent?: boolean;
  };
}

const getResourceIcon = (kind: string) => {
  switch (kind.toLowerCase()) {
    case 'pod':
      return <DatabaseOutlined />;
    case 'deployment':
      return <AppstoreOutlined />;
    case 'service':
      return <GlobalOutlined />;
    case 'replicaset':
      return <CloudServerOutlined />;
    case 'configmap':
      return <SettingOutlined />;
    case 'secret':
      return <KeyOutlined />;
    case 'job':
    case 'cronjob':
      return <ScheduleOutlined />;
    case 'persistentvolumeclaim':
      return <HddOutlined />;
    default:
      return <AppstoreOutlined />;
  }
};

const getStatusColor = (status?: string) => {
  switch (status?.toLowerCase()) {
    case 'running':
    case 'ready':
    case 'active':
      return 'green';
    case 'pending':
      return 'orange';
    case 'failed':
    case 'error':
      return 'red';
    default:
      return 'default';
  }
};

const ResourceNodeComponent: React.FC<ResourceNodeProps> = ({ data }) => {
  const { resource, isParent } = data;

  return (
    <div style={{ minWidth: 200 }}>
      <Handle type="target" position={Position.Top} />

      <Card
        size="small"
        style={{
          border: isParent ? '2px solid #1890ff' : '1px solid #d9d9d9',
          borderRadius: 8,
          backgroundColor: isParent ? '#f0f8ff' : 'white',
          boxShadow: '0 2px 8px rgba(0, 0, 0, 0.1)',
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', marginBottom: 8 }}>
          <span style={{ marginRight: 8, fontSize: 16 }}>
            {getResourceIcon(resource.kind)}
          </span>
          <div style={{ flex: 1 }}>
            <div style={{ fontWeight: 'bold', fontSize: 14 }}>
              {resource.name}
            </div>
            <div style={{ fontSize: 12, color: '#666' }}>
              {resource.kind}
            </div>
          </div>
        </div>

        <div style={{ marginBottom: 8 }}>
          {resource.namespace && (
            <Tag color="blue">
              {resource.namespace}
            </Tag>
          )}
          {resource.status && (
            <Tag color={getStatusColor(resource.status)}>
              {resource.status}
            </Tag>
          )}
        </div>

        <Tooltip
          title={
            <div>
              <div><strong>API Version:</strong> {resource.apiVersion}</div>
              <div><strong>UID:</strong> {resource.uid}</div>
              <div><strong>Created:</strong> {resource.creationTime}</div>
              {resource.labels && Object.keys(resource.labels).length > 0 && (
                <div>
                  <strong>Labels:</strong>
                  {Object.entries(resource.labels).map(([key, value]) => (
                    <div key={key} style={{ marginLeft: 8 }}>
                      {key}: {value}
                    </div>
                  ))}
                </div>
              )}
            </div>
          }
        >
          <div style={{ fontSize: 11, color: '#999', cursor: 'pointer' }}>
            Click for details
          </div>
        </Tooltip>
      </Card>

      <Handle type="source" position={Position.Bottom} />
    </div>
  );
};

export default ResourceNodeComponent;
