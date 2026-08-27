import { invoke } from '@tauri-apps/api/core';
import { GraphResponse } from '../types/graph';

/**
 * Invokes the Tauri backend command parse_state_file
 */
export async function parseStateFile(path: string): Promise<GraphResponse> {
  try {
    const raw = await invoke<string>('parse_state_file', { path });
    return JSON.parse(raw) as GraphResponse;
  } catch (err: any) {
    throw new Error(typeof err === 'string' ? err : err?.message || 'Failed to parse state file');
  }
}

/**
 * Invokes the Tauri backend command parse_code_dir
 */
export async function parseCodeDir(path: string): Promise<GraphResponse> {
  try {
    const raw = await invoke<string>('parse_code_dir', { path });
    return JSON.parse(raw) as GraphResponse;
  } catch (err: any) {
    throw new Error(typeof err === 'string' ? err : err?.message || 'Failed to parse code directory');
  }
}

/**
 * Invokes the Tauri backend command run_drift_analysis
 */
export async function runDriftAnalysis(statePath: string, codePath: string): Promise<GraphResponse> {
  try {
    const raw = await invoke<string>('run_drift_analysis', { statePath, codePath });
    return JSON.parse(raw) as GraphResponse;
  } catch (err: any) {
    throw new Error(typeof err === 'string' ? err : err?.message || 'Failed to run drift analysis');
  }
}

/**
 * Invokes the Tauri backend command sync_graph_to_code
 */
export async function syncGraphToCode(graph: GraphResponse, codePath: string): Promise<any> {
  try {
    const graphJson = JSON.stringify(graph);
    const raw = await invoke<string>('sync_graph_to_code', { graphJson, codePath });
    try {
      return JSON.parse(raw);
    } catch {
      return { status: 'success', raw };
    }
  } catch (err: any) {
    throw new Error(typeof err === 'string' ? err : err?.message || 'Failed to sync graph to code');
  }
}
