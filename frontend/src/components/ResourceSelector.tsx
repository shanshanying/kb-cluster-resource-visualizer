import React, { useState, useEffect } from 'react';
import { Select, Button, Card, List, Typography, Space, Tag, message } from 'antd';
import { SearchOutlined, ReloadOutlined } from '@ant-design/icons';
import { ResourceNode } from '../types';
import apiService from '../services/api';

const { Option } = Select;
const { Title } = Typography;

interface ResourceSelectorProps {
  onResourceSelect: (resourceType: string, resourceName: string, namespace?: string) => void;
  loading?: boolean;
}

const COMMON_RESOURCE_TYPES = [
  // Standard Kubernetes resources
  'deployment',
  'replicaset',
  'statefulset',
  'daemonset',
  'pod',
  'service',
  'job',
  'cronjob',
  'configmap',
  'secret',
  'ingress',
  'persistentvolumeclaim',

  // KubeBlocks custom resources
  'cluster',
  'component',
  'backuppolicy',
  'backup',
  'backupschedule',
  'restore',
  'opsrequest',
  'instance',
  'instanceset'
];

const ResourceSelector: React.FC<ResourceSelectorProps> = ({
  onResourceSelect
}) => {
  const [resourceType, setResourceType] = useState<string>('deployment');
  const [namespace, setNamespace] = useState<string>('');
  const [namespaces, setNamespaces] = useState<string[]>([]);
  const [resources, setResources] = useState<ResourceNode[]>([]);
  const [resourcesLoading, setResourcesLoading] = useState(false);
  const [selectedResource, setSelectedResource] = useState<ResourceNode | null>(null);

  // Load namespaces on component mount
  useEffect(() => {
    loadNamespaces();
  }, []);

  // Load resources when resource type or namespace changes
  useEffect(() => {
    if (resourceType) {
      loadResources();
    }
  }, [resourceType, namespace]);

  const loadNamespaces = async () => {
    try {
      const namespaceList = await apiService.getNamespaces();
      setNamespaces(namespaceList);
    } catch (error) {
      console.error('Failed to load namespaces:', error);
      message.error('Failed to load namespaces');
    }
  };

  const loadResources = async () => {
    if (!resourceType) return;

    setResourcesLoading(true);
    try {
      const resourceList = await apiService.getResourcesByType(resourceType, namespace || undefined);
      setResources(resourceList);
      setSelectedResource(null);
    } catch (error) {
      console.error('Failed to load resources:', error);
      message.error(`Failed to load ${resourceType} resources`);
      setResources([]);
    } finally {
      setResourcesLoading(false);
    }
  };

  const handleResourceSelect = (resource: ResourceNode) => {
    setSelectedResource(resource);
    onResourceSelect(resource.kind, resource.name, resource.namespace);
  };

  const handleRefresh = () => {
    loadResources();
  };

  return (
    <div style={{ padding: '16px', height: '100%', overflow: 'auto' }}>
      <Card>
        <Title level={4}>Resource Selector</Title>

        <Space direction="vertical" style={{ width: '100%' }} size="middle">
          {/* Resource Type Selection */}
          <div>
            <label style={{ display: 'block', marginBottom: 8, fontWeight: 'bold' }}>
              Resource Type:
            </label>
            <Select
              style={{ width: '100%' }}
              value={resourceType}
              onChange={setResourceType}
              placeholder="Select resource type"
              showSearch
              filterOption={(input, option) =>
                (option?.children as unknown as string)?.toLowerCase().includes(input.toLowerCase())
              }
            >
              {COMMON_RESOURCE_TYPES.map(type => (
                <Option key={type} value={type}>
                  {type.charAt(0).toUpperCase() + type.slice(1)}
                </Option>
              ))}
            </Select>
          </div>

          {/* Namespace Selection */}
          <div>
            <label style={{ display: 'block', marginBottom: 8, fontWeight: 'bold' }}>
              Namespace (optional):
            </label>
            <Select
              style={{ width: '100%' }}
              value={namespace}
              onChange={setNamespace}
              placeholder="All namespaces"
              allowClear
              showSearch
              filterOption={(input, option) =>
                (option?.children as unknown as string)?.toLowerCase().includes(input.toLowerCase())
              }
            >
              {namespaces.map(ns => (
                <Option key={ns} value={ns}>
                  {ns}
                </Option>
              ))}
            </Select>
          </div>

          {/* Refresh Button */}
          <Button
            type="default"
            icon={<ReloadOutlined />}
            onClick={handleRefresh}
            loading={resourcesLoading}
            style={{ width: '100%' }}
          >
            Refresh Resources
          </Button>
        </Space>
      </Card>

      {/* Resources List */}
      <Card style={{ marginTop: 16 }} title="Resources" extra={
        <Tag color="blue">{resources.length} found</Tag>
      }>
        <List
          loading={resourcesLoading}
          dataSource={resources}
          renderItem={(resource) => (
            <List.Item
              style={{
                cursor: 'pointer',
                backgroundColor: selectedResource?.uid === resource.uid ? '#f0f8ff' : 'transparent',
                padding: '12px',
                borderRadius: '4px',
                border: selectedResource?.uid === resource.uid ? '1px solid #1890ff' : '1px solid transparent',
              }}
              onClick={() => handleResourceSelect(resource)}
            >
              <List.Item.Meta
                title={
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                    <span>{resource.name}</span>
                    {resource.namespace && (
                      <Tag color="blue">{resource.namespace}</Tag>
                    )}
                    {resource.status && (
                      <Tag
                        color={
                          resource.status.toLowerCase() === 'running' || resource.status.toLowerCase() === 'ready'
                            ? 'green'
                            : resource.status.toLowerCase() === 'pending'
                            ? 'orange'
                            : 'default'
                        }
                      >
                        {resource.status}
                      </Tag>
                    )}
                  </div>
                }
                description={
                  <div>
                    <div style={{ fontSize: '12px', color: '#666' }}>
                      {resource.kind} • {resource.creationTime}
                    </div>
                  </div>
                }
              />
            </List.Item>
          )}
          locale={{
            emptyText: resourceType
              ? `No ${resourceType} resources found${namespace ? ` in namespace ${namespace}` : ''}`
              : 'Select a resource type to view resources'
          }}
        />
      </Card>

      {/* Instructions */}
      <Card style={{ marginTop: 16 }} size="small">
        <Typography.Text type="secondary" style={{ fontSize: '12px' }}>
          <SearchOutlined /> Select a resource type and optionally a namespace to view available resources.
          Click on a resource to visualize its ownerReference relationships.
        </Typography.Text>
      </Card>
    </div>
  );
};

export default ResourceSelector;
