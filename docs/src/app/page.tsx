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
} from 'lucide-react';

export default function HomePage() {
  const [installTab, setInstallTab] = useState<'curl' | 'macos' | 'linux' | 'brew' | 'go'>('curl');
  const [copiedInstall, setCopiedInstall] = useState(false);

  const installCommands = {
    curl: 'curl -fsSL https://raw.githubusercontent.com/vishnunandan555/mayfly/main/install.sh | bash',
    macos: 'curl -fsSL https://raw.githubusercontent.com/vishnunandan555/mayfly/main/install.sh | bash',
    linux: 'curl -fsSL https://raw.githubusercontent.com/vishnunandan555/mayfly/main/install.sh | bash',
    brew: '# Or clone & build via standard library\ngit clone https://github.com/vishnunandan555/mayfly.git && cd mayfly && ./install.sh',
    go: 'go install mayfly/cmd/mayfly@latest',
  };

  const copyToClipboard = () => {
    navigator.clipboard.writeText(installCommands[installTab]);
    setCopiedInstall(true);
    setTimeout(() => setCopiedInstall(false), 2000);
  };

  return (
    <div className="relative min-h-screen bg-[#080a09] text-neutral-100 selection:bg-amber-950/60 selection:text-amber-200">
      {/* Warm Amber / Gold Radial Glows */}
      <div className="absolute inset-0 bg-[radial-gradient(ellipse_80%_60%_at_50%_-20%,rgba(245,158,11,0.12),transparent_70%)] pointer-events-none" />
      <div className="absolute top-[700px] left-1/2 -translate-x-1/2 w-[750px] h-[380px] bg-amber-500/5 blur-[140px] pointer-events-none rounded-full" />

      {/* Navigation Header */}
      <header className="sticky top-0 z-50 border-b border-white/[0.06] bg-[#080a09]/80 backdrop-blur-xl">
        <div className="mx-auto flex h-16 max-w-6xl items-center justify-between px-6">
          <div className="flex items-center gap-3">
            <Link href="/" className="flex items-center gap-2.5 group">
              <span className="inline-flex size-5 items-center justify-center rounded-full bg-amber-500/10 ring-1 ring-inset ring-amber-500/30">
                <span className="size-2 rounded-full bg-amber-400 shadow-[0_0_8px_2px_rgba(245,158,11,0.6)] group-hover:scale-125 transition-transform" />
              </span>
              <span className="font-mono text-sm font-bold tracking-widest text-neutral-100">MAYFLY</span>
            </Link>
          </div>

          <nav className="hidden md:flex items-center gap-7 text-sm font-normal">
            <Link href="/docs/why-mayfly" className="text-neutral-400 transition-colors hover:text-amber-400">
              Why MayFly
            </Link>
            <Link href="/docs/concepts" className="text-neutral-400 transition-colors hover:text-amber-400">
              Concepts
            </Link>
            <Link href="/docs/architecture/security-model" className="text-neutral-400 transition-colors hover:text-amber-400">
              Security Model
            </Link>
            <Link href="/docs/architecture/audit-chain" className="text-neutral-400 transition-colors hover:text-amber-400">
              Audit Trail
            </Link>
            <Link href="/docs/cli/overview" className="text-neutral-400 transition-colors hover:text-amber-400">
              Reference
            </Link>
          </nav>

          <div className="flex items-center gap-3">
            <Link
              href="/docs"
              className="inline-flex items-center gap-1.5 rounded-lg border border-white/[0.08] bg-white/[0.03] px-3.5 py-1.5 text-xs font-medium text-neutral-300 transition-colors hover:border-amber-800/50 hover:bg-white/[0.06] hover:text-neutral-100"
            >
              <BookOpen className="size-3.5 text-amber-400" />
              <span>Docs</span>
            </Link>
            <a
              href="https://github.com/vishnunandan555/mayfly"
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-1.5 rounded-lg bg-amber-400 px-3.5 py-1.5 text-xs font-semibold text-neutral-950 transition-all hover:bg-amber-300 shadow-[0_0_15px_rgba(245,158,11,0.3)]"
            >
              <span>GitHub</span>
              <ExternalLink className="size-3 text-neutral-950" />
            </a>
          </div>
        </div>
      </header>

      {/* Main Hero Section (2 Columns) */}
      <main className="relative mx-auto max-w-6xl px-6 pt-16 pb-20">
        <div className="grid grid-cols-1 lg:grid-cols-12 gap-10 lg:gap-8 items-start">
          
          {/* Left Column: Headline, Description, Badges, Install Tabs */}
          <div className="lg:col-span-6 flex flex-col justify-center pt-2">
            
            {/* Serif Main Headline */}
            <h1 className="font-serif-display text-4xl sm:text-5xl lg:text-[3.5rem] font-normal tracking-tight text-neutral-100 leading-[1.08]">
              Zero-disk secrets sandbox for terminal workflows.
            </h1>

            {/* Value Proposition Description */}
            <p className="mt-5 text-sm sm:text-base text-neutral-400 leading-relaxed font-normal">
              MayFly wraps any command in an authenticated execution sandbox that injects credentials directly into child process RAM. Default-deny disk writes, hardware inode binding, and tamper-evident audit trail for AI coding agents, CI, and local dev.
            </p>

            {/* Badges Row */}
            <div className="mt-6 flex flex-wrap items-center gap-2">
              <span className="inline-flex items-center gap-1.5 rounded-full border border-amber-900/60 bg-amber-950/40 px-3 py-1 font-mono text-[11px] text-amber-400">
                <span className="size-1.5 rounded-full bg-amber-400" />
                zero-dep 100% Go stdlib
              </span>
              <span className="rounded-full border border-white/[0.08] bg-white/[0.03] px-3 py-1 font-mono text-[11px] text-neutral-400">
                release v1.0.0
              </span>
              <span className="rounded-full border border-white/[0.08] bg-white/[0.03] px-3 py-1 font-mono text-[11px] text-neutral-400">
                license AGPL-3.0
              </span>
            </div>

            {/* Divider */}
            <div className="my-8 border-t border-white/[0.08]" />

            {/* Install Selector Header */}
            <div className="flex items-center gap-3 mb-3">
              <span className="font-mono text-[11px] font-semibold text-amber-400 tracking-wider">INSTALL</span>
              <div className="flex items-center gap-1">
                {(
                  [
                    { id: 'curl', label: 'curl' },
                    { id: 'macos', label: 'macOS' },
                    { id: 'linux', label: 'Linux' },
                    { id: 'brew', label: 'Source' },
                    { id: 'go', label: 'Go' },
                  ] as const
                ).map((tab) => (
                  <button
                    key={tab.id}
                    onClick={() => setInstallTab(tab.id)}
                    className={`rounded-full px-3 py-1 text-xs font-mono transition-all ${
                      installTab === tab.id
                        ? 'bg-amber-400 text-neutral-950 font-semibold shadow-sm'
                        : 'text-neutral-400 hover:text-neutral-200'
                    }`}
                  >
                    {tab.label}
                  </button>
                ))}
              </div>
            </div>

            {/* Install Command Box */}
            <div className="relative rounded-xl border border-white/[0.08] bg-[#0c100e] p-4 font-mono text-xs text-neutral-300">
              <div className="text-[10px] text-neutral-500 mb-1.5 uppercase tracking-wider">
                {installTab === 'macos' ? 'macOS • Darwin Intel & Silicon' : installTab === 'linux' ? 'Linux • All Distros' : installTab === 'go' ? 'Go Toolchain' : 'One-Line Installer'}
              </div>
              <div className="flex items-center justify-between gap-3">
                <div className="truncate text-amber-200 font-medium">
                  <span className="text-neutral-600 mr-2 select-none">$</span>
                  {installCommands[installTab]}
                </div>
                <button
                  onClick={copyToClipboard}
                  className="inline-flex size-7 shrink-0 items-center justify-center rounded-lg border border-white/[0.08] bg-white/[0.04] text-neutral-400 hover:border-amber-800/60 hover:text-amber-300 transition-colors"
                  title="Copy command"
                >
                  {copiedInstall ? <Check className="size-3.5 text-amber-400" /> : <Copy className="size-3.5" />}
                </button>
              </div>
            </div>

            {/* Platform Note */}
            <p className="mt-3 text-xs text-neutral-500">
              Single ~12MB static binary. Secrets remain encrypted with AES-256-GCM at rest and reside strictly in volatile process memory during execution.
            </p>
          </div>

          {/* Right Column: Simulated Live Terminal Session (Fence Style) */}
          <div className="lg:col-span-6 lg:pl-4">
            <div className="rounded-2xl border border-white/[0.08] bg-[#0c0f0d] p-5 font-mono text-xs shadow-2xl ring-1 ring-amber-900/20">
              
              {/* Terminal Window Top Bar */}
              <div className="flex items-center justify-between pb-3.5 mb-3.5 border-b border-white/[0.06]">
                <div className="flex items-center gap-1.5">
                  <span className="size-2.5 rounded-full bg-[#ff5f56]" />
                  <span className="size-2.5 rounded-full bg-[#ffbd2e]" />
                  <span className="size-2.5 rounded-full bg-[#27c93f]" />
                  <span className="ml-2 text-[11px] text-neutral-500">~/my-project</span>
                </div>
                <span className="text-[10px] text-amber-400/80 font-mono">in-memory execution</span>
              </div>

              {/* Terminal Session Content */}
              <div className="space-y-3.5 text-left text-neutral-300 leading-relaxed">
                <div>
                  <div className="flex items-center gap-2">
                    <span className="text-amber-400 font-bold select-none">$</span>
                    <span className="text-neutral-100 font-semibold">mf npm run dev</span>
                  </div>
                </div>

                {/* MayFly Injection Block */}
                <div className="border-l-2 border-amber-500 pl-3 py-0.5 space-y-1 bg-amber-950/10 rounded-r">
                  <div className="text-[11px] font-bold text-amber-400 tracking-wider">MAYFLY SANDBOX</div>
                  <div className="text-[11px] text-neutral-400">resolving hardware inode 1048576...</div>
                  <div className="text-[11px] text-neutral-400">unlocking AES-256-GCM vault in volatile RAM...</div>
                  <div className="text-[11px] text-amber-300 font-medium">&rarr; injected STRIPE_SECRET (sk_live_***)</div>
                  <div className="text-[11px] text-amber-300 font-medium">&rarr; injected DATABASE_URL (postgres://***)</div>
                  <div className="text-[11px] text-neutral-500">&rarr; spawned child process via os/exec (pid 48120)</div>
                </div>

                {/* Subprocess Output */}
                <div className="space-y-1">
                  <div className="text-neutral-200 font-semibold text-[11px]">NEXT.JS 15</div>
                  <div className="text-neutral-400 text-[11px]">Ready in 1.8s on http://localhost:3000</div>
                </div>

                {/* Proof of Zero Disk Trace */}
                <div className="pt-2 border-t border-white/[0.04] space-y-1">
                  <div className="flex items-center gap-2">
                    <span className="text-amber-400 font-bold select-none">$</span>
                    <span className="text-neutral-100">cat .env</span>
                  </div>
                  <div className="text-red-400 text-[11px] pl-4">
                    &larr; cat: .env: No such file or directory
                  </div>
                </div>

                {/* Audit Verification */}
                <div className="pt-2 space-y-1">
                  <div className="flex items-center gap-2">
                    <span className="text-amber-400 font-bold select-none">$</span>
                    <span className="text-neutral-100">mf audit verify</span>
                  </div>
                  <div className="text-amber-400 text-[11px] pl-4 font-medium">
                    &larr; Audit log hash chain verified successfully.
                  </div>
                </div>

                {/* Audit Record */}
                <div className="border-l-2 border-neutral-700 pl-3 py-0.5 bg-white/[0.02] text-[10px] text-neutral-400 rounded-r">
                  <span className="text-neutral-300 font-bold">AUDIT TRAIL</span>
                  <div className="truncate text-neutral-500 mt-0.5">
                    #14 2026-08-31T05:14:00Z EXEC project=mayfly hash=a3f9e1b2...
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        {/* Section 2: Quick Start Card (Fence Style "Get one workflow working") */}
        <section className="mt-28">
          <div className="rounded-2xl border border-white/[0.08] bg-[#0b0e0c] p-6 sm:p-10 relative overflow-hidden">
            
            {/* Header */}
            <div className="max-w-2xl">
              <span className="font-mono text-xs font-semibold text-amber-400 tracking-wider block mb-2">QUICK START</span>
              <h2 className="font-serif-display text-2xl sm:text-3xl font-normal text-neutral-100">
                Get one workflow working.
              </h2>
              <p className="mt-2 text-sm text-neutral-400 leading-relaxed">
                Install MayFly with the instructions above. Initialize your project vault, inject secrets in memory, and observe zero plaintext files on disk.
              </p>
            </div>

            {/* 4 Process Steps */}
            <div className="mt-10 grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6 relative">
              
              {/* Step 1 */}
              <div className="flex flex-col justify-between rounded-xl border border-white/[0.06] bg-white/[0.02] p-5">
                <div>
                  <div className="flex items-center gap-2.5 mb-3">
                    <span className="size-6 rounded-full border border-amber-500/40 bg-amber-950/40 flex items-center justify-center font-mono text-xs font-bold text-amber-400">
                      1
                    </span>
                    <h3 className="text-sm font-semibold text-neutral-100">Initialize vault</h3>
                  </div>
                  <p className="text-xs text-neutral-400 leading-relaxed mb-4">
                    Binds the current directory inode to a dedicated encrypted project key space.
                  </p>
                </div>
                <div className="rounded bg-[#070908] px-3 py-2 font-mono text-xs text-amber-300">
                  <span className="text-neutral-600 mr-1.5">$</span>mf init
                </div>
              </div>

              {/* Step 2 */}
              <div className="flex flex-col justify-between rounded-xl border border-white/[0.06] bg-white/[0.02] p-5">
                <div>
                  <div className="flex items-center gap-2.5 mb-3">
                    <span className="size-6 rounded-full border border-amber-500/40 bg-amber-950/40 flex items-center justify-center font-mono text-xs font-bold text-amber-400">
                      2
                    </span>
                    <h3 className="text-sm font-semibold text-neutral-100">Store secrets</h3>
                  </div>
                  <p className="text-xs text-neutral-400 leading-relaxed mb-4">
                    Encrypts variable with AES-256-GCM without writing any <code className="text-neutral-300">.env</code> to disk.
                  </p>
                </div>
                <div className="rounded bg-[#070908] px-3 py-2 font-mono text-xs text-amber-300">
                  <span className="text-neutral-600 mr-1.5">$</span>mf set STRIPE_KEY
                </div>
              </div>

              {/* Step 3 */}
              <div className="flex flex-col justify-between rounded-xl border border-white/[0.06] bg-white/[0.02] p-5">
                <div>
                  <div className="flex items-center gap-2.5 mb-3">
                    <span className="size-6 rounded-full border border-amber-500/40 bg-amber-950/40 flex items-center justify-center font-mono text-xs font-bold text-amber-400">
                      3
                    </span>
                    <h3 className="text-sm font-semibold text-neutral-100">Run target app</h3>
                  </div>
                  <p className="text-xs text-neutral-400 leading-relaxed mb-4">
                    Spawns your runtime with secrets injected directly into process memory.
                  </p>
                </div>
                <div className="rounded bg-[#070908] px-3 py-2 font-mono text-xs text-amber-300">
                  <span className="text-neutral-600 mr-1.5">$</span>mf npm run dev
                </div>
              </div>

              {/* Step 4 */}
              <div className="flex flex-col justify-between rounded-xl border border-white/[0.06] bg-white/[0.02] p-5">
                <div>
                  <div className="flex items-center gap-2.5 mb-3">
                    <span className="size-6 rounded-full border border-amber-500/40 bg-amber-950/40 flex items-center justify-center font-mono text-xs font-bold text-amber-400">
                      4
                    </span>
                    <h3 className="text-sm font-semibold text-neutral-100">Verify audit</h3>
                  </div>
                  <p className="text-xs text-neutral-400 leading-relaxed mb-4">
                    Validates the cryptographic SHA-256 hash chain across all access logs.
                  </p>
                </div>
                <div className="rounded bg-[#070908] px-3 py-2 font-mono text-xs text-amber-300">
                  <span className="text-neutral-600 mr-1.5">$</span>mf audit verify
                </div>
              </div>
            </div>

            {/* Optional Shortcut */}
            <div className="mt-8 pt-6 border-t border-white/[0.06] flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3 text-xs">
              <div className="text-neutral-400">
                <span className="font-mono text-amber-400 font-semibold uppercase mr-2 text-[10px]">VISUAL DASHBOARD</span>
                Prefer an interactive UI? Launch the zero-dependency full-screen TUI:
              </div>
              <div className="rounded-lg bg-[#070908] px-3 py-1.5 font-mono text-xs text-amber-300 border border-white/[0.06]">
                <span className="text-neutral-600 mr-1.5">$</span>mf
              </div>
            </div>
          </div>
        </section>

        {/* Section 3: The Attack Vector vs MayFly Defense */}
        <section className="mt-28">
          <div className="text-center max-w-2xl mx-auto mb-12">
            <h2 className="font-serif-display text-3xl sm:text-4xl font-normal text-neutral-100">
              The Supply-Chain Problem
            </h2>
            <p className="mt-2 text-sm text-neutral-400 leading-relaxed">
              Why traditional plaintext configuration files fail against modern supply-chain credential theft.
            </p>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            {/* The Plaintext Risk */}
            <div className="rounded-2xl border border-red-950/40 bg-[#0d0a0a]/60 p-6 sm:p-8 relative overflow-hidden">
              <div className="flex items-center gap-3 mb-5">
                <div className="size-9 rounded-lg border border-red-900/50 bg-red-950/40 flex items-center justify-center text-red-400">
                  <AlertTriangle className="size-4.5" />
                </div>
                <div>
                  <h3 className="font-semibold text-neutral-100 text-base">The Plaintext .env Risk</h3>
                  <p className="text-xs text-neutral-500 font-mono">Traditional Development Workflow</p>
                </div>
              </div>

              <ul className="space-y-3.5 text-sm text-neutral-400">
                <li className="flex items-start gap-2.5">
                  <span className="text-red-400 font-bold select-none">&times;</span>
                  <span><strong>Unencrypted on Disk:</strong> API keys and database credentials reside as readable plaintext files on developer SSDs.</span>
                </li>
                <li className="flex items-start gap-2.5">
                  <span className="text-red-400 font-bold select-none">&times;</span>
                  <span><strong>Package Install Theft:</strong> Malicious <code className="text-neutral-300 font-mono text-xs">npm postinstall</code> or Python setup scripts scan parent directories and exfiltrate credentials before code runs.</span>
                </li>
                <li className="flex items-start gap-2.5">
                  <span className="text-red-400 font-bold select-none">&times;</span>
                  <span><strong>Accidental Leaks:</strong> Plaintext files get accidentally staged to Git, indexed by backup agents, or dumped in crash logs.</span>
                </li>
                <li className="flex items-start gap-2.5">
                  <span className="text-red-400 font-bold select-none">&times;</span>
                  <span><strong>Zero Attribution:</strong> No tamper-proof log of which process or user accessed secrets.</span>
                </li>
              </ul>
            </div>

            {/* The MayFly Defense */}
            <div className="rounded-2xl border border-amber-900/40 bg-[#0d0f0c]/60 p-6 sm:p-8 relative overflow-hidden">
              <div className="flex items-center gap-3 mb-5">
                <div className="size-9 rounded-lg border border-amber-900/50 bg-amber-950/40 flex items-center justify-center text-amber-400">
                  <Shield className="size-4.5" />
                </div>
                <div>
                  <h3 className="font-semibold text-neutral-100 text-base">The MayFly Defense</h3>
                  <p className="text-xs text-amber-400 font-mono">Zero-Disk Ephemeral Injection</p>
                </div>
              </div>

              <ul className="space-y-3.5 text-sm text-neutral-400">
                <li className="flex items-start gap-2.5">
                  <span className="text-amber-400 font-bold select-none">&#10003;</span>
                  <span><strong>Zero Disk Footprint:</strong> Secrets are strictly encrypted at rest with AES-256-GCM. Unencrypted secrets never touch the filesystem.</span>
                </li>
                <li className="flex items-start gap-2.5">
                  <span className="text-amber-400 font-bold select-none">&#10003;</span>
                  <span><strong>In-Memory OS Spawning:</strong> Direct <code className="text-neutral-300 font-mono text-xs">os/exec</code> injection bypasses shell history, subshell leaks, and install-time scanner scripts.</span>
                </li>
                <li className="flex items-start gap-2.5">
                  <span className="text-amber-400 font-bold select-none">&#10003;</span>
                  <span><strong>Hardware Inode Isolation:</strong> Secrets are tied to the directory inode number, preventing cross-project secret contamination.</span>
                </li>
                <li className="flex items-start gap-2.5">
                  <span className="text-amber-400 font-bold select-none">&#10003;</span>
                  <span><strong>Cryptographic Audit Log:</strong> SHA-256 hash-chained event trail detects any modification or unauthorized access attempts.</span>
                </li>
              </ul>
            </div>
          </div>
        </section>

        {/* Section 4: Feature Grid */}
        <section className="mt-28">
          <div className="text-center max-w-2xl mx-auto mb-12">
            <h2 className="font-serif-display text-3xl sm:text-4xl font-normal text-neutral-100">
              Built for Security-Conscious Engineering
            </h2>
            <p className="mt-2 text-sm text-neutral-400 leading-relaxed">
              Every design decision in MayFly prioritizes zero-disk footprint, host performance, and supply-chain resilience.
            </p>
          </div>

          <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
            {[
              {
                icon: <Shield className="size-4.5" />,
                title: 'Zero-Disk Attack Surface',
                description: 'No unencrypted credentials ever touch disk storage. Decrypted secrets reside strictly in ephemeral process memory.',
              },
              {
                icon: <Lock className="size-4.5" />,
                title: 'AES-256-GCM + PBKDF2',
                description: 'Authenticated AEAD encryption with 600,000 PBKDF2-HMAC-SHA256 iterations and per-write cryptographically random nonces.',
              },
              {
                icon: <Cpu className="size-4.5" />,
                title: 'Deterministic Inode Binding',
                description: 'Filesystem-level device and inode resolution binds secrets to the physical project directory with zero cross-contamination.',
              },
              {
                icon: <Terminal className="size-4.5" />,
                title: 'Direct Process Execution',
                description: 'Processes spawn via direct operating system primitives without intermediate shell wrappers, preventing subshell leaks.',
              },
              {
                icon: <CheckCircle2 className="size-4.5" />,
                title: 'Cryptographic Audit Trail',
                description: 'SHA-256 hash-chained event logs record timestamps and actions, detecting any historical modification or deletion.',
              },
              {
                icon: <Zap className="size-4.5" />,
                title: 'Zero External Dependencies',
                description: 'Pure Go standard library. Immune to upstream npm, pip, or cargo supply-chain vulnerabilities and dependency hijacking.',
              },
            ].map((card) => (
              <div
                key={card.title}
                className="group rounded-2xl border border-white/[0.06] bg-[#0a0d0c] p-6 transition-all duration-200 hover:border-amber-900/50 hover:bg-[#0e110f]"
              >
                <div className="mb-4 inline-flex size-9 items-center justify-center rounded-lg border border-white/[0.08] bg-white/[0.02] text-amber-400 group-hover:border-amber-900/50 group-hover:bg-amber-950/30 transition-colors duration-200">
                  {card.icon}
                </div>
                <h3 className="font-semibold text-neutral-100 text-sm mb-2">{card.title}</h3>
                <p className="text-xs text-neutral-400 leading-relaxed">{card.description}</p>
              </div>
            ))}
          </div>
        </section>

        {/* Section 5: Technical Comparison Matrix */}
        <section className="mt-28">
          <div className="text-center max-w-2xl mx-auto mb-12">
            <h2 className="font-serif-display text-3xl sm:text-4xl font-normal text-neutral-100">
              Technical Comparison
            </h2>
            <p className="mt-2 text-sm text-neutral-400 leading-relaxed">
              How MayFly compares to existing configuration and secrets management solutions.
            </p>
          </div>

          <div className="overflow-x-auto rounded-2xl border border-white/[0.08] bg-[#090c0b]">
            <table className="w-full text-left text-xs">
              <thead className="border-b border-white/[0.08] bg-white/[0.02] text-neutral-400 font-mono">
                <tr>
                  <th className="py-3.5 px-4 font-semibold text-neutral-300">Capability</th>
                  <th className="py-3.5 px-4">Plaintext .env</th>
                  <th className="py-3.5 px-4">Doppler / Infisical</th>
                  <th className="py-3.5 px-4">HashiCorp Vault</th>
                  <th className="py-3.5 px-4 text-amber-400 font-semibold bg-amber-950/30">MayFly</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-white/[0.04] text-neutral-300">
                <tr>
                  <td className="py-3.5 px-4 font-medium text-neutral-200">Zero Disk Plaintext</td>
                  <td className="py-3.5 px-4 text-red-400">No (plaintext)</td>
                  <td className="py-3.5 px-4 text-neutral-400">Varies (env files)</td>
                  <td className="py-3.5 px-4 text-neutral-400">Yes (API read)</td>
                  <td className="py-3.5 px-4 text-amber-400 font-semibold bg-amber-950/20">&#10003; 100% In-Memory</td>
                </tr>
                <tr>
                  <td className="py-3.5 px-4 font-medium text-neutral-200">Zero Cloud Dependency (100% Offline)</td>
                  <td className="py-3.5 px-4 text-amber-400">&#10003; Local</td>
                  <td className="py-3.5 px-4 text-red-400">No (Cloud SaaS)</td>
                  <td className="py-3.5 px-4 text-red-400">No (Server setup)</td>
                  <td className="py-3.5 px-4 text-amber-400 font-semibold bg-amber-950/20">&#10003; Local Binary</td>
                </tr>
                <tr>
                  <td className="py-3.5 px-4 font-medium text-neutral-200">Supply-Chain Attack Immunity</td>
                  <td className="py-3.5 px-4 text-red-400">Vulnerable</td>
                  <td className="py-3.5 px-4 text-neutral-400">Moderate</td>
                  <td className="py-3.5 px-4 text-neutral-400">Moderate</td>
                  <td className="py-3.5 px-4 text-amber-400 font-semibold bg-amber-950/20">&#10003; Zero Dependencies</td>
                </tr>
                <tr>
                  <td className="py-3.5 px-4 font-medium text-neutral-200">Cryptographic Audit Trail</td>
                  <td className="py-3.5 px-4 text-red-400">None</td>
                  <td className="py-3.5 px-4 text-neutral-400">Server Logs</td>
                  <td className="py-3.5 px-4 text-neutral-400">Audit Devices</td>
                  <td className="py-3.5 px-4 text-amber-400 font-semibold bg-amber-950/20">&#10003; SHA-256 Hash Chain</td>
                </tr>
                <tr>
                  <td className="py-3.5 px-4 font-medium text-neutral-200">Direct Process Injection</td>
                  <td className="py-3.5 px-4 text-neutral-400">dotenv wrappers</td>
                  <td className="py-3.5 px-4 text-neutral-400">CLI exec</td>
                  <td className="py-3.5 px-4 text-neutral-400">Envconsul</td>
                  <td className="py-3.5 px-4 text-amber-400 font-semibold bg-amber-950/20">&#10003; Host-Native os/exec</td>
                </tr>
                <tr>
                  <td className="py-3.5 px-4 font-medium text-neutral-200">Binary Size & Footprint</td>
                  <td className="py-3.5 px-4 text-neutral-400">N/A</td>
                  <td className="py-3.5 px-4 text-neutral-400">~30MB</td>
                  <td className="py-3.5 px-4 text-neutral-400">~150MB server</td>
                  <td className="py-3.5 px-4 text-amber-400 font-semibold bg-amber-950/20">&#10003; ~12MB Single Binary</td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>

        {/* Section 6: Zero-Dependency Verification Block */}
        <section className="mt-28 rounded-2xl border border-amber-900/40 bg-[#0b0e0c] p-8">
          <div className="flex flex-col md:flex-row items-start md:items-center justify-between gap-6">
            <div className="max-w-xl">
              <div className="inline-flex items-center gap-2 rounded-full border border-amber-900/60 bg-amber-950/40 px-3 py-1 text-xs font-mono text-amber-400 mb-4">
                <Shield className="size-3" />
                Zero-Dependency Audit Proof
              </div>
              <h2 className="font-serif-display text-2xl sm:text-3xl font-normal text-neutral-100">
                100% Go Standard Library
              </h2>
              <p className="mt-2 text-xs sm:text-sm text-neutral-400 leading-relaxed">
                MayFly contains 0 third-party packages in its entire dependency tree. Run <code className="text-neutral-200 font-mono">go list -m all</code> on the repository to verify that no external packages are imported.
              </p>
            </div>

            <div className="w-full md:w-auto">
              <Link
                href="/docs/reference/zero-dependency-audit"
                className="inline-flex items-center gap-2 rounded-lg border border-amber-800/60 bg-amber-950/40 px-5 py-2.5 text-xs font-mono text-amber-300 hover:bg-amber-900/40 transition-colors"
              >
                <span>Inspect Audit Proof</span>
                <ChevronRight className="size-4" />
              </Link>
            </div>
          </div>
        </section>
      </main>

      {/* Footer */}
      <footer className="border-t border-white/[0.06] bg-[#060807] py-12 text-xs text-neutral-500 font-mono">
        <div className="mx-auto max-w-6xl px-6 flex flex-col sm:flex-row items-center justify-between gap-4">
          <div>
            MayFly &bull; Released under AGPL-3.0 &bull; Built for the Zero-Dependency Hackathon
          </div>
          <div className="flex items-center gap-5 text-neutral-400">
            <Link href="/docs" className="hover:text-amber-400 transition-colors">Documentation</Link>
            <Link href="/docs/quickstart" className="hover:text-amber-400 transition-colors">Quickstart</Link>
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
