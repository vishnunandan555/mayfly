import React from 'react';

interface IconProps extends React.SVGProps<SVGSVGElement> {
  className?: string;
}

export function UniversalIcon({ className = 'size-4', ...props }: IconProps) {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className={className} {...props}>
      <rect width="20" height="16" x="2" y="4" rx="3" />
      <path d="m7 10 3 2-3 2" />
      <path d="M13 14h4" />
    </svg>
  );
}

export function NodejsIcon({ className = 'size-4', ...props }: IconProps) {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" className={className} {...props}>
      <path d="M12 2.5 20.5 7.4v9.8L12 22.1 3.5 17.2V7.4L12 2.5z" />
      <path d="M12 7.5v9M8.5 9.5l7 4M8.5 14.5l7-4" opacity="0.6" />
      <circle cx="12" cy="12" r="2.2" fill="currentColor" />
    </svg>
  );
}

export function PythonIcon({ className = 'size-4', ...props }: IconProps) {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" className={className} {...props}>
      <path d="M12 2.5c-3.2 0-5 1.5-5 3.8v2.2h5v1.2H4.5c-2.3 0-3.8 1.8-3.8 5s1.5 5 3.8 5H7v-2.2c0-2.3 1.8-3.8 3.8-3.8h4.4c2 0 3.8-1.5 3.8-3.8V7.5c0-3.2-1.8-5-5-5h-2z" />
      <circle cx="8.5" cy="5.2" r="0.8" fill="currentColor" />
      <path d="M12 21.5c3.2 0 5-1.5 5-3.8v-2.2h-5v-1.2h7.5c2.3 0 3.8-1.8 3.8-5s-1.5-5-3.8-5H17v2.2c0 2.3-1.8 3.8-3.8 3.8H8.8c-2 0-3.8 1.5-3.8 3.8v2.4c0 3.2 1.8 5 5 5h2z" />
      <circle cx="15.5" cy="18.8" r="0.8" fill="currentColor" />
    </svg>
  );
}

export function GolangIcon({ className = 'size-4', ...props }: IconProps) {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" className={className} {...props}>
      <path d="M2.5 12a5.5 5.5 0 1 1 10.5 2.5h-5" />
      <circle cx="18" cy="12" r="4.5" />
      <path d="M1.5 8h4M1.5 10.5h2.5M1.5 13.5h3" strokeWidth="1.4" opacity="0.7" />
    </svg>
  );
}

export function RustIcon({ className = 'size-4', ...props }: IconProps) {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" strokeLinejoin="round" className={className} {...props}>
      <circle cx="12" cy="12" r="7.5" />
      <circle cx="12" cy="12" r="2.5" />
      <path d="M12 2v2.5M12 19.5V22M2 12h2.5M19.5 12H22M4.9 4.9l1.8 1.8M17.3 17.3l1.8 1.8M4.9 19.1l1.8-1.8M17.3 6.7l1.8-1.8" />
    </svg>
  );
}

export function JvmIcon({ className = 'size-4', ...props }: IconProps) {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" className={className} {...props}>
      <path d="M17 8h1a3 3 0 0 1 3 3v1a3 3 0 0 1-3 3h-1" />
      <path d="M5 8h12v7a4 4 0 0 1-4 4H9a4 4 0 0 1-4-4V8z" />
      <path d="M5 21h14" />
      <path d="M8 2c0 1.5 1 2.5 1 4M12 2c0 1.5 1 2.5 1 4M16 2c0 1.5 1 2.5 1 4" strokeWidth="1.4" opacity="0.6" />
    </svg>
  );
}

export function RubyIcon({ className = 'size-4', ...props }: IconProps) {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" className={className} {...props}>
      <path d="M6 3.5h12l4 6.5-10 11.5L2 10l4-6.5z" />
      <path d="M2 10h20M7 3.5 12 21.5 17 3.5M6 10l6 11.5 6-11.5" opacity="0.5" />
    </svg>
  );
}

export function PhpIcon({ className = 'size-4', ...props }: IconProps) {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" className={className} {...props}>
      <ellipse cx="12" cy="12" rx="10.5" ry="6.5" />
      <path d="M7 9.5v5M7 9.5h2a1.5 1.5 0 0 1 0 3H7M12 9.5v5M12 12h2.5M14.5 9.5v5M17 9.5v5M17 9.5h2a1.5 1.5 0 0 1 0 3h-2" strokeWidth="1.4" />
    </svg>
  );
}

export function DotnetIcon({ className = 'size-4', ...props }: IconProps) {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" className={className} {...props}>
      <rect width="18" height="18" x="3" y="3" rx="4" />
      <circle cx="7.5" cy="15.5" r="1" fill="currentColor" />
      <path d="M9.5 8.5v7M9.5 8.5l5 7v-7" />
    </svg>
  );
}

export function BunIcon({ className = 'size-4', ...props }: IconProps) {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" className={className} {...props}>
      <path d="M12 3.5c-5.5 0-9.5 3.5-9.5 9 0 4.5 4 8 9.5 8s9.5-3.5 9.5-8c0-5.5-4-9-9-9z" />
      <circle cx="8.5" cy="13.5" r="1" fill="currentColor" />
      <circle cx="15.5" cy="13.5" r="1" fill="currentColor" />
      <path d="M10.5 16.5c.8.6 2.2.6 3 0" />
      <path d="M12 3.5c0 2.5-1 3.5-2 4.5M12 3.5c0 2.5 1 3.5 2 4.5" strokeWidth="1.2" opacity="0.6" />
    </svg>
  );
}

export function DockerIcon({ className = 'size-4', ...props }: IconProps) {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" strokeLinejoin="round" className={className} {...props}>
      <path d="M2.5 13.5c1.2-1.5 3-2 5.5-2 3.5 0 5 2 8.5 2 2.5 0 4-1 5-2.5 0 5-3.5 8-9.5 8s-9.5-3-9.5-5.5z" />
      <rect width="2.2" height="2" x="5" y="8" rx="0.3" />
      <rect width="2.2" height="2" x="8" y="8" rx="0.3" />
      <rect width="2.2" height="2" x="11" y="8" rx="0.3" />
      <rect width="2.2" height="2" x="8" y="5.2" rx="0.3" />
      <rect width="2.2" height="2" x="11" y="5.2" rx="0.3" />
    </svg>
  );
}

export const STACK_ICONS: Record<string, React.FC<IconProps>> = {
  Universal: UniversalIcon,
  Nodejs: NodejsIcon,
  Python: PythonIcon,
  Golang: GolangIcon,
  Rust: RustIcon,
  Jvm: JvmIcon,
  Ruby: RubyIcon,
  Php: PhpIcon,
  Dotnet: DotnetIcon,
  Bun: BunIcon,
  Docker: DockerIcon,
};
