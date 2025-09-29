/**
 * Tree Layout Algorithms for K8s Resource Visualization
 *
 * This module implements various tree layout algorithms optimized for
 * Kubernetes resource tree visualization.
 */

import { Node, Edge } from 'reactflow';
import { FlowNode, FlowEdge } from '../types';

export interface LayoutConfig {
  nodeWidth: number;
  nodeHeight: number;
  horizontalSpacing: number;
  verticalSpacing: number;
  direction: 'TB' | 'LR';
}

export interface LayoutResult {
  nodes: FlowNode[];
  edges: FlowEdge[];
}

// Internal node structure for RT algorithm
interface RTNode {
  id: string;
  children: RTNode[];
  parent: RTNode | null;

  // RT algorithm specific properties
  x: number;
  y: number;
  prelim: number;        // preliminary x coordinate
  mod: number;           // modifier for subtree positioning
  thread: RTNode | null; // thread to next node in contour
  ancestor: RTNode;      // ancestor for lowest common ancestor queries
  change: number;        // change in position
  shift: number;         // shift in position
  number: number;        // number in preorder traversal

  // Original node data
  data: any;
  level: number;
}

/**
 * Reingold-Tilford Tree Layout Algorithm
 *
 * This is the gold standard for tree layout, ensuring:
 * - No edge crossings
 * - Optimal space utilization
 * - Aesthetically pleasing symmetric layout
 * - O(n) time complexity
 */
export class ReingoldTilfordLayout {
  private config: LayoutConfig;
  private nodeMap: Map<string, RTNode> = new Map();

  constructor(config: LayoutConfig) {
    this.config = config;
  }

  public layout(nodes: Node[], edges: Edge[]): LayoutResult {
    if (nodes.length === 0) {
      return { nodes: [], edges: [] };
    }

    // Build tree structure
    const rootNode = this.buildTree(nodes, edges);

    // Apply Reingold-Tilford algorithm
    this.initializeNodes(rootNode);
    this.firstWalk(rootNode);
    this.secondWalk(rootNode, 0);

    // Convert back to ReactFlow format
    return this.convertToReactFlow(nodes, edges);
  }

  private buildTree(nodes: Node[], edges: Edge[]): RTNode {
    // Clear previous state
    this.nodeMap.clear();

    // Create RT nodes
    nodes.forEach(node => {
      const rtNode: RTNode = {
        id: node.id,
        children: [],
        parent: null,
        x: 0,
        y: 0,
        prelim: 0,
        mod: 0,
        thread: null,
        ancestor: null as any, // Will be set to self in initializeNodes
        change: 0,
        shift: 0,
        number: 0,
        data: node.data,
        level: 0
      };
      this.nodeMap.set(node.id, rtNode);
    });

    // Build parent-child relationships
    edges.forEach(edge => {
      const parent = this.nodeMap.get(edge.source);
      const child = this.nodeMap.get(edge.target);

      if (parent && child) {
        parent.children.push(child);
        child.parent = parent;
      }
    });

    // Find root node (node with no parent)
    let rootNode = Array.from(this.nodeMap.values()).find(node => node.parent === null);

    if (!rootNode) {
      // If no clear root, use the first node
      rootNode = this.nodeMap.get(nodes[0].id)!;
    }

    // Assign levels
    this.assignLevels(rootNode, 0);

    return rootNode;
  }

  private assignLevels(node: RTNode, level: number): void {
    node.level = level;
    node.children.forEach(child => {
      this.assignLevels(child, level + 1);
    });
  }

  private initializeNodes(root: RTNode): void {
    let number = 0;

    const traverse = (node: RTNode) => {
      node.number = number++;
      node.ancestor = node;
      node.children.forEach(traverse);
    };

    traverse(root);
  }

  /**
   * First walk: Calculate preliminary positions and modifiers
   * This is a post-order traversal that calculates the preliminary x-coordinate
   * for each node and the modifier for each subtree.
   */
  private firstWalk(node: RTNode): void {
    if (node.children.length === 0) {
      // Leaf node
      node.prelim = 0;
      if (this.leftSibling(node)) {
        node.prelim = this.leftSibling(node)!.prelim + this.getNodeSeparation();
      }
    } else {
      // Internal node
      node.children.forEach(child => this.firstWalk(child));

      const leftmost = node.children[0];
      const rightmost = node.children[node.children.length - 1];

      // Center over children
      node.prelim = (leftmost.prelim + rightmost.prelim) / 2;

      if (this.leftSibling(node)) {
        node.prelim = this.leftSibling(node)!.prelim + this.getNodeSeparation();
        node.mod = node.prelim - (leftmost.prelim + rightmost.prelim) / 2;
      }

      if (node.children.length > 0 && this.leftSibling(node)) {
        this.apportion(node);
      }
    }
  }

  /**
   * Second walk: Calculate final positions
   * This is a pre-order traversal that calculates the final x-coordinate
   * for each node by adding the accumulated modifier.
   */
  private secondWalk(node: RTNode, modSum: number): void {
    const isHorizontal = this.config.direction === 'LR';

    if (isHorizontal) {
      node.y = node.prelim + modSum;
      node.x = node.level * (this.config.nodeHeight + this.config.verticalSpacing);
    } else {
      node.x = node.prelim + modSum;
      node.y = node.level * (this.config.nodeHeight + this.config.verticalSpacing);
    }

    node.children.forEach(child => {
      this.secondWalk(child, modSum + node.mod);
    });
  }

  /**
   * Apportion: Resolve conflicts between subtrees
   * This is the heart of the RT algorithm that ensures no overlaps.
   */
  private apportion(node: RTNode): void {
    let leftmost: RTNode | null = node.children[0];
    let neighbor: RTNode | null = this.leftSibling(leftmost);
    let compareDepth = 1;
    const maxDepth = this.getMaxDepth() - node.level;

    while (leftmost && neighbor && compareDepth <= maxDepth) {
      let leftModSum = 0;
      let rightModSum = 0;
      let ancestorLeftmost = leftmost;
      let ancestorNeighbor = neighbor;

      for (let i = 0; i < compareDepth; i++) {
        ancestorLeftmost = ancestorLeftmost.parent!;
        ancestorNeighbor = ancestorNeighbor.parent!;
        rightModSum += ancestorLeftmost.mod;
        leftModSum += ancestorNeighbor.mod;
      }

      const moveDistance = neighbor.prelim + leftModSum + this.getNodeSeparation() -
                          (leftmost.prelim + rightModSum);

      if (moveDistance > 0) {
        const ancestor = this.nextLeft(node.children[0]);
      if (ancestor) {
          this.moveSubtree(ancestor, node, moveDistance);
      }
        rightModSum += moveDistance;
      }

      compareDepth++;

      if (leftmost.children.length === 0) {
        leftmost = this.nextLeft(leftmost);
      } else {
        leftmost = leftmost.children[0];
      }

      if (neighbor.children.length === 0) {
        neighbor = this.nextRight(neighbor);
      } else {
        neighbor = neighbor.children[neighbor.children.length - 1];
      }

      // Break if we can't continue
      if (!leftmost || !neighbor) {
        break;
      }
    }
  }

  /**
   * Move subtree to resolve conflicts
   */
  private moveSubtree(wl: RTNode, wr: RTNode, shift: number): void {
    const subtrees = wr.number - wl.number;
    wr.change -= shift / subtrees;
    wr.shift += shift;
    wl.change += shift / subtrees;
    wr.prelim += shift;
    wr.mod += shift;
  }

  // Helper methods
  private leftSibling(node: RTNode): RTNode | null {
    if (!node.parent) return null;

    const siblings = node.parent.children;
    const index = siblings.indexOf(node);

    return index > 0 ? siblings[index - 1] : null;
  }

  private nextLeft(node: RTNode): RTNode | null {
    return node.children.length > 0 ? node.children[0] : node.thread;
  }

  private nextRight(node: RTNode): RTNode | null {
    const children = node.children;
    return children.length > 0 ? children[children.length - 1] : node.thread;
  }

  private getNodeSeparation(): number {
    return this.config.nodeWidth + this.config.horizontalSpacing;
  }

  private getMaxDepth(): number {
    let maxDepth = 0;
    this.nodeMap.forEach(node => {
      maxDepth = Math.max(maxDepth, node.level);
    });
    return maxDepth;
  }

  private convertToReactFlow(originalNodes: Node[], originalEdges: Edge[]): LayoutResult {
    const flowNodes: FlowNode[] = originalNodes.map(node => {
      const rtNode = this.nodeMap.get(node.id)!;

      return {
        ...node,
        position: { x: rtNode.x, y: rtNode.y },
        data: {
          ...node.data,
          level: rtNode.level,
          isRoot: rtNode.parent === null,
          layoutDirection: this.config.direction
        }
      } as FlowNode;
    });

    const flowEdges: FlowEdge[] = originalEdges.map(edge => ({
      ...edge,
      type: 'smoothstep',
      style: {
        stroke: '#bbb', // Changed from #666 to lighter gray
        strokeWidth: 2,
      },
      animated: false,
      label: (edge.label as string) || '',
      labelStyle: { fontSize: '11px', fill: '#666' },
      labelBgStyle: { fill: 'white', fillOpacity: 0.8 }
    }));

    return { nodes: flowNodes, edges: flowEdges };
  }
}

/**
 * Simple Hierarchical Layout (current implementation)
 * Kept for comparison and as fallback
 */
export class HierarchicalLayout {
  private config: LayoutConfig;

  constructor(config: LayoutConfig) {
    this.config = config;
  }

  public layout(nodes: Node[], edges: Edge[]): LayoutResult {
    const isHorizontal = this.config.direction === 'LR';

    // Build adjacency list
    const adjacencyList: { [key: string]: string[] } = {};
    const inDegree: { [key: string]: number } = {};

    nodes.forEach(node => {
      adjacencyList[node.id] = [];
      inDegree[node.id] = 0;
    });

    edges.forEach(edge => {
      adjacencyList[edge.source].push(edge.target);
      inDegree[edge.target] = (inDegree[edge.target] || 0) + 1;
    });

    // Find root nodes
    const roots = nodes.filter(node => inDegree[node.id] === 0);
    const rootNode = roots.length > 0 ? roots[0] : nodes[0];

    // BFS to assign levels
    const positions: { [key: string]: { x: number; y: number; level: number } } = {};
    const levelWidth: { [level: number]: number } = {};

    const queue = [{ id: rootNode.id, level: 0 }];
    const visited = new Set<string>();

    while (queue.length > 0) {
      const { id, level } = queue.shift()!;

      if (visited.has(id)) continue;
      visited.add(id);

      levelWidth[level] = (levelWidth[level] || 0) + 1;

      adjacencyList[id].forEach(childId => {
        if (!visited.has(childId)) {
          queue.push({ id: childId, level: level + 1 });
        }
      });
    }

    // Calculate positions
    const levelIndex: { [level: number]: number } = {};

    queue.push({ id: rootNode.id, level: 0 });
    visited.clear();

    while (queue.length > 0) {
      const { id, level } = queue.shift()!;

      if (visited.has(id)) continue;
      visited.add(id);

      levelIndex[level] = (levelIndex[level] || 0);
      const indexInLevel = levelIndex[level]++;

      const levelNodes = levelWidth[level];
      const totalWidth = levelNodes * this.config.nodeWidth + (levelNodes - 1) * this.config.horizontalSpacing;
      const startX = -totalWidth / 2;

      if (isHorizontal) {
        positions[id] = {
          x: level * (this.config.nodeHeight + this.config.verticalSpacing),
          y: startX + indexInLevel * (this.config.nodeWidth + this.config.horizontalSpacing),
          level
        };
      } else {
        positions[id] = {
          x: startX + indexInLevel * (this.config.nodeWidth + this.config.horizontalSpacing),
          y: level * (this.config.nodeHeight + this.config.verticalSpacing),
          level
        };
      }

      adjacencyList[id].forEach(childId => {
        if (!visited.has(childId)) {
          queue.push({ id: childId, level: level + 1 });
        }
      });
    }

    // Convert to ReactFlow format
    const flowNodes: FlowNode[] = nodes.map(node => ({
      ...node,
      type: node.type || 'resourceNode',
      position: {
        x: positions[node.id]?.x || 0,
        y: positions[node.id]?.y || 0
      },
      data: {
        ...node.data,
        level: positions[node.id]?.level || 0,
        isRoot: inDegree[node.id] === 0,
        layoutDirection: this.config.direction
      }
    }));

    const flowEdges: FlowEdge[] = edges.map(edge => ({
      ...edge,
      type: 'smoothstep',
      style: {
        stroke: '#bbb', // Changed from #666 to lighter gray
        strokeWidth: 2,
      },
      animated: false,
      label: (edge.label as string) || '',
      labelStyle: { fontSize: '11px', fill: '#666' },
      labelBgStyle: { fill: 'white', fillOpacity: 0.8 }
    }));

    return { nodes: flowNodes, edges: flowEdges };
  }
}

/**
 * Enhanced Tree Layout Algorithm
 *
 * This algorithm provides a tree layout with the following features:
 * - Maintains strict tree structure
 * - Prevents edge overlaps using intelligent spacing
 * - Optimizes for visual clarity and readability
 * - Uses layered positioning with conflict resolution
 */
export class EnhancedTreeLayout {
  private config: LayoutConfig;
  private nodeMap: Map<string, TreeLayoutNode> = new Map();
  private levelNodes: Map<number, TreeLayoutNode[]> = new Map();

  constructor(config: LayoutConfig) {
    this.config = config;
  }

  public layout(nodes: Node[], edges: Edge[]): LayoutResult {
    if (nodes.length === 0) {
      return { nodes: [], edges: [] };
    }

    // Clear previous state
    this.nodeMap.clear();
    this.levelNodes.clear();

    // Build tree structure
    const rootNode = this.buildTreeStructure(nodes, edges);

    // Calculate positions using enhanced algorithm
    this.calculatePositions(rootNode);

    // Convert back to ReactFlow format
    return this.convertToReactFlow(nodes, edges);
  }

  private buildTreeStructure(nodes: Node[], edges: Edge[]): TreeLayoutNode {
    // Create tree nodes
    nodes.forEach(node => {
      const treeNode: TreeLayoutNode = {
        id: node.id,
        children: [],
        parent: null,
        x: 0,
        y: 0,
        level: 0,
        subtreeWidth: 0,
        data: node.data,
        leftmostChild: null,
        rightmostChild: null,
        index: 0
      };
      this.nodeMap.set(node.id, treeNode);
    });

    // Build parent-child relationships
    edges.forEach(edge => {
      const parent = this.nodeMap.get(edge.source);
      const child = this.nodeMap.get(edge.target);

      if (parent && child) {
        parent.children.push(child);
        child.parent = parent;
      }
    });

    // Find root node
    let rootNode = Array.from(this.nodeMap.values()).find(node => node.parent === null);
    if (!rootNode) {
      rootNode = this.nodeMap.get(nodes[0].id)!;
    }

    // Assign levels and organize nodes by level
    this.assignLevels(rootNode, 0);

    // Sort children by their subtree sizes for better layout
    this.sortChildrenBySubtreeSize(rootNode);

    return rootNode;
  }

  private assignLevels(node: TreeLayoutNode, level: number): void {
    node.level = level;

    if (!this.levelNodes.has(level)) {
      this.levelNodes.set(level, []);
    }
    this.levelNodes.get(level)!.push(node);

    node.children.forEach(child => {
      this.assignLevels(child, level + 1);
    });
  }

  private sortChildrenBySubtreeSize(node: TreeLayoutNode): void {
    // First calculate subtree sizes
    this.calculateSubtreeSize(node);

    // Sort children by subtree size (larger subtrees in the middle)
    node.children.sort((a, b) => b.subtreeWidth - a.subtreeWidth);

    // Recursively sort children's children
    node.children.forEach(child => this.sortChildrenBySubtreeSize(child));

    // Update leftmost and rightmost references
    if (node.children.length > 0) {
      node.leftmostChild = node.children[0];
      node.rightmostChild = node.children[node.children.length - 1];
    }
  }

  private calculateSubtreeSize(node: TreeLayoutNode): number {
    if (node.children.length === 0) {
      node.subtreeWidth = 1;
      return 1;
    }

    let totalWidth = 0;
    node.children.forEach(child => {
      totalWidth += this.calculateSubtreeSize(child);
    });

    node.subtreeWidth = Math.max(totalWidth, 1);
    return node.subtreeWidth;
  }

  private calculatePositions(rootNode: TreeLayoutNode): void {
    const isHorizontal = this.config.direction === 'LR';

    // First pass: calculate relative positions within each subtree
    this.calculateSubtreePositions(rootNode, 0);

    // Second pass: resolve conflicts between subtrees at each level
    this.resolveConflicts();

    // Third pass: center the entire tree
    this.centerTree(rootNode);

    // Final pass: set absolute positions
    this.setAbsolutePositions(isHorizontal);
  }

  private calculateSubtreePositions(node: TreeLayoutNode, startX: number): number {
    if (node.children.length === 0) {
      node.x = startX;
      return startX + this.getNodeSeparation();
    }

    let currentX = startX;
    const childPositions: number[] = [];

    // Position children
    node.children.forEach((child, index) => {
      child.index = index;
      childPositions.push(currentX);
      currentX = this.calculateSubtreePositions(child, currentX);
    });

    // Position parent at the center of children
    const firstChildX = childPositions[0];
    const lastChildX = childPositions[childPositions.length - 1];
    node.x = (firstChildX + lastChildX) / 2;

    return currentX;
  }

  private resolveConflicts(): void {
    // Process each level to resolve horizontal conflicts
    for (const [, nodesAtLevel] of this.levelNodes.entries()) {
      if (nodesAtLevel.length <= 1) continue;

      // Sort nodes by their current x position
      nodesAtLevel.sort((a, b) => a.x - b.x);

      // Resolve overlaps
      for (let i = 1; i < nodesAtLevel.length; i++) {
        const prevNode = nodesAtLevel[i - 1];
        const currentNode = nodesAtLevel[i];

        const minDistance = this.getNodeSeparation();
        const requiredX = prevNode.x + minDistance;

        if (currentNode.x < requiredX) {
          const shift = requiredX - currentNode.x;
          this.shiftSubtree(currentNode, shift);
        }
      }
    }
  }

  private shiftSubtree(node: TreeLayoutNode, shift: number): void {
    node.x += shift;
    node.children.forEach(child => this.shiftSubtree(child, shift));
  }

  private centerTree(_rootNode: TreeLayoutNode): void {
    // Find the bounds of the tree
    let minX = Number.MAX_VALUE;
    let maxX = Number.MIN_VALUE;

    this.nodeMap.forEach(node => {
      minX = Math.min(minX, node.x);
      maxX = Math.max(maxX, node.x);
    });

    // Center the tree
    const treeWidth = maxX - minX;
    const centerShift = -treeWidth / 2;

    this.nodeMap.forEach(node => {
      node.x += centerShift;
    });
  }

  private setAbsolutePositions(isHorizontal: boolean): void {
    this.nodeMap.forEach(node => {
      if (isHorizontal) {
        const temp = node.x;
        node.x = node.level * (this.config.nodeHeight + this.config.verticalSpacing);
        node.y = temp;
      } else {
        node.y = node.level * (this.config.nodeHeight + this.config.verticalSpacing);
      }
    });
  }

  private getNodeSeparation(): number {
    return this.config.nodeWidth + this.config.horizontalSpacing;
  }

  private convertToReactFlow(originalNodes: Node[], originalEdges: Edge[]): LayoutResult {
    const flowNodes: FlowNode[] = originalNodes.map(node => {
      const treeNode = this.nodeMap.get(node.id)!;

      return {
        ...node,
        position: { x: treeNode.x, y: treeNode.y },
        data: {
          ...node.data,
          level: treeNode.level,
          isRoot: treeNode.parent === null,
          layoutDirection: this.config.direction
        }
      } as FlowNode;
    });

    const flowEdges: FlowEdge[] = originalEdges.map(edge => ({
      ...edge,
      type: 'smoothstep',
      style: {
        stroke: '#bbb',
        strokeWidth: 2,
      },
      animated: false,
      label: (edge.label as string) || '',
      labelStyle: { fontSize: '11px', fill: '#666' },
      labelBgStyle: { fill: 'white', fillOpacity: 0.8 }
    }));

    return { nodes: flowNodes, edges: flowEdges };
  }
}

// Internal node structure for Enhanced Tree Layout
interface TreeLayoutNode {
  id: string;
  children: TreeLayoutNode[];
  parent: TreeLayoutNode | null;
  x: number;
  y: number;
  level: number;
  subtreeWidth: number;
  data: any;
  leftmostChild: TreeLayoutNode | null;
  rightmostChild: TreeLayoutNode | null;
  index: number; // Index among siblings
}

/**
 * Layout Engine Factory
 * Creates appropriate layout algorithm based on type
 */
export function createLayoutEngine(
  algorithm: string,
  config: LayoutConfig
): HierarchicalLayout | ReingoldTilfordLayout | EnhancedTreeLayout {
  switch (algorithm) {
    case 'reingold-tilford':
      return new ReingoldTilfordLayout(config);
    case 'enhanced-tree':
      return new EnhancedTreeLayout(config);
    case 'hierarchical':
    default:
      return new HierarchicalLayout(config);
  }
}
