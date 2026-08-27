import React from 'react';

export type DriftStatus = 'IN_SYNC' | 'MODIFIED' | 'MISSING_IN_STATE' | 'MISSING_IN_CODE' | 'unknown';

export interface Position {
  x: number;
  y: number;
}

export interface SecurityIssue {
  ruleId: string;
  severity: 'CRITICAL' | 'HIGH' | 'MEDIUM' | 'LOW' | string;
  title: string;
  description: string;
}

export interface NodeData extends Record<string, unknown> {
  label: string;
  provider: string;
  resourceType: string;
  module: string;
  isDataSource: boolean;
  isContainer?: boolean;
  driftStatus: DriftStatus;
  attributes?: Record<string, any>;
  securityIssues?: SecurityIssue[];
}

export interface Node {
  id: string;
  type: string;
  position: Position;
  data: NodeData;
  parentId?: string;
  extent?: 'parent' | [[number, number], [number, number]] | null;
  style?: React.CSSProperties;
}

export interface Edge {
  id: string;
  source: string;
  target: string;
  type: string;
  animated: boolean;
}

export interface GraphResponse {
  nodes: Node[];
  edges: Edge[];
}
