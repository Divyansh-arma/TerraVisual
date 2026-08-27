import React from 'react';
import { GraphResponse } from '../types/graph';
import { Layers, GitFork, CheckCircle2, AlertCircle, PlusCircle, MinusCircle } from 'lucide-react';

interface StatsSummaryProps {
  graph: GraphResponse | null;
}

export const StatsSummary: React.FC<StatsSummaryProps> = ({ graph }) => {
  if (!graph) return null;

  const totalNodes = graph.nodes.length;
  const totalEdges = graph.edges.length;

  const inSyncCount = graph.nodes.filter(n => n.data.driftStatus === 'IN_SYNC').length;
  const modifiedCount = graph.nodes.filter(n => n.data.driftStatus === 'MODIFIED').length;
  const missingInStateCount = graph.nodes.filter(n => n.data.driftStatus === 'MISSING_IN_STATE').length;
  const missingInCodeCount = graph.nodes.filter(n => n.data.driftStatus === 'MISSING_IN_CODE').length;

  return (
    <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-3">
      <div className="bg-slate-900/80 border border-slate-800 rounded-xl p-3 flex items-center gap-3">
        <div className="p-2 bg-indigo-500/10 text-indigo-400 rounded-lg">
          <Layers className="w-5 h-5" />
        </div>
        <div>
          <div className="text-xs text-slate-400 font-medium uppercase tracking-wider">Nodes</div>
          <div className="text-xl font-bold text-white">{totalNodes}</div>
        </div>
      </div>

      <div className="bg-slate-900/80 border border-slate-800 rounded-xl p-3 flex items-center gap-3">
        <div className="p-2 bg-sky-500/10 text-sky-400 rounded-lg">
          <GitFork className="w-5 h-5" />
        </div>
        <div>
          <div className="text-xs text-slate-400 font-medium uppercase tracking-wider">Edges</div>
          <div className="text-xl font-bold text-white">{totalEdges}</div>
        </div>
      </div>

      <div className="bg-slate-900/80 border border-slate-800 rounded-xl p-3 flex items-center gap-3">
        <div className="p-2 bg-emerald-500/10 text-emerald-400 rounded-lg">
          <CheckCircle2 className="w-5 h-5" />
        </div>
        <div>
          <div className="text-xs text-slate-400 font-medium uppercase tracking-wider">In Sync</div>
          <div className="text-xl font-bold text-emerald-400">{inSyncCount}</div>
        </div>
      </div>

      <div className="bg-slate-900/80 border border-slate-800 rounded-xl p-3 flex items-center gap-3">
        <div className="p-2 bg-amber-500/10 text-amber-400 rounded-lg">
          <AlertCircle className="w-5 h-5" />
        </div>
        <div>
          <div className="text-xs text-slate-400 font-medium uppercase tracking-wider">Modified</div>
          <div className="text-xl font-bold text-amber-400">{modifiedCount}</div>
        </div>
      </div>

      <div className="bg-slate-900/80 border border-slate-800 rounded-xl p-3 flex items-center gap-3">
        <div className="p-2 bg-blue-500/10 text-blue-400 rounded-lg">
          <PlusCircle className="w-5 h-5" />
        </div>
        <div>
          <div className="text-xs text-slate-400 font-medium uppercase tracking-wider leading-tight">No State</div>
          <div className="text-xl font-bold text-blue-400">{missingInStateCount}</div>
        </div>
      </div>

      <div className="bg-slate-900/80 border border-slate-800 rounded-xl p-3 flex items-center gap-3">
        <div className="p-2 bg-rose-500/10 text-rose-400 rounded-lg">
          <MinusCircle className="w-5 h-5" />
        </div>
        <div>
          <div className="text-xs text-slate-400 font-medium uppercase tracking-wider leading-tight">No Code</div>
          <div className="text-xl font-bold text-rose-400">{missingInCodeCount}</div>
        </div>
      </div>
    </div>
  );
};
