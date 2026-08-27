import React, { memo } from 'react';
import { Handle, Position, NodeProps } from '@xyflow/react';
import { NodeData, DriftStatus } from '../types/graph';
import { Server, Database, Cloud, Network, HardDrive, Shield, ShieldAlert, Box, Globe, Layers } from 'lucide-react';

const getResourceIcon = (resourceType?: string) => {
  if (!resourceType) return <Cloud className="w-3.5 h-3.5 text-indigo-400" />;
  const rt = resourceType.toLowerCase();
  if (rt === 'module') {
    return <Box className="w-3.5 h-3.5 text-purple-400" />;
  }
  if (rt === 'aws_availability_zone') {
    return <Globe className="w-3.5 h-3.5 text-amber-400" />;
  }
  if (rt.includes('vpc') || rt.includes('subnet') || rt.includes('route') || rt.includes('network') || rt.includes('gateway')) {
    return <Network className="w-3.5 h-3.5 text-sky-400" />;
  }
  if (rt.includes('db') || rt.includes('dynamo') || rt.includes('rds') || rt.includes('sql') || rt.includes('table')) {
    return <Database className="w-3.5 h-3.5 text-purple-400" />;
  }
  if (rt.includes('instance') || rt.includes('ecs') || rt.includes('lambda') || rt.includes('machine') || rt.includes('eks') || rt.includes('cluster')) {
    return <Server className="w-3.5 h-3.5 text-emerald-400" />;
  }
  if (rt.includes('node_group') || rt.includes('nodegroup')) {
    return <Layers className="w-3.5 h-3.5 text-cyan-400" />;
  }
  if (rt.includes('s3') || rt.includes('bucket') || rt.includes('storage')) {
    return <HardDrive className="w-3.5 h-3.5 text-amber-400" />;
  }
  if (rt.includes('security') || rt.includes('iam') || rt.includes('firewall')) {
    return <Shield className="w-3.5 h-3.5 text-rose-400" />;
  }
  return <Cloud className="w-3.5 h-3.5 text-indigo-400" />;
};

const getDriftStyles = (status?: DriftStatus, isParent: boolean = false, isModule: boolean = false) => {
  if (isModule && !isParent) {
    return {
      cardBorder: 'border-indigo-500/80 bg-slate-900/95 shadow-md shadow-indigo-950/40',
      badge: 'bg-indigo-500/20 text-indigo-300 border-indigo-500/40',
      handle: '!bg-indigo-400 !border-slate-950',
    };
  }

  switch (status) {
    case 'CREATE':
      return {
        cardBorder: isParent
          ? 'border-emerald-500/50 border-dashed bg-emerald-950/20 shadow-xl shadow-emerald-950/30'
          : 'border-emerald-500/90 border-2 bg-slate-900/95 shadow-md shadow-emerald-950/40',
        badge: 'bg-emerald-500/20 text-emerald-300 border-emerald-500/40 font-bold',
        handle: '!bg-emerald-400 !border-slate-950',
      };
    case 'DESTROY':
      return {
        cardBorder: isParent
          ? 'border-rose-500/50 border-dashed bg-rose-950/20'
          : 'border-rose-500/90 border-dashed border-2 bg-rose-950/30 shadow-md shadow-rose-950/40',
        badge: 'bg-rose-500/20 text-rose-300 border-rose-500/40 font-bold',
        handle: '!bg-rose-400 !border-slate-950',
      };
    case 'IN_SYNC':
      return {
        cardBorder: isParent
          ? (isModule ? 'border-indigo-500/50 border-dashed bg-indigo-950/20 shadow-xl shadow-indigo-950/30' : 'border-emerald-500/40 bg-emerald-950/10')
          : 'border-emerald-500/80 bg-slate-900/95 shadow-md shadow-emerald-950/40',
        badge: 'bg-emerald-500/20 text-emerald-300 border-emerald-500/30',
        handle: '!bg-emerald-400 !border-slate-950',
      };
    case 'MODIFIED':
      return {
        cardBorder: isParent
          ? (isModule ? 'border-amber-500/50 border-dashed bg-amber-950/20' : 'border-amber-500/40 bg-amber-950/10')
          : 'border-amber-500/90 bg-slate-900/95 shadow-md shadow-amber-950/40',
        badge: 'bg-amber-500/20 text-amber-300 border-amber-500/30',
        handle: '!bg-amber-400 !border-slate-950',
      };
    case 'MISSING_IN_CODE':
      return {
        cardBorder: isParent
          ? 'border-rose-500/40 border-dashed bg-rose-950/10'
          : 'border-rose-500/80 border-dashed border-2 bg-rose-950/20 shadow-md shadow-rose-950/40',
        badge: 'bg-rose-500/20 text-rose-300 border-rose-500/30',
        handle: '!bg-rose-400 !border-slate-950',
      };
    case 'MISSING_IN_STATE':
      return {
        cardBorder: isParent
          ? 'border-blue-500/40 border-dashed bg-blue-950/10'
          : 'border-blue-500/80 border-dashed border-2 bg-blue-950/20 shadow-md shadow-blue-950/40',
        badge: 'bg-blue-500/20 text-blue-300 border-blue-500/30',
        handle: '!bg-blue-400 !border-slate-950',
      };
    default:
      return {
        cardBorder: isParent
          ? (isModule ? 'border-indigo-500/50 border-dashed bg-indigo-950/20' : 'border-slate-700/50 bg-slate-900/30')
          : 'border-slate-700/80 bg-slate-900/95 shadow-md shadow-slate-950/40',
        badge: 'bg-slate-800 text-slate-400 border-slate-700',
        handle: '!bg-slate-400 !border-slate-950',
      };
  }
};

export const InfrastructureNode: React.FC<NodeProps> = memo(({ data, selected }) => {
  const nodeData = (data || {}) as unknown as Partial<NodeData>;
  const resourceType = nodeData?.resourceType || 'unknown';
  const label = nodeData?.label || 'resource';
  const provider = nodeData?.provider || 'aws';
  const driftStatus = nodeData?.driftStatus || 'unknown';
  const isModule = resourceType === 'module';

  // Determine if this node acts as a container (subgraph parent with children)
  const isContainer = Boolean(nodeData?.isContainer);
  const styles = getDriftStyles(driftStatus, isContainer, isModule);
  const issues = nodeData?.securityIssues || [];
  const hasSecurityIssues = issues.length > 0;

  // Subnet public vs private identification
  const isPublicSubnet =
    resourceType === 'aws_subnet' &&
    (nodeData?.attributes?.subnet_type === 'public' || label.toLowerCase().includes('public'));
  const isPrivateSubnet =
    resourceType === 'aws_subnet' &&
    (nodeData?.attributes?.subnet_type === 'private' || label.toLowerCase().includes('private'));

  // Compact Floating Security Badge
  const renderSecurityBadge = () => {
    if (!hasSecurityIssues) return null;
    return (
      <div className="absolute -top-2 -right-2 z-30 group/sec">
        <div className="flex items-center gap-1 bg-rose-600 hover:bg-rose-500 text-white text-[9px] font-bold px-1.5 py-0.5 rounded-full shadow-lg shadow-rose-950 border border-rose-400/50 cursor-pointer animate-pulse">
          <ShieldAlert className="w-3 h-3" />
          <span>{issues.length}</span>
        </div>

        {/* Hover Tooltip / Popover */}
        <div className="hidden group-hover/sec:block absolute right-0 top-5 w-60 bg-slate-900/95 border border-rose-500/50 rounded-xl p-2.5 shadow-2xl backdrop-blur-md z-50 text-left pointer-events-none space-y-1.5 animate-in fade-in zoom-in-95">
          <div className="flex items-center justify-between border-b border-slate-800 pb-1 text-[10px] font-bold text-rose-400">
            <span className="flex items-center gap-1">
              <ShieldAlert className="w-3 h-3" /> Security Alerts
            </span>
            <span className="bg-rose-950/80 text-rose-300 px-1 py-0.2 rounded text-[8px] font-mono">
              {issues.length}
            </span>
          </div>
          <div className="space-y-1 max-h-40 overflow-y-auto">
            {issues.map((issue, idx) => (
              <div key={idx} className="space-y-0.5 text-[9px]">
                <div className="flex items-center gap-1 font-bold">
                  <span
                    className={`px-1 py-0.2 rounded text-[7px] uppercase tracking-wider font-mono ${
                      issue?.severity === 'CRITICAL'
                        ? 'bg-red-500/20 text-red-300 border border-red-500/40'
                        : issue?.severity === 'HIGH'
                        ? 'bg-rose-500/20 text-rose-300 border border-rose-500/40'
                        : 'bg-amber-500/20 text-amber-300 border border-amber-500/40'
                    }`}
                  >
                    {issue?.severity || 'MEDIUM'}
                  </span>
                  <span className="text-slate-400 font-mono">{issue?.ruleId || 'Rule'}</span>
                </div>
                <div className="text-slate-200 font-medium leading-snug">
                  {issue?.title || 'Security notice'}
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    );
  };

  // Case 1: Container / Subgraph Node (VPC, AZ, Subnet, Module)
  if (isContainer) {
    const isVPC =
      resourceType === 'aws_vpc' ||
      resourceType === 'azurerm_virtual_network' ||
      resourceType === 'google_compute_network';
    const isAZ = resourceType === 'aws_availability_zone';

    let containerStyle = styles.cardBorder;
    if (isVPC) {
      containerStyle = 'border-2 border-dashed border-sky-500/70 bg-slate-950/80 rounded-3xl shadow-2xl backdrop-blur-md';
    } else if (isAZ) {
      containerStyle = 'border-2 border-dashed border-amber-500/50 bg-slate-900/40 rounded-2xl shadow-inner';
    } else if (isPublicSubnet) {
      containerStyle = 'border-2 border-emerald-500/80 bg-emerald-950/30 rounded-xl shadow-lg';
    } else if (isPrivateSubnet) {
      containerStyle = 'border-2 border-indigo-500/80 bg-indigo-950/30 rounded-xl shadow-lg';
    } else if (isModule) {
      containerStyle = 'border-2 border-dashed border-indigo-500/50 bg-indigo-950/20 rounded-2xl shadow-xl shadow-indigo-950/30';
    }

    const cidrText = nodeData?.attributes?.cidr_block || nodeData?.attributes?.cidr;

    return (
      <div
        className={`relative w-full h-full min-w-[280px] min-h-[160px] transition-all duration-200 p-3.5 flex flex-col justify-between cursor-pointer ${
          containerStyle
        } ${
          selected
            ? 'ring-2 ring-sky-400 ring-offset-2 ring-offset-slate-950 scale-[1.01]'
            : 'hover:border-slate-400'
        }`}
      >
        {renderSecurityBadge()}

        <Handle
          type="target"
          position={Position.Top}
          className={`!w-2.5 !h-2.5 !border-2 ${styles.handle}`}
        />

        {/* Top Header */}
        <div className="flex items-center justify-between gap-2 border-b border-slate-800/80 pb-2">
          <div className="flex items-center gap-2 min-w-0">
            <div
              className={`p-1.5 rounded-lg border shrink-0 ${
                isModule
                  ? 'bg-purple-500/10 border-purple-500/30 text-purple-400'
                  : isAZ
                  ? 'bg-amber-500/10 border-amber-500/30 text-amber-400'
                  : isPublicSubnet
                  ? 'bg-emerald-500/10 border-emerald-500/30 text-emerald-400'
                  : isPrivateSubnet
                  ? 'bg-indigo-500/10 border-indigo-500/30 text-indigo-400'
                  : 'bg-sky-500/10 border-sky-500/30 text-sky-400'
              }`}
            >
              {isModule ? (
                <Box className="w-4 h-4" />
              ) : isAZ ? (
                <Globe className="w-4 h-4" />
              ) : (
                <Network className="w-4 h-4" />
              )}
            </div>
            <div className="min-w-0">
              <div
                className={`text-[9px] font-mono font-bold uppercase tracking-wider ${
                  isModule
                    ? 'text-purple-400'
                    : isAZ
                    ? 'text-amber-400'
                    : isPublicSubnet
                    ? 'text-emerald-400'
                    : isPrivateSubnet
                    ? 'text-indigo-400'
                    : 'text-sky-400'
                }`}
              >
                {isModule
                  ? 'TERRAFORM MODULE'
                  : isAZ
                  ? 'AVAILABILITY ZONE'
                  : isPublicSubnet
                  ? 'PUBLIC SUBNET'
                  : isPrivateSubnet
                  ? 'PRIVATE SUBNET'
                  : 'AWS VIRTUAL PRIVATE CLOUD'}
              </div>
              <div className="text-xs font-bold text-white font-mono truncate">
                {isVPC ? (
                  <span className="flex items-center gap-1.5">
                    <span>{label}</span>
                    {cidrText && !label.includes(String(cidrText)) && (
                      <span className="text-[10px] text-sky-300 font-normal">
                        ({String(cidrText)})
                      </span>
                    )}
                  </span>
                ) : (
                  label
                )}
              </div>
            </div>
          </div>

          <span
            className={`px-1.5 py-0.5 text-[8px] font-bold rounded-full border uppercase tracking-wider shrink-0 ${styles.badge}`}
          >
            {driftStatus}
          </span>
        </div>

        {/* Footer Badge */}
        <div className="flex items-center justify-between text-[9px] text-slate-400 font-mono pt-2 pointer-events-none">
          <span>
            {isModule
              ? 'Module Subgraph'
              : isAZ
              ? 'Logical Zone'
              : isPublicSubnet
              ? 'Public Subnet'
              : isPrivateSubnet
              ? 'Private Subnet'
              : 'VPC Boundary'}
          </span>
          {cidrText ? (
            <span className="text-slate-300 bg-slate-900 px-1.5 py-0.5 rounded border border-slate-700 font-mono text-[8px]">
              {String(cidrText)}
            </span>
          ) : isAZ && nodeData?.attributes?.availability_zone ? (
            <span className="text-amber-300 bg-amber-950/80 px-1.5 py-0.5 rounded border border-amber-800/60 font-mono text-[8px]">
              {String(nodeData.attributes.availability_zone)}
            </span>
          ) : null}
        </div>

        <Handle
          type="source"
          position={Position.Bottom}
          className={`!w-2.5 !h-2.5 !border-2 ${styles.handle}`}
        />
      </div>
    );
  }

  // Case 2: Standalone Module Card (Module WITHOUT child resources)
  if (isModule) {
    return (
      <div
        className={`relative w-[260px] h-[75px] rounded-xl border p-2.5 transition-all duration-200 cursor-pointer flex flex-col justify-between ${
          styles.cardBorder
        } ${
          selected
            ? 'ring-2 ring-indigo-400 ring-offset-2 ring-offset-slate-950 scale-[1.02]'
            : 'hover:scale-[1.01]'
        }`}
      >
        {renderSecurityBadge()}

        <Handle
          type="target"
          position={Position.Top}
          className={`!w-2.5 !h-2.5 !border-2 ${styles.handle}`}
        />

        {/* Top Row: Icon + Label + Module Badge */}
        <div className="flex items-center justify-between gap-1.5">
          <div className="flex items-center gap-1.5 min-w-0">
            <div className="p-1 rounded bg-indigo-950/80 border border-indigo-800/60 text-purple-400 shrink-0">
              <Box className="w-3.5 h-3.5" />
            </div>
            <span className="text-[9px] font-mono text-indigo-400 font-bold uppercase tracking-wider truncate">
              TERRAFORM
            </span>
          </div>
          <span className="px-1.5 py-0.5 text-[8px] font-bold rounded border uppercase tracking-wider shrink-0 bg-indigo-500/20 text-indigo-300 border-indigo-500/40">
            MODULE
          </span>
        </div>

        {/* Bottom Row: Module Name + Source */}
        <div className="flex items-baseline justify-between gap-2 min-w-0">
          <div className="text-xs font-bold text-white font-mono truncate">
            {label}
          </div>
          {nodeData?.attributes?.source && (
            <span className="text-[8px] text-slate-400 font-mono truncate max-w-[100px]">
              {String(nodeData.attributes.source)}
            </span>
          )}
        </div>

        <Handle
          type="source"
          position={Position.Bottom}
          className={`!w-2.5 !h-2.5 !border-2 ${styles.handle}`}
        />
      </div>
    );
  }

  // Case 3: Public / Private Subnet Card (Fixed 240px x 75px)
  if (isPublicSubnet || isPrivateSubnet) {
    const cardBorder = isPublicSubnet
      ? 'border-2 border-emerald-500/80 bg-emerald-950/40 shadow-md shadow-emerald-950/40'
      : 'border-2 border-indigo-500/80 bg-indigo-950/40 shadow-md shadow-indigo-950/40';
    const badgeStyle = isPublicSubnet
      ? 'bg-emerald-500/20 text-emerald-300 border-emerald-500/40'
      : 'bg-indigo-500/20 text-indigo-300 border-indigo-500/40';
    const badgeText = isPublicSubnet ? 'PUBLIC SUBNET' : 'PRIVATE SUBNET';

    return (
      <div
        className={`relative w-[240px] h-[75px] rounded-xl p-2.5 transition-all duration-200 cursor-pointer flex flex-col justify-between ${
          cardBorder
        } ${
          selected
            ? 'ring-2 ring-sky-400 ring-offset-2 ring-offset-slate-950 scale-[1.02]'
            : 'hover:scale-[1.01]'
        }`}
      >
        {renderSecurityBadge()}

        <Handle
          type="target"
          position={Position.Top}
          className={`!w-2.5 !h-2.5 !border-2 ${styles.handle}`}
        />

        {/* Top Row: Subnet Badge + CIDR */}
        <div className="flex items-center justify-between gap-1.5">
          <div className="flex items-center gap-1.5 min-w-0">
            <div
              className={`p-0.5 rounded ${
                isPublicSubnet ? 'bg-emerald-500/20 text-emerald-300' : 'bg-indigo-500/20 text-indigo-300'
              }`}
            >
              <Network className="w-3.5 h-3.5" />
            </div>
            <span
              className={`px-1.5 py-0.5 text-[8px] font-bold rounded border uppercase tracking-wider ${badgeStyle}`}
            >
              {badgeText}
            </span>
          </div>

          <span
            className={`px-1.5 py-0.5 text-[8px] font-bold rounded border uppercase tracking-wider shrink-0 ${styles.badge}`}
          >
            {driftStatus}
          </span>
        </div>

        {/* Bottom Row: Label / CIDR */}
        <div className="min-w-0">
          <div className="text-xs font-bold text-white font-mono truncate leading-tight">
            {label}
          </div>
          {nodeData?.attributes?.cidr_block && (
            <div className="text-[9px] font-mono text-slate-400 truncate leading-none mt-0.5">
              {String(nodeData.attributes.cidr_block)}
            </div>
          )}
        </div>

        <Handle
          type="source"
          position={Position.Bottom}
          className={`!w-2.5 !h-2.5 !border-2 ${styles.handle}`}
        />
      </div>
    );
  }

  // Case 4: Standard Resource Card (Fixed 240px x 75px)
  return (
    <div
      className={`relative w-[240px] h-[75px] rounded-xl border p-2.5 transition-all duration-200 cursor-pointer flex flex-col justify-between ${
        styles.cardBorder
      } ${
        selected
          ? 'ring-2 ring-sky-400 ring-offset-2 ring-offset-slate-950 scale-[1.02]'
          : 'hover:scale-[1.01]'
      }`}
    >
      {renderSecurityBadge()}

      <Handle
        type="target"
        position={Position.Top}
        className={`!w-2.5 !h-2.5 !border-2 ${styles.handle}`}
      />

      {/* Top Row: Provider Icon + Provider Name + Drift Status */}
      <div className="flex items-center justify-between gap-1.5">
        <div className="flex items-center gap-1.5 min-w-0">
          <div className="p-0.5 rounded bg-slate-800/80 text-slate-300 shrink-0">
            {getResourceIcon(resourceType)}
          </div>
          <span className="text-[9px] font-mono text-slate-400 truncate uppercase tracking-wider">
            {provider}
          </span>
        </div>

        <span
          className={`px-1.5 py-0.5 text-[8px] font-bold rounded border uppercase tracking-wider shrink-0 ${styles.badge}`}
        >
          {driftStatus}
        </span>
      </div>

      {/* Bottom Row: Resource Type & Label */}
      <div className="min-w-0">
        <div className="text-[9px] font-mono text-slate-400 truncate leading-none mb-0.5">
          {resourceType}
        </div>
        <div className="text-xs font-bold text-white font-mono truncate leading-tight">
          {label}
        </div>
      </div>

      <Handle
        type="source"
        position={Position.Bottom}
        className={`!w-2.5 !h-2.5 !border-2 ${styles.handle}`}
      />
    </div>
  );
});
