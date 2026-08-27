import dagre from 'dagre';
import { Node, Edge, Position } from '@xyflow/react';
import { NodeData } from '../types/graph';

export const standardCardWidth = 240;
export const standardCardHeight = 75;
export const subnetCardWidth = 330;
export const subnetCardHeight = 85;
export const computeCardWidth = 330;
export const computeCardHeight = 85;
export const moduleCardWidth = 260;
export const moduleCardHeight = 75;

export const minSubnetWidth = 280;
export const minSubnetHeight = 150;
export const minAZWidth = 370;
export const minAZHeight = 275;
export const minVPCWidth = 840;
export const minVPCHeight = 560;

interface NodeSize {
  width: number;
  height: number;
}

/**
 * Calculates a hierarchical compound layout for React Flow subgraphs.
 * 
 * Two-pass layout engine:
 * Pass 1: Bottom-up layout of containers from leaf nodes up to root containers:
 *         - AZ Containers: Vertical columns (Public Subnet top y:52, Private Subnet bottom y:160, 370x275)
 *         - VPC Container: IGW at top y:56, AZ columns at y:135, EKS Control Plane & Node Groups at y:440 (840x560)
 * Pass 2: Global Dagre layout of root containers and external resources (S3, DynamoDB at x:900+).
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
  const nodeMap = new Map<string, Node<NodeData>>();
  const parentSet = new Set<string>();
  const childrenMap = new Map<string, string[]>(); // parentId -> childIds

  // 1. Index all nodes and build parent-child adjacency
  nodes.forEach((n) => {
    if (!n?.id) return;
    nodeMap.set(n.id, { ...n });
    if (n.parentId) {
      parentSet.add(n.parentId);
      const existing = childrenMap.get(n.parentId) || [];
      existing.push(n.id);
      childrenMap.set(n.parentId, existing);
    }
  });

  // Calculate ancestor depth for each node (0 = root, 1 = child of root, 2 = grandchild, etc.)
  const getDepth = (id: string): number => {
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

  const nodeDepths = new Map<string, number>();
  nodes.forEach((n) => {
    if (n?.id) {
      nodeDepths.set(n.id, getDepth(n.id));
    }
  });

  let maxDepth = 0;
  nodeDepths.forEach((d) => {
    if (d > maxDepth) maxDepth = d;
  });

  const nodeDimensions = new Map<string, NodeSize>();

  // Helper to get base size for leaf cards
  const getBaseCardSize = (n: Node<NodeData>): NodeSize => {
    const resType = n.data?.resourceType || '';
    if (resType === 'aws_subnet' || resType === 'azurerm_subnet') {
      return { width: subnetCardWidth, height: subnetCardHeight };
    }
    if (resType.includes('eks') || resType.includes('cluster') || resType.includes('node_group')) {
      return { width: computeCardWidth, height: computeCardHeight };
    }
    if (resType === 'aws_internet_gateway') {
      return { width: 240, height: 60 };
    }
    const isModule = resType === 'module';
    return {
      width: isModule ? moduleCardWidth : standardCardWidth,
      height: isModule ? moduleCardHeight : standardCardHeight,
    };
  };

  // 2. PASS 1: Bottom-up layout of containers from deepest children to top level
  for (let d = maxDepth; d >= 0; d--) {
    nodes.forEach((n) => {
      if (!n?.id || nodeDepths.get(n.id) !== d) return;

      const childIds = childrenMap.get(n.id) || [];
      const isContainer = parentSet.has(n.id) || Boolean(n.data?.isContainer);

      if (!isContainer || childIds.length === 0) {
        // Leaf node or empty container
        const base = getBaseCardSize(n);
        nodeDimensions.set(n.id, base);
        return;
      }

      const resType = n.data?.resourceType || '';
      const isAZ = resType === 'aws_availability_zone';
      const isVPC = resType === 'aws_vpc' || resType === 'azurerm_virtual_network' || resType === 'google_compute_network' || n.id === 'vpc-main';

      if (isAZ) {
        // Layout children vertically inside AZ container (Public Subnet on Top, Private Subnet on Bottom)
        const pubSubnets: string[] = [];
        const privSubnets: string[] = [];
        const otherAZChildren: string[] = [];

        childIds.forEach((cid) => {
          const child = nodeMap.get(cid);
          if (!child) return;
          const subType = child.data?.attributes?.subnet_type || child.data?.tier || '';
          const label = (child.data?.label || '').toLowerCase();
          if (subType === 'public' || label.includes('public')) {
            pubSubnets.push(cid);
          } else if (subType === 'private' || label.includes('private')) {
            privSubnets.push(cid);
          } else {
            otherAZChildren.push(cid);
          }
        });

        const ordered = [...pubSubnets, ...privSubnets, ...otherAZChildren];
        let currentY = 52;
        const padX = 20;

        ordered.forEach((cid) => {
          const childNode = nodeMap.get(cid);
          if (!childNode) return;

          childNode.position = { x: padX, y: currentY };
          childNode.extent = 'parent';
          childNode.style = { ...childNode.style, width: subnetCardWidth, height: subnetCardHeight };
          nodeMap.set(cid, childNode);

          currentY += subnetCardHeight + 20;
        });

        const azW = Math.max(minAZWidth, padX * 2 + subnetCardWidth);
        const azH = Math.max(minAZHeight, currentY + 10);
        nodeDimensions.set(n.id, { width: azW, height: azH });

        const updatedAZ = nodeMap.get(n.id)!;
        updatedAZ.style = { ...updatedAZ.style, width: azW, height: azH };
        nodeMap.set(n.id, updatedAZ);
        return;
      }

      if (isVPC) {
        // Layout VPC Container:
        // Tier 1 (y: 56): Internet Gateway (centered at x: 300)
        // Tier 2 (y: 135): AZ Containers side-by-side (x: 30, x: 435)
        // Tier 3 (y: 440): EKS Cluster (x: 50) and EKS Node Group (x: 455)
        const igwNodes: string[] = [];
        const azNodes: string[] = [];
        const computeNodes: string[] = [];
        const fallbackNodes: string[] = [];

        childIds.forEach((cid) => {
          const child = nodeMap.get(cid);
          if (!child) return;
          const cType = child.data?.resourceType || '';
          if (cType === 'aws_internet_gateway' || cid.includes('igw')) {
            igwNodes.push(cid);
          } else if (cType === 'aws_availability_zone' || cid.startsWith('az-')) {
            azNodes.push(cid);
          } else if (cType.includes('eks') || cType.includes('cluster') || cType.includes('node_group')) {
            computeNodes.push(cid);
          } else {
            fallbackNodes.push(cid);
          }
        });

        // 1. Position Internet Gateway at top center
        igwNodes.forEach((cid) => {
          const childNode = nodeMap.get(cid);
          if (!childNode) return;
          childNode.position = { x: 300, y: 56 };
          childNode.extent = 'parent';
          childNode.style = { ...childNode.style, width: 240, height: 60 };
          nodeMap.set(cid, childNode);
        });

        // 2. Position AZ Containers side-by-side as vertical columns
        const azPadX = 30;
        const azGapX = 35;
        let azY = 135;
        let maxAZH = minAZHeight;

        azNodes.forEach((cid, idx) => {
          const childNode = nodeMap.get(cid);
          if (!childNode) return;
          const dim = nodeDimensions.get(cid) || { width: minAZWidth, height: minAZHeight };
          const posX = azPadX + idx * (dim.width + azGapX);

          childNode.position = { x: posX, y: azY };
          childNode.extent = 'parent';
          childNode.style = { ...childNode.style, width: dim.width, height: dim.height };
          nodeMap.set(cid, childNode);

          if (dim.height > maxAZH) maxAZH = dim.height;
        });

        // 3. Position EKS Cluster & Node Group below AZ columns
        let computeY = azY + maxAZH + 25;
        computeNodes.forEach((cid) => {
          const childNode = nodeMap.get(cid);
          if (!childNode) return;
          const cType = childNode.data?.resourceType || '';
          const isCluster = cType === 'aws_eks_cluster' || cid.includes('cluster');
          const posX = isCluster ? 50 : 455;

          childNode.position = { x: posX, y: computeY };
          childNode.extent = 'parent';
          childNode.style = { ...childNode.style, width: computeCardWidth, height: computeCardHeight };
          nodeMap.set(cid, childNode);
        });

        // 4. Position fallback nodes if any
        let fallbackY = computeY + (computeNodes.length > 0 ? computeCardHeight + 20 : 0);
        fallbackNodes.forEach((cid, idx) => {
          const childNode = nodeMap.get(cid);
          if (!childNode) return;
          childNode.position = { x: 30 + idx * 260, y: fallbackY };
          childNode.extent = 'parent';
          childNode.style = { ...childNode.style, width: standardCardWidth, height: standardCardHeight };
          nodeMap.set(cid, childNode);
        });

        const totalVPCW = Math.max(minVPCWidth, azPadX * 2 + azNodes.length * (minAZWidth + azGapX));
        const totalVPCH = Math.max(minVPCHeight, fallbackY + (fallbackNodes.length > 0 ? standardCardHeight + 24 : 30));

        nodeDimensions.set(n.id, { width: totalVPCW, height: totalVPCH });
        const updatedVPC = nodeMap.get(n.id)!;
        updatedVPC.style = { ...updatedVPC.style, width: totalVPCW, height: totalVPCH };
        nodeMap.set(n.id, updatedVPC);
        return;
      }

      // Generic container fallback
      let currentX = 24;
      let currentY = 60;
      childIds.forEach((cid) => {
        const childNode = nodeMap.get(cid);
        if (!childNode) return;
        const dim = nodeDimensions.get(cid) || getBaseCardSize(childNode);
        childNode.position = { x: currentX, y: currentY };
        childNode.extent = 'parent';
        childNode.style = { ...childNode.style, width: dim.width, height: dim.height };
        nodeMap.set(cid, childNode);
        currentY += dim.height + 16;
      });

      const genericW = 380;
      const genericH = Math.max(240, currentY + 20);
      nodeDimensions.set(n.id, { width: genericW, height: genericH });
      const updatedContainer = nodeMap.get(n.id)!;
      updatedContainer.style = { ...updatedContainer.style, width: genericW, height: genericH };
      nodeMap.set(n.id, updatedContainer);
    });
  }

  // 3. PASS 2: Global Dagre layout for root nodes (nodes with !parentId)
  const rootNodes = nodes.filter((n) => !n?.parentId);

  if (rootNodes.length > 0) {
    try {
      const dagreGraph = new dagre.graphlib.Graph();
      dagreGraph.setDefaultEdgeLabel(() => ({}));
      dagreGraph.setGraph({
        rankdir: isHorizontal ? 'LR' : 'TB',
        ranksep: 100, // Explicit vertical spacing
        nodesep: 80,  // Explicit horizontal spacing
      });

      rootNodes.forEach((rn) => {
        const dim = nodeDimensions.get(rn.id) || { width: standardCardWidth, height: standardCardHeight };
        dagreGraph.setNode(rn.id, { width: dim.width, height: dim.height });
      });

      // Add root-level edges
      const rootSet = new Set(rootNodes.map((r) => r.id));
      (edges || []).forEach((e) => {
        const getRootAncestor = (id: string): string => {
          let curr = id;
          const visited = new Set<string>();
          while (curr) {
            if (visited.has(curr)) break;
            visited.add(curr);
            const p = nodeMap.get(curr)?.parentId;
            if (!p) return curr;
            curr = p;
          }
          return curr;
        };

        const rootSrc = getRootAncestor(e.source);
        const rootTgt = getRootAncestor(e.target);

        if (rootSrc && rootTgt && rootSrc !== rootTgt && rootSet.has(rootSrc) && rootSet.has(rootTgt)) {
          dagreGraph.setEdge(rootSrc, rootTgt);
        }
      });

      dagre.layout(dagreGraph);

      rootNodes.forEach((rn, idx) => {
        const dim = nodeDimensions.get(rn.id) || { width: standardCardWidth, height: standardCardHeight };
        const dagreNode = dagreGraph.node(rn.id) || {
          x: idx * (dim.width + 80),
          y: 100,
          width: dim.width,
          height: dim.height,
        };

        const updated = nodeMap.get(rn.id)!;
        updated.position = {
          x: dagreNode.x - dim.width / 2,
          y: dagreNode.y - dim.height / 2,
        };
        updated.targetPosition = isHorizontal ? Position.Left : Position.Top;
        updated.sourcePosition = isHorizontal ? Position.Right : Position.Bottom;
        updated.style = {
          ...updated.style,
          width: dim.width,
          height: dim.height,
        };
        nodeMap.set(rn.id, updated);
      });
    } catch (e) {
      console.warn('Root Dagre layout error, falling back to grid:', e);
    }
  }

  // 4. Assemble final node array sorted parents before children
  const resultNodes: Node<NodeData>[] = Array.from(nodeMap.values());
  resultNodes.sort((a, b) => {
    const depthA = nodeDepths.get(a.id) || 0;
    const depthB = nodeDepths.get(b.id) || 0;
    if (depthA !== depthB) {
      return depthA - depthB;
    }
    return a.id.localeCompare(b.id);
  });

  return {
    nodes: resultNodes,
    edges: edges || [],
  };
};
