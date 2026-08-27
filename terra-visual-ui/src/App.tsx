import React, { useState } from 'react';
import { Toolbar } from './components/Toolbar';
import { GraphCanvas } from './components/GraphCanvas';
import { JsonPreview } from './components/JsonPreview';
import { ErrorBoundary } from './components/ErrorBoundary';
import { parseStateFile, parseCodeDir, parsePlanFile, runDriftAnalysis, syncGraphToCode } from './services/tauriBridge';
import { GraphResponse } from './types/graph';
import { AlertTriangle, X, Code2 } from 'lucide-react';

export const App: React.FC = () => {
  const [statePath, setStatePath] = useState<string>('');
  const [codePath, setCodePath] = useState<string>('');
  const [planPath, setPlanPath] = useState<string>('');
  const [graph, setGraph] = useState<GraphResponse | null>(null);
  const [isLoading, setIsLoading] = useState<boolean>(false);
  const [isSyncing, setIsSyncing] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);
  const [showJsonInspector, setShowJsonInspector] = useState<boolean>(false);
  const [isDriftDrawerOpen, setIsDriftDrawerOpen] = useState<boolean>(false);

  const handleSelectPreset = (preset: 'basic' | 'drift' | 'plan') => {
    setError(null);
    if (preset === 'basic') {
      setStatePath('../terra-parser/testdata/mock_tfstate.json');
      setCodePath('../terra-parser/testdata/mock_tf');
      setPlanPath('');
    } else if (preset === 'drift') {
      setStatePath('../terra-parser/testdata/drift_test/mock_state.json');
      setCodePath('../terra-parser/testdata/drift_test/code');
      setPlanPath('');
    } else {
      setPlanPath('../terra-parser/testdata/mock_tfplan.json');
      setStatePath('');
      setCodePath('');
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

  const handleParsePlan = async () => {
    if (!planPath) return;
    setIsLoading(true);
    setError(null);
    try {
      const res = await parsePlanFile(planPath);
      setGraph(res);
    } catch (err: any) {
      setError(err?.message || 'Failed to parse plan file');
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
      setIsDriftDrawerOpen(true);
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

  const handleResetAll = () => {
    setStatePath('');
    setCodePath('');
    setPlanPath('');
    setGraph(null);
    setError(null);
    setIsDriftDrawerOpen(false);
  };

  return (
    <div className="h-screen w-screen bg-slate-950 text-slate-100 flex flex-col overflow-hidden">
      {/* 1. Slim Compact Top Bar (Height ~52px) */}
      <Toolbar
        statePath={statePath}
        setStatePath={setStatePath}
        codePath={codePath}
        setCodePath={setCodePath}
        planPath={planPath}
        setPlanPath={setPlanPath}
        onParseState={handleParseState}
        onParseCode={handleParseCode}
        onParsePlan={handleParsePlan}
        onRunDrift={handleRunDrift}
        onResetAll={handleResetAll}
        onOpenDriftSummary={() => setIsDriftDrawerOpen(true)}
        isLoading={isLoading}
        graph={graph}
        onSelectPreset={handleSelectPreset}
      />

      {/* Error Alert Banner */}
      {error && (
        <div className="px-4 py-2 bg-rose-950/90 border-b border-rose-500/40 text-rose-200 text-xs flex items-center justify-between z-40 animate-in fade-in">
          <div className="flex items-center gap-2">
            <AlertTriangle className="w-4 h-4 text-rose-400 shrink-0" />
            <span>{error}</span>
          </div>
          <button
            type="button"
            onClick={() => setError(null)}
            className="p-1 text-rose-400 hover:text-white transition"
          >
            <X className="w-4 h-4" />
          </button>
        </div>
      )}

      {/* 2. Maximized Viewport Canvas Area (>85% vertical screen space) */}
      <main className="flex-1 w-full relative overflow-hidden p-2">
        <ErrorBoundary fallbackTitle="Canvas Rendering Issue">
          <GraphCanvas
            graph={graph}
            codePath={codePath}
            onSyncToCode={handleSyncToCode}
            isSyncing={isSyncing}
            isDriftDrawerOpen={isDriftDrawerOpen}
            onCloseDriftDrawer={() => setIsDriftDrawerOpen(false)}
          />
        </ErrorBoundary>

        {/* Floating JSON Contract Inspector toggle button at bottom right */}
        {graph && (
          <button
            type="button"
            onClick={() => setShowJsonInspector(!showJsonInspector)}
            className="absolute bottom-4 right-4 z-30 flex items-center gap-1.5 px-3 py-1.5 text-xs font-semibold bg-slate-900/90 hover:bg-slate-800 text-slate-300 hover:text-white rounded-xl border border-slate-700/80 shadow-xl backdrop-blur transition"
          >
            <Code2 className="w-3.5 h-3.5 text-sky-400" />
            <span>{showJsonInspector ? 'Hide JSON Contract' : 'Inspect JSON'}</span>
          </button>
        )}
      </main>

      {/* JSON Inspector Modal / Slide-over */}
      {showJsonInspector && (
        <div className="fixed inset-x-4 bottom-4 top-16 bg-slate-950/95 border border-slate-800 rounded-2xl shadow-2xl backdrop-blur-xl z-50 p-4 flex flex-col animate-in fade-in zoom-in-95">
          <div className="flex items-center justify-between border-b border-slate-800 pb-2 mb-3">
            <h3 className="text-sm font-bold text-white flex items-center gap-2">
              <Code2 className="w-4 h-4 text-sky-400" />
              Terraform Visual Graph JSON Contract
            </h3>
            <button
              type="button"
              onClick={() => setShowJsonInspector(false)}
              className="p-1 text-slate-400 hover:text-white rounded-lg bg-slate-900 border border-slate-800"
            >
              <X className="w-4 h-4" />
            </button>
          </div>
          <div className="flex-1 overflow-auto">
            <JsonPreview graph={graph} />
          </div>
        </div>
      )}
    </div>
  );
};
