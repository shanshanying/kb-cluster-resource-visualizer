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
import { ResourceRelationship } from '../types';
import { FlowNode, FlowEdge } from '../types';

const nodeTypes = {
  resourceNode: ResourceNodeComponent,
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
    node.targetPosition = isHorizontal ? 'left' : 'top';
    node.sourcePosition = isHorizontal ? 'right' : 'bottom';

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

interface ResourceFlowProps {
  relationship?: ResourceRelationship;
  loading?: boolean;
}

const ResourceFlow: React.FC<ResourceFlowProps> = ({ relationship, loading }) => {
  const [nodes, setNodes, onNodesChange] = useNodesState([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState([]);
  const [layoutDirection, setLayoutDirection] = useState<'TB' | 'LR'>('TB');

  const onConnect = useCallback(
    (params: Connection) => setEdges((eds) => addEdge(params, eds)),
    [setEdges]
  );

  useEffect(() => {
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
      ...children.map((child, index) => ({
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
    const { nodes: layoutedNodes, edges: layoutedEdges } = getLayoutedElements(
      flowNodes,
      flowEdges,
      layoutDirection
    );

    setNodes(layoutedNodes);
    setEdges(layoutedEdges);
  }, [relationship, layoutDirection, setNodes, setEdges]);

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

  if (!relationship || relationship.children.length === 0) {
    return (
      <div style={{
        height: '100%',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        fontSize: 16,
        color: '#666'
      }}>
        {relationship
          ? 'No child resources found with ownerReference relationships.'
          : 'Select a resource to visualize its relationships.'
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
          <button
            onClick={() => onLayout('TB')}
            style={{
              marginRight: 8,
              padding: '4px 8px',
              border: '1px solid #d9d9d9',
              background: layoutDirection === 'TB' ? '#1890ff' : 'white',
              color: layoutDirection === 'TB' ? 'white' : 'black',
              borderRadius: '4px',
              cursor: 'pointer',
            }}
          >
            Vertical
          </button>
          <button
            onClick={() => onLayout('LR')}
            style={{
              padding: '4px 8px',
              border: '1px solid #d9d9d9',
              background: layoutDirection === 'LR' ? '#1890ff' : 'white',
              color: layoutDirection === 'LR' ? 'white' : 'black',
              borderRadius: '4px',
              cursor: 'pointer',
            }}
          >
            Horizontal
          </button>
        </div>
      </ReactFlow>
    </div>
  );
};

export default ResourceFlow;
