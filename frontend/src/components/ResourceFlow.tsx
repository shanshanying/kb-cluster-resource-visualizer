import React, { useCallback, useEffect, useState } from 'react';
import ReactFlow, {
  addEdge,
  ConnectionMode,
  useNodesState,
  useEdgesState,
  Controls,
  Background,
  BackgroundVariant,
  Connection,
  MiniMap,
} from 'reactflow';
import 'reactflow/dist/style.css';

import ResourceNodeComponent from './ResourceNode';
import { TreeNode } from '../types';
import { FlowNode, FlowEdge } from '../types';
import { LayoutAlgorithm } from '../App';
import { createLayoutEngine, LayoutConfig } from '../utils/layoutAlgorithms';

const nodeTypes = {
  resourceNode: ResourceNodeComponent,
};

// Helper function to count total nodes in tree structure
const countTreeNodes = (nodes: TreeNode[]): number => {
  return nodes.reduce((count, node) => {
    return count + 1 + countTreeNodes(node.children);
  }, 0);
};


// Node dimensions moved to layoutAlgorithms.ts


// Removed old getTreeLayout function - now using layoutAlgorithms.ts

// Convert tree structure to React Flow nodes and edges
const convertTreeToFlow = (treeNodes: TreeNode[], layoutDirection: 'TB' | 'LR' = 'TB'): { nodes: FlowNode[], edges: FlowEdge[] } => {
  const flowNodes: FlowNode[] = [];
  const flowEdges: FlowEdge[] = [];

  const processNode = (treeNode: TreeNode, level: number = 0, isRoot: boolean = false) => {
    // Convert new format to legacy ResourceNode for compatibility
    const legacyResource = {
      name: treeNode.resource.metadata.name,
      kind: treeNode.resource.kind,
      apiVersion: treeNode.resource.apiVersion,
      namespace: treeNode.resource.metadata.namespace,
      uid: treeNode.resource.metadata.uid,
      labels: treeNode.resource.metadata.labels,
      annotations: treeNode.resource.metadata.annotations,
      creationTime: treeNode.resource.metadata.creationTimestamp,
      status: treeNode.resource.status ? 'Running' : 'Unknown', // Simplified status mapping
    };

    // Add the current node
    flowNodes.push({
      id: treeNode.resource.metadata.uid,
      type: 'resourceNode',
      position: { x: 0, y: 0 }, // Will be set by layout algorithm
      data: {
        resource: legacyResource,
        isParent: isRoot,
        level: level,
        isRoot: isRoot,
        layoutDirection: layoutDirection,
      },
    });

    // Process children and create edges
    treeNode.children.forEach((childNode, index) => {
      // Determine edge style based on resource types
      const getEdgeStyle = (parentKind: string, childKind: string) => {
        const parentType = parentKind.toLowerCase();
        const childType = childKind.toLowerCase();

        // All edges now use consistent light gray colors
        // Strong ownership relationships - slightly darker gray
        if ((parentType === 'cluster' && childType === 'component') ||
            (parentType === 'component' && childType === 'instance') ||
            (parentType === 'deployment' && childType === 'replicaset') ||
            (parentType === 'replicaset' && childType === 'pod')) {
          return {
            type: 'smoothstep',
            style: { stroke: '#999', strokeWidth: 3 }, // Changed from blue to medium gray
            animated: false,
          };
        }

        // Service relationships - dashed light gray
        if (parentType === 'service' || childType === 'service') {
          return {
            type: 'smoothstep',
            style: { stroke: '#aaa', strokeWidth: 2, strokeDasharray: '5,5' }, // Changed from purple to light gray
            animated: false,
          };
        }

        // Config relationships - dotted light gray
        if (parentType === 'configmap' || parentType === 'secret' ||
            childType === 'configmap' || childType === 'secret') {
          return {
            type: 'smoothstep',
            style: { stroke: '#ccc', strokeWidth: 2, strokeDasharray: '3,3' }, // Changed from yellow to very light gray
            animated: false,
          };
        }

        // Default relationship - standard light gray
        return {
          type: 'smoothstep',
          style: { stroke: '#bbb', strokeWidth: 2 }, // Changed from green to light gray
          animated: false,
        };
      };

      const edgeStyle = getEdgeStyle(treeNode.resource.kind, childNode.resource.kind);

      // Create edge from parent to child
      flowEdges.push({
        id: `${treeNode.resource.metadata.uid}-${childNode.resource.metadata.uid}`,
        source: treeNode.resource.metadata.uid,
        target: childNode.resource.metadata.uid,
        ...edgeStyle,
        label: index === 0 ? `${treeNode.children.length}` : undefined,
        labelStyle: { fontSize: 12, fontWeight: 'bold' },
        labelBgStyle: { fill: 'white', fillOpacity: 0.8 },
      });

      // Recursively process child
      processNode(childNode, level + 1, false);
    });
  };

  // Process all root nodes (should typically be just one)
  treeNodes.forEach((rootNode) => {
    processNode(rootNode, 0, true);
  });

  return { nodes: flowNodes, edges: flowEdges };
};

interface ResourceFlowProps {
  treeNodes?: TreeNode[];
  loading?: boolean;
  layoutAlgorithm?: LayoutAlgorithm;
}

const ResourceFlow: React.FC<ResourceFlowProps> = ({
  treeNodes,
  loading,
  layoutAlgorithm = 'hierarchical',
}) => {
  const [nodes, setNodes, onNodesChange] = useNodesState([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState([]);
  const [layoutDirection, setLayoutDirection] = useState<'TB' | 'LR'>('TB');

  const onConnect = useCallback(
    (params: Connection) => setEdges((eds) => addEdge(params, eds)),
    [setEdges]
  );

  useEffect(() => {
    // Handle tree structure data
    if (treeNodes && treeNodes.length > 0) {
      console.log('Processing tree nodes in ResourceFlow:', treeNodes);
      const { nodes: flowNodes, edges: flowEdges } = convertTreeToFlow(treeNodes, layoutDirection);
      console.log('Converted to flow nodes:', flowNodes.length, 'edges:', flowEdges.length);

      // Apply layout using selected algorithm
      const config: LayoutConfig = {
        nodeWidth: 280,
        nodeHeight: 140,
        horizontalSpacing: 80, // Reduced from 140 to 80 for closer node spacing
        verticalSpacing: 180, // Reduced from 280 to 180 for closer layer spacing
        direction: layoutDirection
      };

      const layoutEngine = createLayoutEngine(layoutAlgorithm, config);
      const { nodes: layoutedNodes, edges: layoutedEdges } = layoutEngine.layout(flowNodes, flowEdges);

      setNodes(layoutedNodes);
      setEdges(layoutedEdges);
      return;
    }

    // Clear nodes and edges if no tree data
    setNodes([]);
    setEdges([]);
  }, [treeNodes, layoutDirection, setNodes, setEdges]);

  const onLayout = useCallback(
    (direction: 'TB' | 'LR') => {
      setLayoutDirection(direction);
    },
    []
  );

  if (loading) {
    return (
      <div style={{
        height: '100%',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        fontSize: 16,
        color: '#666'
      }}>
        Loading resource relationships...
      </div>
    );
  }

  // Check if we have data to display
  const hasData = treeNodes && treeNodes.length > 0;

  if (!hasData) {
    return (
      <div style={{
        height: '100%',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        fontSize: 16,
        color: '#666'
      }}>
        {treeNodes
          ? 'No resource tree found.'
          : 'Select a resource to use as root node for the tree visualization.'
        }
      </div>
    );
  }

  return (
    <div style={{ height: '100%', width: '100%' }}>
      <ReactFlow
        nodes={nodes}
        edges={edges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        onConnect={onConnect}
        nodeTypes={nodeTypes}
        connectionMode={ConnectionMode.Loose}
        fitView
        attributionPosition="bottom-left"
      >
        <Controls />
        <MiniMap
          nodeColor={(node) => {
            if (node.data?.isRoot) return '#1890ff';
            if (node.data?.isParent) return '#52c41a';
            const level = node.data?.level || 0;
            const levelColors = ['#f0f0f0', '#f6ffed', '#e6f7ff', '#fff2e8', '#f9f0ff'];
            return levelColors[Math.min(level, levelColors.length - 1)];
          }}
          nodeStrokeColor={(node) => {
            if (node.data?.isRoot) return '#1890ff';
            if (node.data?.isParent) return '#52c41a';
            return '#d9d9d9';
          }}
          nodeStrokeWidth={2}
          maskColor="rgba(240, 248, 255, 0.8)"
          style={{
            backgroundColor: 'rgba(255, 255, 255, 0.9)',
            border: '1px solid #d9d9d9',
          }}
        />
        <Background variant={BackgroundVariant.Dots} gap={12} size={1} />

        {/* Enhanced Layout Controls */}
        <div style={{
          position: 'absolute',
          top: 15,
          right: 15,
          zIndex: 4,
          background: 'rgba(255, 255, 255, 0.95)',
          padding: '12px',
          borderRadius: '8px',
          boxShadow: '0 4px 16px rgba(0, 0, 0, 0.15)',
          backdropFilter: 'blur(10px)',
          border: '1px solid rgba(0, 0, 0, 0.1)',
          minWidth: 200,
        }}>
          {/* Header */}
          <div style={{ marginBottom: '12px', fontSize: '13px', fontWeight: '600', color: '#262626' }}>
            🌳 Tree Visualization
          </div>

          {/* Statistics */}
          {treeNodes && treeNodes.length > 0 && (
            <div style={{
              marginBottom: '12px',
              padding: '8px',
              background: '#f0f8ff',
              borderRadius: '6px',
              border: '1px solid #e6f7ff'
            }}>
              <div style={{ fontSize: '11px', color: '#1890ff', fontWeight: '500' }}>
                📊 {countTreeNodes(treeNodes)} total nodes
              </div>
              <div style={{ fontSize: '11px', color: '#666', marginTop: '2px' }}>
                🌱 {treeNodes.length} root node{treeNodes.length > 1 ? 's' : ''}
              </div>
              <div style={{ fontSize: '11px', color: '#666', marginTop: '2px' }}>
                🔗 {edges.length} edges
              </div>
            </div>
          )}

          {/* Layout Algorithm */}
          <div style={{ marginBottom: '8px', fontSize: '11px', color: '#8c8c8c', fontWeight: '500' }}>
            LAYOUT ALGORITHM
          </div>
          <div style={{ marginBottom: '8px', fontSize: '12px', color: '#666' }}>
            {layoutAlgorithm === 'hierarchical' ? '📊 Hierarchical' :
             layoutAlgorithm === 'reingold-tilford' ? '🌳 Tree (RT)' :
             layoutAlgorithm === 'enhanced-tree' ? '🌲 Enhanced Tree' :
             layoutAlgorithm === 'force-directed' ? '⚡ Force' : '🎯 Radial'}
          </div>
          <div style={{ marginBottom: '12px', fontSize: '10px', color: '#999' }}>
            📏 Layer spacing: 180px | Node spacing: 80px
          </div>

          {/* Direction Controls */}
          <div style={{ marginBottom: '8px', fontSize: '11px', color: '#8c8c8c', fontWeight: '500' }}>
            DIRECTION
          </div>
          <div style={{ display: 'flex', gap: '6px' }}>
            <button
              onClick={() => onLayout('TB')}
              style={{
                flex: 1,
                padding: '6px 8px',
                border: layoutDirection === 'TB' ? '2px solid #1890ff' : '1px solid #d9d9d9',
                background: layoutDirection === 'TB' ? '#e6f7ff' : 'white',
                color: layoutDirection === 'TB' ? '#1890ff' : '#666',
                borderRadius: '6px',
                cursor: 'pointer',
                fontSize: '11px',
                fontWeight: layoutDirection === 'TB' ? '600' : '400',
                transition: 'all 0.2s ease',
              }}
            >
              ⬇️ Vertical
            </button>
            <button
              onClick={() => onLayout('LR')}
              style={{
                flex: 1,
                padding: '6px 8px',
                border: layoutDirection === 'LR' ? '2px solid #1890ff' : '1px solid #d9d9d9',
                background: layoutDirection === 'LR' ? '#e6f7ff' : 'white',
                color: layoutDirection === 'LR' ? '#1890ff' : '#666',
                borderRadius: '6px',
                cursor: 'pointer',
                fontSize: '11px',
                fontWeight: layoutDirection === 'LR' ? '600' : '400',
                transition: 'all 0.2s ease',
              }}
            >
              ➡️ Horizontal
            </button>
          </div>

          {/* Resource Type Colors Legend */}
          <div style={{ marginTop: '12px', paddingTop: '12px', borderTop: '1px solid #f0f0f0' }}>
            <div style={{ marginBottom: '6px', fontSize: '11px', color: '#8c8c8c', fontWeight: '500' }}>
              RESOURCE TYPES
            </div>
            <div style={{ display: 'flex', flexDirection: 'column', gap: '3px' }}>
              <div style={{ display: 'flex', alignItems: 'center', fontSize: '10px', color: '#666' }}>
                <div style={{ width: '10px', height: '10px', background: '#e6f7ff', border: '1px solid #1890ff', marginRight: '6px', borderRadius: '2px' }}></div>
                Workload
              </div>
              <div style={{ display: 'flex', alignItems: 'center', fontSize: '10px', color: '#666' }}>
                <div style={{ width: '10px', height: '10px', background: '#f9f0ff', border: '1px solid #722ed1', marginRight: '6px', borderRadius: '2px' }}></div>
                Network
              </div>
              <div style={{ display: 'flex', alignItems: 'center', fontSize: '10px', color: '#666' }}>
                <div style={{ width: '10px', height: '10px', background: '#feffe6', border: '1px solid #fadb14', marginRight: '6px', borderRadius: '2px' }}></div>
                Config
              </div>
              <div style={{ display: 'flex', alignItems: 'center', fontSize: '10px', color: '#666' }}>
                <div style={{ width: '10px', height: '10px', background: '#f6ffed', border: '1px solid #52c41a', marginRight: '6px', borderRadius: '2px' }}></div>
                Storage
              </div>
              <div style={{ display: 'flex', alignItems: 'center', fontSize: '10px', color: '#666' }}>
                <div style={{ width: '10px', height: '10px', background: '#e6fffb', border: '1px solid #13c2c2', marginRight: '6px', borderRadius: '2px' }}></div>
                KubeBlocks
              </div>
              <div style={{ display: 'flex', alignItems: 'center', fontSize: '10px', color: '#666' }}>
                <div style={{ width: '10px', height: '10px', background: '#f6f2ed', border: '1px solid #a0845c', marginRight: '6px', borderRadius: '2px' }}></div>
                Backup
              </div>
            </div>
          </div>
        </div>
      </ReactFlow>
    </div>
  );
};

export default ResourceFlow;
