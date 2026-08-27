import React from 'react';
import { open } from '@tauri-apps/plugin-dialog';
import { FileCode2, FolderGit2, RefreshCw, Play, Sparkles, AlertTriangle, FolderOpen, FileSpreadsheet } from 'lucide-react';

interface ControlPanelProps {
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
  isLoading: boolean;
  error: string | null;
  onSelectPreset: (preset: 'basic' | 'drift' | 'plan') => void;
}

export const ControlPanel: React.FC<ControlPanelProps> = ({
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
  isLoading,
  error,
  onSelectPreset,
}) => {
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
      console.error('Failed to open native state file dialog:', err);
      const entered = window.prompt('Enter absolute path to terraform.tfstate:');
      if (entered) setStatePath(entered);
    }
  };

  const handleSelectCodeDirectory = async () => {
    try {
      const selected = await open({
        directory: true,
      });
      if (selected) {
        const path = Array.isArray(selected) ? selected[0] : selected;
        if (path) setCodePath(path);
      }
    } catch (err) {
      console.error('Failed to open native code directory dialog:', err);
      const entered = window.prompt('Enter absolute path to Terraform HCL directory:');
      if (entered) setCodePath(entered);
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
      console.error('Failed to open native plan file dialog:', err);
      const entered = window.prompt('Enter absolute path to tfplan.json:');
      if (entered) setPlanPath(entered);
    }
  };

  return (
    <div className="bg-slate-900/90 border border-slate-800 rounded-2xl p-5 shadow-xl backdrop-blur">
      <div className="flex flex-wrap items-center justify-between gap-4 pb-4 border-b border-slate-800">
        <div>
          <h2 className="text-lg font-semibold text-white flex items-center gap-2">
            <Sparkles className="w-5 h-5 text-sky-400" />
            Infrastructure Control Panel
          </h2>
          <p className="text-xs text-slate-400">
            Ingest local Terraform state, plan diffs, or HCL code to generate visual topology
          </p>
        </div>

        {/* Presets */}
        <div className="flex items-center gap-2">
          <span className="text-xs text-slate-400 font-medium">Quick Presets:</span>
          <button
            type="button"
            onClick={() => onSelectPreset('basic')}
            className="px-2.5 py-1 text-xs font-medium bg-slate-800 hover:bg-slate-700 text-slate-200 rounded-lg transition"
          >
            Basic AWS VPC
          </button>
          <button
            type="button"
            onClick={() => onSelectPreset('drift')}
            className="px-2.5 py-1 text-xs font-medium bg-indigo-950/60 hover:bg-indigo-900/80 text-indigo-300 border border-indigo-800/60 rounded-lg transition"
          >
            Drift Scenario
          </button>
          <button
            type="button"
            onClick={() => onSelectPreset('plan')}
            className="px-2.5 py-1 text-xs font-medium bg-emerald-950/60 hover:bg-emerald-900/80 text-emerald-300 border border-emerald-800/60 rounded-lg transition"
          >
            Plan (tfplan.json)
          </button>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 my-4">
        {/* State File Path */}
        <div className="space-y-1.5">
          <label className="text-xs font-semibold text-slate-300 flex items-center gap-1.5">
            <FileCode2 className="w-4 h-4 text-emerald-400" />
            State File (.tfstate)
          </label>
          <div className="flex gap-2">
            <input
              type="text"
              value={statePath}
              onChange={(e) => setStatePath(e.target.value)}
              placeholder="/path/to/terraform.tfstate"
              className="flex-1 bg-slate-950 border border-slate-800 rounded-lg px-3 py-2 text-xs font-mono text-slate-200 placeholder-slate-600 focus:outline-none focus:border-sky-500 transition"
            />
            <button
              type="button"
              onClick={handleSelectStateFile}
              className="flex items-center gap-1 px-2.5 py-2 text-xs font-medium bg-slate-800 hover:bg-slate-700 text-slate-300 hover:text-white rounded-lg border border-slate-700 transition shrink-0"
              title="Select State File"
            >
              <FolderOpen className="w-3.5 h-3.5" />
            </button>
          </div>
        </div>

        {/* Code Directory Path */}
        <div className="space-y-1.5">
          <label className="text-xs font-semibold text-slate-300 flex items-center gap-1.5">
            <FolderGit2 className="w-4 h-4 text-sky-400" />
            HCL Directory (.tf)
          </label>
          <div className="flex gap-2">
            <input
              type="text"
              value={codePath}
              onChange={(e) => setCodePath(e.target.value)}
              placeholder="/path/to/terraform/code"
              className="flex-1 bg-slate-950 border border-slate-800 rounded-lg px-3 py-2 text-xs font-mono text-slate-200 placeholder-slate-600 focus:outline-none focus:border-sky-500 transition"
            />
            <button
              type="button"
              onClick={handleSelectCodeDirectory}
              className="flex items-center gap-1 px-2.5 py-2 text-xs font-medium bg-slate-800 hover:bg-slate-700 text-slate-300 hover:text-white rounded-lg border border-slate-700 transition shrink-0"
              title="Select Code Directory"
            >
              <FolderOpen className="w-3.5 h-3.5" />
            </button>
          </div>
        </div>

        {/* Plan File Path */}
        <div className="space-y-1.5">
          <label className="text-xs font-semibold text-slate-300 flex items-center gap-1.5">
            <FileSpreadsheet className="w-4 h-4 text-amber-400" />
            Plan File (tfplan.json)
          </label>
          <div className="flex gap-2">
            <input
              type="text"
              value={planPath}
              onChange={(e) => setPlanPath(e.target.value)}
              placeholder="/path/to/tfplan.json"
              className="flex-1 bg-slate-950 border border-slate-800 rounded-lg px-3 py-2 text-xs font-mono text-slate-200 placeholder-slate-600 focus:outline-none focus:border-sky-500 transition"
            />
            <button
              type="button"
              onClick={handleSelectPlanFile}
              className="flex items-center gap-1 px-2.5 py-2 text-xs font-medium bg-slate-800 hover:bg-slate-700 text-slate-300 hover:text-white rounded-lg border border-slate-700 transition shrink-0"
              title="Select Plan JSON File"
            >
              <FolderOpen className="w-3.5 h-3.5" />
            </button>
          </div>
        </div>
      </div>

      {/* Action Buttons */}
      <div className="flex flex-wrap items-center gap-3 pt-2">
        <button
          type="button"
          onClick={onParseState}
          disabled={isLoading || !statePath}
          className="flex items-center gap-2 px-4 py-2 text-xs font-semibold bg-emerald-600 hover:bg-emerald-500 disabled:opacity-50 disabled:cursor-not-allowed text-white rounded-lg transition shadow-lg shadow-emerald-950"
        >
          {isLoading ? <RefreshCw className="w-3.5 h-3.5 animate-spin" /> : <Play className="w-3.5 h-3.5 fill-current" />}
          <span>{isLoading ? 'Parsing State...' : 'Parse State File'}</span>
        </button>

        <button
          type="button"
          onClick={onParseCode}
          disabled={isLoading || !codePath}
          className="flex items-center gap-2 px-4 py-2 text-xs font-semibold bg-sky-600 hover:bg-sky-500 disabled:opacity-50 disabled:cursor-not-allowed text-white rounded-lg transition shadow-lg shadow-sky-950"
        >
          {isLoading ? <RefreshCw className="w-3.5 h-3.5 animate-spin" /> : <Play className="w-3.5 h-3.5 fill-current" />}
          <span>{isLoading ? 'Parsing HCL...' : 'Parse HCL Code'}</span>
        </button>

        <button
          type="button"
          onClick={onParsePlan}
          disabled={isLoading || !planPath}
          className="flex items-center gap-2 px-4 py-2 text-xs font-semibold bg-amber-600 hover:bg-amber-500 disabled:opacity-50 disabled:cursor-not-allowed text-white rounded-lg transition shadow-lg shadow-amber-950"
        >
          {isLoading ? <RefreshCw className="w-3.5 h-3.5 animate-spin" /> : <Play className="w-3.5 h-3.5 fill-current" />}
          <span>{isLoading ? 'Parsing Plan...' : 'Parse Plan (tfplan.json)'}</span>
        </button>

        <button
          type="button"
          onClick={onRunDrift}
          disabled={isLoading || !statePath || !codePath}
          className="flex items-center gap-2 px-4 py-2 text-xs font-semibold bg-gradient-to-r from-indigo-600 to-purple-600 hover:from-indigo-500 hover:to-purple-500 disabled:opacity-50 disabled:cursor-not-allowed text-white rounded-lg transition shadow-lg shadow-indigo-950"
        >
          {isLoading ? <RefreshCw className="w-3.5 h-3.5 animate-spin" /> : <Sparkles className="w-3.5 h-3.5" />}
          <span>{isLoading ? 'Analyzing Drift...' : 'Run Drift Analysis'}</span>
        </button>
      </div>

      {/* Error display */}
      {error && (
        <div className="mt-4 p-3 bg-rose-500/10 border border-rose-500/20 text-rose-300 rounded-lg text-xs flex items-start gap-2">
          <AlertTriangle className="w-4 h-4 text-rose-400 shrink-0 mt-0.5" />
          <span>{error}</span>
        </div>
      )}
    </div>
  );
};
