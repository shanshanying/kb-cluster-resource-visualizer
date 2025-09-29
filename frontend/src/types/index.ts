export interface ResourceNode {
  name: string;
  kind: string;
  apiVersion: string;
  namespace?: string;
  uid: string;
  labels?: Record<string, string>;
  annotations?: Record<string, string>;
  creationTime: string;
  status?: string;
}


export interface TreeNode {
  resource: {
    metadata: {
      name: string;
      namespace?: string;
      uid: string;
      labels?: Record<string, string>;
      annotations?: Record<string, string>;
      creationTimestamp: string;
    };
    kind: string;
    apiVersion: string;
    status?: any;
  };
  children: TreeNode[];
}

export interface FlowNode {
  id: string;
  type: string;
  position: { x: number; y: number };
  data: {
    resource: ResourceNode;
    isParent?: boolean;
    level?: number;
    isRoot?: boolean;
    layoutDirection?: 'TB' | 'LR';
  };
}

export interface FlowEdge {
  id: string;
  source: string;
  target: string;
  type: string;
  style?: React.CSSProperties;
  animated?: boolean;
  label?: string;
  labelStyle?: React.CSSProperties;
  labelBgStyle?: React.CSSProperties;
}
