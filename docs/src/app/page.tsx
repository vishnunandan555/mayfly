import Link from 'next/link';
import React from 'react';
import { Shield, Lock, Terminal, Cpu, CheckCircle2, ArrowRight, ExternalLink } from 'lucide-react';

export default function HomePage() {
  return (
    <div className="relative min-h-screen bg-neutral-950 text-neutral-100 selection:bg-emerald-900/50 selection:text-emerald-200">
      {/* Emerald radial glow — top center */}
      <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_50%_-20%,rgba(16,185,129,0.12),transparent_60%)] pointer-events-none" />

      {/* Navigation */}
      <header className="sticky top-0 z-40 border-b border-neutral-800/60 bg-neutral-950/85 backdrop-blur-md">
        <div className="mx-auto flex h-16 max-w-6xl items-center justify-between px-6">
          <div className="flex items-center gap-2.5">
            <span className="inline-flex size-5 items-center justify-center">
              <span className="size-2 rounded-full bg-emerald-400 shadow-[0_0_6px_2px_rgba(16,185,129,0.4)]" />
            </span>
            <span className="font-mono text-sm font-bold tracking-widest text-neutral-100">MAYFLY</span>
            <span className="rounded border border-emerald-900/60 bg-emerald-950/40 px-1.5 py-0.5 font-mono text-[10px] text-emerald-500">
              ZERO-DEP
            </span>
          </div>
          <nav className="flex items-center gap-6 text-sm">
            <Link href="/docs" className="text-neutral-400 transition-colors hover:text-emerald-400">
              Docs
            </Link>
            <Link href="/docs/architecture/security-model" className="text-neutral-400 transition-colors hover:text-emerald-400">
              Security Model
            </Link>
            <Link href="/docs/cli/overview" className="text-neutral-400 transition-colors hover:text-emerald-400">
              CLI Reference
            </Link>
            <a
              href="https://github.com/vishnunandan555/mayfly"
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-1 text-neutral-400 transition-colors hover:text-emerald-400"
            >
              GitHub <ExternalLink className="size-3.5" />
            </a>
          </nav>
        </div>
      </header>

      {/* Hero */}
      <main className="relative mx-auto max-w-6xl px-6 pt-20 pb-20">
        <div className="flex flex-col items-center text-center">

          {/* Badge */}
          <div className="inline-flex items-center gap-2 rounded-full border border-emerald-900/50 bg-emerald-950/30 px-3.5 py-1 text-xs text-emerald-500 mb-10 font-mono">
            <span className="inline-block size-1.5 rounded-full bg-emerald-400" />
            Go Standard Library Only &bull; 0 External Dependencies
          </div>

          {/* Headline — punchy two-liner */}
          <h1 className="max-w-3xl text-4xl font-semibold tracking-tight text-neutral-100 sm:text-[3.5rem] sm:leading-[1.1]">
            Your{' '}
            <span className="font-mono text-neutral-400">.env</span>
            {' '}file is a security hole.
            <br />
            <span className="text-emerald-400">MayFly closes it.</span>
          </h1>

          <p className="mt-6 max-w-xl text-base text-neutral-400 leading-relaxed">
            Secrets live encrypted at rest and are injected directly into child process memory at runtime. Nothing written to disk. No shell wrappers. No third-party dependencies.
          </p>

          {/* Terminal snippet — immediately below headline */}
          <div className="mt-10 w-full max-w-xl rounded-xl border border-neutral-800 bg-[#0c0e0d] p-4 font-mono text-xs text-left shadow-2xl ring-1 ring-emerald-900/20">
            <div className="flex items-center gap-1.5 pb-3 mb-3 border-b border-neutral-800/80">
              <span className="size-2.5 rounded-full bg-neutral-700" />
              <span className="size-2.5 rounded-full bg-neutral-700" />
              <span className="size-2.5 rounded-full bg-neutral-700" />
              <span className="ml-auto text-[10px] text-neutral-600 font-mono">bash</span>
            </div>
            <div className="space-y-3">
              <div>
                <span className="text-neutral-600"># store secrets in encrypted vault</span>
                <div className="text-neutral-300 mt-0.5">mayfly set STRIPE_SECRET=sk_live_...</div>
              </div>
              <div>
                <span className="text-neutral-600"># inject into process memory at runtime</span>
                <div className="text-emerald-400 mt-0.5">mayfly run npm run dev</div>
              </div>
              <div>
                <span className="text-neutral-600"># verify cryptographic audit trail</span>
                <div className="text-neutral-300 mt-0.5">mayfly audit verify</div>
              </div>
            </div>
          </div>

          {/* CTA Buttons */}
          <div className="mt-8 flex flex-wrap items-center justify-center gap-3">
            <Link
              href="/docs"
              className="inline-flex items-center gap-2 rounded-lg bg-emerald-500 px-5 py-2.5 text-sm font-semibold text-black transition-colors hover:bg-emerald-400"
            >
              Read Documentation <ArrowRight className="size-4" />
            </Link>
            <Link
              href="/docs/quickstart"
              className="inline-flex items-center gap-2 rounded-lg border border-neutral-800 bg-neutral-900 px-5 py-2.5 text-sm font-medium text-neutral-300 transition-colors hover:border-emerald-900 hover:bg-neutral-800 hover:text-neutral-100"
            >
              Quickstart Guide
            </Link>
          </div>
        </div>

        {/* Feature Grid */}
        <section className="mt-32 grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
          {[
            {
              icon: <Shield className="size-4.5" />,
              title: 'Zero-Disk Attack Surface',
              description: 'No unencrypted files are ever created on disk. Decrypted secrets reside strictly in ephemeral process memory.',
            },
            {
              icon: <Lock className="size-4.5" />,
              title: 'AES-256-GCM + PBKDF2',
              description: 'Authenticated encryption with 600,000 PBKDF2-HMAC-SHA256 iterations and a fresh random nonce per write.',
            },
            {
              icon: <Cpu className="size-4.5" />,
              title: 'Inode Project Binding',
              description: 'Secrets are bound to hardware filesystem inodes, not directory names. Project A secrets cannot bleed into Project B.',
            },
            {
              icon: <Terminal className="size-4.5" />,
              title: 'Direct OS Execution',
              description: 'Commands are spawned via os/exec, bypassing shell history logging, variable expansion, and subshell leakage.',
            },
            {
              icon: <CheckCircle2 className="size-4.5" />,
              title: 'Tamper-Evident Audit Log',
              description: 'SHA-256 hash-chained event log detects any modification, deletion, or reordering of historical access records.',
            },
            {
              icon: <span className="font-mono text-[11px] font-bold">0-DEP</span>,
              title: 'Zero External Dependencies',
              description: 'Pure Go standard library. No go.mod require blocks. Immune to supply-chain dependency hijacking by design.',
            },
          ].map((card) => (
            <div
              key={card.title}
              className="group rounded-xl border border-neutral-800/60 bg-neutral-900/30 p-6 transition-all duration-200 hover:border-emerald-900/50 hover:bg-neutral-900/60"
            >
              <div className="mb-4 inline-flex size-9 items-center justify-center rounded-lg border border-neutral-800 bg-neutral-900 text-emerald-500 group-hover:border-emerald-900/50 group-hover:bg-emerald-950/30 transition-colors duration-200">
                {card.icon}
              </div>
              <h3 className="font-semibold text-neutral-100 text-sm mb-2">{card.title}</h3>
              <p className="text-sm text-neutral-500 leading-relaxed">{card.description}</p>
            </div>
          ))}
        </section>
      </main>

      {/* Footer */}
      <footer className="border-t border-neutral-800/50 py-10 text-center text-xs text-neutral-600 font-mono">
        MayFly &bull; AGPL-3.0 &bull; Zero-Dependency Hackathon &bull;{' '}
        <a
          href="https://github.com/vishnunandan555/mayfly"
          className="text-neutral-500 hover:text-emerald-500 transition-colors"
          target="_blank"
          rel="noopener noreferrer"
        >
          github.com/vishnunandan555/mayfly
        </a>
      </footer>
    </div>
  );
}
