import React from 'react';

interface LogoProps extends React.SVGProps<SVGSVGElement> {
  size?: number;
  className?: string;
}

export const Logo: React.FC<LogoProps> = ({ size = 32, className = '', ...props }) => {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 512 512"
      width={size}
      height={size}
      fill="none"
      className={className}
      {...props}
    >
      <defs>
        {/* Brand Linear Gradient: Electric Cyan to Hyper Indigo */}
        <linearGradient id="terraCyanIndigo" x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stopColor="#0ea5e9" />
          <stop offset="100%" stopColor="#6366f1" />
        </linearGradient>

        {/* Node Stroke Gradient */}
        <linearGradient id="terraNodeStroke" x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stopColor="#38bdf8" stopOpacity="0.9" />
          <stop offset="100%" stopColor="#818cf8" stopOpacity="0.9" />
        </linearGradient>

        {/* Glassmorphic Node Fill Gradient */}
        <linearGradient id="terraGlassFill" x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stopColor="#1e293b" stopOpacity="0.95" />
          <stop offset="100%" stopColor="#0f172a" stopOpacity="0.98" />
        </linearGradient>

        {/* Outer Glow & Drop Shadow Filter */}
        <filter id="terraDropShadow" x="-20%" y="-20%" width="140%" height="140%">
          <feDropShadow dx="0" dy="10" stdDeviation="16" floodColor="#0ea5e9" floodOpacity="0.4" />
          <feDropShadow dx="0" dy="4" stdDeviation="6" floodColor="#6366f1" floodOpacity="0.3" />
        </filter>

        {/* Ambient Blur Filter */}
        <filter id="terraAmbientBlur" x="-30%" y="-30%" width="160%" height="160%">
          <feGaussianBlur stdDeviation="14" />
        </filter>
      </defs>

      {/* Ambient Background Glow behind the Topology T */}
      <g filter="url(#terraAmbientBlur)" opacity="0.45">
        <circle cx="132" cy="140" r="50" fill="#0ea5e9" />
        <circle cx="380" cy="140" r="50" fill="#6366f1" />
        <circle cx="256" cy="372" r="50" fill="#0ea5e9" />
        <circle cx="256" cy="140" r="40" fill="#6366f1" />
      </g>

      {/* Glowing Connection Vector Paths */}
      <g>
        {/* Glow Underlay Strokes */}
        <path
          d="M 132 140 L 380 140"
          stroke="#0ea5e9"
          strokeWidth="24"
          strokeLinecap="round"
          opacity="0.35"
          filter="url(#terraAmbientBlur)"
        />
        <path
          d="M 256 140 L 256 372"
          stroke="#6366f1"
          strokeWidth="24"
          strokeLinecap="round"
          opacity="0.35"
          filter="url(#terraAmbientBlur)"
        />

        {/* Primary Glowing Circuit Paths */}
        <path
          d="M 132 140 L 380 140"
          stroke="url(#terraCyanIndigo)"
          strokeWidth="14"
          strokeLinecap="round"
        />
        <path
          d="M 256 140 L 256 372"
          stroke="url(#terraCyanIndigo)"
          strokeWidth="14"
          strokeLinecap="round"
        />
      </g>

      {/* Central Junction Hub */}
      <circle cx="256" cy="140" r="22" fill="#0f172a" stroke="url(#terraNodeStroke)" strokeWidth="4" />
      <circle cx="256" cy="140" r="10" fill="url(#terraCyanIndigo)" />

      {/* Node 1: Top-Left Node */}
      <g filter="url(#terraDropShadow)">
        <rect
          x="76"
          y="84"
          width="112"
          height="112"
          rx="28"
          fill="url(#terraGlassFill)"
          stroke="url(#terraNodeStroke)"
          strokeWidth="4"
        />
        {/* Inner Node Accent */}
        <circle cx="132" cy="140" r="18" fill="url(#terraCyanIndigo)" opacity="0.25" />
        <circle cx="132" cy="140" r="10" fill="url(#terraCyanIndigo)" />
        <rect x="96" y="104" width="20" height="4" rx="2" fill="#38bdf8" opacity="0.8" />
      </g>

      {/* Node 2: Top-Right Node */}
      <g filter="url(#terraDropShadow)">
        <rect
          x="324"
          y="84"
          width="112"
          height="112"
          rx="28"
          fill="url(#terraGlassFill)"
          stroke="url(#terraNodeStroke)"
          strokeWidth="4"
        />
        {/* Inner Node Accent */}
        <circle cx="380" cy="140" r="18" fill="url(#terraCyanIndigo)" opacity="0.25" />
        <circle cx="380" cy="140" r="10" fill="url(#terraCyanIndigo)" />
        <rect x="344" y="104" width="20" height="4" rx="2" fill="#818cf8" opacity="0.8" />
      </g>

      {/* Node 3: Bottom-Center Node */}
      <g filter="url(#terraDropShadow)">
        <rect
          x="200"
          y="316"
          width="112"
          height="112"
          rx="28"
          fill="url(#terraGlassFill)"
          stroke="url(#terraNodeStroke)"
          strokeWidth="4"
        />
        {/* Inner Node Accent */}
        <circle cx="256" cy="372" r="18" fill="url(#terraCyanIndigo)" opacity="0.25" />
        <circle cx="256" cy="372" r="10" fill="url(#terraCyanIndigo)" />
        <rect x="220" y="336" width="20" height="4" rx="2" fill="#38bdf8" opacity="0.8" />
      </g>
    </svg>
  );
};
