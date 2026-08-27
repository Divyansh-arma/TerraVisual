import React, { useState } from 'react';
import { GraphResponse, DriftStatus } from '../types/graph';
import { Code, Check, Copy, Network, ListTree, ArrowRight, Box } from 'lucide-react';

interface JsonPreviewProps {
  graph: GraphResponse | null;
}

export const JsonPreview: React.FC<JsonPreviewProps> = ({ graph }) => {
  const [activeTab, setActiveTab] = useState<'json' | 'nodes' | 'edges'>('json');
  const [copied, setCopied] = useState(false);

  if (!graph) {
    return (
      <div className="bg-slate-900/60 border border-dashed border-slate-800 rounded-2xl p-12 text-center text-slate-500">
        <Network className="w-12 h-12 mx-auto mb-3 text-slate-700 animate-pulse" />
        <h3 className="text-sm font-semibold text-slate-400 mb-1">No Graph Loaded</h3>
        <p className="text-xs text-slate-600 max-w-sm mx-auto">
          Specify a Terraform state file or HCL directory above and click Parse to inspect the React Flow graph contract.
        </p>
      </div>
    );
  }

  const jsonString = JSON.stringify(graph, null, 2);

  const handleCopy = () => {
    navigator.clipboard.writeText(jsonString);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const getDriftBadge = (status: DriftStatus) => {
    switch (status) {
      case 'IN_SYNC':
        return <span className="px-2 py-0.5 text-[10px] font-bold rounded bg-emerald-500/20 text-emerald-300 border border-emerald-500/30">IN_SYNC</span>;
      case 'MODIFIED':
        return <span className="px-2 py-0.5 text-[10px] font-bold rounded bg-amber-500/20 text-amber-300 border border-amber-500/30">MODIFIED</span>;
      case 'MISSING_IN_STATE':
        return <span className="px-2 py-0.5 text-[10px] font-bold rounded bg-blue-500/20 text-blue-300 border border-blue-500/30">MISSING_IN_STATE</span>;
      case 'MISSING_IN_CODE':
        return <span className="px-2 py-0.5 text-[10px] font-bold rounded bg-rose-500/20 text-rose-300 border border-rose-500/30">MISSING_IN_CODE</span>;
      default:
        return <span className="px-2 py-0.5 text-[10px] font-bold rounded bg-slate-700 text-slate-300">UNKNOWN</span>;
    }
  };

  return (
    <div className="bg-slate-900/90 border border-slate-800 rounded-2xl overflow-hidden shadow-2xl flex flex-col">
      {/* Header Tabs */}
      <div className="bg-slate-950 px-4 py-3 border-b border-slate-800 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={() => setActiveTab('json')}
            className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-semibold transition ${
              activeTab === 'json' ? 'bg-slate-800 text-white' : 'text-slate-400 hover:text-slate-200'
            }`}
          >
            <Code className="w-3.5 h-3.5" />
            Standard JSON
          </button>
          <button
            type="button"
            onClick={() => setActiveTab('nodes')}
            className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-semibold transition ${
              activeTab === 'nodes' ? 'bg-slate-800 text-white' : 'text-slate-400 hover:text-slate-200'
            }`}
          >
            <Box className="w-3.5 h-3.5" />
            Nodes ({graph.nodes.length})
          </button>
          <button
            type="button"
            onClick={() => setActiveTab('edges')}
            className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-semibold transition ${
              activeTab === 'edges' ? 'bg-slate-800 text-white' : 'text-slate-400 hover:text-slate-200'
            }`}
          >
            <ListTree className="w-3.5 h-3.5" />
            Edges ({graph.edges.length})
          </button>
        </div>

        <button
          type="button"
          onClick={handleCopy}
          className="flex items-center gap-1.5 px-3 py-1.5 bg-slate-800 hover:bg-slate-700 text-slate-300 hover:text-white rounded-lg text-xs font-medium transition border border-slate-700"
        >
          {copied ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
          {copied ? 'Copied' : 'Copy JSON'}
        </button>
      </div>

      {/* Content Area */}
      <div className="p-4 max-h-[500px] overflow-auto">
        {activeTab === 'json' && (
          <pre className="text-xs font-mono text-emerald-400/90 leading-relaxed bg-slate-950 p-4 rounded-xl border border-slate-800/80 overflow-x-auto selection:bg-emerald-900 selection:text-emerald-100">
            {jsonString}
          </pre>
        )}

        {activeTab === 'nodes' && (
          <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
            {graph.nodes.map((node) => (
              <div
                key={node.id}
                className="bg-slate-950/80 border border-slate-800/90 rounded-xl p-3.5 space-y-2 hover:border-slate-700 transition"
              >
                <div className="flex items-start justify-between gap-2">
                  <div>
                    <div className="text-xs font-bold text-white font-mono">{node.id}</div>
                    <div className="text-[11px] text-slate-400">
                      Type: <span className="text-sky-400 font-mono">{node.data.resourceType}</span> | Provider: <span className="text-amber-400">{node.data.provider}</span>
                    </div>
                  </div>
                  {getDriftBadge(node.data.driftStatus)}
                </div>

                {node.data.attributes && Object.keys(node.data.attributes).length > 0 && (
                  <div className="pt-2 border-t border-slate-800/60">
                    <div className="text-[10px] uppercase font-semibold text-slate-500 mb-1">Key Attributes:</div>
                    <div className="bg-slate-900/90 p-2 rounded-lg text-[11px] font-mono text-slate-300 space-y-1">
                      {Object.entries(node.data.attributes).slice(0, 4).map(([k, v]) => (
                        <div key={k} className="flex items-baseline justify-between gap-2 truncate">
                          <span className="text-slate-500">{k}:</span>
                          <span className="text-slate-200 truncate">{typeof v === 'object' ? JSON.stringify(v) : String(v)}</span>
                        </div>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            ))}
          </div>
        )}

        {activeTab === 'edges' && (
          <div className="space-y-2">
            {graph.edges.length === 0 ? (
              <div className="text-xs text-slate-500 text-center py-6">No dependency edges found in this graph.</div>
            ) : (
              graph.edges.map((edge) => (
                <div
                  key={edge.id}
                  className="bg-slate-950/80 border border-slate-800 rounded-xl p-3 flex items-center justify-between text-xs font-mono"
                >
                  <div className="flex items-center gap-3">
                    <span className="text-sky-400 font-semibold">{edge.source}</span>
                    <ArrowRight className="w-4 h-4 text-slate-500" />
                    <span className="text-emerald-400 font-semibold">{edge.target}</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="text-[10px] text-slate-500 bg-slate-900 px-2 py-0.5 rounded border border-slate-800">
                      {edge.type}
                    </span>
                    {edge.animated && (
                      <span className="text-[10px] text-indigo-400 bg-indigo-950/50 px-2 py-0.5 rounded border border-indigo-900">
                        animated
                      </span>
                    )}
                  </div>
                </div>
              ))
            )}
          </div>
        )}
      </div>
    </div>
  );
};
