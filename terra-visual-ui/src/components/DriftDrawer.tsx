import React, { useState, useEffect } from 'react';
import { Node } from '@xyflow/react';
import { NodeData } from '../types/graph';
import {
  X,
  Sparkles,
  CheckCircle2,
  MinusCircle,
  PlusCircle,
  AlertCircle,
  Database,
  Server,
  Cloud,
  Network,
  HardDrive,
  Shield,
  Box,
  ChevronLeft,
  ChevronRight,
  ArrowUpRight,
} from 'lucide-react';

interface DriftDrawerProps {
  isOpen: boolean;
  onClose: () => void;
  nodes: Node<NodeData>[];
  onSelectNode: (nodeId: string) => void;
}

type DriftSection = 'overview' | 'divergence' | 'unmanaged' | 'unapplied' | 'item_detail';

const getResourceIcon = (resourceType?: string) => {
  if (!resourceType) return <Cloud className="w-3.5 h-3.5 text-indigo-400" />;
  const rt = resourceType.toLowerCase();
  if (rt === 'module') return <Box className="w-3.5 h-3.5 text-purple-400" />;
  if (rt.includes('vpc') || rt.includes('subnet') || rt.includes('network') || rt.includes('gateway')) {
    return <Network className="w-3.5 h-3.5 text-sky-400" />;
  }
  if (rt.includes('db') || rt.includes('dynamo') || rt.includes('rds') || rt.includes('sql') || rt.includes('table')) {
    return <Database className="w-3.5 h-3.5 text-purple-400" />;
  }
  if (rt.includes('eks') || rt.includes('instance') || rt.includes('cluster') || rt.includes('node_group')) {
    return <Server className="w-3.5 h-3.5 text-emerald-400" />;
  }
  if (rt.includes('s3') || rt.includes('bucket') || rt.includes('storage')) {
    return <HardDrive className="w-3.5 h-3.5 text-amber-400" />;
  }
  if (rt.includes('security') || rt.includes('iam') || rt.includes('firewall')) {
    return <Shield className="w-3.5 h-3.5 text-rose-400" />;
  }
  return <Cloud className="w-3.5 h-3.5 text-indigo-400" />;
};

export const DriftDrawer: React.FC<DriftDrawerProps> = ({
  isOpen,
  onClose,
  nodes,
  onSelectNode,
}) => {
  const [selectedSection, setSelectedSection] = useState<DriftSection>('overview');
  const [selectedItemId, setSelectedItemId] = useState<string | null>(null);

  // Reset navigation when drawer reopens
  useEffect(() => {
    if (isOpen) {
      setSelectedSection('overview');
      setSelectedItemId(null);
    }
  }, [isOpen]);

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        if (selectedSection !== 'overview') {
          setSelectedSection('overview');
          setSelectedItemId(null);
        } else {
          onClose();
        }
      }
    };
    if (isOpen) {
      window.addEventListener('keydown', handleKeyDown);
    }
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [isOpen, onClose, selectedSection]);

  if (!isOpen) return null;

  const totalNodes = nodes.length;
  const inSyncNodes = nodes.filter((n) => n.data?.driftStatus === 'IN_SYNC');
  const modifiedNodes = nodes.filter((n) => n.data?.driftStatus === 'MODIFIED');
  const missingInCodeNodes = nodes.filter(
    (n) => n.data?.driftStatus === 'MISSING_IN_CODE' || n.data?.driftStatus === 'DESTROY'
  );
  const missingInStateNodes = nodes.filter(
    (n) => n.data?.driftStatus === 'MISSING_IN_STATE' || n.data?.driftStatus === 'CREATE'
  );

  const selectedNode = selectedItemId ? nodes.find((n) => n.id === selectedItemId) : null;

  const getSectionTitle = () => {
    switch (selectedSection) {
      case 'divergence':
        return 'Attribute Divergence';
      case 'unmanaged':
        return 'Unmanaged in Code';
      case 'unapplied':
        return 'Unapplied Declarations';
      case 'item_detail':
        return selectedNode?.data?.label || selectedNode?.id || 'Resource Detail';
      default:
        return 'Overview';
    }
  };

  return (
    <aside className="fixed top-[52px] right-0 bottom-0 w-[420px] bg-slate-950/98 border-l border-slate-800 shadow-2xl backdrop-blur-2xl z-50 flex flex-col animate-in slide-in-from-right duration-200 text-slate-200">
      {/* 1. Header with Breadcrumb & Back Navigation */}
      <div className="p-4 border-b border-slate-800 flex items-center justify-between gap-3 bg-slate-900/60 shrink-0">
        <div className="flex items-center gap-2 min-w-0">
          {selectedSection !== 'overview' ? (
            <button
              type="button"
              onClick={() => {
                setSelectedSection('overview');
                setSelectedItemId(null);
              }}
              className="p-1 rounded-lg bg-slate-800 hover:bg-slate-700 text-sky-400 hover:text-white transition flex items-center gap-1 text-xs font-semibold shrink-0"
              title="Back to Overview"
            >
              <ChevronLeft className="w-4 h-4" />
              <span>Back</span>
            </button>
          ) : (
            <div className="p-2 rounded-xl bg-gradient-to-br from-indigo-500/20 to-purple-500/20 border border-indigo-500/40 text-indigo-400 shrink-0">
              <Sparkles className="w-4 h-4" />
            </div>
          )}

          <div className="min-w-0">
            {/* Breadcrumb path */}
            <div className="flex items-center gap-1 text-[10px] font-mono text-slate-400 truncate">
              <span
                onClick={() => {
                  setSelectedSection('overview');
                  setSelectedItemId(null);
                }}
                className={`cursor-pointer hover:text-white transition ${
                  selectedSection === 'overview' ? 'text-sky-400 font-bold' : ''
                }`}
              >
                Drift Summary
              </span>
              {selectedSection !== 'overview' && (
                <>
                  <span>/</span>
                  <span className="text-slate-200 truncate">{getSectionTitle()}</span>
                </>
              )}
            </div>
            <h3 className="text-sm font-bold text-white tracking-tight truncate mt-0.5">
              {selectedSection === 'overview' ? 'Drift Reconciliation Summary' : getSectionTitle()}
            </h3>
          </div>
        </div>

        <button
          type="button"
          onClick={onClose}
          className="p-1 rounded-lg hover:bg-slate-800 text-slate-400 hover:text-white transition shrink-0"
          title="Close Drawer (Esc)"
        >
          <X className="w-4 h-4" />
        </button>
      </div>

      {/* 2. Top Summary Metric Cards (Interactive click to drill down) */}
      <div className="p-4 border-b border-slate-800/80 bg-slate-950/50 grid grid-cols-4 gap-2 shrink-0 text-center font-mono">
        <button
          type="button"
          onClick={() => {
            setSelectedSection('overview');
            setSelectedItemId(null);
          }}
          className={`p-2 rounded-xl border transition ${
            selectedSection === 'overview'
              ? 'bg-slate-800 border-sky-500/50 ring-1 ring-sky-400'
              : 'bg-slate-900/80 border-slate-800 hover:border-slate-700'
          }`}
        >
          <div className="text-[9px] text-slate-400 uppercase tracking-wider">Total</div>
          <div className="text-sm font-bold text-white mt-0.5">{totalNodes}</div>
        </button>

        <button
          type="button"
          onClick={() => {
            setSelectedSection('overview');
            setSelectedItemId(null);
          }}
          className="p-2 rounded-xl bg-emerald-950/30 border border-emerald-500/30 hover:border-emerald-500/60 transition"
        >
          <div className="text-[9px] text-emerald-400 uppercase tracking-wider">In Sync</div>
          <div className="text-sm font-bold text-emerald-300 mt-0.5">{inSyncNodes.length}</div>
        </button>

        <button
          type="button"
          onClick={() => {
            setSelectedSection('divergence');
            setSelectedItemId(null);
          }}
          className={`p-2 rounded-xl border transition ${
            selectedSection === 'divergence'
              ? 'bg-amber-950/60 border-amber-500 ring-1 ring-amber-400'
              : 'bg-amber-950/30 border-amber-500/30 hover:border-amber-500/60'
          }`}
        >
          <div className="text-[9px] text-amber-400 uppercase tracking-wider">Modified</div>
          <div className="text-sm font-bold text-amber-300 mt-0.5">{modifiedNodes.length}</div>
        </button>

        <button
          type="button"
          onClick={() => {
            setSelectedSection('unmanaged');
            setSelectedItemId(null);
          }}
          className={`p-2 rounded-xl border transition ${
            selectedSection === 'unmanaged' || selectedSection === 'unapplied'
              ? 'bg-rose-950/60 border-rose-500 ring-1 ring-rose-400'
              : 'bg-rose-950/30 border-rose-500/30 hover:border-rose-500/60'
          }`}
        >
          <div className="text-[9px] text-rose-400 uppercase tracking-wider">Drifted</div>
          <div className="text-sm font-bold text-rose-300 mt-0.5">
            {missingInCodeNodes.length + missingInStateNodes.length}
          </div>
        </button>
      </div>

      {/* 3. Section Content based on selectedSection */}
      <div className="flex-1 overflow-y-auto p-4 space-y-5 text-xs">
        {/* VIEW: Overview Mode */}
        {selectedSection === 'overview' && (
          <>
            {/* Modified preview */}
            {modifiedNodes.length > 0 && (
              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <h4 className="text-[11px] font-bold text-amber-400 uppercase tracking-wider flex items-center gap-1.5 font-mono">
                    <AlertCircle className="w-3.5 h-3.5" />
                    Attribute Divergence ({modifiedNodes.length})
                  </h4>
                  <button
                    type="button"
                    onClick={() => setSelectedSection('divergence')}
                    className="text-[10px] text-amber-300 hover:underline flex items-center gap-0.5"
                  >
                    View All <ChevronRight className="w-3 h-3" />
                  </button>
                </div>

                <div className="space-y-2">
                  {modifiedNodes.slice(0, 3).map((node) => (
                    <div
                      key={node.id}
                      onClick={() => {
                        setSelectedItemId(node.id);
                        setSelectedSection('item_detail');
                        onSelectNode(node.id);
                      }}
                      className="bg-slate-900/90 border border-amber-500/40 hover:border-amber-400 rounded-xl p-2.5 transition cursor-pointer space-y-1.5"
                    >
                      <div className="flex items-center justify-between gap-2">
                        <div className="flex items-center gap-1.5 min-w-0">
                          {getResourceIcon(node.data?.resourceType)}
                          <span className="font-bold text-slate-100 font-mono truncate">
                            {node.data?.label || node.id}
                          </span>
                        </div>
                        <span className="px-1.5 py-0.2 rounded text-[8px] font-mono font-bold bg-amber-500/20 text-amber-300 border border-amber-500/40">
                          MODIFIED
                        </span>
                      </div>
                      <div className="text-[10px] text-slate-400 font-mono truncate">{node.id}</div>
                      {node.data?.driftDiffs && node.data.driftDiffs.length > 0 && (
                        <div className="text-[10px] text-amber-300/90 font-mono">
                          {node.data.driftDiffs.length} attribute field(s) diverged
                        </div>
                      )}
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Unmanaged preview */}
            {missingInCodeNodes.length > 0 && (
              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <h4 className="text-[11px] font-bold text-rose-400 uppercase tracking-wider flex items-center gap-1.5 font-mono">
                    <MinusCircle className="w-3.5 h-3.5" />
                    Unmanaged in Code ({missingInCodeNodes.length})
                  </h4>
                  <button
                    type="button"
                    onClick={() => setSelectedSection('unmanaged')}
                    className="text-[10px] text-rose-300 hover:underline flex items-center gap-0.5"
                  >
                    View All <ChevronRight className="w-3 h-3" />
                  </button>
                </div>

                <div className="space-y-2">
                  {missingInCodeNodes.slice(0, 2).map((node) => (
                    <div
                      key={node.id}
                      onClick={() => onSelectNode(node.id)}
                      className="bg-rose-950/20 border border-rose-500/30 hover:border-rose-400/60 rounded-xl p-2.5 space-y-1 transition cursor-pointer"
                    >
                      <div className="flex items-center justify-between gap-2">
                        <div className="flex items-center gap-1.5 min-w-0">
                          {getResourceIcon(node.data?.resourceType)}
                          <span className="font-bold text-rose-200 font-mono truncate">
                            {node.data?.label || node.id}
                          </span>
                        </div>
                        <span className="px-1.5 py-0.2 rounded text-[8px] font-mono font-bold bg-rose-500/20 text-rose-300 border border-rose-500/40">
                          MISSING IN CODE
                        </span>
                      </div>
                      <div className="text-[10px] text-slate-400 font-mono">{node.id}</div>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Unapplied preview */}
            {missingInStateNodes.length > 0 && (
              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <h4 className="text-[11px] font-bold text-sky-400 uppercase tracking-wider flex items-center gap-1.5 font-mono">
                    <PlusCircle className="w-3.5 h-3.5" />
                    Unapplied in State ({missingInStateNodes.length})
                  </h4>
                  <button
                    type="button"
                    onClick={() => setSelectedSection('unapplied')}
                    className="text-[10px] text-sky-300 hover:underline flex items-center gap-0.5"
                  >
                    View All <ChevronRight className="w-3 h-3" />
                  </button>
                </div>

                <div className="space-y-2">
                  {missingInStateNodes.slice(0, 2).map((node) => (
                    <div
                      key={node.id}
                      onClick={() => onSelectNode(node.id)}
                      className="bg-sky-950/20 border border-sky-500/30 hover:border-sky-400/60 rounded-xl p-2.5 space-y-1 transition cursor-pointer"
                    >
                      <div className="flex items-center justify-between gap-2">
                        <div className="flex items-center gap-1.5 min-w-0">
                          {getResourceIcon(node.data?.resourceType)}
                          <span className="font-bold text-sky-200 font-mono truncate">
                            {node.data?.label || node.id}
                          </span>
                        </div>
                        <span className="px-1.5 py-0.2 rounded text-[8px] font-mono font-bold bg-sky-500/20 text-sky-300 border border-sky-500/40">
                          MISSING IN STATE
                        </span>
                      </div>
                      <div className="text-[10px] text-slate-400 font-mono">{node.id}</div>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* All In Sync Notice */}
            {modifiedNodes.length === 0 && missingInCodeNodes.length === 0 && missingInStateNodes.length === 0 && (
              <div className="p-6 text-center space-y-2 bg-emerald-950/20 border border-emerald-500/30 rounded-2xl">
                <CheckCircle2 className="w-8 h-8 text-emerald-400 mx-auto" />
                <h4 className="font-bold text-emerald-300">100% Infrastructure In Sync</h4>
                <p className="text-[11px] text-slate-400">
                  All {totalNodes} resources match perfectly across HCL code and Terraform state.
                </p>
              </div>
            )}
          </>
        )}

        {/* VIEW: Attribute Divergence Full List */}
        {selectedSection === 'divergence' && (
          <div className="space-y-3">
            <p className="text-[11px] text-slate-400">
              Resources where declared configuration in .tf code differs from active cloud state.
            </p>
            {modifiedNodes.map((node) => {
              const diffs = node.data?.driftDiffs || [];
              return (
                <div
                  key={node.id}
                  className="bg-slate-900/90 border border-amber-500/40 rounded-xl p-3 space-y-2 hover:border-amber-400 transition cursor-pointer"
                  onClick={() => onSelectNode(node.id)}
                >
                  <div className="flex items-center justify-between gap-2">
                    <div className="flex items-center gap-1.5 min-w-0">
                      {getResourceIcon(node.data?.resourceType)}
                      <span className="font-bold text-slate-100 font-mono truncate">
                        {node.data?.label || node.id}
                      </span>
                    </div>
                    <button
                      type="button"
                      onClick={(e) => {
                        e.stopPropagation();
                        onSelectNode(node.id);
                      }}
                      className="p-1 rounded bg-slate-800 hover:bg-slate-700 text-sky-400"
                      title="Focus on Canvas"
                    >
                      <ArrowUpRight className="w-3.5 h-3.5" />
                    </button>
                  </div>

                  <div className="text-[10px] text-slate-400 font-mono">{node.id}</div>

                  {diffs.length > 0 ? (
                    <div className="rounded-lg overflow-hidden border border-slate-800 bg-slate-950 font-mono text-[10px]">
                      <div className="grid grid-cols-3 bg-slate-900/90 p-1.5 text-[9px] font-bold text-slate-400 border-b border-slate-800">
                        <span>Field</span>
                        <span>Code (.tf)</span>
                        <span>State</span>
                      </div>
                      <div className="divide-y divide-slate-800/60">
                        {diffs.map((d, i) => (
                          <div key={i} className="grid grid-cols-3 p-1.5 items-center gap-1">
                            <span className="text-amber-400 font-semibold truncate">{d.field}</span>
                            <span className="text-sky-300 truncate">
                              {typeof d.codeValue === 'object' ? JSON.stringify(d.codeValue) : String(d.codeValue)}
                            </span>
                            <span className="text-slate-400 truncate">
                              {typeof d.stateValue === 'object' ? JSON.stringify(d.stateValue) : String(d.stateValue)}
                            </span>
                          </div>
                        ))}
                      </div>
                    </div>
                  ) : (
                    <div className="text-[10px] text-slate-400 italic">
                      Dependencies or configuration mismatch detected.
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        )}

        {/* VIEW: Unmanaged in Code Full List */}
        {selectedSection === 'unmanaged' && (
          <div className="space-y-3">
            <p className="text-[11px] text-slate-400">
              Resources present in Terraform state or provisioned out-of-band that have no corresponding HCL block.
            </p>
            {missingInCodeNodes.map((node) => (
              <div
                key={node.id}
                onClick={() => onSelectNode(node.id)}
                className="bg-rose-950/20 border border-rose-500/30 hover:border-rose-400/60 rounded-xl p-3 space-y-1.5 transition cursor-pointer"
              >
                <div className="flex items-center justify-between gap-2">
                  <div className="flex items-center gap-1.5 min-w-0">
                    {getResourceIcon(node.data?.resourceType)}
                    <span className="font-bold text-rose-200 font-mono truncate">
                      {node.data?.label || node.id}
                    </span>
                  </div>
                  <button
                    type="button"
                    onClick={(e) => {
                      e.stopPropagation();
                      onSelectNode(node.id);
                    }}
                    className="p-1 rounded bg-slate-800 hover:bg-slate-700 text-sky-400"
                    title="Focus on Canvas"
                  >
                    <ArrowUpRight className="w-3.5 h-3.5" />
                  </button>
                </div>
                <div className="text-[10px] text-slate-400 font-mono">{node.id}</div>
                <div className="text-[10px] text-rose-300/80">
                  Exists in State / Cloud, but missing from .tf files.
                </div>
              </div>
            ))}
          </div>
        )}

        {/* VIEW: Unapplied Full List */}
        {selectedSection === 'unapplied' && (
          <div className="space-y-3">
            <p className="text-[11px] text-slate-400">
              Resources defined in HCL code that have not yet been created in cloud state.
            </p>
            {missingInStateNodes.map((node) => (
              <div
                key={node.id}
                onClick={() => onSelectNode(node.id)}
                className="bg-sky-950/20 border border-sky-500/30 hover:border-sky-400/60 rounded-xl p-3 space-y-1.5 transition cursor-pointer"
              >
                <div className="flex items-center justify-between gap-2">
                  <div className="flex items-center gap-1.5 min-w-0">
                    {getResourceIcon(node.data?.resourceType)}
                    <span className="font-bold text-sky-200 font-mono truncate">
                      {node.data?.label || node.id}
                    </span>
                  </div>
                  <button
                    type="button"
                    onClick={(e) => {
                      e.stopPropagation();
                      onSelectNode(node.id);
                    }}
                    className="p-1 rounded bg-slate-800 hover:bg-slate-700 text-sky-400"
                    title="Focus on Canvas"
                  >
                    <ArrowUpRight className="w-3.5 h-3.5" />
                  </button>
                </div>
                <div className="text-[10px] text-slate-400 font-mono">{node.id}</div>
                <div className="text-[10px] text-sky-300/80">
                  Declared in HCL code, pending `terraform apply`.
                </div>
              </div>
            ))}
          </div>
        )}

        {/* VIEW: Individual Item Detail */}
        {selectedSection === 'item_detail' && selectedNode && (
          <div className="space-y-4">
            <div className="bg-slate-900/90 border border-slate-800 rounded-xl p-3 space-y-2">
              <div className="flex items-center justify-between">
                <span className="font-mono text-[10px] text-slate-400 uppercase">
                  {selectedNode.data?.provider} &middot; {selectedNode.data?.resourceType}
                </span>
                <span className="px-1.5 py-0.2 rounded text-[8px] font-mono font-bold bg-amber-500/20 text-amber-300 border border-amber-500/40">
                  {selectedNode.data?.driftStatus}
                </span>
              </div>
              <h4 className="text-sm font-bold text-white font-mono">
                {selectedNode.data?.label || selectedNode.id}
              </h4>
              <div className="text-[10px] text-slate-400 font-mono break-all">{selectedNode.id}</div>
            </div>

            {/* Full diff breakdown */}
            {selectedNode.data?.driftDiffs && selectedNode.data.driftDiffs.length > 0 && (
              <div className="space-y-2">
                <h5 className="text-[11px] font-bold text-amber-400 uppercase tracking-wider font-mono">
                  Attribute Diffs ({selectedNode.data.driftDiffs.length})
                </h5>
                <div className="rounded-xl overflow-hidden border border-slate-800 bg-slate-950 font-mono text-[11px]">
                  <div className="grid grid-cols-3 bg-slate-900 p-2 text-[10px] font-bold text-slate-400 border-b border-slate-800">
                    <span>Field</span>
                    <span>Code (.tf)</span>
                    <span>State</span>
                  </div>
                  <div className="divide-y divide-slate-800/80">
                    {selectedNode.data.driftDiffs.map((d, i) => (
                      <div key={i} className="grid grid-cols-3 p-2 items-center gap-1 text-[10px]">
                        <span className="text-amber-400 font-bold truncate">{d.field}</span>
                        <span className="text-sky-300 truncate">
                          {typeof d.codeValue === 'object' ? JSON.stringify(d.codeValue) : String(d.codeValue)}
                        </span>
                        <span className="text-slate-400 truncate">
                          {typeof d.stateValue === 'object' ? JSON.stringify(d.stateValue) : String(d.stateValue)}
                        </span>
                      </div>
                    ))}
                  </div>
                </div>
              </div>
            )}
          </div>
        )}
      </div>
    </aside>
  );
};
