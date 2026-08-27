import React, { useEffect } from 'react';
import { Node, Edge } from '@xyflow/react';
import { NodeData } from '../types/graph';
import {
  X,
  Layers,
  Server,
  Database,
  Cloud,
  Network,
  HardDrive,
  Shield,
  ShieldAlert,
  Box,
  Globe,
  ArrowUpRight,
  ArrowDownLeft,
  FileCode,
  Tag,
  Link,
} from 'lucide-react';

interface InspectorDrawerProps {
  selectedNode: Node<NodeData> | null;
  allNodes: Node<NodeData>[];
  allEdges: Edge[];
  onClose: () => void;
  onSelectNode: (nodeId: string) => void;
}

const getResourceIcon = (resourceType?: string) => {
  if (!resourceType) return <Cloud className="w-4 h-4 text-indigo-400" />;
  const rt = resourceType.toLowerCase();
  if (rt === 'module') return <Box className="w-4 h-4 text-purple-400" />;
  if (rt === 'aws_availability_zone') return <Globe className="w-4 h-4 text-amber-400" />;
  if (rt.includes('vpc') || rt.includes('subnet') || rt.includes('network') || rt.includes('gateway')) {
    return <Network className="w-4 h-4 text-sky-400" />;
  }
  if (rt.includes('db') || rt.includes('dynamo') || rt.includes('rds') || rt.includes('sql') || rt.includes('table')) {
    return <Database className="w-4 h-4 text-purple-400" />;
  }
  if (rt.includes('eks') || rt.includes('instance') || rt.includes('cluster') || rt.includes('node_group')) {
    return <Server className="w-4 h-4 text-emerald-400" />;
  }
  if (rt.includes('s3') || rt.includes('bucket') || rt.includes('storage')) {
    return <HardDrive className="w-4 h-4 text-amber-400" />;
  }
  if (rt.includes('security') || rt.includes('iam') || rt.includes('firewall')) {
    return <Shield className="w-4 h-4 text-rose-400" />;
  }
  return <Cloud className="w-4 h-4 text-indigo-400" />;
};

export const InspectorDrawer: React.FC<InspectorDrawerProps> = ({
  selectedNode,
  allNodes,
  allEdges,
  onClose,
  onSelectNode,
}) => {
  // Handle Escape key to close drawer
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onClose();
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [onClose]);

  if (!selectedNode) return null;

  const nodeData = selectedNode.data;
  const resourceType = nodeData?.resourceType || 'Resource';
  const label = nodeData?.label || selectedNode.id;
  const provider = nodeData?.provider || 'aws';
  const driftStatus = nodeData?.driftStatus || 'IN_SYNC';
  const attributes = nodeData?.attributes || {};
  const securityIssues = nodeData?.securityIssues || [];

  // Relationships
  const parentNode = selectedNode.parentId
    ? allNodes.find((n) => n.id === selectedNode.parentId)
    : null;
  const childNodes = allNodes.filter((n) => n.parentId === selectedNode.id);
  const incomingEdges = allEdges.filter((e) => e.target === selectedNode.id);
  const outgoingEdges = allEdges.filter((e) => e.source === selectedNode.id);

  const getStatusBadge = () => {
    switch (driftStatus) {
      case 'CREATE':
        return 'bg-emerald-500/20 text-emerald-300 border-emerald-500/40';
      case 'DESTROY':
        return 'bg-rose-500/20 text-rose-300 border-rose-500/40';
      case 'MODIFIED':
        return 'bg-amber-500/20 text-amber-300 border-amber-500/40';
      case 'MISSING_IN_STATE':
        return 'bg-blue-500/20 text-blue-300 border-blue-500/40';
      case 'MISSING_IN_CODE':
        return 'bg-rose-500/20 text-rose-300 border-rose-500/40';
      default:
        return 'bg-slate-800 text-slate-300 border-slate-700';
    }
  };

  return (
    <aside className="fixed top-[52px] right-0 bottom-0 w-[380px] bg-slate-950/95 border-l border-slate-800 shadow-2xl backdrop-blur-xl z-50 flex flex-col animate-in slide-in-from-right duration-200 text-slate-200">
      {/* 1. Header */}
      <div className="p-4 border-b border-slate-800 flex items-start justify-between gap-3 bg-slate-900/60">
        <div className="flex items-start gap-2.5 min-w-0">
          <div className="p-2 rounded-xl bg-slate-800/80 border border-slate-700/80 text-sky-400 shrink-0 mt-0.5">
            {getResourceIcon(resourceType)}
          </div>
          <div className="min-w-0">
            <div className="flex items-center gap-1.5">
              <span className="text-[10px] font-mono text-slate-400 uppercase tracking-wider">
                {provider}
              </span>
              <span
                className={`text-[8px] font-mono font-bold px-1.5 py-0.2 rounded border uppercase ${getStatusBadge()}`}
              >
                {driftStatus}
              </span>
            </div>
            <h3 className="text-sm font-bold text-white font-mono truncate mt-0.5 leading-tight">
              {label}
            </h3>
            <div className="text-[10px] font-mono text-slate-400 truncate mt-0.5">
              {resourceType}
            </div>
          </div>
        </div>

        <button
          type="button"
          onClick={onClose}
          className="p-1 rounded-lg hover:bg-slate-800 text-slate-400 hover:text-white transition shrink-0"
          title="Close Inspector (Esc)"
        >
          <X className="w-4 h-4" />
        </button>
      </div>

      {/* Body: Scrollable sections */}
      <div className="flex-1 overflow-y-auto p-4 space-y-5 text-xs">
        {/* Security Alerts (if any) */}
        {securityIssues.length > 0 && (
          <div className="space-y-2 bg-rose-950/30 border border-rose-500/40 rounded-xl p-3">
            <div className="flex items-center justify-between text-rose-300 font-bold text-xs">
              <span className="flex items-center gap-1.5">
                <ShieldAlert className="w-4 h-4 text-rose-400" />
                Security Alerts ({securityIssues.length})
              </span>
            </div>
            <div className="space-y-2 mt-1 max-h-48 overflow-y-auto">
              {securityIssues.map((issue, idx) => (
                <div key={idx} className="bg-slate-900/90 rounded-lg p-2 border border-rose-500/30 space-y-1">
                  <div className="flex items-center justify-between text-[10px]">
                    <span className="font-mono text-rose-400 font-bold">{issue.ruleId}</span>
                    <span className="px-1.5 py-0.2 rounded text-[8px] font-bold bg-rose-500/20 text-rose-300 border border-rose-500/40">
                      {issue.severity}
                    </span>
                  </div>
                  <div className="text-slate-200 font-medium leading-snug">{issue.title}</div>
                  {issue.description && (
                    <div className="text-[10px] text-slate-400 leading-relaxed">{issue.description}</div>
                  )}
                </div>
              ))}
            </div>
          </div>
        )}

        {/* 2. Configuration Attributes Table */}
        <div className="space-y-2">
          <h4 className="text-[11px] font-bold text-slate-400 uppercase tracking-wider flex items-center gap-1.5">
            <Tag className="w-3.5 h-3.5 text-sky-400" />
            Configuration Attributes
          </h4>
          {Object.keys(attributes).length === 0 ? (
            <div className="text-slate-500 italic text-[11px] py-1">No attributes declared</div>
          ) : (
            <div className="bg-slate-900/80 border border-slate-800 rounded-xl overflow-hidden divide-y divide-slate-800/60 font-mono text-[11px]">
              {Object.entries(attributes).map(([k, v]) => (
                <div key={k} className="p-2 flex flex-col gap-0.5">
                  <span className="text-slate-400 text-[10px]">{k}</span>
                  <span className="text-slate-200 break-words">
                    {typeof v === 'object' ? JSON.stringify(v, null, 1) : String(v)}
                  </span>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* 3. Relationships & Topology Hierarchy */}
        <div className="space-y-2">
          <h4 className="text-[11px] font-bold text-slate-400 uppercase tracking-wider flex items-center gap-1.5">
            <Link className="w-3.5 h-3.5 text-indigo-400" />
            Relationships &amp; Topology
          </h4>
          <div className="space-y-2 font-mono text-[11px]">
            {/* Parent Container */}
            {parentNode && (
              <div className="bg-slate-900/80 border border-slate-800 rounded-xl p-2.5 space-y-1">
                <span className="text-[10px] text-slate-400 uppercase tracking-wider flex items-center gap-1">
                  <Layers className="w-3 h-3 text-sky-400" /> Parent Container:
                </span>
                <button
                  type="button"
                  onClick={() => onSelectNode(parentNode.id)}
                  className="w-full text-left font-bold text-sky-300 hover:text-sky-200 hover:underline truncate"
                >
                  {parentNode.data?.label || parentNode.id}
                </button>
              </div>
            )}

            {/* Child Resources */}
            {childNodes.length > 0 && (
              <div className="bg-slate-900/80 border border-slate-800 rounded-xl p-2.5 space-y-1.5">
                <span className="text-[10px] text-slate-400 uppercase tracking-wider">
                  Contains ({childNodes.length} resources):
                </span>
                <div className="space-y-1 max-h-32 overflow-y-auto">
                  {childNodes.map((child) => (
                    <button
                      key={child.id}
                      type="button"
                      onClick={() => onSelectNode(child.id)}
                      className="w-full text-left text-slate-300 hover:text-white hover:underline truncate block"
                    >
                      • {child.data?.label || child.id}
                    </button>
                  ))}
                </div>
              </div>
            )}

            {/* Incoming Dependencies */}
            {incomingEdges.length > 0 && (
              <div className="bg-slate-900/80 border border-slate-800 rounded-xl p-2.5 space-y-1">
                <span className="text-[10px] text-slate-400 uppercase tracking-wider flex items-center gap-1">
                  <ArrowDownLeft className="w-3 h-3 text-emerald-400" /> Depended on by:
                </span>
                <div className="space-y-1">
                  {incomingEdges.map((e) => (
                    <button
                      key={e.id}
                      type="button"
                      onClick={() => onSelectNode(e.source)}
                      className="w-full text-left text-emerald-300 hover:underline truncate block"
                    >
                      ← {e.source}
                    </button>
                  ))}
                </div>
              </div>
            )}

            {/* Outgoing Dependencies */}
            {outgoingEdges.length > 0 && (
              <div className="bg-slate-900/80 border border-slate-800 rounded-xl p-2.5 space-y-1">
                <span className="text-[10px] text-slate-400 uppercase tracking-wider flex items-center gap-1">
                  <ArrowUpRight className="w-3 h-3 text-purple-400" /> Depends on:
                </span>
                <div className="space-y-1">
                  {outgoingEdges.map((e) => (
                    <button
                      key={e.id}
                      type="button"
                      onClick={() => onSelectNode(e.target)}
                      className="w-full text-left text-purple-300 hover:underline truncate block"
                    >
                      → {e.target}
                    </button>
                  ))}
                </div>
              </div>
            )}
          </div>
        </div>

        {/* 4. Origin / Module info */}
        <div className="space-y-1.5 pt-2 border-t border-slate-800/80 text-[10px] font-mono text-slate-400">
          <div className="flex items-center gap-1.5">
            <FileCode className="w-3.5 h-3.5 text-slate-400" />
            <span className="text-slate-400">Resource Address:</span>
          </div>
          <div className="bg-slate-900 px-2 py-1 rounded border border-slate-800 text-slate-300 truncate">
            {selectedNode.id}
          </div>
          {nodeData?.module && (
            <div className="text-slate-400">
              Module: <span className="text-slate-300">{nodeData.module}</span>
            </div>
          )}
        </div>
      </div>
    </aside>
  );
};
