import React, { useState } from 'react';
import { ControlPanel } from './components/ControlPanel';
import { StatsSummary } from './components/StatsSummary';
import { GraphCanvas } from './components/GraphCanvas';
import { JsonPreview } from './components/JsonPreview';
import { ErrorBoundary } from './components/ErrorBoundary';
import { parseStateFile, parseCodeDir, runDriftAnalysis, syncGraphToCode } from './services/tauriBridge';
import { GraphResponse } from './types/graph';
import { Logo } from './components/Logo';
import { ShieldCheck, Code2, ChevronDown, ChevronUp } from 'lucide-react';

export const App: React.FC = () => {
  const [statePath, setStatePath] = useState<string>('');
  const [codePath, setCodePath] = useState<string>('');
  const [graph, setGraph] = useState<GraphResponse | null>(null);
  const [isLoading, setIsLoading] = useState<boolean>(false);
  const [isSyncing, setIsSyncing] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);
  const [showJsonInspector, setShowJsonInspector] = useState<boolean>(false);

  const handleSelectPreset = (preset: 'basic' | 'drift') => {
    setError(null);
    if (preset === 'basic') {
      setStatePath('../terra-parser/testdata/mock_tfstate.json');
      setCodePath('../terra-parser/testdata/mock_tf');
    } else {
      setStatePath('../terra-parser/testdata/drift_test/mock_state.json');
      setCodePath('../terra-parser/testdata/drift_test/code');
    }
  };

  const handleParseState = async () => {
    if (!statePath) return;
    setIsLoading(true);
    setError(null);
    try {
      const res = await parseStateFile(statePath);
      setGraph(res);
    } catch (err: any) {
      setError(err?.message || 'Failed to parse state file');
    } finally {
      setIsLoading(false);
    }
  };

  const handleParseCode = async () => {
    if (!codePath) return;
    setIsLoading(true);
    setError(null);
    try {
      const res = await parseCodeDir(codePath);
      setGraph(res);
    } catch (err: any) {
      setError(err?.message || 'Failed to parse code directory');
    } finally {
      setIsLoading(false);
    }
  };

  const handleRunDrift = async () => {
    if (!statePath || !codePath) return;
    setIsLoading(true);
    setError(null);
    try {
      const res = await runDriftAnalysis(statePath, codePath);
      setGraph(res);
    } catch (err: any) {
      setError(err?.message || 'Failed to run drift analysis');
    } finally {
      setIsLoading(false);
    }
  };

  const handleSyncToCode = async (currentGraph: GraphResponse) => {
    if (!codePath) {
      setError('Please specify a Terraform HCL Code Directory to sync to.');
      return;
    }
    setIsSyncing(true);
    setError(null);
    try {
      await syncGraphToCode(currentGraph, codePath);
      // Re-parse code or drift to refresh graph state
      if (statePath) {
        const refreshed = await runDriftAnalysis(statePath, codePath);
        setGraph(refreshed);
      } else {
        const refreshed = await parseCodeDir(codePath);
        setGraph(refreshed);
      }
    } catch (err: any) {
      setError(err?.message || 'Failed to sync graph to code');
      throw err;
    } finally {
      setIsSyncing(false);
    }
  };

  return (
    <div className="min-h-screen bg-slate-950 text-slate-100 flex flex-col">
      {/* Top Navigation */}
      <header className="border-b border-slate-800/80 bg-slate-950/80 backdrop-blur sticky top-0 z-50">
        <div className="max-w-7xl mx-auto px-6 h-14 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <Logo className="w-8 h-8 shrink-0" />
            <div>
              <span className="font-bold text-sm text-white tracking-tight">Terra Visual</span>
              <span className="text-[10px] ml-2 px-2 py-0.5 rounded-full bg-slate-800 text-sky-400 font-mono">v1.0.0</span>
            </div>
          </div>

          <div className="flex items-center gap-4 text-xs">
            <div className="flex items-center gap-1.5 text-emerald-400 bg-emerald-500/10 border border-emerald-500/20 px-2.5 py-1 rounded-full font-medium">
              <ShieldCheck className="w-3.5 h-3.5" />
              <span>Local-First &amp; Air-Gapped</span>
            </div>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="flex-1 max-w-7xl mx-auto w-full px-6 py-6 space-y-6">
        {/* Control Panel */}
        <ControlPanel
          statePath={statePath}
          setStatePath={setStatePath}
          codePath={codePath}
          setCodePath={setCodePath}
          onParseState={handleParseState}
          onParseCode={handleParseCode}
          onRunDrift={handleRunDrift}
          isLoading={isLoading}
          error={error}
          onSelectPreset={handleSelectPreset}
        />

        {/* Stats Summary */}
        <StatsSummary graph={graph} />

        {/* Interactive React Flow Canvas wrapped in ErrorBoundary */}
        <div className="space-y-2">
          <div className="flex items-center justify-between px-1">
            <h3 className="text-sm font-semibold text-slate-300 flex items-center gap-2">
              <span>Interactive Architecture Canvas</span>
            </h3>
            {graph && (
              <button
                type="button"
                onClick={() => setShowJsonInspector(!showJsonInspector)}
                className="flex items-center gap-1.5 text-xs text-slate-400 hover:text-slate-200 px-2.5 py-1 rounded-lg bg-slate-900 border border-slate-800 hover:bg-slate-800 transition"
              >
                <Code2 className="w-3.5 h-3.5 text-sky-400" />
                <span>{showJsonInspector ? 'Hide JSON Contract' : 'Inspect JSON Contract'}</span>
                {showJsonInspector ? <ChevronUp className="w-3.5 h-3.5" /> : <ChevronDown className="w-3.5 h-3.5" />}
              </button>
            )}
          </div>

          <ErrorBoundary fallbackTitle="Canvas Rendering Issue">
            <GraphCanvas
              graph={graph}
              codePath={codePath}
              onSyncToCode={handleSyncToCode}
              isSyncing={isSyncing}
            />
          </ErrorBoundary>
        </div>

        {/* Expandable JSON Inspector */}
        {showJsonInspector && (
          <div className="pt-2 animate-in fade-in duration-200">
            <JsonPreview graph={graph} />
          </div>
        )}
      </main>
    </div>
  );
};
