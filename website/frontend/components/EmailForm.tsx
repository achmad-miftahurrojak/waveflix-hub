"use client";

import { useRouter } from "next/navigation";
import { type FormEvent, useState } from "react";
import { BACKEND } from "@/lib/helpers";

export default function EmailForm() {
  const [email, setEmail] = useState("");
  const [loading, setLoading] = useState(false);
  const router = useRouter();

  const handleSubmit = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    if (!email) return;

    setLoading(true);
    try {
      // Cek apakah email sudah terdaftar di database
      const res = await fetch(
        `${BACKEND}/api/auth/check-email?email=${encodeURIComponent(email)}`
      );
      const data = await res.json();

      if (data.exists) {
        // Email sudah ada → arahkan ke halaman Login dengan email pre-filled
        router.push(`/masuk?email=${encodeURIComponent(email)}`);
      } else {
        // Email belum ada → arahkan ke halaman Daftar dengan email pre-filled
        router.push(`/daftar?email=${encodeURIComponent(email)}`);
      }
    } catch {
      // Kalau server mati, fallback ke halaman daftar
      router.push(`/daftar?email=${encodeURIComponent(email)}`);
    } finally {
      setLoading(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="mt-4 flex flex-col md:flex-row items-center gap-2 max-w-3xl mx-auto w-full">
      <div className="relative w-full flex-1">
        <input
          type="email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          placeholder="Alamat email"
          required
          className="w-full h-14 flex-1 rounded border border-white/40 bg-black/50 px-4 py-4 text-white placeholder-white/60 backdrop-blur outline-none transition focus:border-white"
        />
      </div>
      <button
        type="submit"
        disabled={loading}
        className="flex w-fit items-center justify-center gap-2 rounded bg-accent px-7 h-14 text-xl font-semibold text-black hover:bg-accent-dark transition whitespace-nowrap disabled:opacity-60"
      >
        {loading ? "Mengecek…" : "Mulai"}
        {!loading && (
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3" className="w-5 h-5">
            <path strokeLinecap="round" strokeLinejoin="round" d="m9 18 6-6-6-6" />
          </svg>
        )}
      </button>
    </form>
  );
}
