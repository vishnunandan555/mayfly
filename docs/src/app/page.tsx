'use client';

import React, { useState } from 'react';
import Link from 'next/link';
import {
  Shield,
  Lock,
  Terminal,
  Cpu,
  CheckCircle2,
  ArrowRight,
  ExternalLink,
  Copy,
  Check,
  Zap,
  AlertTriangle,
  FileCode,
  Layers,
  ChevronRight,
  BookOpen,
  Sparkles,
  Eye,
  KeyRound,
  FileSearch,
  History,
  Monitor,
  CheckCircle,
  HelpCircle,
  Database,
  RefreshCw,
  Menu,
  X,
  Code,
  ShieldAlert,
  GitBranch,
  FolderGit2,
  Search,
  WifiOff,
} from 'lucide-react';

export default function HomePage() {
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
  const [installOs, setInstallOs] = useState<'unix' | 'windows' | 'source'>('unix');
  const [copiedInstall, setCopiedInstall] = useState(false);
  const [copiedHash, setCopiedHash] = useState(false);
  const [terminalTab, setTerminalTab] = useState<'injection' | 'tui' | 'scanner' | 'audit'>('injection');

  const installCommands = {
    unix: 'curl -fsSL https://raw.githubusercontent.com/vishnunandan555/mayfly/main/install.sh | bash',
    windows: 'irm https://raw.githubusercontent.com/vishnunandan555/mayfly/main/install.ps1 | iex',
    source: 'git clone https://github.com/vishnunandan555/mayfly.git && cd mayfly && make install',
  };

  const copyToClipboard = (text: string, setCopied: (v: boolean) => void) => {
    navigator.clipboard.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const substitutions = [
    {
      name: 'godotenv / dotenv',
      downloads: '10.4M+ weekly',
      category: 'In-Memory Process Injection',
      whatItMeans: 'Stops saving .env files on disk',
      stdlib: 'os.Environ, os/exec.CommandContext, RAM zeroing',
      desc: 'Instead of saving API keys in readable plaintext files on your SSD, MayFly loads them directly into your app’s memory and zeroes memory buffers upon exit.',
      file: 'pkg/executor/process.go',
    },
    {
      name: 'golang.org/x/crypto/pbkdf2',
      downloads: '5.2M+ weekly',
      category: 'Key Derivation Function (KDF)',
      whatItMeans: 'Standard password key derivation',
      stdlib: 'crypto/hmac, crypto/sha256, encoding/binary',
      desc: 'Hand-rolled the official RFC 8018 password algorithm with 600,000 security rounds so your master password cannot be brute-forced by GPUs.',
      file: 'pkg/vault/kdf.go',
    },
    {
      name: 'bubbletea / tview',
      downloads: '1.5M+ weekly',
      category: 'Terminal UI Engine',
      whatItMeans: 'Interactive visual dashboard',
      stdlib: 'syscall.SYS_IOCTL (termios), bytes.Buffer, 2D Canvas',
      desc: 'Built a lightweight 2D double-buffered terminal interface from scratch using standard operating system calls, without heavy UI framework packages.',
      file: 'pkg/tui/',
    },
    {
      name: 'fatih/color / chalk',
      downloads: '300M+ weekly',
      category: 'Terminal Styling & Colors',
      whatItMeans: 'Clean, accessible terminal colors',
      stdlib: 'Raw ANSI SGR builders, os.Getenv("NO_COLOR")',
      desc: 'Built-in 16-color ANSI sequence builder with bold, dim, and underline modes that automatically respects accessibility and NO_COLOR rules.',
      file: 'pkg/tui/terminal/terminal.go',
    },
    {
      name: 'atotto/clipboard',
      downloads: '800k+ weekly',
      category: 'System Clipboard Controller',
      whatItMeans: 'One-key secret copying',
      stdlib: 'encoding/base64, ANSI OSC 52 escape sequences',
      desc: 'Uses native terminal clipboard escape codes (supported in iTerm2, Alacritty, Kitty, Windows Terminal, VS Code) to copy secrets safely without native plugins.',
      file: 'pkg/tui/terminal/clipboard.go',
    },
    {
      name: 'trufflehog / gitleaks',
      downloads: 'Multi-tool staple',
      category: 'Plaintext Credential Scanner',
      whatItMeans: 'Finds accidental password leaks',
      stdlib: 'path/filepath.WalkDir, regexp, bufio.Scanner',
      desc: 'Built-in code crawler that scans your project for accidentally hardcoded API keys or unencrypted .env files, respecting your .mayflyignore file.',
      file: 'pkg/scanner/scanner.go',
    },
  ];

  const useCases = [
    {
      icon: <Code className="size-4.5 text-amber-400" />,
      badge: 'Web & Backend Apps',
      title: 'Local Development & Hot-Reload',
      description:
        'Run Next.js, Vite, Django, Express, or FastAPI with secrets injected strictly into memory. When the dev server shuts down, volatile RAM is zeroed.',
      command: 'mf npm run dev',
      link: '/docs/guides/nodejs',
    },
    {
      icon: <ShieldAlert className="size-4.5 text-red-400" />,
      badge: 'Supply-Chain Defense',
      title: 'Untrusted Package Installs',
      description:
        'Protect against malicious npm postinstall or Python setup scripts that scan disks for .env files and exfiltrate API keys before you even launch your app.',
      command: 'npm install (0 plaintext .env on disk)',
      link: '/docs/why-mayfly',
    },
    {
      icon: <GitBranch className="size-4.5 text-emerald-400" />,
      badge: 'DevOps & CI/CD',
      title: 'CI Test Runners & Docker',
      description:
        'Pass credentials to ephemeral CI test runners and containerized builds without burning plaintext API tokens into disk caches or image layers.',
      command: 'mf docker compose up',
      link: '/docs/guides/docker',
    },
    {
      icon: <FolderGit2 className="size-4.5 text-purple-400" />,
      badge: 'Hardware Inode Binding',
      title: 'Multi-Project & Monorepo Isolation',
      description:
        'Seamlessly switch between multiple microservices or client repositories. Secrets automatically bind to the folder inode without manual profile switching.',
      command: 'cd ../api && mf run ./server',
      link: '/docs/concepts',
    },
    {
      icon: <Search className="size-4.5 text-sky-400" />,
      badge: 'Codebase Auditing',
      title: 'Plaintext Credential Crawling',
      description:
        'Scan your entire codebase, configuration files, and legacy projects for hardcoded tokens, OpenAI keys, or orphaned .env files in milliseconds.',
      command: 'mf scan',
      link: '/docs/cli/scan',
    },
    {
      icon: <WifiOff className="size-4.5 text-amber-400" />,
      badge: '100% Offline & Air-Gapped',
      title: 'Air-Gapped Workstations & Flights',
      description:
        'No cloud accounts, no API rate limits, and zero internet requirement. Work securely on airplanes, remote locations, and strict air-gapped networks.',
      command: 'mf (Works with 0 network calls)',
      link: '/docs/reference/zero-dependency-audit',
    },
  ];

  return (
    <div className="relative min-h-screen bg-[#0d100f] text-neutral-100 selection:bg-amber-950/60 selection:text-amber-200 font-sans">
      {/* Warm Ambient Radial Glows */}
      <div className="absolute inset-0 bg-[radial-gradient(ellipse_80%_60%_at_50%_-20%,rgba(245,158,11,0.12),transparent_70%)] pointer-events-none" />
      <div className="absolute top-[600px] left-1/2 -translate-x-1/2 w-[700px] h-[350px] bg-amber-500/5 blur-[140px] pointer-events-none rounded-full" />

      {/* Navigation Header */}
      <header className="sticky top-0 z-50 border-b border-white/[0.07] bg-[#0a0d0c]/90 backdrop-blur-xl">
        <div className="mx-auto flex h-16 max-w-6xl items-center justify-between px-4 sm:px-6">
          <div className="flex items-center gap-3">
            <Link href="/" className="flex items-center gap-2.5 group" id="nav-brand-logo">
              <img
                src="/icon.png"
                alt="MayFly Logo"
                width={26}
                height={26}
                className="size-6 object-contain shrink-0 group-hover:scale-105 transition-transform"
              />
              <span className="font-mono text-sm font-bold tracking-widest text-neutral-100">MAYFLY</span>
            </Link>
          </div>

          {/* Desktop Navigation Links */}
          <nav className="hidden md:flex items-center gap-5 lg:gap-7 text-sm font-subtext font-normal" aria-label="Main Navigation">
            <Link href="/docs/why-mayfly" className="text-neutral-400 transition-colors hover:text-amber-400">
              Why MayFly
            </Link>
            <Link href="#use-cases" className="text-neutral-400 transition-colors hover:text-amber-400">
              Use Cases
            </Link>
            <Link href="/docs/quickstart" className="text-neutral-400 transition-colors hover:text-amber-400">
              Quickstart
            </Link>
            <Link href="/docs/concepts" className="text-neutral-400 transition-colors hover:text-amber-400">
              How It Works
            </Link>
            <Link href="/docs/architecture/security-model" className="text-neutral-400 transition-colors hover:text-amber-400">
              Security Model
            </Link>
            <Link href="/docs/reference/zero-dependency-audit" className="text-neutral-400 transition-colors hover:text-amber-400">
              Zero-Dep Audit
            </Link>
            <Link href="/docs/cli/overview" className="text-neutral-400 transition-colors hover:text-amber-400">
              CLI
            </Link>
          </nav>

          {/* Right Action Buttons */}
          <div className="flex items-center gap-2 sm:gap-3">
            <Link
              href="/docs"
              id="header-docs-btn"
              className="inline-flex items-center gap-1.5 rounded-lg border border-white/[0.08] bg-white/[0.03] px-3 sm:px-3.5 py-1.5 text-xs font-subtext font-medium text-neutral-300 transition-colors hover:border-amber-800/50 hover:bg-white/[0.06] hover:text-neutral-100"
            >
              <BookOpen className="size-3.5 text-amber-400" />
              <span>Docs</span>
            </Link>
            <a
              href="https://github.com/vishnunandan555/mayfly"
              target="_blank"
              rel="noopener noreferrer"
              id="header-github-btn"
              className="hidden sm:inline-flex items-center gap-1.5 rounded-lg bg-amber-400 px-3.5 py-1.5 text-xs font-subtext font-semibold text-neutral-950 transition-all hover:bg-amber-300 shadow-[0_0_15px_rgba(245,158,11,0.25)]"
            >
              <span>GitHub</span>
              <ExternalLink className="size-3 text-neutral-950" />
            </a>

            {/* Mobile Hamburger Toggle Button */}
            <button
              onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
              id="mobile-menu-toggle-btn"
              aria-label={mobileMenuOpen ? 'Close navigation menu' : 'Open navigation menu'}
              aria-expanded={mobileMenuOpen}
              className="md:hidden inline-flex size-9 items-center justify-center rounded-lg border border-white/[0.08] bg-white/[0.04] text-neutral-300 hover:text-amber-400 hover:border-amber-800/50 transition-colors"
            >
              {mobileMenuOpen ? <X className="size-5" /> : <Menu className="size-5" />}
            </button>
          </div>
        </div>

        {/* Mobile Navigation Drawer / Dropdown */}
        {mobileMenuOpen && (
          <div className="md:hidden border-b border-white/[0.08] bg-[#0a0d0c]/98 backdrop-blur-2xl px-5 py-6 space-y-4 animate-in fade-in slide-in-from-top-3 duration-200">
            <nav className="flex flex-col space-y-3 font-subtext text-sm">
              <Link
                href="/docs/why-mayfly"
                onClick={() => setMobileMenuOpen(false)}
                className="flex items-center justify-between py-2 text-neutral-300 hover:text-amber-400 border-b border-white/[0.04]"
              >
                <span>Why MayFly</span>
                <ChevronRight className="size-4 text-neutral-600" />
              </Link>
              <Link
                href="#use-cases"
                onClick={() => setMobileMenuOpen(false)}
                className="flex items-center justify-between py-2 text-neutral-300 hover:text-amber-400 border-b border-white/[0.04]"
              >
                <span>Use Cases &amp; Workflows</span>
                <ChevronRight className="size-4 text-neutral-600" />
              </Link>
              <Link
                href="/docs/quickstart"
                onClick={() => setMobileMenuOpen(false)}
                className="flex items-center justify-between py-2 text-neutral-300 hover:text-amber-400 border-b border-white/[0.04]"
              >
                <span>Quickstart (30-Sec Guide)</span>
                <ChevronRight className="size-4 text-neutral-600" />
              </Link>
              <Link
                href="/docs/concepts"
                onClick={() => setMobileMenuOpen(false)}
                className="flex items-center justify-between py-2 text-neutral-300 hover:text-amber-400 border-b border-white/[0.04]"
              >
                <span>How It Works (Concepts)</span>
                <ChevronRight className="size-4 text-neutral-600" />
              </Link>
              <Link
                href="/docs/architecture/security-model"
                onClick={() => setMobileMenuOpen(false)}
                className="flex items-center justify-between py-2 text-neutral-300 hover:text-amber-400 border-b border-white/[0.04]"
              >
                <span>Security &amp; Cryptography Model</span>
                <ChevronRight className="size-4 text-neutral-600" />
              </Link>
              <Link
                href="/docs/reference/zero-dependency-audit"
                onClick={() => setMobileMenuOpen(false)}
                className="flex items-center justify-between py-2 text-neutral-300 hover:text-amber-400 border-b border-white/[0.04]"
              >
                <span>Zero-Dep Audit Proof</span>
                <ChevronRight className="size-4 text-neutral-600" />
              </Link>
              <Link
                href="/docs/cli/overview"
                onClick={() => setMobileMenuOpen(false)}
                className="flex items-center justify-between py-2 text-neutral-300 hover:text-amber-400"
              >
                <span>CLI Reference &amp; Options</span>
                <ChevronRight className="size-4 text-neutral-600" />
              </Link>
            </nav>

            <div className="pt-3 border-t border-white/[0.08] flex items-center gap-3">
              <Link
                href="/docs"
                onClick={() => setMobileMenuOpen(false)}
                className="flex-1 inline-flex items-center justify-center gap-2 rounded-lg border border-white/[0.1] bg-white/[0.05] py-2.5 text-xs font-subtext font-semibold text-neutral-200 hover:bg-white/[0.1]"
              >
                <BookOpen className="size-4 text-amber-400" />
                <span>Documentation</span>
              </Link>
              <a
                href="https://github.com/vishnunandan555/mayfly"
                target="_blank"
                rel="noopener noreferrer"
                className="flex-1 inline-flex items-center justify-center gap-2 rounded-lg bg-amber-400 py-2.5 text-xs font-subtext font-semibold text-neutral-950 hover:bg-amber-300"
              >
                <span>GitHub</span>
                <ExternalLink className="size-3.5 text-neutral-950" />
              </a>
            </div>
          </div>
        )}
      </header>

      {/* Main Hero Section */}
      <main className="relative mx-auto max-w-6xl px-4 sm:px-6 pt-10 sm:pt-16 pb-20" id="main-content">
        <div className="grid grid-cols-1 lg:grid-cols-12 gap-8 lg:gap-8 items-stretch">
          
          {/* Left Column: Headline, Value Proposition, Badges, Quick Install */}
          <section className="lg:col-span-6 flex flex-col justify-between" aria-labelledby="hero-title">
            <div>
              {/* Main Headline */}
              <h1 id="hero-title" className="font-serif-display text-3xl sm:text-4xl md:text-5xl lg:text-[3.35rem] font-normal tracking-tight text-neutral-100 leading-[1.15] sm:leading-[1.1]">
                Never store passwords in <span className="text-amber-400 italic">.env</span> files again.
              </h1>

              {/* Plain English Description */}
              <p className="mt-4 sm:mt-5 text-sm sm:text-base text-neutral-300 font-subtext leading-relaxed font-normal">
                MayFly keeps your API keys in an encrypted local vault. When you start your app (<code className="text-amber-300 font-mono text-xs font-semibold">mf npm run dev</code>), MayFly injects them straight into memory and wipes them when you finish.
              </p>
              <p className="mt-2 text-xs sm:text-sm text-neutral-400 font-subtext leading-relaxed">
                Zero plaintext files on your hard drive, zero cloud dependencies, and zero third-party packages.
              </p>

              {/* Badges Row */}
              <div className="mt-5 sm:mt-6 flex flex-wrap items-center gap-2">
                <span className="inline-flex items-center gap-1.5 rounded-full border border-amber-900/60 bg-amber-950/40 px-3 py-1 font-mono text-[11px] text-amber-400">
                  <span className="size-1.5 rounded-full bg-amber-400" />
                  100% Go Standard Library
                </span>
                <span className="rounded-full border border-purple-900/60 bg-purple-950/40 px-3 py-1 font-mono text-[11px] text-purple-300">
                  AES-256-GCM + PBKDF2
                </span>
                <span className="rounded-full border border-emerald-900/60 bg-emerald-950/40 px-3 py-1 font-mono text-[11px] text-emerald-300">
                  Deterministic Build
                </span>
                <span className="rounded-full border border-white/[0.08] bg-white/[0.03] px-3 py-1 font-mono text-[11px] text-neutral-400">
                  AGPL-3.0 License
                </span>
              </div>
            </div>

            {/* Bottom Install Card of Left Column */}
            <div className="mt-8 pt-6 border-t border-white/[0.08]">
              {/* Quick Install Switcher */}
              <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2.5 mb-3">
                <div className="flex items-center gap-2">
                  <span className="font-mono text-[11px] font-semibold text-amber-400 tracking-wider">INSTALL</span>
                  <div className="flex items-center gap-1 bg-[#131715] p-0.5 rounded-full border border-white/[0.06] overflow-x-auto scrollbar-none" role="tablist">
                    <button
                      onClick={() => setInstallOs('unix')}
                      id="install-unix-tab"
                      role="tab"
                      aria-selected={installOs === 'unix'}
                      className={`rounded-full px-3 py-1 text-xs font-subtext transition-all whitespace-nowrap ${
                        installOs === 'unix'
                          ? 'bg-amber-400 text-neutral-950 font-semibold shadow-sm'
                          : 'text-neutral-400 hover:text-neutral-200'
                      }`}
                    >
                      macOS &amp; Linux
                    </button>
                    <button
                      onClick={() => setInstallOs('windows')}
                      id="install-windows-tab"
                      role="tab"
                      aria-selected={installOs === 'windows'}
                      className={`rounded-full px-3 py-1 text-xs font-subtext transition-all whitespace-nowrap ${
                        installOs === 'windows'
                          ? 'bg-amber-400 text-neutral-950 font-semibold shadow-sm'
                          : 'text-neutral-400 hover:text-neutral-200'
                      }`}
                    >
                      Windows
                    </button>
                    <button
                      onClick={() => setInstallOs('source')}
                      id="install-source-tab"
                      role="tab"
                      aria-selected={installOs === 'source'}
                      className={`rounded-full px-3 py-1 text-xs font-subtext transition-all whitespace-nowrap ${
                        installOs === 'source'
                          ? 'bg-amber-400 text-neutral-950 font-semibold shadow-sm'
                          : 'text-neutral-400 hover:text-neutral-200'
                      }`}
                    >
                      From Source
                    </button>
                  </div>
                </div>

                {/* Quickstart Jump Link */}
                <Link
                  href="/docs/quickstart"
                  className="hidden sm:inline-flex items-center gap-1 text-xs font-subtext font-medium text-amber-400 hover:text-amber-300 transition-colors"
                >
                  <span>30-Sec Guide</span>
                  <ArrowRight className="size-3.5" />
                </Link>
              </div>

              {/* Install Command Box */}
              <div className="relative rounded-xl border border-white/[0.08] bg-[#131715] p-3.5 sm:p-4 font-mono text-xs text-neutral-300">
                <div className="text-[10px] text-neutral-500 mb-1.5 font-subtext uppercase tracking-wider">
                  {installOs === 'unix'
                    ? 'Works on bash, zsh, and fish (installs mayfly and mf to ~/.local/bin)'
                    : installOs === 'windows'
                    ? 'PowerShell installer (sets PATH automatically)'
                    : 'Zero external dependencies • Builds static reproducible binary'}
                </div>
                <div className="flex items-center justify-between gap-3">
                  <div className="overflow-x-auto scrollbar-none text-amber-200 font-medium whitespace-nowrap pr-2">
                    <span className="text-neutral-600 mr-2 select-none">{installOs === 'windows' ? '>' : '$'}</span>
                    {installCommands[installOs]}
                  </div>
                  <button
                    onClick={() => copyToClipboard(installCommands[installOs], setCopiedInstall)}
                    className="inline-flex size-8 shrink-0 items-center justify-center rounded-lg border border-white/[0.08] bg-white/[0.04] text-neutral-400 hover:border-amber-800/60 hover:text-amber-300 transition-colors active:scale-95"
                    title="Copy install command"
                    aria-label="Copy install command"
                  >
                    {copiedInstall ? <Check className="size-3.5 text-amber-400" /> : <Copy className="size-3.5" />}
                  </button>
                </div>
              </div>

              {/* Platform Note */}
              <div className="mt-3 flex items-center justify-between text-xs text-neutral-500 font-subtext">
                <span>Single ~12MB standalone binary. Gives you both <code className="text-neutral-300 font-mono">mayfly</code> and <code className="text-amber-400 font-mono font-bold">mf</code>.</span>
                <Link href="/docs/quickstart" className="sm:hidden text-amber-400 font-medium whitespace-nowrap ml-2">
                  Quickstart &rarr;
                </Link>
              </div>
            </div>
          </section>

          {/* Right Column: Interactive Live Terminal Preview */}
          <section className="lg:col-span-6 lg:pl-2 flex flex-col" aria-label="Interactive Terminal Preview">
            <div className="h-full rounded-2xl border border-white/[0.08] bg-[#131715] p-4 sm:p-5 font-mono text-xs shadow-2xl ring-1 ring-amber-900/20 flex flex-col justify-between">
              
              {/* Terminal Top Bar with Non-Wrapping Scrollable Tabs */}
              <div>
                <div className="flex items-center justify-between pb-3.5 mb-3.5 border-b border-white/[0.06] gap-2">
                  <div className="flex items-center gap-1.5 shrink-0">
                    <span className="size-2.5 rounded-full bg-[#ff5f56]" />
                    <span className="size-2.5 rounded-full bg-[#ffbd2e]" />
                    <span className="size-2.5 rounded-full bg-[#27c93f]" />
                    <span className="ml-2 text-[11px] text-neutral-500 hidden sm:inline">~/my-project</span>
                  </div>
                  
                  {/* Clean Workflow Switcher Tabs */}
                  <div className="flex items-center gap-1 bg-[#0d100f] p-0.5 rounded-lg border border-white/[0.06] overflow-x-auto scrollbar-none max-w-full" role="tablist">
                    <button
                      onClick={() => setTerminalTab('injection')}
                      id="term-tab-injection"
                      role="tab"
                      aria-selected={terminalTab === 'injection'}
                      className={`px-2.5 py-1 rounded text-[11px] font-subtext transition-colors whitespace-nowrap ${
                        terminalTab === 'injection'
                          ? 'bg-amber-400/15 text-amber-400 font-semibold border border-amber-500/40'
                          : 'text-neutral-400 hover:text-neutral-200'
                      }`}
                    >
                      Run App
                    </button>
                    <button
                      onClick={() => setTerminalTab('tui')}
                      id="term-tab-tui"
                      role="tab"
                      aria-selected={terminalTab === 'tui'}
                      className={`px-2.5 py-1 rounded text-[11px] font-subtext transition-colors whitespace-nowrap ${
                        terminalTab === 'tui'
                          ? 'bg-amber-400/15 text-amber-400 font-semibold border border-amber-500/40'
                          : 'text-neutral-400 hover:text-neutral-200'
                      }`}
                    >
                      Visual TUI
                    </button>
                    <button
                      onClick={() => setTerminalTab('scanner')}
                      id="term-tab-scanner"
                      role="tab"
                      aria-selected={terminalTab === 'scanner'}
                      className={`px-2.5 py-1 rounded text-[11px] font-subtext transition-colors whitespace-nowrap ${
                        terminalTab === 'scanner'
                          ? 'bg-amber-400/15 text-amber-400 font-semibold border border-amber-500/40'
                          : 'text-neutral-400 hover:text-neutral-200'
                      }`}
                    >
                      Leak Finder
                    </button>
                    <button
                      onClick={() => setTerminalTab('audit')}
                      id="term-tab-audit"
                      role="tab"
                      aria-selected={terminalTab === 'audit'}
                      className={`px-2.5 py-1 rounded text-[11px] font-subtext transition-colors whitespace-nowrap ${
                        terminalTab === 'audit'
                          ? 'bg-amber-400/15 text-amber-400 font-semibold border border-amber-500/40'
                          : 'text-neutral-400 hover:text-neutral-200'
                      }`}
                    >
                      Audit Chain
                    </button>
                  </div>
                </div>

                {/* Tab 1: RAM Injection Simulation */}
                {terminalTab === 'injection' && (
                  <div className="space-y-3 text-left text-neutral-300 leading-relaxed min-h-[280px]">
                    <div>
                      <div className="flex items-center gap-2">
                        <span className="text-amber-400 font-bold select-none">$</span>
                        <span className="text-neutral-100 font-semibold break-all">mf set STRIPE_SECRET=sk_live_51Msz...</span>
                      </div>
                      <div className="text-amber-400/90 text-[11px] pl-4 mt-0.5">
                        [saved] Secret STRIPE_SECRET encrypted to vault
                      </div>
                    </div>

                    <div className="pt-1">
                      <div className="flex items-center gap-2">
                        <span className="text-amber-400 font-bold select-none">$</span>
                        <span className="text-neutral-100 font-semibold">mf npm run dev</span>
                      </div>
                      
                      <div className="border-l-2 border-amber-500 pl-3 py-1 my-1.5 bg-amber-950/20 rounded-r text-[11px] space-y-0.5">
                        <div className="text-amber-400 font-semibold">[mayfly] unlocked vault in memory (aes-256-gcm)</div>
                        <div className="text-neutral-400">[mayfly] injected 3 secret(s) directly into process RAM</div>
                      </div>

                      <div className="pl-4 text-[11px] space-y-0.5 text-neutral-400">
                        <div className="text-neutral-300">&gt; my-app@0.1.0 dev</div>
                        <div>&gt; next dev</div>
                        <div className="text-neutral-300 font-medium">   ▲ Next.js 15.5.24</div>
                        <div>   - Local:        http://localhost:3000</div>
                        <div className="text-emerald-400 font-medium">   [ready] in 1.4s (Secrets loaded from RAM)</div>
                      </div>
                    </div>

                    <div className="pt-2 border-t border-white/[0.04] space-y-0.5">
                      <div className="flex items-center gap-2">
                        <span className="text-amber-400 font-bold select-none">$</span>
                        <span className="text-neutral-100">cat .env</span>
                      </div>
                      <div className="text-red-400 text-[11px] pl-4">
                        cat: .env: No such file or directory (Protected against disk scrapers)
                      </div>
                    </div>
                  </div>
                )}

                {/* Tab 2: Visual TUI Dashboard Simulation */}
                {terminalTab === 'tui' && (
                  <div className="space-y-2 text-left text-neutral-300 leading-tight min-h-[280px]">
                    <div className="flex items-center justify-between text-neutral-400 text-[11px] pb-1 border-b border-white/[0.04]">
                      <span className="text-amber-400 font-bold">MAYFLY INTERACTIVE DASHBOARD</span>
                      <span className="text-[10px] text-neutral-500 hidden sm:inline">Run &apos;mf&apos; from anywhere</span>
                    </div>

                    <div className="grid grid-cols-1 sm:grid-cols-2 gap-2 my-2">
                      <div className="rounded border border-amber-500/60 bg-amber-950/20 p-2 text-[10px]">
                        <div className="text-amber-300 font-bold flex items-center justify-between">
                          <span>[current] my-project</span>
                          <span className="text-emerald-400">ACTIVE</span>
                        </div>
                        <div className="text-neutral-400 mt-1">3 configured secrets</div>
                        <div className="text-neutral-500 text-[9px] truncate">Inode: #19842013</div>
                      </div>
                      <div className="rounded border border-white/[0.06] bg-[#0d100f] p-2 text-[10px]">
                        <div className="text-neutral-300 font-bold">backend-api</div>
                        <div className="text-neutral-400 mt-1">5 configured secrets</div>
                        <div className="text-neutral-500 text-[9px] truncate">Inode: #10948271</div>
                      </div>
                    </div>

                    <div className="rounded border border-white/[0.06] bg-[#0d100f] p-2.5 space-y-1.5 text-[11px]">
                      <div className="text-neutral-400 text-[10px] uppercase tracking-wider font-semibold">Secrets in Active Project:</div>
                      <div className="flex justify-between items-center text-neutral-200">
                        <span className="text-amber-300 font-mono">DATABASE_URL</span>
                        <span className="text-neutral-500 font-mono">••••••••••••••••••••</span>
                      </div>
                      <div className="flex justify-between items-center text-neutral-200">
                        <span className="text-amber-300 font-mono">STRIPE_SECRET</span>
                        <span className="text-neutral-500 font-mono">••••••••••••••••••••</span>
                      </div>
                      <div className="flex justify-between items-center text-neutral-200">
                        <span className="text-amber-300 font-mono">OPENAI_KEY</span>
                        <span className="text-neutral-500 font-mono">••••••••••••••••••••</span>
                      </div>
                    </div>

                    <div className="pt-2 text-[10px] text-neutral-400 flex flex-wrap gap-2 border-t border-white/[0.04]">
                      <span className="text-amber-400">[Enter] Open</span>
                      <span className="text-amber-400">[V] Reveal</span>
                      <span className="text-amber-400">[C] Copy</span>
                      <span className="text-amber-400">[N] New</span>
                      <span className="text-amber-400">[S] Scan</span>
                      <span className="text-amber-400">[Q] Exit</span>
                    </div>
                  </div>
                )}

                {/* Tab 3: Plaintext Leak Scanner */}
                {terminalTab === 'scanner' && (
                  <div className="space-y-3 text-left text-neutral-300 leading-relaxed min-h-[280px]">
                    <div>
                      <div className="flex items-center gap-2">
                        <span className="text-amber-400 font-bold select-none">$</span>
                        <span className="text-neutral-100 font-semibold">mf scan</span>
                      </div>
                      <div className="text-neutral-400 text-[11px] pl-4 mt-0.5">
                        Checking project files for accidental plaintext credentials...
                      </div>
                    </div>

                    <div className="rounded-lg border border-red-900/40 bg-red-950/20 p-3 space-y-2 text-[11px]">
                      <div className="flex items-center gap-2 text-red-400 font-semibold">
                        <AlertTriangle className="size-3.5 shrink-0" />
                        <span>[CRITICAL LEAK] Unencrypted .env file found on disk</span>
                      </div>
                      <div className="pl-5 text-neutral-300 font-mono">
                        File: <code className="text-red-300 bg-red-950/40 px-1 py-0.5 rounded">.env</code> (Line 1)
                        <div className="text-neutral-400 text-[10px] mt-0.5">Warning: Malicious packages can read this file during install.</div>
                      </div>
                    </div>

                    <div className="rounded-lg border border-amber-900/40 bg-amber-950/20 p-3 space-y-1.5 text-[11px]">
                      <div className="flex items-center gap-2 text-amber-400 font-semibold">
                        <KeyRound className="size-3.5 shrink-0" />
                        <span>[WARNING] Hardcoded API key in code:</span>
                      </div>
                      <div className="pl-5 text-neutral-300 font-mono text-[10px] break-all">
                        File: <code className="text-amber-200">src/services/api.ts:24</code> (Stripe Key pattern)
                      </div>
                    </div>

                    <div className="pl-4 text-[11px] text-emerald-400 font-medium">
                      Action: Run <code className="bg-emerald-950/40 px-1 py-0.5 rounded">mf import .env &amp;&amp; rm .env</code> to encrypt.
                    </div>
                  </div>
                )}

                {/* Tab 4: Cryptographic Audit Chain */}
                {terminalTab === 'audit' && (
                  <div className="space-y-3 text-left text-neutral-300 leading-relaxed min-h-[280px]">
                    <div>
                      <div className="flex items-center gap-2">
                        <span className="text-amber-400 font-bold select-none">$</span>
                        <span className="text-neutral-100 font-semibold">mf audit verify</span>
                      </div>
                      <div className="text-emerald-400 text-[11px] pl-4 mt-0.5 font-medium">
                        [ok] Audit log hash chain verified intact (42 events checked).
                      </div>
                    </div>

                    <div className="rounded-lg border border-white/[0.06] bg-[#0d100f] p-3 space-y-2 text-[10px] font-mono">
                      <div className="text-neutral-400 uppercase tracking-wider text-[9px] font-semibold">Cryptographic Event History:</div>
                      <div className="border-l-2 border-emerald-500 pl-2 py-0.5 bg-white/[0.02] text-neutral-300 truncate">
                        #40 2026-08-31T05:12:01Z SET project=my-project key=STRIPE_KEY hash=9f8c1a...
                      </div>
                      <div className="border-l-2 border-emerald-500 pl-2 py-0.5 bg-white/[0.02] text-neutral-300 truncate">
                        #41 2026-08-31T05:13:45Z GET project=my-project key=DATABASE_URL hash=b2e4d0...
                      </div>
                      <div className="border-l-2 border-emerald-500 pl-2 py-0.5 bg-white/[0.02] text-neutral-300 truncate">
                        #42 2026-08-31T05:14:00Z EXEC project=my-project cmd=&quot;npm dev&quot; hash=4a1e7e...
                      </div>
                    </div>

                    <div className="text-[11px] text-neutral-400 pl-4">
                      Every event is mathematically linked to the previous one using SHA-256. If anyone tampers with a log line, verification fails.
                    </div>
                  </div>
                )}
              </div>

              <div className="pt-2.5 border-t border-white/[0.04] text-[10px] text-neutral-500 flex items-center justify-between font-subtext">
                <span>Direct OS In-Memory Injection</span>
                <span className="text-amber-400/80">AES-256-GCM + PBKDF2</span>
              </div>
            </div>
          </section>
        </div>

        {/* Section 2: Quick Start Steps */}
        <section className="mt-16 sm:mt-28" id="quickstart" aria-labelledby="quickstart-heading">
          <div className="rounded-2xl border border-white/[0.08] bg-[#131715] p-5 sm:p-10 relative overflow-hidden">
            
            <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
              <div className="max-w-2xl">
                <span className="font-mono text-xs font-semibold text-amber-400 tracking-wider block mb-1.5 uppercase">
                  Simple 4-Step Workflow
                </span>
                <h2 id="quickstart-heading" className="font-serif-display text-2xl sm:text-3xl font-normal text-neutral-100">
                  How to use MayFly in 30 seconds.
                </h2>
                <p className="mt-1 text-sm text-neutral-400 font-subtext leading-relaxed">
                  Initialize your folder, save your secrets, run your app, and verify the tamper-proof security log.
                </p>
              </div>

              <Link
                href="/docs/quickstart"
                id="quickstart-full-guide-btn"
                className="inline-flex items-center gap-2 rounded-lg bg-amber-400 px-4 py-2 text-xs font-subtext font-semibold text-neutral-950 hover:bg-amber-300 transition-colors shrink-0 shadow-sm"
              >
                <span>Full Quickstart Guide</span>
                <ArrowRight className="size-3.5" />
              </Link>
            </div>

            <div className="mt-8 grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 sm:gap-5 relative">
              
              <article className="flex flex-col justify-between rounded-xl border border-white/[0.06] bg-[#0d100f] p-4 sm:p-5">
                <div>
                  <div className="flex items-center gap-2.5 mb-2.5">
                    <span className="size-6 rounded-full border border-amber-500/40 bg-amber-950/40 flex items-center justify-center font-mono text-xs font-bold text-amber-400">
                      1
                    </span>
                    <h3 className="text-sm font-semibold text-neutral-100 font-subtext">Initialize Vault</h3>
                  </div>
                  <p className="text-xs text-neutral-400 font-subtext leading-relaxed mb-3.5">
                    Binds the current folder to its own private, encrypted key space.
                  </p>
                </div>
                <div className="rounded bg-[#171c1a] px-3 py-2 font-mono text-xs text-amber-300">
                  <span className="text-neutral-600 mr-1.5">$</span>mf init
                </div>
              </article>

              <article className="flex flex-col justify-between rounded-xl border border-white/[0.06] bg-[#0d100f] p-4 sm:p-5">
                <div>
                  <div className="flex items-center gap-2.5 mb-2.5">
                    <span className="size-6 rounded-full border border-amber-500/40 bg-amber-950/40 flex items-center justify-center font-mono text-xs font-bold text-amber-400">
                      2
                    </span>
                    <h3 className="text-sm font-semibold text-neutral-100 font-subtext">Save Secrets</h3>
                  </div>
                  <p className="text-xs text-neutral-400 font-subtext leading-relaxed mb-3.5">
                    Encrypts your variable with AES-256-GCM without writing plaintext to disk.
                  </p>
                </div>
                <div className="rounded bg-[#171c1a] px-3 py-2 font-mono text-xs text-amber-300">
                  <span className="text-neutral-600 mr-1.5">$</span>mf set STRIPE_KEY
                </div>
              </article>

              <article className="flex flex-col justify-between rounded-xl border border-white/[0.06] bg-[#0d100f] p-4 sm:p-5">
                <div>
                  <div className="flex items-center gap-2.5 mb-2.5">
                    <span className="size-6 rounded-full border border-amber-500/40 bg-amber-950/40 flex items-center justify-center font-mono text-xs font-bold text-amber-400">
                      3
                    </span>
                    <h3 className="text-sm font-semibold text-neutral-100 font-subtext">Run Your App</h3>
                  </div>
                  <p className="text-xs text-neutral-400 font-subtext leading-relaxed mb-3.5">
                    Spawns your app with secrets injected directly into volatile RAM.
                  </p>
                </div>
                <div className="rounded bg-[#171c1a] px-3 py-2 font-mono text-xs text-amber-300">
                  <span className="text-neutral-600 mr-1.5">$</span>mf npm run dev
                </div>
              </article>

              <article className="flex flex-col justify-between rounded-xl border border-white/[0.06] bg-[#0d100f] p-4 sm:p-5">
                <div>
                  <div className="flex items-center gap-2.5 mb-2.5">
                    <span className="size-6 rounded-full border border-amber-500/40 bg-amber-950/40 flex items-center justify-center font-mono text-xs font-bold text-amber-400">
                      4
                    </span>
                    <h3 className="text-sm font-semibold text-neutral-100 font-subtext">Verify Audit</h3>
                  </div>
                  <p className="text-xs text-neutral-400 font-subtext leading-relaxed mb-3.5">
                    Mathematically validates that no security log entries were modified.
                  </p>
                </div>
                <div className="rounded bg-[#171c1a] px-3 py-2 font-mono text-xs text-amber-300">
                  <span className="text-neutral-600 mr-1.5">$</span>mf audit verify
                </div>
              </article>
            </div>

            <div className="mt-7 pt-5 border-t border-white/[0.06] flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3 text-xs font-subtext">
              <div className="text-neutral-400">
                <span className="font-mono text-amber-400 font-semibold uppercase mr-2 text-[10px]">VISUAL DASHBOARD</span>
                Prefer a full-screen interactive terminal dashboard? Just run:
              </div>
              <div className="rounded-lg bg-[#0d100f] px-3 py-1.5 font-mono text-xs text-amber-300 border border-white/[0.06]">
                <span className="text-neutral-600 mr-1.5">$</span>mf
              </div>
            </div>
          </div>
        </section>

        {/* Section 3: Practical Developer Use Cases */}
        <section className="mt-16 sm:mt-28" id="use-cases" aria-labelledby="usecases-heading">
          <div className="text-center max-w-2xl mx-auto mb-8 sm:mb-10">
            <span className="font-mono text-xs font-semibold text-amber-400 tracking-wider block mb-1.5 uppercase">
              Developer Workflows
            </span>
            <h2 id="usecases-heading" className="font-serif-display text-3xl sm:text-4xl font-normal text-neutral-100">
              Built for real-world scenarios.
            </h2>
            <p className="mt-2 text-sm text-neutral-400 font-subtext leading-relaxed">
              How modern engineering teams and independent developers use MayFly to secure their environments.
            </p>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 sm:gap-5">
            {useCases.map((uc) => (
              <article
                key={uc.title}
                className="group rounded-2xl border border-white/[0.08] bg-[#131715] p-5 sm:p-6 flex flex-col justify-between transition-all duration-200 hover:border-amber-900/50 hover:bg-[#161c19]"
              >
                <div>
                  <div className="flex items-center justify-between gap-2 mb-3">
                    <div className="inline-flex size-9 items-center justify-center rounded-lg border border-white/[0.08] bg-white/[0.02] group-hover:bg-amber-950/30 transition-colors">
                      {uc.icon}
                    </div>
                    <span className="font-mono text-[10px] text-amber-400/90 bg-amber-950/40 px-2 py-0.5 rounded border border-amber-900/40">
                      {uc.badge}
                    </span>
                  </div>

                  <h3 className="font-semibold text-neutral-100 text-sm mb-2 font-subtext">{uc.title}</h3>
                  <p className="text-xs text-neutral-400 font-subtext leading-relaxed mb-4">{uc.description}</p>
                </div>

                <div className="pt-3 border-t border-white/[0.04] flex items-center justify-between gap-2">
                  <code className="text-[11px] font-mono text-amber-300 bg-[#0d100f] px-2 py-1 rounded border border-white/[0.04] truncate">
                    {uc.command}
                  </code>
                  <Link
                    href={uc.link}
                    className="text-neutral-400 group-hover:text-amber-400 transition-colors inline-flex items-center"
                    aria-label={`Read guide for ${uc.title}`}
                  >
                    <ArrowRight className="size-3.5" />
                  </Link>
                </div>
              </article>
            ))}
          </div>
        </section>

        {/* Section 4: The Supply-Chain Threat Explained */}
        <section className="mt-16 sm:mt-28" id="why-mayfly" aria-labelledby="why-heading">
          <div className="text-center max-w-2xl mx-auto mb-8 sm:mb-10">
            <span className="font-mono text-xs font-semibold text-amber-400 tracking-wider block mb-1.5 uppercase">
              The Security Challenge
            </span>
            <h2 id="why-heading" className="font-serif-display text-3xl sm:text-4xl font-normal text-neutral-100">
              Why .env files create supply-chain risk.
            </h2>
            <p className="mt-2 text-sm text-neutral-400 font-subtext leading-relaxed">
              Third-party install scripts execute with local user permissions during build and install steps.
            </p>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-5 sm:gap-6">
            {/* The Plaintext Risk */}
            <article className="rounded-2xl border border-red-950/40 bg-[#131111] p-5 sm:p-8 relative overflow-hidden">
              <div className="flex items-center gap-3 mb-5">
                <div className="size-9 rounded-lg border border-red-900/50 bg-red-950/40 flex items-center justify-center text-red-400 shrink-0">
                  <AlertTriangle className="size-4.5" />
                </div>
                <div>
                  <h3 className="font-semibold text-neutral-100 text-base font-subtext">Plaintext .env Exposure</h3>
                  <p className="text-xs text-neutral-500 font-mono">Traditional Development</p>
                </div>
              </div>

              <ul className="space-y-3 text-sm text-neutral-400 font-subtext">
                <li className="flex items-start gap-2.5">
                  <span className="text-red-400 font-bold select-none shrink-0">[x]</span>
                  <span><strong>Unencrypted on Disk:</strong> Passwords and API keys sit in plain text where local processes can read them.</span>
                </li>
                <li className="flex items-start gap-2.5">
                  <span className="text-red-400 font-bold select-none shrink-0">[x]</span>
                  <span><strong>Install-Time Exfiltration:</strong> Malicious <code className="text-neutral-300 font-mono text-xs">npm postinstall</code> or Python setup scripts can scan drives and exfiltrate credentials before your app runs.</span>
                </li>
                <li className="flex items-start gap-2.5">
                  <span className="text-red-400 font-bold select-none shrink-0">[x]</span>
                  <span><strong>Accidental Git Commits:</strong> An unintended <code className="text-neutral-300 font-mono text-xs">git add .</code> risks exposing production credentials to remote repositories.</span>
                </li>
                <li className="flex items-start gap-2.5">
                  <span className="text-red-400 font-bold select-none shrink-0">[x]</span>
                  <span><strong>No Audit History:</strong> Traditional files provide no tamper-proof record of which process accessed secrets.</span>
                </li>
              </ul>
            </article>

            {/* The MayFly Defense */}
            <article className="rounded-2xl border border-amber-900/40 bg-[#141815] p-5 sm:p-8 relative overflow-hidden">
              <div className="flex items-center gap-3 mb-5">
                <div className="size-9 rounded-lg border border-amber-900/50 bg-amber-950/40 flex items-center justify-center text-amber-400 shrink-0">
                  <Shield className="size-4.5" />
                </div>
                <div>
                  <h3 className="font-semibold text-neutral-100 text-base font-subtext">The MayFly Architecture</h3>
                  <p className="text-xs text-amber-400 font-mono">In-Memory Secret Injection</p>
                </div>
              </div>

              <ul className="space-y-3 text-sm text-neutral-400 font-subtext">
                <li className="flex items-start gap-2.5">
                  <span className="text-amber-400 font-bold select-none shrink-0">[ok]</span>
                  <span><strong>Zero Files on Disk:</strong> Secrets are strictly encrypted with AES-256-GCM at rest. No plaintext file touches disk.</span>
                </li>
                <li className="flex items-start gap-2.5">
                  <span className="text-amber-400 font-bold select-none shrink-0">[ok]</span>
                  <span><strong>Direct RAM Injection:</strong> Secrets reside in volatile process memory while your application is active, and memory buffers are zeroed on exit.</span>
                </li>
                <li className="flex items-start gap-2.5">
                  <span className="text-amber-400 font-bold select-none shrink-0">[ok]</span>
                  <span><strong>Folder Inode Binding:</strong> Secrets are bound to your folder’s filesystem identity, preventing cross-project leaks.</span>
                </li>
                <li className="flex items-start gap-2.5">
                  <span className="text-amber-400 font-bold select-none shrink-0">[ok]</span>
                  <span><strong>Tamper-Proof Audit Trail:</strong> Every secret access is signed into a cryptographic SHA-256 hash chain that cannot be altered.</span>
                </li>
              </ul>
            </article>
          </div>
        </section>

        {/* Section 5: Zero-Dependency Architecture */}
        <section className="mt-16 sm:mt-28" id="package-killer" aria-labelledby="package-killer-heading">
          <div className="text-center max-w-2xl mx-auto mb-8 sm:mb-10">
            <span className="font-mono text-xs font-semibold text-amber-400 tracking-wider uppercase block mb-1.5">
              Zero-Dependency Architecture
            </span>
            <h2 id="package-killer-heading" className="font-serif-display text-3xl sm:text-4xl font-normal text-neutral-100">
              Core Subsystems Implemented in Pure Go
            </h2>
            <p className="mt-2 text-sm text-neutral-400 font-subtext leading-relaxed">
              Every subsystem was implemented from first principles using standard library primitives to eliminate upstream supply-chain risk.
            </p>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 sm:gap-5">
            {substitutions.map((sub) => (
              <article
                key={sub.name}
                className="rounded-2xl border border-white/[0.08] bg-[#131715] p-5 flex flex-col justify-between transition-all duration-200 hover:border-amber-900/50 hover:bg-[#161c19]"
              >
                <div>
                  <div className="flex items-center justify-between gap-2 mb-2">
                    <span className="font-mono text-xs font-bold text-amber-400 bg-amber-950/40 px-2 py-0.5 rounded border border-amber-900/50 truncate">
                      {sub.name}
                    </span>
                    <span className="text-[10px] text-neutral-400 font-mono shrink-0">
                      {sub.downloads}
                    </span>
                  </div>

                  <div className="text-xs font-semibold text-neutral-200 mb-0.5 font-subtext">
                    {sub.category}
                  </div>
                  <div className="text-[11px] text-amber-400/90 font-subtext font-medium mb-2">
                    {sub.whatItMeans}
                  </div>

                  <p className="text-xs text-neutral-400 font-subtext leading-relaxed mb-3 font-normal">
                    {sub.desc}
                  </p>
                </div>

                <div className="pt-3 border-t border-white/[0.04] space-y-1 font-mono text-[10px]">
                  <div className="text-neutral-500">Built with: <span className="text-neutral-300">{sub.stdlib}</span></div>
                  <div className="text-amber-400/80 truncate">Source: <span className="text-neutral-400">{sub.file}</span></div>
                </div>
              </article>
            ))}
          </div>

          <div className="mt-6 text-center">
            <Link
              href="/docs/reference/zero-dependency-audit"
              id="view-full-matrix-btn"
              className="inline-flex items-center gap-2 text-xs font-mono text-amber-400 hover:text-amber-300 transition-colors p-2"
            >
              <span>View the complete 12-entry STDLIB substitution matrix &rarr;</span>
            </Link>
          </div>
        </section>

        {/* Section 6: Cryptographic Build Determinism */}
        <section className="mt-16 sm:mt-28 rounded-2xl border border-emerald-900/40 bg-[#101713] p-6 sm:p-8 relative overflow-hidden" id="reproducible-proof" aria-labelledby="reproducible-heading">
          <div className="flex flex-col lg:flex-row items-start lg:items-center justify-between gap-6">
            <div className="max-w-xl">
              <div className="inline-flex items-center gap-2 rounded-full border border-emerald-900/60 bg-emerald-950/40 px-3 py-1 text-xs font-mono text-emerald-300 mb-3">
                <CheckCircle className="size-3.5" />
                Cryptographic Build Determinism
              </div>
              <h2 id="reproducible-heading" className="font-serif-display text-2xl sm:text-3xl font-normal text-neutral-100">
                100% Deterministic Binaries
              </h2>
              <p className="mt-2 text-xs sm:text-sm text-neutral-300 font-subtext leading-relaxed">
                Build MayFly twice on any machine using <code className="text-emerald-300 font-mono">make reproducible</code> to independently verify byte-identical SHA-256 cryptographic hashes.
              </p>
            </div>

            <div className="w-full lg:w-auto font-mono text-xs">
              <div className="rounded-xl border border-emerald-800/40 bg-[#0d1310] p-4 space-y-2">
                <div className="text-[10px] text-neutral-400 uppercase tracking-wider font-subtext">Published SHA-256 Binary Checksum:</div>
                <div className="flex flex-col sm:flex-row sm:items-center gap-3">
                  <span className="text-emerald-300 text-[11px] break-all font-semibold font-mono">
                    34a93967e7a8dbdadc649dbfeecde1d36b816e3627aed480c09876e8acb582ec
                  </span>
                  <button
                    onClick={() => copyToClipboard('34a93967e7a8dbdadc649dbfeecde1d36b816e3627aed480c09876e8acb582ec', setCopiedHash)}
                    className="self-start sm:self-auto inline-flex size-7 shrink-0 items-center justify-center rounded border border-white/[0.08] bg-white/[0.04] text-neutral-400 hover:text-emerald-300 transition-colors"
                    title="Copy Checksum"
                    aria-label="Copy SHA-256 Checksum"
                  >
                    {copiedHash ? <Check className="size-3 text-emerald-400" /> : <Copy className="size-3" />}
                  </button>
                </div>
                <div className="text-[10px] text-neutral-500 pt-1 border-t border-white/[0.04]">
                  Flags: <code className="text-neutral-400">-trimpath -ldflags=&quot;-s -w -buildid=&quot;</code>
                </div>
              </div>
            </div>
          </div>
        </section>

        {/* Section 7: Core Features */}
        <section className="mt-16 sm:mt-28" id="features" aria-labelledby="features-heading">
          <div className="text-center max-w-2xl mx-auto mb-8 sm:mb-10">
            <span className="font-mono text-xs font-semibold text-amber-400 tracking-wider block mb-1.5 uppercase">
              Core Capabilities
            </span>
            <h2 id="features-heading" className="font-serif-display text-3xl sm:text-4xl font-normal text-neutral-100">
              Built for security, performance, and developer efficiency.
            </h2>
            <p className="mt-2 text-sm text-neutral-400 font-subtext leading-relaxed">
              Engineered for robust local credential management with zero runtime dependencies.
            </p>
          </div>

          <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
            {[
              {
                icon: <Shield className="size-4.5" />,
                title: 'Zero Plaintext on Disk',
                description: 'Unencrypted secrets never touch storage. Decrypted credentials exist exclusively in ephemeral process memory while your app runs.',
              },
              {
                icon: <Lock className="size-4.5" />,
                title: 'AES-256-GCM + PBKDF2',
                description: 'Authenticated encryption with 600,000 security rounds ensures your vault remains protected against offline GPU brute-force attempts.',
              },
              {
                icon: <Cpu className="size-4.5" />,
                title: 'Hardware Inode Isolation',
                description: 'Secrets are automatically bound to the physical storage device and directory inode, preventing accidental project cross-contamination.',
              },
              {
                icon: <Terminal className="size-4.5" />,
                title: 'Native Process Spawning',
                description: 'Spawns your processes directly via operating system primitives (os/exec) without subshell wrappers, keeping your terminal history clean.',
              },
              {
                icon: <CheckCircle2 className="size-4.5" />,
                title: 'Tamper-Proof Audit Trail',
                description: 'Every secret read, write, or execution is signed into a cryptographic SHA-256 hash chain so unauthorized tampering is immediately detected.',
              },
              {
                icon: <Zap className="size-4.5" />,
                title: '100% Local & Offline',
                description: 'No cloud accounts, no network daemons, and zero third-party packages. Operates completely locally on your workstation.',
              },
            ].map((card) => (
              <article
                key={card.title}
                className="group rounded-2xl border border-white/[0.06] bg-[#131715] p-5 sm:p-6 transition-all duration-200 hover:border-amber-900/50 hover:bg-[#161c19]"
              >
                <div className="mb-4 inline-flex size-9 items-center justify-center rounded-lg border border-white/[0.08] bg-white/[0.02] text-amber-400 group-hover:border-amber-900/50 group-hover:bg-amber-950/30 transition-colors duration-200">
                  {card.icon}
                </div>
                <h3 className="font-semibold text-neutral-100 text-sm mb-2 font-subtext">{card.title}</h3>
                <p className="text-xs text-neutral-400 font-subtext leading-relaxed">{card.description}</p>
              </article>
            ))}
          </div>
        </section>

        {/* Section 8: Technical Comparison Matrix */}
        <section className="mt-16 sm:mt-28" id="comparison" aria-labelledby="comparison-heading">
          <div className="text-center max-w-2xl mx-auto mb-6 sm:mb-10">
            <span className="font-mono text-xs font-semibold text-amber-400 tracking-wider block mb-1.5 uppercase">
              Technical Comparison
            </span>
            <h2 id="comparison-heading" className="font-serif-display text-3xl sm:text-4xl font-normal text-neutral-100">
              Architectural Comparison
            </h2>
            <p className="mt-2 text-sm text-neutral-400 font-subtext leading-relaxed">
              How MayFly compares to traditional plaintext .env files and cloud secret managers.
            </p>
          </div>

          {/* Mobile Horizontal Swipe Hint */}
          <div className="flex items-center justify-center gap-2 text-xs text-amber-400/80 mb-3 md:hidden font-subtext">
            <span>&larr;</span>
            <span>Swipe table horizontally to view full matrix</span>
            <span>&rarr;</span>
          </div>

          <div className="overflow-x-auto scrollbar-none rounded-2xl border border-white/[0.08] bg-[#131715] shadow-inner">
            <table className="w-full text-left text-xs font-subtext min-w-[620px]">
              <thead className="border-b border-white/[0.08] bg-white/[0.02] text-neutral-400 font-mono">
                <tr>
                  <th className="py-3.5 px-4 font-semibold text-neutral-300 sticky left-0 bg-[#131715] z-10">Feature</th>
                  <th className="py-3.5 px-4">Plaintext .env</th>
                  <th className="py-3.5 px-4">Doppler / Infisical</th>
                  <th className="py-3.5 px-4">HashiCorp Vault</th>
                  <th className="py-3.5 px-4 text-amber-400 font-semibold bg-amber-950/30">MayFly</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-white/[0.04] text-neutral-300">
                <tr>
                  <td className="py-3.5 px-4 font-medium text-neutral-200 sticky left-0 bg-[#131715] z-10 shadow-[2px_0_5px_rgba(0,0,0,0.4)]">No Plaintext on Disk</td>
                  <td className="py-3.5 px-4 text-red-400">Plaintext on disk</td>
                  <td className="py-3.5 px-4 text-neutral-400">Downloads .env files</td>
                  <td className="py-3.5 px-4 text-neutral-400">In-memory via API</td>
                  <td className="py-3.5 px-4 text-amber-400 font-semibold bg-amber-950/20">100% In-Memory RAM</td>
                </tr>
                <tr>
                  <td className="py-3.5 px-4 font-medium text-neutral-200 sticky left-0 bg-[#131715] z-10 shadow-[2px_0_5px_rgba(0,0,0,0.4)]">100% Offline Architecture</td>
                  <td className="py-3.5 px-4 text-emerald-400">Offline</td>
                  <td className="py-3.5 px-4 text-red-400">Cloud SaaS dependent</td>
                  <td className="py-3.5 px-4 text-red-400">Requires daemon server</td>
                  <td className="py-3.5 px-4 text-amber-400 font-semibold bg-amber-950/20">Single Local Binary</td>
                </tr>
                <tr>
                  <td className="py-3.5 px-4 font-medium text-neutral-200 sticky left-0 bg-[#131715] z-10 shadow-[2px_0_5px_rgba(0,0,0,0.4)]">Supply-Chain Attack Resilience</td>
                  <td className="py-3.5 px-4 text-red-400">Vulnerable to theft</td>
                  <td className="py-3.5 px-4 text-neutral-400">Partial</td>
                  <td className="py-3.5 px-4 text-neutral-400">Partial</td>
                  <td className="py-3.5 px-4 text-amber-400 font-semibold bg-amber-950/20">Full (No disk target)</td>
                </tr>
                <tr>
                  <td className="py-3.5 px-4 font-medium text-neutral-200 sticky left-0 bg-[#131715] z-10 shadow-[2px_0_5px_rgba(0,0,0,0.4)]">Tamper-Proof Audit Log</td>
                  <td className="py-3.5 px-4 text-red-400">None</td>
                  <td className="py-3.5 px-4 text-neutral-400">Cloud Web UI</td>
                  <td className="py-3.5 px-4 text-neutral-400">Server Log</td>
                  <td className="py-3.5 px-4 text-amber-400 font-semibold bg-amber-950/20">SHA-256 Hash Chain</td>
                </tr>
                <tr>
                  <td className="py-3.5 px-4 font-medium text-neutral-200 sticky left-0 bg-[#131715] z-10 shadow-[2px_0_5px_rgba(0,0,0,0.4)]">Direct Process Injection</td>
                  <td className="py-3.5 px-4 text-neutral-400">dotenv libraries</td>
                  <td className="py-3.5 px-4 text-neutral-400">CLI exec wrapper</td>
                  <td className="py-3.5 px-4 text-neutral-400">Envconsul tool</td>
                  <td className="py-3.5 px-4 text-amber-400 font-semibold bg-amber-950/20">Built-in os/exec</td>
                </tr>
                <tr>
                  <td className="py-3.5 px-4 font-medium text-neutral-200 sticky left-0 bg-[#131715] z-10 shadow-[2px_0_5px_rgba(0,0,0,0.4)]">Dependencies &amp; Overhead</td>
                  <td className="py-3.5 px-4 text-neutral-400">N/A</td>
                  <td className="py-3.5 px-4 text-neutral-400">Requires SDKs</td>
                  <td className="py-3.5 px-4 text-neutral-400">150MB+ Server setup</td>
                  <td className="py-3.5 px-4 text-amber-400 font-semibold bg-amber-950/20">0 Deps (12MB binary)</td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>

        {/* Section 9: Zero-Dependency Verification CTA */}
        <section className="mt-16 sm:mt-28 rounded-2xl border border-amber-900/40 bg-[#131715] p-6 sm:p-8" id="audit-cta">
          <div className="flex flex-col md:flex-row items-start md:items-center justify-between gap-6">
            <div className="max-w-xl">
              <div className="inline-flex items-center gap-2 rounded-full border border-amber-900/60 bg-amber-950/40 px-3 py-1 text-xs font-mono text-amber-400 mb-4">
                <Shield className="size-3" />
                Zero-Dependency Audit
              </div>
              <h2 className="font-serif-display text-2xl sm:text-3xl font-normal text-neutral-100">
                100% Go Standard Library
              </h2>
              <p className="mt-2 text-xs sm:text-sm text-neutral-400 font-subtext leading-relaxed">
                MayFly contains zero third-party packages in its dependency manifest. Run <code className="text-neutral-200 font-mono">go list -m all</code> on the repository to verify that only standard library packages are used.
              </p>
            </div>

            <div className="w-full md:w-auto">
              <Link
                href="/docs/reference/zero-dependency-audit"
                id="inspect-audit-proof-btn"
                className="w-full sm:w-auto inline-flex items-center justify-center gap-2 rounded-lg border border-amber-800/60 bg-amber-950/40 px-5 py-2.5 text-xs font-mono text-amber-300 hover:bg-amber-900/40 transition-colors"
              >
                <span>Inspect Audit Proof</span>
                <ChevronRight className="size-4" />
              </Link>
            </div>
          </div>
        </section>
      </main>

      {/* Footer */}
      <footer className="border-t border-white/[0.06] bg-[#0a0d0c] py-10 sm:py-12 text-xs text-neutral-500 font-subtext" role="contentinfo">
        <div className="mx-auto max-w-6xl px-4 sm:px-6 flex flex-col sm:flex-row items-center justify-between gap-4 text-center sm:text-left">
          <div>
            MayFly &bull; Released under AGPL-3.0 &bull; Built with Pure Go Standard Library
          </div>
          <div className="flex flex-wrap justify-center items-center gap-4 sm:gap-5 text-neutral-400">
            <Link href="/docs" className="hover:text-amber-400 transition-colors">Documentation</Link>
            <Link href="/docs/quickstart" className="hover:text-amber-400 transition-colors">Quickstart</Link>
            <Link href="/docs/reference/zero-dependency-audit" className="hover:text-amber-400 transition-colors">Zero-Dep Audit</Link>
            <a
              href="https://github.com/vishnunandan555/mayfly"
              target="_blank"
              rel="noopener noreferrer"
              className="hover:text-amber-400 transition-colors"
            >
              GitHub
            </a>
          </div>
        </div>
      </footer>
    </div>
  );
}
