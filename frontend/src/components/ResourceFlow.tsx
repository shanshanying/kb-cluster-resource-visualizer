import React, { useCallback, useEffect, useState } from 'react';
import ReactFlow, {
  Node,
  Edge,
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
import dagre from 'dagre';
import 'reactflow/dist/style.css';

import ResourceNodeComponent from './ResourceNode';
import { ResourceRelationship, TreeNode } from '../types';
import { FlowNode, FlowEdge } from '../types';

const nodeTypes = {
  resourceNode: ResourceNodeComponent,
};

// Helper function to count total nodes in tree structure
const countTreeNodes = (nodes: TreeNode[]): number => {
  return nodes.reduce((count, node) => {
    return count + 1 + countTreeNodes(node.children);
  }, 0);
};

const dagreGraph = new dagre.graphlib.Graph();
dagreGraph.setDefaultEdgeLabel(() => ({}));

const nodeWidth = 220;
const nodeHeight = 120;

const getLayoutedElements = (nodes: Node[], edges: Edge[], direction = 'TB') => {
  const isHorizontal = direction === 'LR';
  dagreGraph.setGraph({ rankdir: direction });

  nodes.forEach((node) => {
    dagreGraph.setNode(node.id, { width: nodeWidth, height: nodeHeight });
  });

  edges.forEach((edge) => {
    dagreGraph.setEdge(edge.source, edge.target);
  });

  dagre.layout(dagreGraph);

  nodes.forEach((node) => {
    const nodeWithPosition = dagreGraph.node(node.id);
    node.targetPosition = isHorizontal ? ('left' as any) : ('top' as any);
    node.sourcePosition = isHorizontal ? ('right' as any) : ('bottom' as any);

    // We are shifting the dagre node position (anchor=center center) to the top left
    // so it matches the React Flow node anchor point (top left).
    node.position = {
      x: nodeWithPosition.x - nodeWidth / 2,
      y: nodeWithPosition.y - nodeHeight / 2,
    };

    return node;
  });

  return { nodes, edges };
};

// Tree layout algorithm for hierarchical tree structure
const getTreeLayout = (nodes: Node[], edges: Edge[], direction = 'TB') => {
  const isHorizontal = direction === 'LR';

  // Build adjacency list to understand tree structure
  const adjacencyList: { [key: string]: string[] } = {};
  const inDegree: { [key: string]: number } = {};

  // Initialize
  nodes.forEach(node => {
    adjacencyList[node.id] = [];
    inDegree[node.id] = 0;
  });

  // Build adjacency list and calculate in-degrees
  edges.forEach(edge => {
    adjacencyList[edge.source].push(edge.target);
    inDegree[edge.target] = (inDegree[edge.target] || 0) + 1;
  });

  // Find root nodes (nodes with in-degree 0)
  const roots = nodes.filter(node => inDegree[node.id] === 0);

  // If no clear root, use the first node
  const rootNode = roots.length > 0 ? roots[0] : nodes[0];

  // Calculate positions using tree layout
  const positions: { [key: string]: { x: number; y: number; level: number } } = {};
  const levelWidth: { [level: number]: number } = {};

  // BFS to assign levels
  const queue = [{ id: rootNode.id, level: 0 }];
  const visited = new Set<string>();

  while (queue.length > 0) {
    const { id, level } = queue.shift()!;

    if (visited.has(id)) continue;
    visited.add(id);

    // Count nodes at this level
    levelWidth[level] = (levelWidth[level] || 0) + 1;

    // Add children to queue
    adjacencyList[id].forEach(childId => {
      if (!visited.has(childId)) {
        queue.push({ id: childId, level: level + 1 });
      }
    });
  }

  // Calculate positions
  const levelCounters: { [level: number]: number } = {};
  const spacing = { x: nodeWidth + 50, y: nodeHeight + 80 };

  const assignPosition = (nodeId: string, level: number) => {
    if (positions[nodeId]) return positions[nodeId];

    levelCounters[level] = (levelCounters[level] || 0) + 1;
    const nodeIndex = levelCounters[level] - 1;
    const totalNodesAtLevel = levelWidth[level];

    let x, y;

    if (isHorizontal) {
      x = level * spacing.x;
      y = (nodeIndex - (totalNodesAtLevel - 1) / 2) * spacing.y;
    } else {
      x = (nodeIndex - (totalNodesAtLevel - 1) / 2) * spacing.x;
      y = level * spacing.y;
    }

    positions[nodeId] = { x, y, level };
    return positions[nodeId];
  };

  // Assign positions using BFS again
  const positionQueue = [{ id: rootNode.id, level: 0 }];
  const positionVisited = new Set<string>();

  while (positionQueue.length > 0) {
    const { id, level } = positionQueue.shift()!;

    if (positionVisited.has(id)) continue;
    positionVisited.add(id);

    assignPosition(id, level);

    adjacencyList[id].forEach(childId => {
      if (!positionVisited.has(childId)) {
        positionQueue.push({ id: childId, level: level + 1 });
      }
    });
  }

  // Apply positions to nodes
  nodes.forEach(node => {
    const pos = positions[node.id] || { x: 0, y: 0, level: 0 };
    node.position = { x: pos.x, y: pos.y };
    node.targetPosition = isHorizontal ? ('left' as any) : ('top' as any);
    node.sourcePosition = isHorizontal ? ('right' as any) : ('bottom' as any);

    // Add level information to node data
    node.data = {
      ...node.data,
      level: pos.level,
      isRoot: pos.level === 0,
    };
  });

  return { nodes, edges };
};

// Convert tree structure to React Flow nodes and edges
const convertTreeToFlow = (treeNodes: TreeNode[]): { nodes: FlowNode[], edges: FlowEdge[] } => {
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
      },
    });

    // Process children and create edges
    treeNode.children.forEach(childNode => {
      // Create edge from parent to child
      flowEdges.push({
        id: `${treeNode.resource.metadata.uid}-${childNode.resource.metadata.uid}`,
        source: treeNode.resource.metadata.uid,
        target: childNode.resource.metadata.uid,
        type: 'smoothstep',
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
  relationship?: ResourceRelationship;
  treeNodes?: TreeNode[];
  loading?: boolean;
  useTreeLayout?: boolean;
}

const ResourceFlow: React.FC<ResourceFlowProps> = ({
  relationship,
  treeNodes,
  loading,
  useTreeLayout = false
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
    if (useTreeLayout && treeNodes && treeNodes.length > 0) {
      console.log('Processing tree nodes in ResourceFlow:', treeNodes);
      const { nodes: flowNodes, edges: flowEdges } = convertTreeToFlow(treeNodes);
      console.log('Converted to flow nodes:', flowNodes.length, 'edges:', flowEdges.length);

      // Apply tree layout
      const { nodes: layoutedNodes, edges: layoutedEdges } = getTreeLayout(
        flowNodes,
        flowEdges,
        layoutDirection
      );

      setNodes(layoutedNodes);
      setEdges(layoutedEdges);
      return;
    }

    // Handle legacy relationship data
    if (!relationship) {
      setNodes([]);
      setEdges([]);
      return;
    }

    const { parent, children } = relationship;

    // Create nodes
    const flowNodes: FlowNode[] = [
      {
        id: parent.uid,
        type: 'resourceNode',
        position: { x: 0, y: 0 },
        data: {
          resource: parent,
          isParent: true,
        },
      },
      ...children.map((child) => ({
        id: child.uid,
        type: 'resourceNode',
        position: { x: 0, y: 0 },
        data: {
          resource: child,
          isParent: false,
        },
      })),
    ];

    // Create edges
    const flowEdges: FlowEdge[] = children.map((child) => ({
      id: `${parent.uid}-${child.uid}`,
      source: parent.uid,
      target: child.uid,
      type: 'smoothstep',
    }));

    // Apply layout
    const { nodes: layoutedNodes, edges: layoutedEdges } = useTreeLayout
      ? getTreeLayout(flowNodes, flowEdges, layoutDirection)
      : getLayoutedElements(flowNodes, flowEdges, layoutDirection);

    setNodes(layoutedNodes);
    setEdges(layoutedEdges);
  }, [relationship, treeNodes, useTreeLayout, layoutDirection, setNodes, setEdges]);

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
  const hasData = useTreeLayout
    ? (treeNodes && treeNodes.length > 0)
    : (relationship && relationship.children.length > 0);

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
        {useTreeLayout
          ? (treeNodes ? 'No resource tree found.' : 'Select a resource to use as root node for the tree visualization.')
          : (relationship
              ? 'No child resources found with ownerReference relationships.'
              : 'Select a resource to visualize its relationships.')
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
            if (node.data?.isParent) return '#1890ff';
            return '#f0f0f0';
          }}
          nodeStrokeWidth={3}
        />
        <Background variant={BackgroundVariant.Dots} gap={12} size={1} />

        {/* Layout controls */}
        <div style={{
          position: 'absolute',
          top: 10,
          right: 10,
          zIndex: 4,
          background: 'white',
          padding: '8px',
          borderRadius: '4px',
          boxShadow: '0 2px 8px rgba(0, 0, 0, 0.1)',
        }}>
          <div style={{ marginBottom: '8px', fontSize: '12px', color: '#666' }}>
            🌳 Layout: {useTreeLayout ? 'Tree Structure' : 'Dagre'}
            {useTreeLayout && treeNodes && treeNodes.length > 0 && (
              <div style={{ marginTop: '4px', fontSize: '11px', color: '#999' }}>
                📊 {countTreeNodes(treeNodes)} nodes
              </div>
            )}
          </div>
          <button
            onClick={() => onLayout('TB')}
            style={{
              marginRight: 8,
              padding: '4px 10px',
              border: layoutDirection === 'TB' ? '2px solid #1890ff' : '1px solid #d9d9d9',
              background: layoutDirection === 'TB' ? '#e6f7ff' : 'white',
              color: layoutDirection === 'TB' ? '#1890ff' : '#666',
              borderRadius: '4px',
              cursor: 'pointer',
              fontSize: '11px',
              fontWeight: layoutDirection === 'TB' ? '500' : 'normal',
            }}
          >
            ⬇️ Vertical
          </button>
          <button
            onClick={() => onLayout('LR')}
            style={{
              padding: '4px 10px',
              border: layoutDirection === 'LR' ? '2px solid #1890ff' : '1px solid #d9d9d9',
              background: layoutDirection === 'LR' ? '#e6f7ff' : 'white',
              color: layoutDirection === 'LR' ? '#1890ff' : '#666',
              borderRadius: '4px',
              cursor: 'pointer',
              fontSize: '11px',
              fontWeight: layoutDirection === 'LR' ? '500' : 'normal',
            }}
          >
            ➡️ Horizontal
          </button>
        </div>
      </ReactFlow>
    </div>
  );
};

export default ResourceFlow;
