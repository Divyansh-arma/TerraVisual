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
import { ResourceTemplate } from '../constants/resourceTemplates';
import { getLayoutedElements } from '../utils/layout';
import {
  Maximize2,
  Minimize2,
  Layers,
  ArrowDownUp,
  ArrowLeftRight,
  Info,
  X,
  Search,
  PlusCircle,
  UploadCloud,
  Trash2,
  CheckCircle2,
  ShieldAlert,
} from 'lucide-react';

interface GraphCanvasProps {
  graph: GraphResponse | null;
  codePath: string;
  onSyncToCode: (currentGraph: GraphResponse) => Promise<void>;
  isSyncing: boolean;
}

const GraphCanvasInner: React.FC<GraphCanvasProps> = ({
  graph,
  codePath,
  onSyncToCode,
  isSyncing,
}) => {
  const [direction, setDirection] = useState<'TB' | 'LR'>('TB');
  const [selectedNode, setSelectedNode] = useState<Node<NodeData> | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [isFullScreen, setIsFullScreen] = useState(false);
  const [syncSuccess, setSyncSuccess] = useState<string | null>(null);
  const [isAddModalOpen, setIsAddModalOpen] = useState<boolean>(false);

  const { screenToFlowPosition, getViewport, fitView } = useReactFlow();

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

      // Drop near center of current viewport
      const viewport = getViewport();
      const position = screenToFlowPosition({
        x: window.innerWidth / 2 + (Math.random() * 80 - 40),
        y: window.innerHeight / 2 + (Math.random() * 80 - 40),
      }) || {
        x: (-viewport.x + 300) / viewport.zoom,
        y: (-viewport.y + 200) / viewport.zoom,
      };

      const newNode: Node<NodeData> = {
        id: newId,
        type: 'infrastructureNode',
        position,
        parentId,
        extent: parentId ? 'parent' : undefined,
        data: {
          label: customName,
          provider: template.provider,
          resourceType: template.resourceType,
          module: 'root',
          isDataSource: false,
          driftStatus: 'MISSING_IN_STATE',
          attributes,
        },
      };

      setNodes((nds) => [...nds, newNode]);
      setSelectedNode(newNode);
    },
    [getViewport, screenToFlowPosition, setNodes]
  );

  // Delete selected node
  const handleDeleteSelected = useCallback(() => {
    if (!selectedNode) return;
    setNodes((nds) => nds.filter((n) => n.id !== selectedNode.id));
    setEdges((eds) => eds.filter((e) => e.source !== selectedNode.id && e.target !== selectedNode.id));
    setSelectedNode(null);
  }, [selectedNode, setNodes, setEdges]);

  // Handle Sync to Code
  const handleSync = async () => {
    if (!codePath) return;

    const currentGraph: GraphResponse = {
      nodes: nodes.map((n) => ({
        id: n.id,
        type: n.type || 'infrastructureNode',
        position: n.position,
        data: n.data,
        parentId: n.parentId,
        extent: n.extent === 'parent' ? 'parent' : undefined,
      })),
      edges: edges.map((e) => ({
        id: e.id,
        source: e.source,
        target: e.target,
        type: e.type || 'smoothstep',
        animated: e.animated ?? true,
      })),
    };

    try {
      await onSyncToCode(currentGraph);
      setSyncSuccess('HCL code updated successfully!');
      setTimeout(() => setSyncSuccess(null), 4000);
    } catch {
      // Error handled upstream
    }
  };

  // Filtered nodes based on search query
  const displayedNodes = useMemo(() => {
    if (!searchQuery.trim()) return nodes;
    const q = searchQuery.toLowerCase();
    return nodes.map((node) => ({
      ...node,
      selected:
        node.id.toLowerCase().includes(q) ||
        node.data.label.toLowerCase().includes(q) ||
        node.data.resourceType.toLowerCase().includes(q),
    }));
  }, [nodes, searchQuery]);

  if (!graph || graph.nodes.length === 0) {
    return (
      <div className="w-full h-[550px] bg-slate-950 border border-slate-800/90 rounded-2xl flex flex-col items-center justify-center text-slate-500 p-8">
        <Layers className="w-12 h-12 mb-3 text-slate-700 animate-pulse" />
        <h3 className="text-base font-semibold text-slate-300 mb-1">Canvas Ready</h3>
        <p className="text-xs text-slate-500 text-center max-w-md">
          Load a Terraform state file or HCL directory above to visualize and edit the infrastructure graph.
        </p>
      </div>
    );
  }

  return (
    <div
      className={`relative w-full bg-slate-950 border border-slate-800/90 rounded-2xl overflow-hidden shadow-2xl transition-all ${
        isFullScreen ? 'fixed inset-4 z-50 h-[calc(100vh-2rem)]' : 'h-[620px]'
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
        <Controls className="!bg-slate-900 !border-slate-800 !text-slate-200 fill-current shadow-xl" />
        <MiniMap
          nodeColor={(node) => {
            const data = node.data as unknown as NodeData;
            if (data?.driftStatus === 'IN_SYNC') return '#10b981';
            if (data?.driftStatus === 'MODIFIED') return '#f59e0b';
            if (data?.driftStatus === 'MISSING_IN_STATE') return '#3b82f6';
            if (data?.driftStatus === 'MISSING_IN_CODE') return '#f43f5e';
            return '#64748b';
          }}
          className="!bg-slate-900/90 !border-slate-800 !rounded-xl overflow-hidden shadow-xl"
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

        {/* Top-Right Sync to Code Action Panel */}
        <Panel position="top-right" className="flex items-center gap-2 bg-slate-900/90 p-2 rounded-xl border border-slate-800 shadow-xl backdrop-blur">
          {syncSuccess && (
            <div className="flex items-center gap-1 text-emerald-400 text-xs font-medium px-2 py-1 bg-emerald-500/10 rounded-lg border border-emerald-500/20 animate-in fade-in">
              <CheckCircle2 className="w-3.5 h-3.5" />
              <span>{syncSuccess}</span>
            </div>
          )}

          <button
            type="button"
            onClick={handleSync}
            disabled={isSyncing || !codePath}
            className="flex items-center gap-2 px-3.5 py-1.5 text-xs font-bold bg-gradient-to-r from-emerald-600 to-teal-600 hover:from-emerald-500 hover:to-teal-500 disabled:opacity-40 disabled:cursor-not-allowed text-white rounded-lg shadow-lg shadow-emerald-950 transition"
            title="Save and synchronize visual canvas changes back to HCL (.tf) code"
          >
            <UploadCloud className="w-4 h-4" />
            <span>{isSyncing ? 'Syncing AST...' : 'Sync to Code'}</span>
          </button>
        </Panel>

        {/* Legend Panel */}
        <Panel position="bottom-left" className="bg-slate-900/90 px-3 py-2 rounded-xl border border-slate-800 shadow-xl backdrop-blur flex items-center gap-3 text-[11px] font-mono">
          <span className="flex items-center gap-1 text-emerald-400">
            <span className="w-2.5 h-2.5 rounded-full bg-emerald-500"></span> IN_SYNC
          </span>
          <span className="flex items-center gap-1 text-amber-400">
            <span className="w-2.5 h-2.5 rounded-full bg-amber-500"></span> MODIFIED
          </span>
          <span className="flex items-center gap-1 text-blue-400">
            <span className="w-2.5 h-2.5 rounded-full border border-dashed border-blue-400"></span> MISSING_STATE
          </span>
          <span className="flex items-center gap-1 text-rose-400">
            <span className="w-2.5 h-2.5 rounded-full border border-dashed border-rose-400"></span> MISSING_CODE
          </span>
        </Panel>
      </ReactFlow>

      {/* Selected Node Details Drawer */}
      {selectedNode && (
        <div className="absolute top-16 right-4 w-80 bg-slate-900/95 border border-slate-700 rounded-2xl p-4 shadow-2xl backdrop-blur z-50 space-y-3 animate-in fade-in duration-150">
          <div className="flex items-start justify-between gap-2 border-b border-slate-800 pb-3">
            <div>
              <div className="text-xs font-bold text-white font-mono">{selectedNode.id}</div>
              <div className="text-[11px] text-slate-400">
                {selectedNode.data.provider} &middot; {selectedNode.data.resourceType}
              </div>
            </div>
            <div className="flex items-center gap-1">
              <button
                type="button"
                onClick={handleDeleteSelected}
                className="p-1 text-rose-400 hover:text-rose-300 rounded-lg bg-rose-950/40 hover:bg-rose-900/50 border border-rose-900/40 transition"
                title="Delete this node from graph"
              >
                <Trash2 className="w-4 h-4" />
              </button>
              <button
                type="button"
                onClick={() => setSelectedNode(null)}
                className="p-1 text-slate-400 hover:text-white rounded-lg bg-slate-800 hover:bg-slate-700 transition"
              >
                <X className="w-4 h-4" />
              </button>
            </div>
          </div>

          <div className="space-y-2 text-xs">
            <div className="flex items-center justify-between">
              <span className="text-slate-400 font-medium">Drift Status:</span>
              <span className="font-bold font-mono text-sky-400">{selectedNode.data.driftStatus}</span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-slate-400 font-medium">Module:</span>
              <span className="font-mono text-slate-200">{selectedNode.data.module || 'root'}</span>
            </div>
          </div>

          {selectedNode.data.attributes && Object.keys(selectedNode.data.attributes).length > 0 && (
            <div className="space-y-1 pt-2 border-t border-slate-800">
              <div className="text-[10px] uppercase font-bold text-slate-400 flex items-center gap-1">
                <Info className="w-3 h-3" /> Attributes
              </div>
              <div className="bg-slate-950 p-2.5 rounded-xl border border-slate-800/80 max-h-48 overflow-auto text-[11px] font-mono space-y-1">
                {Object.entries(selectedNode.data.attributes).map(([k, v]) => (
                  <div key={k} className="flex flex-col gap-0.5 border-b border-slate-900 pb-1 last:border-0">
                    <span className="text-slate-500 font-semibold">{k}:</span>
                    <span className="text-slate-200 break-all">
                      {typeof v === 'object' ? JSON.stringify(v, null, 2) : String(v)}
                    </span>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Security Misconfigurations */}
          {selectedNode.data.securityIssues && selectedNode.data.securityIssues.length > 0 && (
            <div className="space-y-1.5 pt-2 border-t border-slate-800">
              <div className="text-[10px] uppercase font-bold text-rose-400 flex items-center justify-between">
                <span className="flex items-center gap-1">
                  <ShieldAlert className="w-3.5 h-3.5" /> Security Alerts
                </span>
                <span className="bg-rose-950 text-rose-300 px-1.5 py-0.2 rounded font-mono text-[9px] border border-rose-800/60">
                  {selectedNode.data.securityIssues.length}
                </span>
              </div>
              <div className="space-y-2 max-h-48 overflow-y-auto pr-1">
                {selectedNode.data.securityIssues.map((issue, idx) => (
                  <div key={idx} className="bg-rose-950/20 border border-rose-500/30 rounded-xl p-2 space-y-1 text-xs">
                    <div className="flex items-center justify-between">
                      <span className="text-[9px] uppercase font-mono px-1.5 py-0.2 rounded bg-rose-500/20 text-rose-300 font-bold border border-rose-500/30">
                        {issue.severity}
                      </span>
                      <span className="text-[10px] font-mono text-slate-400">{issue.ruleId}</span>
                    </div>
                    <div className="font-semibold text-slate-200 text-[11px] leading-snug">{issue.title}</div>
                    {issue.description && (
                      <p className="text-[10px] text-slate-400 leading-tight">{issue.description}</p>
                    )}
                  </div>
                ))}
              </div>
            </div>
          )}

          <div className="pt-2 text-[10px] text-slate-500 italic">
            Tip: Press <kbd className="px-1 py-0.5 bg-slate-800 rounded text-slate-300 font-mono">Delete</kbd> or <kbd className="px-1 py-0.5 bg-slate-800 rounded text-slate-300 font-mono">Backspace</kbd> to delete selected node.
          </div>
        </div>
      )}

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
