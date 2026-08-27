import { Component, ErrorInfo, ReactNode } from 'react';
import { AlertTriangle, RefreshCw, RotateCcw } from 'lucide-react';

interface Props {
  children: ReactNode;
  fallbackTitle?: string;
  isTopLevel?: boolean;
}

interface State {
  hasError: boolean;
  error: Error | null;
}

export class ErrorBoundary extends Component<Props, State> {
  public state: State = {
    hasError: false,
    error: null,
  };

  public static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error };
  }

  public componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    console.error('Terra Visual ErrorBoundary caught an error:', error, errorInfo);
  }

  public handleReset = () => {
    if (this.props.isTopLevel) {
      window.location.reload();
    } else {
      this.setState({ hasError: false, error: null });
    }
  };

  public render() {
    if (this.state.hasError) {
      if (this.props.isTopLevel) {
        return (
          <div className="min-h-screen bg-slate-950 text-slate-100 flex flex-col items-center justify-center p-6 select-none">
            <div className="max-w-md w-full bg-slate-900 border border-rose-900/60 rounded-3xl p-8 shadow-2xl flex flex-col items-center text-center space-y-5 animate-in fade-in zoom-in-95">
              <div className="w-14 h-14 rounded-2xl bg-rose-500/10 border border-rose-500/30 flex items-center justify-center text-rose-400 shadow-lg shadow-rose-950/50">
                <AlertTriangle className="w-8 h-8" />
              </div>
              <div className="space-y-1.5">
                <h1 className="text-xl font-bold text-white tracking-tight">Something went wrong</h1>
                <p className="text-xs text-slate-400">
                  Terra Visual encountered an unexpected error. You can reload the application safely.
                </p>
              </div>
              {this.state.error && (
                <div className="w-full bg-slate-950 border border-slate-800 rounded-xl p-3 text-left overflow-auto max-h-36">
                  <pre className="text-[11px] font-mono text-rose-300 whitespace-pre-wrap break-all">
                    {this.state.error.message}
                  </pre>
                </div>
              )}
              <button
                type="button"
                onClick={this.handleReset}
                className="w-full flex items-center justify-center gap-2 px-5 py-2.5 rounded-xl bg-gradient-to-r from-sky-600 to-indigo-600 hover:from-sky-500 hover:to-indigo-500 text-white text-xs font-bold transition shadow-lg shadow-sky-950"
              >
                <RotateCcw className="w-4 h-4" />
                <span>Reload Application</span>
              </button>
            </div>
          </div>
        );
      }

      return (
        <div className="w-full h-[550px] bg-slate-950 border border-rose-900/60 rounded-2xl flex flex-col items-center justify-center text-slate-300 p-8 shadow-2xl space-y-4">
          <div className="p-3 rounded-2xl bg-rose-500/10 text-rose-400 border border-rose-500/20">
            <AlertTriangle className="w-8 h-8" />
          </div>
          <div className="text-center space-y-1 max-w-md">
            <h3 className="text-base font-bold text-white">
              {this.props.fallbackTitle || 'Canvas Rendering Error'}
            </h3>
            <p className="text-xs text-slate-400">
              An unexpected error occurred while rendering the interactive canvas.
            </p>
            {this.state.error && (
              <pre className="mt-3 p-3 rounded-xl bg-slate-900 border border-slate-800 text-[11px] font-mono text-rose-300 text-left overflow-auto max-h-32">
                {this.state.error.message}
              </pre>
            )}
          </div>
          <button
            type="button"
            onClick={this.handleReset}
            className="flex items-center gap-1.5 px-4 py-2 rounded-lg bg-slate-800 hover:bg-slate-700 text-xs font-semibold text-white border border-slate-700 transition shadow-lg"
          >
            <RefreshCw className="w-3.5 h-3.5" />
            <span>Retry Rendering</span>
          </button>
        </div>
      );
    }

    return this.props.children;
  }
}
