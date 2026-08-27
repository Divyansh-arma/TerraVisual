import React, { useEffect, useState, useMemo, useCallback } from 'react';
import {
  ReactFlow,
  Background,
  Controls,
  MiniMap,
  useNodesState,
  useEdgesState,
  Panel,
  Node,
  Edge,
  MarkerType,
  useReactFlow,
  ReactFlowProvider,
} from '@xyflow/react';
import '@xyflow/react/dist/style.css';

import { GraphResponse, NodeData } from '../types/graph';
import { InfrastructureNode } from './InfrastructureNode';
import { AddResourceModal } from './AddResourceModal';
import { InspectorDrawer } from './InspectorDrawer';
import { DriftDrawer } from './DriftDrawer';
import { ExportButton } from './ExportButton';
import { ResourceTemplate } from '../constants/resourceTemplates';
import { getLayoutedElements } from '../utils/layout';
import {
  Maximize2,
  Minimize2,
  ArrowDownUp,
  ArrowLeftRight,
  Search,
  PlusCircle,
  UploadCloud,
  CheckCircle2,
} from 'lucide-react';

interface GraphCanvasProps {
  graph: GraphResponse | null;
  codePath: string;
  onSyncToCode: (currentGraph: GraphResponse) => Promise<void>;
  isSyncing: boolean;
  isDriftDrawerOpen?: boolean;
  onCloseDriftDrawer?: () => void;
}

const GraphCanvasInner: React.FC<GraphCanvasProps> = ({
  graph,
  codePath,
  onSyncToCode,
  isSyncing,
  isDriftDrawerOpen = false,
  onCloseDriftDrawer = () => {},
}) => {
  const [direction, setDirection] = useState<'TB' | 'LR'>('TB');
  const [selectedNode, setSelectedNode] = useState<Node<NodeData> | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [isFullScreen, setIsFullScreen] = useState(false);
  const [syncSuccess, setSyncSuccess] = useState<string | null>(null);
  const [isAddModalOpen, setIsAddModalOpen] = useState<boolean>(false);

  const { fitView } = useReactFlow();

  const nodeTypes = useMemo(
    () => ({
      infrastructureNode: InfrastructureNode,
    }),
    []
  );

  const [nodes, setNodes, onNodesChange] = useNodesState<Node<NodeData>>([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([]);

  // Update nodes and edges whenever graph data changes
  useEffect(() => {
    if (!graph || graph.nodes.length === 0) {
      setNodes([]);
      setEdges([]);
      setSelectedNode(null);
      return;
    }

    const flowNodes: Node<NodeData>[] = graph.nodes.map((n) => ({
      id: n.id,
      type: 'infrastructureNode',
      position: n.position || { x: 0, y: 0 },
      data: n.data,
      parentId: n.parentId,
      extent: n.extent,
    }));

    const flowEdges: Edge[] = graph.edges.map((e) => ({
      id: e.id,
      source: e.source,
      target: e.target,
      type: e.type || 'smoothstep',
      animated: e.animated ?? true,
      style: {
        stroke: '#38bdf8',
        strokeWidth: 2,
      },
      markerEnd: {
        type: MarkerType.ArrowClosed,
        width: 16,
        height: 16,
        color: '#38bdf8',
      },
    }));

    const layouted = getLayoutedElements(flowNodes, flowEdges, direction);
    setNodes(layouted.nodes);
    setEdges(layouted.edges);

    // Automatically center and scale viewport smoothly
    setTimeout(() => {
      fitView({ padding: 0.2, duration: 400 });
    }, 50);
  }, [graph, direction, setNodes, setEdges, fitView]);

  const onNodeClick = useCallback((_: React.MouseEvent, node: Node<NodeData>) => {
    setSelectedNode(node);
  }, []);

  const onPaneClick = useCallback(() => {
    setSelectedNode(null);
  }, []);

  const toggleDirection = () => {
    setDirection((prev) => (prev === 'TB' ? 'LR' : 'TB'));
  };

  const handleSelectNodeById = useCallback(
    (nodeId: string) => {
      const target = nodes.find((n) => n.id === nodeId);
      if (target) {
        setSelectedNode(target);
      }
    },
    [nodes]
  );

  // Add resource from Catalog Modal
  const handleAddResourceFromCatalog = useCallback(
    (template: ResourceTemplate, customName: string, attributes: Record<string, any>) => {
      const newId = `${template.resourceType}.${customName}`;

      let parentId: string | undefined = undefined;
      if (attributes.vpc_id && typeof attributes.vpc_id === 'string') {
        const parts = attributes.vpc_id.split('.');
        if (parts.length >= 2) {
          parentId = `${parts[0]}.${parts[1]}`;
        } else {
          parentId = attributes.vpc_id;
        }
      }

      const newNode: Node<NodeData> = {
        id: newId,
        type: 'infrastructureNode',
        position: { x: 50, y: 50 },
        data: {
          label: customName,
          provider: template.provider,
          resourceType: template.resourceType,
          module: 'root',
          isDataSource: false,
          isContainer: template.resourceType === 'aws_vpc' || template.resourceType === 'azurerm_virtual_network',
          driftStatus: 'MISSING_IN_STATE',
          attributes: { ...template.defaultAttributes, ...attributes },
        },
        parentId,
      };

      const newEdges: Edge[] = [];
      if (attributes.vpc_id && typeof attributes.vpc_id === 'string') {
        const vpcId = attributes.vpc_id.split('.')[0] + '.' + (attributes.vpc_id.split('.')[1] || 'main');
        newEdges.push({
          id: `e-${vpcId}-${newId}`,
          source: vpcId,
          target: newId,
          type: 'smoothstep',
          animated: true,
          style: { stroke: '#38bdf8', strokeWidth: 2 },
          markerEnd: { type: MarkerType.ArrowClosed, width: 16, height: 16, color: '#38bdf8' },
        });
      }

      setNodes((prev) => {
        const all = [...prev, newNode];
        const layouted = getLayoutedElements(all, [...edges, ...newEdges], direction);
        return layouted.nodes;
      });

      setEdges((prev) => [...prev, ...newEdges]);
      setIsAddModalOpen(false);
      setSelectedNode(newNode);
    },
    [direction, edges, setEdges, setNodes]
  );

  // Handle Sync to Code
  const handleSync = async () => {
    try {
      const graphResponse: GraphResponse = {
        nodes: nodes.map((n) => ({
          id: n.id,
          type: n.type || 'infrastructureNode',
          position: n.position,
          data: n.data,
          parentId: n.parentId,
          extent: n.extent,
        })),
        edges: edges.map((e) => ({
          id: e.id,
          source: e.source,
          target: e.target,
          type: e.type || 'smoothstep',
          animated: Boolean(e.animated),
        })),
      };

      await onSyncToCode(graphResponse);
      setSyncSuccess('Saved to .tf code!');
      setTimeout(() => setSyncSuccess(null), 3000);
    } catch (err: any) {
      console.error('Failed to sync graph to code:', err);
    }
  };

  // Filter nodes based on search query
  const displayedNodes = useMemo(() => {
    if (!searchQuery.trim()) return nodes;
    const q = searchQuery.toLowerCase();
    return nodes.map((node) => {
      const label = node.data?.label?.toLowerCase() || '';
      const rt = node.data?.resourceType?.toLowerCase() || '';
      const id = node.id.toLowerCase();
      const isMatch = label.includes(q) || rt.includes(q) || id.includes(q);

      return {
        ...node,
        style: {
          ...node.style,
          opacity: isMatch ? 1 : 0.25,
        },
      };
    });
  }, [nodes, searchQuery]);

  return (
    <div
      className={`relative w-full bg-slate-950 border border-slate-800/90 rounded-2xl overflow-hidden shadow-2xl transition-all ${
        isFullScreen ? 'fixed inset-2 z-50 h-[calc(100vh-1rem)]' : 'h-[calc(100vh-72px)] min-h-[580px]'
      }`}
    >
      <ReactFlow
        nodes={displayedNodes}
        edges={edges}
        nodeTypes={nodeTypes}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        onNodeClick={onNodeClick}
        onPaneClick={onPaneClick}
        fitView
        fitViewOptions={{ padding: 0.2 }}
        minZoom={0.2}
        maxZoom={2}
        deleteKeyCode={['Backspace', 'Delete']}
        className="bg-slate-950"
      >
        <Background color="#1e293b" gap={20} size={1.5} />
        <Controls
          position="bottom-left"
          className="!bg-slate-900 !border-slate-800 !text-slate-200 fill-current shadow-xl !mb-4 !ml-4"
        />
        <MiniMap
          nodeColor={(node) => {
            const data = node.data as unknown as NodeData;
            if (data?.driftStatus === 'IN_SYNC') return '#10b981';
            if (data?.driftStatus === 'MODIFIED') return '#f59e0b';
            if (data?.driftStatus === 'CREATE' || data?.driftStatus === 'MISSING_IN_STATE') return '#10b981';
            if (data?.driftStatus === 'DESTROY' || data?.driftStatus === 'MISSING_IN_CODE') return '#f43f5e';
            return '#64748b';
          }}
          className="!bg-slate-900/90 !border-slate-800 !rounded-xl overflow-hidden shadow-xl !mb-4 !mr-4"
          maskColor="rgba(15, 23, 42, 0.7)"
        />

        {/* Top-Left Control Bar Panel */}
        <Panel position="top-left" className="flex items-center gap-2 bg-slate-900/90 p-2 rounded-xl border border-slate-800 shadow-xl backdrop-blur">
          {/* Search */}
          <div className="relative flex items-center">
            <Search className="w-3.5 h-3.5 text-slate-500 absolute left-2.5" />
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Filter resources..."
              className="bg-slate-950 border border-slate-800 rounded-lg pl-8 pr-2.5 py-1 text-xs text-slate-200 placeholder-slate-600 focus:outline-none focus:border-sky-500 w-36 transition font-mono"
            />
          </div>

          {/* Add Resource Catalog Button */}
          <button
            type="button"
            onClick={() => setIsAddModalOpen(true)}
            className="flex items-center gap-1.5 px-3 py-1 text-xs font-semibold bg-gradient-to-r from-sky-600 to-indigo-600 hover:from-sky-500 hover:to-indigo-500 text-white rounded-lg transition shadow-md shadow-sky-950"
            title="Open Multi-Cloud Resource Catalog"
          >
            <PlusCircle className="w-3.5 h-3.5" />
            <span>Add Resource</span>
          </button>

          {/* Direction toggle */}
          <button
            type="button"
            onClick={toggleDirection}
            className="flex items-center gap-1.5 px-2.5 py-1 text-xs font-semibold bg-slate-800 hover:bg-slate-700 text-slate-200 rounded-lg border border-slate-700 transition"
            title="Toggle Layout Direction"
          >
            {direction === 'TB' ? (
              <>
                <ArrowDownUp className="w-3.5 h-3.5 text-sky-400" />
                <span>Vertical</span>
              </>
            ) : (
              <>
                <ArrowLeftRight className="w-3.5 h-3.5 text-indigo-400" />
                <span>Horizontal</span>
              </>
            )}
          </button>

          {/* Fullscreen toggle */}
          <button
            type="button"
            onClick={() => setIsFullScreen(!isFullScreen)}
            className="p-1.5 bg-slate-800 hover:bg-slate-700 text-slate-200 rounded-lg border border-slate-700 transition"
            title={isFullScreen ? 'Exit Full Screen' : 'Full Screen'}
          >
            {isFullScreen ? <Minimize2 className="w-3.5 h-3.5" /> : <Maximize2 className="w-3.5 h-3.5" />}
          </button>
        </Panel>

        {/* Top-Right Sync & Export Action Panel */}
        <Panel position="top-right" className="flex items-center gap-2 bg-slate-900/90 p-2 rounded-xl border border-slate-800 shadow-xl backdrop-blur">
          {syncSuccess && (
            <div className="flex items-center gap-1 text-emerald-400 text-xs font-medium px-2 py-1 bg-emerald-500/10 rounded-lg border border-emerald-500/20 animate-in fade-in">
              <CheckCircle2 className="w-3.5 h-3.5" />
              <span>{syncSuccess}</span>
            </div>
          )}

          {/* High-Res Canvas Export Button */}
          <ExportButton />

          {codePath && (
            <button
              type="button"
              onClick={handleSync}
              disabled={isSyncing || !codePath}
              className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-bold bg-gradient-to-r from-emerald-600 to-teal-600 hover:from-emerald-500 hover:to-teal-500 disabled:opacity-40 disabled:cursor-not-allowed text-white rounded-xl shadow-lg shadow-emerald-950 transition"
              title="Save and synchronize visual canvas changes back to HCL (.tf) code"
            >
              <UploadCloud className="w-3.5 h-3.5" />
              <span>{isSyncing ? 'Syncing AST...' : 'Sync Code'}</span>
            </button>
          )}
        </Panel>
      </ReactFlow>

      {/* Floating Centered Canvas Legend */}
      <div className="absolute bottom-4 left-1/2 -translate-x-1/2 z-20 pointer-events-none">
        <div className="bg-slate-900/95 px-3 py-1.5 rounded-xl border border-slate-800 shadow-xl backdrop-blur flex items-center gap-3 text-[10px] font-mono pointer-events-auto">
          <span className="flex items-center gap-1 text-emerald-400">
            <span className="w-2 h-2 rounded-full bg-emerald-500"></span> IN_SYNC / CREATE
          </span>
          <span className="flex items-center gap-1 text-amber-400">
            <span className="w-2 h-2 rounded-full bg-amber-500"></span> MODIFIED
          </span>
          <span className="flex items-center gap-1 text-rose-400">
            <span className="w-2 h-2 rounded-full bg-rose-500"></span> DESTROY / DRIFT
          </span>
        </div>
      </div>

      {/* Slide-over Resource Inspector Drawer */}
      <InspectorDrawer
        selectedNode={selectedNode}
        allNodes={nodes}
        allEdges={edges}
        onClose={() => setSelectedNode(null)}
        onSelectNode={handleSelectNodeById}
      />

      {/* Slide-over Drift Summary Drawer */}
      <DriftDrawer
        isOpen={isDriftDrawerOpen}
        onClose={onCloseDriftDrawer}
        nodes={nodes}
        onSelectNode={handleSelectNodeById}
      />

      {/* Multi-Cloud Add Resource Modal */}
      <AddResourceModal
        isOpen={isAddModalOpen}
        onClose={() => setIsAddModalOpen(false)}
        onAddResource={handleAddResourceFromCatalog}
      />
    </div>
  );
};

export const GraphCanvas: React.FC<GraphCanvasProps> = (props) => (
  <ReactFlowProvider>
    <GraphCanvasInner {...props} />
  </ReactFlowProvider>
);
