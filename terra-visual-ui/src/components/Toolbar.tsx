import React, { useEffect } from 'react';
import { open } from '@tauri-apps/plugin-dialog';
import { Logo } from './Logo';
import { GraphResponse } from '../types/graph';
import {
  FolderGit2,
  FileCode2,
  FileSpreadsheet,
  Play,
  Sparkles,
  RefreshCw,
  Layers,
  GitFork,
  CheckCircle2,
  AlertCircle,
  PlusCircle,
  MinusCircle,
  X,
  RotateCcw,
} from 'lucide-react';

interface ToolbarProps {
  statePath: string;
  setStatePath: (path: string) => void;
  codePath: string;
  setCodePath: (path: string) => void;
  planPath: string;
  setPlanPath: (path: string) => void;
  onParseState: () => void;
  onParseCode: () => void;
  onParsePlan: () => void;
  onRunDrift: () => void;
  onResetAll: () => void;
  onOpenDriftSummary?: () => void;
  isLoading: boolean;
  graph: GraphResponse | null;
  onSelectPreset: (preset: 'basic' | 'drift' | 'plan') => void;
}

export const Toolbar: React.FC<ToolbarProps> = ({
  statePath,
  setStatePath,
  codePath,
  setCodePath,
  planPath,
  setPlanPath,
  onParseState,
  onParseCode,
  onParsePlan,
  onRunDrift,
  onResetAll,
  onOpenDriftSummary,
  isLoading,
  graph,
  onSelectPreset,
}) => {
  // Global shortcut: Cmd+K / Ctrl+K to Reset All
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault();
        onResetAll();
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [onResetAll]);

  const handleSelectCodeDirectory = async () => {
    try {
      const selected = await open({ directory: true });
      if (selected) {
        const path = Array.isArray(selected) ? selected[0] : selected;
        if (path) setCodePath(path);
      }
    } catch (err) {
      console.error('Failed to open native code dialog:', err);
      const entered = window.prompt('Enter path to Terraform HCL directory:');
      if (entered) setCodePath(entered);
    }
  };

  const handleSelectStateFile = async () => {
    try {
      const selected = await open({
        filters: [{ name: 'State', extensions: ['json', 'tfstate'] }],
      });
      if (selected) {
        const path = Array.isArray(selected) ? selected[0] : selected;
        if (path) setStatePath(path);
      }
    } catch (err) {
      console.error('Failed to open native state dialog:', err);
      const entered = window.prompt('Enter path to terraform.tfstate:');
      if (entered) setStatePath(entered);
    }
  };

  const handleSelectPlanFile = async () => {
    try {
      const selected = await open({
        filters: [{ name: 'Plan', extensions: ['json'] }],
      });
      if (selected) {
        const path = Array.isArray(selected) ? selected[0] : selected;
        if (path) setPlanPath(path);
      }
    } catch (err) {
      console.error('Failed to open native plan dialog:', err);
      const entered = window.prompt('Enter path to tfplan.json:');
      if (entered) setPlanPath(entered);
    }
  };

  const handleClearCode = (e: React.MouseEvent) => {
    e.stopPropagation();
    setCodePath('');
    if (!statePath && !planPath) {
      onResetAll();
    }
  };

  const handleClearState = (e: React.MouseEvent) => {
    e.stopPropagation();
    setStatePath('');
    if (!codePath && !planPath) {
      onResetAll();
    }
  };

  const handleClearPlan = (e: React.MouseEvent) => {
    e.stopPropagation();
    setPlanPath('');
    if (!codePath && !statePath) {
      onResetAll();
    }
  };

  // Metrics calculations
  const totalNodes = graph?.nodes.length || 0;
  const totalEdges = graph?.edges.length || 0;
  const inSyncCount = graph?.nodes.filter((n) => n.data.driftStatus === 'IN_SYNC').length || 0;
  const modifiedCount = graph?.nodes.filter((n) => n.data.driftStatus === 'MODIFIED').length || 0;
  const createCount =
    graph?.nodes.filter(
      (n) => n.data.driftStatus === 'CREATE' || n.data.driftStatus === 'MISSING_IN_STATE'
    ).length || 0;
  const destroyCount =
    graph?.nodes.filter(
      (n) => n.data.driftStatus === 'DESTROY' || n.data.driftStatus === 'MISSING_IN_CODE'
    ).length || 0;

  const getBaseName = (p: string) => {
    if (!p) return '';
    const clean = p.replace(/\\/g, '/');
    const parts = clean.split('/').filter(Boolean);
    return parts.length > 0 ? parts[parts.length - 1] : p;
  };

  const hasAnyPath = Boolean(codePath || statePath || planPath || graph);

  return (
    <header className="border-b border-slate-800/90 bg-slate-950/95 backdrop-blur sticky top-0 z-40 px-4 py-2 flex flex-wrap items-center justify-between gap-3 min-h-[52px]">
      {/* Left: Brand Identity */}
      <div className="flex items-center gap-2.5 shrink-0">
        <Logo className="w-7 h-7" />
        <div className="flex items-center gap-1.5">
          <span className="font-bold text-sm text-white tracking-tight">Terra Visual</span>
          <span className="text-[10px] px-1.5 py-0.2 rounded-full bg-slate-800/80 text-sky-400 font-mono border border-slate-700">
            v1.0
          </span>
        </div>
      </div>

      {/* Middle: Compact File Picker Pills with Clear Buttons & Context-Aware Execution Controls */}
      <div className="flex flex-wrap items-center gap-2">
        {/* Open Code Pill */}
        <div
          onClick={handleSelectCodeDirectory}
          role="button"
          tabIndex={0}
          className={`group flex items-center gap-1.5 px-2.5 py-1 text-xs font-medium rounded-lg border transition cursor-pointer select-none ${
            codePath
              ? 'bg-sky-950/60 border-sky-500/50 text-sky-300 hover:bg-sky-900/60'
              : 'bg-slate-900/80 border-slate-800 text-slate-400 hover:text-slate-200 hover:border-slate-700'
          }`}
          title={codePath ? `Loaded: ${codePath} (Click to replace)` : 'Select Terraform HCL Code Directory'}
        >
          <FolderGit2 className="w-3.5 h-3.5 text-sky-400 shrink-0" />
          <span className="truncate max-w-[120px]">{codePath ? getBaseName(codePath) : 'Open Code'}</span>
          {codePath && (
            <button
              type="button"
              onClick={handleClearCode}
              className="p-0.5 ml-0.5 rounded hover:bg-sky-800/80 text-sky-400 hover:text-white transition shrink-0"
              title="Clear Code Path"
            >
              <X className="w-3 h-3" />
            </button>
          )}
        </div>

        {/* Open State Pill */}
        <div
          onClick={handleSelectStateFile}
          role="button"
          tabIndex={0}
          className={`group flex items-center gap-1.5 px-2.5 py-1 text-xs font-medium rounded-lg border transition cursor-pointer select-none ${
            statePath
              ? 'bg-emerald-950/60 border-emerald-500/50 text-emerald-300 hover:bg-emerald-900/60'
              : 'bg-slate-900/80 border-slate-800 text-slate-400 hover:text-slate-200 hover:border-slate-700'
          }`}
          title={statePath ? `Loaded: ${statePath} (Click to replace)` : 'Select Terraform State File (.tfstate)'}
        >
          <FileCode2 className="w-3.5 h-3.5 text-emerald-400 shrink-0" />
          <span className="truncate max-w-[120px]">{statePath ? getBaseName(statePath) : 'Open State'}</span>
          {statePath && (
            <button
              type="button"
              onClick={handleClearState}
              className="p-0.5 ml-0.5 rounded hover:bg-emerald-800/80 text-emerald-400 hover:text-white transition shrink-0"
              title="Clear State Path"
            >
              <X className="w-3 h-3" />
            </button>
          )}
        </div>

        {/* Open Plan Pill */}
        <div
          onClick={handleSelectPlanFile}
          role="button"
          tabIndex={0}
          className={`group flex items-center gap-1.5 px-2.5 py-1 text-xs font-medium rounded-lg border transition cursor-pointer select-none ${
            planPath
              ? 'bg-amber-950/60 border-amber-500/50 text-amber-300 hover:bg-amber-900/60'
              : 'bg-slate-900/80 border-slate-800 text-slate-400 hover:text-slate-200 hover:border-slate-700'
          }`}
          title={planPath ? `Loaded: ${planPath} (Click to replace)` : 'Select Terraform Plan File (tfplan.json)'}
        >
          <FileSpreadsheet className="w-3.5 h-3.5 text-amber-400 shrink-0" />
          <span className="truncate max-w-[120px]">{planPath ? getBaseName(planPath) : 'Open Plan'}</span>
          {planPath && (
            <button
              type="button"
              onClick={handleClearPlan}
              className="p-0.5 ml-0.5 rounded hover:bg-amber-800/80 text-amber-400 hover:text-white transition shrink-0"
              title="Clear Plan Path"
            >
              <X className="w-3 h-3" />
            </button>
          )}
        </div>

        {/* Reset All Button */}
        {hasAnyPath && (
          <button
            type="button"
            onClick={onResetAll}
            className="flex items-center gap-1 px-2 py-1 text-xs font-medium text-slate-400 hover:text-rose-300 hover:bg-rose-950/40 rounded-lg border border-transparent hover:border-rose-800/40 transition"
            title="Reset All Files & Canvas (⌘K)"
          >
            <RotateCcw className="w-3 h-3" />
            <span className="text-[11px] font-mono">Reset</span>
          </button>
        )}

        <div className="h-4 w-px bg-slate-800 mx-1" />

        {/* Multi-Action Execution Controls */}
        <div className="flex items-center gap-1.5">
          {/* Drift Mode Button (when both Code and State are loaded) */}
          {statePath && codePath && (
            <button
              type="button"
              onClick={onRunDrift}
              disabled={isLoading}
              className="flex items-center gap-1.5 px-3 py-1 text-xs font-semibold bg-gradient-to-r from-indigo-600 to-purple-600 hover:from-indigo-500 hover:to-purple-500 text-white rounded-lg transition shadow-md shadow-indigo-950 disabled:opacity-50"
              title="Run Drift Analysis comparing HCL Code against State"
            >
              {isLoading ? <RefreshCw className="w-3 h-3 animate-spin" /> : <Sparkles className="w-3 h-3" />}
              <span>Run Drift</span>
            </button>
          )}

          {/* Plan Parse Button */}
          {planPath && (
            <button
              type="button"
              onClick={onParsePlan}
              disabled={isLoading}
              className="flex items-center gap-1.5 px-3 py-1 text-xs font-semibold bg-amber-600 hover:bg-amber-500 text-white rounded-lg transition shadow-md shadow-amber-950 disabled:opacity-50"
              title="Parse Plan JSON diffs & topology"
            >
              {isLoading ? <RefreshCw className="w-3 h-3 animate-spin" /> : <Play className="w-3 h-3 fill-current" />}
              <span>Parse Plan</span>
            </button>
          )}

          {/* Code Parse Button (when code is loaded and not currently running drift exclusively) */}
          {codePath && (!statePath || !planPath) && (
            <button
              type="button"
              onClick={onParseCode}
              disabled={isLoading}
              className={`flex items-center gap-1.5 px-2.5 py-1 text-xs font-semibold text-white rounded-lg transition shadow-md shadow-sky-950 disabled:opacity-50 ${
                statePath ? 'bg-slate-800 hover:bg-slate-700 text-sky-300' : 'bg-sky-600 hover:bg-sky-500'
              }`}
              title="Parse HCL Code AST directory"
            >
              {isLoading ? <RefreshCw className="w-3 h-3 animate-spin" /> : <Play className="w-3 h-3 fill-current" />}
              <span>Parse Code</span>
            </button>
          )}

          {/* State Parse Button (when state is loaded and not currently running drift exclusively) */}
          {statePath && (!codePath || !planPath) && (
            <button
              type="button"
              onClick={onParseState}
              disabled={isLoading}
              className={`flex items-center gap-1.5 px-2.5 py-1 text-xs font-semibold text-white rounded-lg transition shadow-md shadow-emerald-950 disabled:opacity-50 ${
                codePath ? 'bg-slate-800 hover:bg-slate-700 text-emerald-300' : 'bg-emerald-600 hover:bg-emerald-500'
              }`}
              title="Parse Terraform State JSON"
            >
              {isLoading ? <RefreshCw className="w-3 h-3 animate-spin" /> : <Play className="w-3 h-3 fill-current" />}
              <span>Parse State</span>
            </button>
          )}

          {/* Dedicated Drift Summary Drawer Trigger */}
          {onOpenDriftSummary && graph && (
            <button
              type="button"
              onClick={onOpenDriftSummary}
              className="flex items-center gap-1.5 px-2.5 py-1 text-xs font-semibold bg-slate-900 hover:bg-slate-800 text-indigo-300 hover:text-indigo-200 border border-indigo-500/40 rounded-lg transition shadow-sm"
              title="View Categorized Drift Summary Drawer"
            >
              <Sparkles className="w-3.5 h-3.5 text-indigo-400" />
              <span>Drift Summary</span>
            </button>
          )}

          {/* Demo Presets when no paths are active */}
          {!codePath && !statePath && !planPath && (
            <div className="flex items-center gap-1.5">
              <span className="text-[11px] text-slate-500 font-mono mr-1">Presets:</span>
              <button
                type="button"
                onClick={() => onSelectPreset('basic')}
                className="px-2 py-0.8 text-[11px] font-medium bg-slate-900 hover:bg-slate-800 text-slate-300 rounded border border-slate-800 transition"
              >
                Demo VPC
              </button>
              <button
                type="button"
                onClick={() => onSelectPreset('drift')}
                className="px-2 py-0.8 text-[11px] font-medium bg-indigo-950/40 hover:bg-indigo-900/60 text-indigo-300 rounded border border-indigo-800/40 transition"
              >
                Demo Drift
              </button>
              <button
                type="button"
                onClick={() => onSelectPreset('plan')}
                className="px-2 py-0.8 text-[11px] font-medium bg-emerald-950/40 hover:bg-emerald-900/60 text-emerald-300 rounded border border-emerald-800/40 transition"
              >
                Demo Plan
              </button>
            </div>
          )}
        </div>
      </div>

      {/* Right: Inline Status Badge Strip */}
      {graph && (
        <div className="flex items-center gap-2 bg-slate-900/90 border border-slate-800 rounded-lg px-2.5 py-1 text-xs font-mono shrink-0">
          <div className="flex items-center gap-1.5 text-slate-300 font-semibold border-r border-slate-800 pr-2">
            <Layers className="w-3.5 h-3.5 text-indigo-400" />
            <span>{totalNodes}</span>
            <span className="text-slate-500">|</span>
            <GitFork className="w-3.5 h-3.5 text-sky-400" />
            <span>{totalEdges}</span>
          </div>

          <div className="flex items-center gap-2 text-[11px]">
            <span className="flex items-center gap-1 text-emerald-400 font-medium" title="In Sync">
              <CheckCircle2 className="w-3 h-3" />
              <span>{inSyncCount}</span>
            </span>

            {modifiedCount > 0 && (
              <span className="flex items-center gap-1 text-amber-400 font-medium" title="Modified">
                <AlertCircle className="w-3 h-3" />
                <span>{modifiedCount}</span>
              </span>
            )}

            {createCount > 0 && (
              <span className="flex items-center gap-1 text-emerald-400 font-medium" title="To Create">
                <PlusCircle className="w-3 h-3" />
                <span>{createCount}</span>
              </span>
            )}

            {destroyCount > 0 && (
              <span className="flex items-center gap-1 text-rose-400 font-medium" title="To Destroy">
                <MinusCircle className="w-3 h-3" />
                <span>{destroyCount}</span>
              </span>
            )}
          </div>
        </div>
      )}
    </header>
  );
};
