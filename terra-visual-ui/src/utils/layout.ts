import dagre from 'dagre';
import { Node, Edge, Position } from '@xyflow/react';
import { NodeData } from '../types/graph';

export const standardCardWidth = 240;
export const standardCardHeight = 75;
export const moduleCardWidth = 260;
export const moduleCardHeight = 75;
export const subnetContainerWidth = 280;
export const subnetContainerHeight = 180;
export const azContainerWidth = 380;
export const azContainerHeight = 260;
export const vpcContainerWidth = 460;
export const vpcContainerHeight = 340;

const getNodeDimensions = (node: Node<NodeData>, parentSet: Set<string>): { width: number; height: number } => {
  const resourceType = node.data?.resourceType || '';
  const isContainer = parentSet.has(node.id) || Boolean(node.data?.isContainer);
  const isModule = resourceType === 'module';

  if (isContainer) {
    if (resourceType === 'aws_vpc' || resourceType === 'azurerm_virtual_network' || resourceType === 'google_compute_network') {
      return { width: vpcContainerWidth, height: vpcContainerHeight };
    }
    if (resourceType === 'aws_availability_zone') {
      return { width: azContainerWidth, height: azContainerHeight };
    }
    if (resourceType === 'aws_subnet' || resourceType === 'azurerm_subnet') {
      return { width: subnetContainerWidth, height: subnetContainerHeight };
    }
    if (isModule) {
      return { width: 400, height: 280 };
    }
    return { width: azContainerWidth, height: azContainerHeight };
  }

  if (isModule) {
    return { width: moduleCardWidth, height: moduleCardHeight };
  }

  return { width: standardCardWidth, height: standardCardHeight };
};

/**
 * Calculates a top-to-bottom (TB) or left-to-right (LR) directed acyclic graph layout using Dagre with compound subgraphs.
 * Handles AWS Topology multi-tier subgraphs (VPC -> AZ -> Subnet -> Resource).
 */
export const getLayoutedElements = (
  nodes: Node<NodeData>[],
  edges: Edge[],
  direction: 'TB' | 'LR' = 'TB'
): { nodes: Node<NodeData>[]; edges: Edge[] } => {
  if (!nodes || nodes.length === 0) {
    return { nodes: [], edges: edges || [] };
  }

  const isHorizontal = direction === 'LR';
  const parentSet = new Set<string>();
  const nodeMap = new Map<string, Node<NodeData>>();

  nodes.forEach((node) => {
    if (node?.id) {
      nodeMap.set(node.id, node);
      if (node.parentId) {
        parentSet.add(node.parentId);
      }
    }
  });

  try {
    // 1. Enable compound graphs in Dagre
    const dagreGraph = new dagre.graphlib.Graph({ compound: true });
    dagreGraph.setDefaultEdgeLabel(() => ({}));

    dagreGraph.setGraph({
      rankdir: direction,
      ranksep: 80, // Explicit vertical spacing
      nodesep: 60, // Explicit horizontal spacing
      compound: true,
    });

    // 2. Register nodes with hierarchical dimensions
    nodes.forEach((node) => {
      if (!node?.id) return;
      const { width, height } = getNodeDimensions(node, parentSet);

      dagreGraph.setNode(node.id, { width, height });

      if (node.parentId) {
        dagreGraph.setParent(node.id, node.parentId);
      }
    });

    // 3. Register edges safely
    (edges || []).forEach((edge) => {
      if (edge?.source && edge?.target) {
        dagreGraph.setEdge(edge.source, edge.target);
      }
    });

    // 4. Run Dagre Layout
    dagre.layout(dagreGraph);

    // 5. Calculate positions and recursive parent offsets
    const layoutedNodes: Node<NodeData>[] = nodes.map((node, index) => {
      if (!node?.id) return node;

      const { width: defaultW, height: defaultH } = getNodeDimensions(node, parentSet);

      const nodeWithPosition = dagreGraph.node(node.id) || {
        x: index * (standardCardWidth + 30),
        y: 100,
        width: defaultW,
        height: defaultH,
      };

      const nWidth = nodeWithPosition.width || defaultW;
      const nHeight = nodeWithPosition.height || defaultH;
      let x = nodeWithPosition.x - nWidth / 2;
      let y = nodeWithPosition.y - nHeight / 2;

      const updatedNode: Node<NodeData> = {
        ...node,
        targetPosition: isHorizontal ? Position.Left : Position.Top,
        sourcePosition: isHorizontal ? Position.Right : Position.Bottom,
      };

      // In React Flow, child position is relative to its immediate parent
      if (node.parentId) {
        const parentWithPosition = dagreGraph.node(node.parentId);
        if (parentWithPosition) {
          const parentDim = nodeMap.get(node.parentId)
            ? getNodeDimensions(nodeMap.get(node.parentId)!, parentSet)
            : { width: azContainerWidth, height: azContainerHeight };
          const pWidth = parentWithPosition.width || parentDim.width;
          const pHeight = parentWithPosition.height || parentDim.height;
          const parentX = parentWithPosition.x - pWidth / 2;
          const parentY = parentWithPosition.y - pHeight / 2;
          x -= parentX;
          y -= parentY;
          updatedNode.extent = 'parent';
        }
      }

      updatedNode.position = { x: isNaN(x) ? 0 : x, y: isNaN(y) ? 0 : y };
      updatedNode.style = {
        ...node.style,
        width: nWidth,
        height: nHeight,
      };

      return updatedNode;
    });

    // 6. Ensure ancestor nodes appear before descendants in the array
    const getAncestorDepth = (id: string): number => {
      let depth = 0;
      let curr = id;
      const visited = new Set<string>();
      while (curr) {
        if (visited.has(curr)) break;
        visited.add(curr);
        const parent = nodeMap.get(curr)?.parentId;
        if (!parent) break;
        depth++;
        curr = parent;
      }
      return depth;
    };

    layoutedNodes.sort((a, b) => {
      const depthA = getAncestorDepth(a.id);
      const depthB = getAncestorDepth(b.id);
      if (depthA !== depthB) {
        return depthA - depthB;
      }
      return a.id.localeCompare(b.id);
    });

    return { nodes: layoutedNodes, edges: edges || [] };
  } catch (error) {
    console.warn('Dagre layout failed, falling back to grid layout:', error);

    // Graceful fallback grid layout
    const fallbackNodes = nodes.map((node, i) => ({
      ...node,
      position: node.position && !isNaN(node.position.x) ? node.position : { x: (i % 4) * 280, y: Math.floor(i / 4) * 120 },
    }));

    return { nodes: fallbackNodes, edges: edges || [] };
  }
};
