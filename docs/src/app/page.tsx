import Link from 'next/link';
import React from 'react';
import { Shield, Lock, Terminal, Cpu, CheckCircle2, ArrowRight, ExternalLink } from 'lucide-react';

export default function HomePage() {
  return (
    <div className="relative min-h-screen bg-neutral-950 text-neutral-100 selection:bg-neutral-800 selection:text-neutral-200">
      {/* Subtle background glow */}
      <div className="absolute inset-0 bg-[radial-gradient(circle_at_50%_0%,rgba(120,119,198,0.1),transparent_50%)] pointer-events-none" />

      {/* Navigation Header */}
      <header className="sticky top-0 z-40 border-b border-neutral-800/80 bg-neutral-950/80 backdrop-blur-md">
        <div className="mx-auto flex h-16 max-w-6xl items-center justify-between px-6">
          <div className="flex items-center gap-3">
            <span className="font-mono text-lg font-bold tracking-wider text-neutral-100">MAYFLY</span>
            <span className="rounded border border-neutral-800 bg-neutral-900 px-2 py-0.5 font-mono text-[11px] text-neutral-400">
              ZERO-DEP
            </span>
          </div>
          <nav className="flex items-center gap-6 text-sm">
            <Link
              href="/docs"
              className="text-neutral-400 transition-colors hover:text-neutral-100"
            >
              Docs
            </Link>
            <Link
              href="/docs/architecture/security-model"
              className="text-neutral-400 transition-colors hover:text-neutral-100"
            >
              Security Model
            </Link>
            <Link
              href="/docs/cli/overview"
              className="text-neutral-400 transition-colors hover:text-neutral-100"
            >
              CLI Reference
            </Link>
            <a
              href="https://github.com/vishnunandan555/mayfly"
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-1 text-neutral-400 transition-colors hover:text-neutral-100"
            >
              GitHub <ExternalLink className="size-3.5" />
            </a>
          </nav>
        </div>
      </header>

      {/* Hero Section */}
      <main className="relative mx-auto max-w-6xl px-6 pt-24 pb-20">
        <div className="flex flex-col items-center text-center">
          <div className="inline-flex items-center gap-2 rounded-full border border-neutral-800 bg-neutral-900/90 px-3.5 py-1 text-xs text-neutral-400 mb-8 font-mono">
            <span className="inline-block size-1.5 rounded-full bg-emerald-500" />
            Go Standard Library Only • 0 External Dependencies
          </div>

          <h1 className="max-w-4xl text-4xl font-semibold tracking-tight text-neutral-100 sm:text-6xl sm:leading-[1.15]">
            The secrets manager that never writes plaintext <span className="font-mono text-neutral-300">.env</span> to disk.
          </h1>

          <p className="mt-6 max-w-2xl text-lg text-neutral-400 leading-relaxed font-normal">
            MayFly eliminates supply-chain credential theft during build and install steps. Secrets remain encrypted at rest and are injected directly into child process RAM via host-level execution.
          </p>

          {/* Action Buttons */}
          <div className="mt-10 flex flex-wrap items-center justify-center gap-4">
            <Link
              href="/docs"
              className="inline-flex items-center gap-2 rounded-lg bg-neutral-100 px-5 py-2.5 text-sm font-medium text-neutral-950 transition-colors hover:bg-neutral-200"
            >
              Read Documentation <ArrowRight className="size-4" />
            </Link>
            <Link
              href="/docs/quickstart"
              className="inline-flex items-center gap-2 rounded-lg border border-neutral-800 bg-neutral-900 px-5 py-2.5 text-sm font-medium text-neutral-300 transition-colors hover:bg-neutral-800 hover:text-neutral-100"
            >
              Quickstart Guide
            </Link>
          </div>

          {/* Quick Terminal Snippet */}
          <div className="mt-12 w-full max-w-2xl rounded-xl border border-neutral-800 bg-neutral-900/80 p-4 font-mono text-xs text-left shadow-2xl backdrop-blur-sm">
            <div className="flex items-center justify-between pb-3 mb-3 border-b border-neutral-800 text-neutral-500">
              <span className="text-[11px]">terminal</span>
              <span className="text-[11px]">bash</span>
            </div>
            <div className="space-y-2 text-neutral-300">
              <div>
                <span className="text-neutral-500"># 1. Store secrets in encrypted vault</span>
                <div className="text-neutral-200 mt-0.5">mayfly set STRIPE_SECRET=sk_live_...</div>
              </div>
              <div className="pt-2">
                <span className="text-neutral-500"># 2. Run target command with in-memory injection</span>
                <div className="text-emerald-400 mt-0.5">mayfly run npm run dev</div>
              </div>
            </div>
          </div>
        </div>

        {/* Feature Grid */}
        <section className="mt-28 grid grid-cols-1 gap-6 md:grid-cols-2 lg:grid-cols-3">
          <div className="rounded-xl border border-neutral-800/80 bg-neutral-900/40 p-6 transition-all hover:border-neutral-700">
            <div className="mb-4 inline-flex size-10 items-center justify-center rounded-lg border border-neutral-800 bg-neutral-900 text-neutral-300">
              <Shield className="size-5" />
            </div>
            <h3 className="font-semibold text-neutral-100 text-base">Zero-Disk Attack Surface</h3>
            <p className="mt-2 text-sm text-neutral-400 leading-relaxed">
              No unencrypted files are ever created on disk. Transient decrypted secrets reside strictly in ephemeral process memory.
            </p>
          </div>

          <div className="rounded-xl border border-neutral-800/80 bg-neutral-900/40 p-6 transition-all hover:border-neutral-700">
            <div className="mb-4 inline-flex size-10 items-center justify-center rounded-lg border border-neutral-800 bg-neutral-900 text-neutral-300">
              <Lock className="size-5" />
            </div>
            <h3 className="font-semibold text-neutral-100 text-base">AES-256-GCM + PBKDF2</h3>
            <p className="mt-2 text-sm text-neutral-400 leading-relaxed">
              Authenticated encryption with 600,000 PBKDF2-HMAC-SHA256 iterations and per-save cryptographically secure nonces.
            </p>
          </div>

          <div className="rounded-xl border border-neutral-800/80 bg-neutral-900/40 p-6 transition-all hover:border-neutral-700">
            <div className="mb-4 inline-flex size-10 items-center justify-center rounded-lg border border-neutral-800 bg-neutral-900 text-neutral-300">
              <Cpu className="size-5" />
            </div>
            <h3 className="font-semibold text-neutral-100 text-base">Deterministic Inode Isolation</h3>
            <p className="mt-2 text-sm text-neutral-400 leading-relaxed">
              Filesystem-level device and inode resolution ensures secrets are bound to the absolute directory path with zero cross-contamination.
            </p>
          </div>

          <div className="rounded-xl border border-neutral-800/80 bg-neutral-900/40 p-6 transition-all hover:border-neutral-700">
            <div className="mb-4 inline-flex size-10 items-center justify-center rounded-lg border border-neutral-800 bg-neutral-900 text-neutral-300">
              <Terminal className="size-5" />
            </div>
            <h3 className="font-semibold text-neutral-100 text-base">Direct Process Execution</h3>
            <p className="mt-2 text-sm text-neutral-400 leading-relaxed">
              Processes are spawned via direct operating system execution primitives without intermediate shell wrapper evaluation.
            </p>
          </div>

          <div className="rounded-xl border border-neutral-800/80 bg-neutral-900/40 p-6 transition-all hover:border-neutral-700">
            <div className="mb-4 inline-flex size-10 items-center justify-center rounded-lg border border-neutral-800 bg-neutral-900 text-neutral-300">
              <CheckCircle2 className="size-5" />
            </div>
            <h3 className="font-semibold text-neutral-100 text-base">Cryptographic Audit Trail</h3>
            <p className="mt-2 text-sm text-neutral-400 leading-relaxed">
              SHA-256 hash-chained tamper-evident logs record access timestamps and actions, detecting any historical modification.
            </p>
          </div>

          <div className="rounded-xl border border-neutral-800/80 bg-neutral-900/40 p-6 transition-all hover:border-neutral-700">
            <div className="mb-4 inline-flex size-10 items-center justify-center rounded-lg border border-neutral-800 bg-neutral-900 text-neutral-300">
              <span className="font-mono text-xs font-bold text-neutral-300">0-DEP</span>
            </div>
            <h3 className="font-semibold text-neutral-100 text-base">Zero External Dependencies</h3>
            <p className="mt-2 text-sm text-neutral-400 leading-relaxed">
              100% Go standard library. Immune to upstream npm/pip/cargo supply-chain vulnerabilities and transitive dependency hijacking.
            </p>
          </div>
        </section>
      </main>

      {/* Footer */}
      <footer className="border-t border-neutral-800 py-10 text-center text-xs text-neutral-500 font-mono">
        MayFly • Released under AGPL-3.0 • Built for the Zero-Dependency Hackathon
      </footer>
    </div>
  );
}
