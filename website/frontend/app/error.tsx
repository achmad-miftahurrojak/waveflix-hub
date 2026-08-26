"use client";

import { useEffect } from "react";

export default function Error({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  useEffect(() => {
    console.error(error);
  }, [error]);
  return (
    <main className="flex min-h-screen flex-col items-center justify-center bg-black px-6 text-center">
      <h1 className="text-4xl font-bold text-white md:text-5xl">
        Terjadi Kesalahan
      </h1>
      <p className="mt-4 max-w-md text-white/60">
        Ada yang tidak beres di sisi kami. Coba muat ulang halaman, atau kembali
        beberapa saat lagi.
      </p>
      <button
        onClick={reset}
        className="mt-8 rounded bg-accent px-6 py-3 font-semibold text-black transition hover:bg-accent-dark"
      >
        Coba Lagi
      </button>
      {error.digest ? (
        <p className="mt-6 text-xs text-white/50">Kode error: {error.digest}</p>
      ) : null}
    </main>
  );
}
