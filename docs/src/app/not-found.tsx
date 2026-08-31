import Link from 'next/link';

export default function NotFound() {
  return (
    <div className="flex min-h-[70vh] flex-col items-center justify-center text-center px-4">
      <div className="inline-flex size-14 items-center justify-center rounded-2xl border border-amber-800/40 bg-amber-950/20 text-amber-400 mb-6 shadow-lg shadow-amber-950/30">
        <span className="font-mono text-xl font-bold">404</span>
      </div>
      <h1 className="text-2xl sm:text-3xl font-semibold tracking-tight text-neutral-100 mb-3">
        Page Not Found
      </h1>
      <p className="max-w-md text-sm text-neutral-400 font-subtext leading-relaxed mb-8">
        The documentation page you are looking for might have been moved, renamed, or does not exist.
      </p>
      <div className="flex flex-wrap items-center justify-center gap-3">
        <Link
          href="/"
          className="inline-flex h-9 items-center justify-center rounded-lg bg-amber-500 px-4 text-xs font-semibold text-black hover:bg-amber-400 transition-colors shadow-sm"
        >
          Return Home
        </Link>
        <Link
          href="/docs"
          className="inline-flex h-9 items-center justify-center rounded-lg border border-neutral-800 bg-neutral-900 px-4 text-xs font-medium text-neutral-300 hover:border-neutral-700 hover:text-white transition-colors"
        >
          Browse Documentation
        </Link>
      </div>
    </div>
  );
}
