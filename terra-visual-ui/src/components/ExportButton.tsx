import React, { useState, useEffect, useRef } from 'react';
import { toPng, toSvg } from 'html-to-image';
import { Download, Image, FileCode, Check, RefreshCw } from 'lucide-react';

export const ExportButton: React.FC = () => {
  const [isOpen, setIsOpen] = useState(false);
  const [isExporting, setIsExporting] = useState(false);
  const [downloadSuccess, setDownloadSuccess] = useState<string | null>(null);
  const menuRef = useRef<HTMLDivElement>(null);

  // Close dropdown on clicking outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    };
    if (isOpen) {
      document.addEventListener('mousedown', handleClickOutside);
    }
    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
    };
  }, [isOpen]);

  const handleExport = async (format: 'png' | 'svg') => {
    try {
      setIsExporting(true);
      setIsOpen(false);

      // Target the React Flow canvas viewport element
      const viewportEl = document.querySelector('.react-flow__viewport') as HTMLElement;
      if (!viewportEl) {
        throw new Error('Canvas viewport element not found');
      }

      const fileName = `terra-visual-architecture-${Date.now()}.${format}`;
      let dataUrl = '';

      if (format === 'png') {
        dataUrl = await toPng(viewportEl, {
          backgroundColor: '#020617', // slate-950 dark background
          pixelRatio: 2, // High resolution (2x Retina)
        });
      } else {
        dataUrl = await toSvg(viewportEl, {
          backgroundColor: '#020617',
        });
      }

      // Trigger native download
      const link = document.createElement('a');
      link.download = fileName;
      link.href = dataUrl;
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);

      setDownloadSuccess(format.toUpperCase());
      setTimeout(() => setDownloadSuccess(null), 3000);
    } catch (err) {
      console.error('Failed to export canvas:', err);
      alert('Failed to export canvas image');
    } finally {
      setIsExporting(false);
    }
  };

  return (
    <div className="relative inline-block" ref={menuRef}>
      <button
        type="button"
        onClick={() => setIsOpen(!isOpen)}
        disabled={isExporting}
        className="flex items-center gap-1.5 px-3 py-1.5 bg-slate-900/95 hover:bg-slate-800 border border-slate-700/80 text-slate-200 hover:text-white rounded-xl shadow-lg backdrop-blur text-xs font-semibold transition disabled:opacity-50"
        title="Export Diagram (High-Res PNG / SVG)"
      >
        {isExporting ? (
          <RefreshCw className="w-3.5 h-3.5 animate-spin text-sky-400" />
        ) : downloadSuccess ? (
          <Check className="w-3.5 h-3.5 text-emerald-400" />
        ) : (
          <Download className="w-3.5 h-3.5 text-sky-400" />
        )}
        <span>{downloadSuccess ? `Exported ${downloadSuccess}` : 'Export'}</span>
      </button>

      {isOpen && (
        <div className="absolute right-0 top-full mt-2 w-52 bg-slate-900/95 border border-slate-700 rounded-xl shadow-2xl backdrop-blur-2xl p-1.5 z-[100] text-xs space-y-1 animate-in fade-in zoom-in-95">
          <div className="px-2 py-1 text-[10px] font-bold uppercase tracking-wider text-slate-400 border-b border-slate-800/80">
            Export Diagram
          </div>
          <button
            type="button"
            onClick={() => handleExport('png')}
            className="w-full flex items-center gap-2 px-2.5 py-2 text-left text-slate-200 hover:text-white hover:bg-sky-600/20 rounded-lg transition"
          >
            <Image className="w-4 h-4 text-sky-400" />
            <div className="flex flex-col">
              <span className="font-semibold leading-tight">High-Res PNG</span>
              <span className="text-[10px] text-slate-400">2x Retina dark background</span>
            </div>
          </button>
          <button
            type="button"
            onClick={() => handleExport('svg')}
            className="w-full flex items-center gap-2 px-2.5 py-2 text-left text-slate-200 hover:text-white hover:bg-indigo-600/20 rounded-lg transition"
          >
            <FileCode className="w-4 h-4 text-indigo-400" />
            <div className="flex flex-col">
              <span className="font-semibold leading-tight">Vector SVG</span>
              <span className="text-[10px] text-slate-400">Scalable vector graphic</span>
            </div>
          </button>
        </div>
      )}
    </div>
  );
};
