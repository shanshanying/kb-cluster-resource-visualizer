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

export interface ResourceRelationship {
  parent: ResourceNode;
  children: ResourceNode[];
}

export interface FlowNode {
  id: string;
  type: string;
  position: { x: number; y: number };
  data: {
    resource: ResourceNode;
    isParent?: boolean;
  };
}

export interface FlowEdge {
  id: string;
  source: string;
  target: string;
  type: string;
}
