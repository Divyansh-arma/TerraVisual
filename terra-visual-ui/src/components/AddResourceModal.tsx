import React, { useState } from 'react';
import {
  RESOURCE_TEMPLATES,
  ResourceTemplate,
  CloudProvider,
  ResourceCategory,
} from '../constants/resourceTemplates';
import {
  X,
  Search,
  Cloud,
  Network,
  Server,
  Database,
  HardDrive,
  Cpu,
  Plus,
  CheckCircle,
  Layers,
} from 'lucide-react';

interface AddResourceModalProps {
  isOpen: boolean;
  onClose: () => void;
  onAddResource: (
    template: ResourceTemplate,
    customName: string,
    attributes: Record<string, any>
  ) => void;
}

const CATEGORIES: ('All' | ResourceCategory)[] = [
  'All',
  'Networking',
  'Compute',
  'Database',
  'Storage',
  'Load Balancers',
];

const PROVIDERS: { id: 'all' | CloudProvider; label: string; color: string }[] = [
  { id: 'all', label: 'All Clouds', color: 'bg-slate-800 text-slate-300' },
  { id: 'aws', label: 'AWS', color: 'bg-amber-500/20 text-amber-300 border-amber-500/30' },
  { id: 'azure', label: 'Azure', color: 'bg-sky-500/20 text-sky-300 border-sky-500/30' },
  { id: 'gcp', label: 'GCP', color: 'bg-emerald-500/20 text-emerald-300 border-emerald-500/30' },
];

const getCategoryIcon = (category: ResourceCategory) => {
  switch (category) {
    case 'Networking':
      return <Network className="w-4 h-4 text-sky-400" />;
    case 'Compute':
      return <Server className="w-4 h-4 text-emerald-400" />;
    case 'Database':
      return <Database className="w-4 h-4 text-purple-400" />;
    case 'Storage':
      return <HardDrive className="w-4 h-4 text-amber-400" />;
    case 'Load Balancers':
      return <Cpu className="w-4 h-4 text-indigo-400" />;
    default:
      return <Cloud className="w-4 h-4 text-slate-400" />;
  }
};

export const AddResourceModal: React.FC<AddResourceModalProps> = ({
  isOpen,
  onClose,
  onAddResource,
}) => {
  const [selectedProvider, setSelectedProvider] = useState<'all' | CloudProvider>('all');
  const [selectedCategory, setSelectedCategory] = useState<'All' | ResourceCategory>('All');
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedTemplate, setSelectedTemplate] = useState<ResourceTemplate>(RESOURCE_TEMPLATES[0]);
  const [customName, setCustomName] = useState(RESOURCE_TEMPLATES[0].defaultName);
  const [customAttributes, setCustomAttributes] = useState<string>(
    JSON.stringify(RESOURCE_TEMPLATES[0].defaultAttributes, null, 2)
  );
  const [jsonError, setJsonError] = useState<string | null>(null);

  if (!isOpen) return null;

  const handleSelectTemplate = (t: ResourceTemplate) => {
    setSelectedTemplate(t);
    setCustomName(t.defaultName);
    setCustomAttributes(JSON.stringify(t.defaultAttributes, null, 2));
    setJsonError(null);
  };

  const filteredTemplates = RESOURCE_TEMPLATES.filter((t) => {
    const matchesProvider = selectedProvider === 'all' || t.provider === selectedProvider;
    const matchesCategory = selectedCategory === 'All' || t.category === selectedCategory;
    const q = searchQuery.toLowerCase().trim();
    const matchesSearch =
      !q ||
      t.name.toLowerCase().includes(q) ||
      t.resourceType.toLowerCase().includes(q) ||
      t.description.toLowerCase().includes(q);

    return matchesProvider && matchesCategory && matchesSearch;
  });

  const handleConfirmAdd = () => {
    let parsedAttributes: Record<string, any> = {};
    if (customAttributes.trim()) {
      try {
        parsedAttributes = JSON.parse(customAttributes);
      } catch (err: any) {
        setJsonError(`Invalid JSON in attributes: ${err.message}`);
        return;
      }
    }

    const cleanName = customName.trim().replace(/[^a-zA-Z0-9_]/g, '_') || selectedTemplate.defaultName;
    onAddResource(selectedTemplate, cleanName, parsedAttributes);
    onClose();
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-950/80 backdrop-blur-md animate-in fade-in duration-150">
      <div className="bg-slate-900 border border-slate-800 rounded-2xl w-full max-w-4xl max-h-[85vh] flex flex-col shadow-2xl overflow-hidden">
        {/* Header */}
        <div className="px-6 py-4 border-b border-slate-800 flex items-center justify-between">
          <div className="flex items-center gap-2.5">
            <div className="p-2 rounded-xl bg-sky-500/10 text-sky-400">
              <Layers className="w-5 h-5" />
            </div>
            <div>
              <h2 className="text-base font-bold text-white tracking-tight">
                Multi-Cloud Resource Catalog
              </h2>
              <p className="text-xs text-slate-400">
                Select an infrastructure service template from AWS, Azure, or GCP to add to the canvas
              </p>
            </div>
          </div>

          <button
            type="button"
            onClick={onClose}
            className="p-1.5 rounded-lg text-slate-400 hover:text-white bg-slate-800 hover:bg-slate-700 transition"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        {/* Filter Bar */}
        <div className="px-6 py-3 bg-slate-950/50 border-b border-slate-800/80 flex flex-wrap items-center justify-between gap-3">
          {/* Provider Tabs */}
          <div className="flex items-center gap-1.5">
            {PROVIDERS.map((p) => (
              <button
                key={p.id}
                type="button"
                onClick={() => setSelectedProvider(p.id)}
                className={`px-3 py-1 text-xs font-semibold rounded-lg border transition ${
                  selectedProvider === p.id
                    ? p.color
                    : 'border-transparent text-slate-400 hover:text-slate-200 hover:bg-slate-800/50'
                }`}
              >
                {p.label}
              </button>
            ))}
          </div>

          {/* Search Box */}
          <div className="relative">
            <Search className="w-3.5 h-3.5 text-slate-500 absolute left-2.5 top-2.5" />
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Search services..."
              className="bg-slate-900 border border-slate-800 rounded-lg pl-8 pr-3 py-1.5 text-xs text-slate-200 placeholder-slate-500 focus:outline-none focus:border-sky-500 w-48 transition font-mono"
            />
          </div>
        </div>

        {/* Category Pills */}
        <div className="px-6 py-2 border-b border-slate-800/40 flex items-center gap-2 overflow-x-auto text-xs">
          <span className="text-[11px] font-semibold text-slate-500 uppercase tracking-wider shrink-0 mr-1">
            Category:
          </span>
          {CATEGORIES.map((cat) => (
            <button
              key={cat}
              type="button"
              onClick={() => setSelectedCategory(cat)}
              className={`px-2.5 py-0.5 rounded-full text-xs transition shrink-0 ${
                selectedCategory === cat
                  ? 'bg-sky-500/20 text-sky-300 font-bold border border-sky-500/30'
                  : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800'
              }`}
            >
              {cat}
            </button>
          ))}
        </div>

        {/* Main Body Grid */}
        <div className="flex-1 grid grid-cols-1 md:grid-cols-12 min-h-[380px] overflow-hidden">
          {/* Service Cards Catalog (Left 7 Cols) */}
          <div className="md:col-span-7 p-4 overflow-y-auto border-r border-slate-800/80 space-y-2">
            {filteredTemplates.length === 0 ? (
              <div className="text-center py-12 text-slate-500 text-xs">
                No resource templates match the selected filters.
              </div>
            ) : (
              filteredTemplates.map((template) => {
                const isSelected = selectedTemplate.id === template.id;
                return (
                  <div
                    key={template.id}
                    onClick={() => handleSelectTemplate(template)}
                    className={`p-3 rounded-xl border cursor-pointer transition-all flex items-start justify-between gap-3 ${
                      isSelected
                        ? 'bg-sky-950/30 border-sky-500/80 shadow-lg shadow-sky-950/50 ring-1 ring-sky-500/40'
                        : 'bg-slate-950/60 border-slate-800/80 hover:border-slate-700 hover:bg-slate-950'
                    }`}
                  >
                    <div className="flex items-start gap-3 min-w-0">
                      <div className="p-2 rounded-lg bg-slate-900 border border-slate-800 shrink-0 mt-0.5">
                        {getCategoryIcon(template.category)}
                      </div>
                      <div className="min-w-0">
                        <div className="flex items-center gap-2">
                          <span className="text-xs font-bold text-white truncate">
                            {template.name}
                          </span>
                          <span className="text-[10px] uppercase font-mono px-1.5 py-0.2 rounded bg-slate-800 text-slate-400 border border-slate-700">
                            {template.provider}
                          </span>
                        </div>
                        <div className="text-[11px] font-mono text-sky-400 truncate">
                          {template.resourceType}
                        </div>
                        <p className="text-[11px] text-slate-400 mt-1 line-clamp-2">
                          {template.description}
                        </p>
                      </div>
                    </div>

                    {isSelected && (
                      <CheckCircle className="w-4 h-4 text-sky-400 shrink-0 mt-1" />
                    )}
                  </div>
                );
              })
            )}
          </div>

          {/* Configuration Form (Right 5 Cols) */}
          <div className="md:col-span-5 p-5 bg-slate-950/30 flex flex-col justify-between overflow-y-auto space-y-4">
            <div className="space-y-4">
              <div>
                <div className="text-xs font-bold text-slate-200 uppercase tracking-wider mb-1">
                  Configure Resource
                </div>
                <div className="text-[11px] font-mono text-slate-400">
                  {selectedTemplate.resourceType}
                </div>
              </div>

              {/* Resource Name Input */}
              <div className="space-y-1.5">
                <label className="text-xs font-semibold text-slate-300">
                  Resource Name (Identifier)
                </label>
                <input
                  type="text"
                  value={customName}
                  onChange={(e) => setCustomName(e.target.value)}
                  placeholder={selectedTemplate.defaultName}
                  className="w-full bg-slate-900 border border-slate-800 rounded-lg px-3 py-2 text-xs font-mono text-white placeholder-slate-600 focus:outline-none focus:border-sky-500 transition"
                />
                <p className="text-[10px] text-slate-500 font-mono">
                  Node ID: {selectedTemplate.resourceType}.{customName || selectedTemplate.defaultName}
                </p>
              </div>

              {/* Attributes Editor */}
              <div className="space-y-1.5">
                <label className="text-xs font-semibold text-slate-300 flex items-center justify-between">
                  <span>Configuration Attributes (JSON)</span>
                  <span className="text-[10px] text-slate-500">Editable</span>
                </label>
                <textarea
                  rows={6}
                  value={customAttributes}
                  onChange={(e) => {
                    setCustomAttributes(e.target.value);
                    setJsonError(null);
                  }}
                  className="w-full bg-slate-900 border border-slate-800 rounded-lg p-2.5 text-xs font-mono text-emerald-400 focus:outline-none focus:border-sky-500 transition resize-none"
                />
                {jsonError && (
                  <p className="text-[10px] text-rose-400 font-semibold">{jsonError}</p>
                )}
              </div>
            </div>

            {/* Actions */}
            <div className="pt-3 border-t border-slate-800 flex items-center justify-end gap-2">
              <button
                type="button"
                onClick={onClose}
                className="px-3.5 py-1.5 text-xs font-semibold text-slate-400 hover:text-slate-200 transition"
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={handleConfirmAdd}
                className="flex items-center gap-1.5 px-4 py-2 bg-gradient-to-r from-sky-600 to-indigo-600 hover:from-sky-500 hover:to-indigo-500 text-white rounded-lg text-xs font-bold transition shadow-lg shadow-sky-950"
              >
                <Plus className="w-3.5 h-3.5" />
                Add to Canvas
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
