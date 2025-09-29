import React, { useState, useEffect } from 'react';
import { Layout, Typography, Alert, Spin, message, Radio, Card } from 'antd';
import ResourceSelector from './components/ResourceSelector';
import ResourceFlow from './components/ResourceFlow';
import { TreeNode } from './types';
import apiService from './services/api';

export type LayoutAlgorithm = 'hierarchical' | 'reingold-tilford' ;

const { Header, Sider, Content } = Layout;
const { Title } = Typography;

const App: React.FC = () => {
  const [treeNodes, setTreeNodes] = useState<TreeNode[] | undefined>();
  const [loading, setLoading] = useState(false);
  const [healthStatus, setHealthStatus] = useState<'checking' | 'healthy' | 'error'>('checking');
  const [collapsed, setCollapsed] = useState(false);
  const [layoutAlgorithm, setLayoutAlgorithm] = useState<LayoutAlgorithm>('reingold-tilford');

  useEffect(() => {
    checkHealth();
  }, []);

  const checkHealth = async () => {
    try {
      await apiService.healthCheck();
      setHealthStatus('healthy');
    } catch (error) {
      console.error('Health check failed:', error);
      setHealthStatus('error');
      message.error('Cannot connect to backend API. Please ensure the backend server is running.');
    }
  };

  const handleResourceSelect = async (resourceType: string, resourceName: string, namespace?: string) => {
    setLoading(true);
    try {
      // Use tree structure API with selected resource as root node
      console.log('Building resource tree with root node:', { resourceType, rootResource: resourceName, namespace });
      const treeData = await apiService.getResourceTree(resourceType, resourceName, namespace);
      console.log('Received tree data:', treeData);
      setTreeNodes(treeData);

      if (treeData.length === 0) {
        message.info(`No resource tree found with ${resourceType}/${resourceName} as root node`);
      } else {
        const totalNodes = countTreeNodes(treeData);
        message.success(`Built resource tree with ${resourceType}/${resourceName} as root containing ${totalNodes} total nodes`);
      }
    } catch (error) {
      console.error('Failed to load resource data:', error);
      message.error('Failed to load resource data');
      setTreeNodes(undefined);
    } finally {
      setLoading(false);
    }
  };

  // Helper function to count total nodes in tree structure
  const countTreeNodes = (nodes: TreeNode[]): number => {
    return nodes.reduce((count, node) => {
      return count + 1 + countTreeNodes(node.children);
    }, 0);
  };

  return (
    <Layout style={{ height: '100vh' }}>
      <Header style={{
        background: '#fff',
        padding: '0 24px',
        borderBottom: '1px solid #f0f0f0',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between'
      }}>
        <Title level={3} style={{ margin: 0, color: '#1890ff' }}>
          Cluster Resource Visualizer
        </Title>

        <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
          {healthStatus === 'checking' && (
            <Spin size="small" />
          )}

          {healthStatus === 'error' && (
            <Alert
              message="Backend Disconnected"
              type="error"
              showIcon
              style={{ margin: 0 }}
              action={
                <button
                  onClick={checkHealth}
                  style={{
                    background: 'none',
                    border: 'none',
                    color: '#ff4d4f',
                    cursor: 'pointer',
                    textDecoration: 'underline'
                  }}
                >
                  Retry
                </button>
              }
            />
          )}
        </div>
      </Header>

      <Layout>
        <Sider
          collapsible
          collapsed={collapsed}
          onCollapse={setCollapsed}
          width={350}
          style={{
            background: '#fff',
            borderRight: '1px solid #f0f0f0',
            overflow: 'auto'
          }}
        >
          {!collapsed && (
            <div style={{ padding: '16px 0' }}>
              <ResourceSelector
                onResourceSelect={handleResourceSelect}
                loading={loading}
              />

              {/* Layout Algorithm Selector */}
              <Card
                title="🌳 Layout Algorithm"
                size="small"
                style={{ marginTop: 16 }}
                bodyStyle={{ padding: '12px' }}
              >
                <Radio.Group
                  value={layoutAlgorithm}
                  onChange={(e) => setLayoutAlgorithm(e.target.value)}
                  style={{ width: '100%' }}
                >
                  <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
                    <Radio value="hierarchical">
                      <span style={{ fontSize: '13px' }}>📊 Hierarchical</span>
                      <div style={{ fontSize: '11px', color: '#666', marginLeft: 20 }}>
                        Simple layered layout
                      </div>
                    </Radio>
                    <Radio value="reingold-tilford">
                      <span style={{ fontSize: '13px' }}>🌳 Tree (RT)</span>
                      <div style={{ fontSize: '11px', color: '#666', marginLeft: 20 }}>
                        Optimized tree layout
                      </div>
                    </Radio>
                  </div>
                </Radio.Group>
              </Card>
            </div>
          )}
        </Sider>

        <Content style={{
          background: '#f5f5f5',
          position: 'relative'
        }}>
          {healthStatus === 'error' ? (
            <div style={{
              height: '100%',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              flexDirection: 'column',
              gap: 16
            }}>
              <Alert
                message="Backend API Unavailable"
                description="Please start the backend server and refresh the page."
                type="error"
                showIcon
                style={{ maxWidth: 400 }}
              />
              <button
                onClick={() => window.location.reload()}
                style={{
                  padding: '8px 16px',
                  background: '#1890ff',
                  color: 'white',
                  border: 'none',
                  borderRadius: '4px',
                  cursor: 'pointer'
                }}
              >
                Refresh Page
              </button>
            </div>
          ) : (
            <ResourceFlow
              treeNodes={treeNodes}
              loading={loading}
              layoutAlgorithm={layoutAlgorithm}
            />
          )}
        </Content>
      </Layout>
    </Layout>
  );
};

export default App;
