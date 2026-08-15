"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { useAuth } from "./AuthProvider";

export default function AuthForm({ mode }: { mode: "login" | "register" }) {
  const router = useRouter();
  const { login, register } = useAuth();
  const [email, setEmail] = useState("");
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const isRegister = mode === "register";

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setLoading(true);
    const err = isRegister
      ? await register(email, username, password)
      : await login(email, password);
    setLoading(false);
    if (err) setError(err);
    else router.push("/");
  };

  const inputCls =
    "w-full rounded-md border border-white/15 bg-surface px-4 py-3 text-sm outline-none focus:border-accent";

  return (
    <div className="mx-auto mt-32 max-w-md px-6">
      <div className="rounded-2xl bg-black/40 p-8 ring-1 ring-white/10">
        <h1 className="mb-1 text-2xl font-bold">
          {isRegister ? "Buat Akun" : "Masuk"}
        </h1>
        <p className="mb-6 text-sm text-white/50">
          {isRegister
            ? "Daftar untuk menyimpan watchlist & riwayat tontonan."
            : "Masuk untuk mengakses Daftar Saya kamu."}
        </p>

        <form onSubmit={submit} className="flex flex-col gap-3">
          <input
            type="email"
            required
            placeholder="Email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            className={inputCls}
          />
          {isRegister && (
            <input
              type="text"
              required
              placeholder="Nama pengguna"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              className={inputCls}
            />
          )}
          <input
            type="password"
            required
            minLength={6}
            placeholder="Password (min. 6 karakter)"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            className={inputCls}
          />

          {error && (
            <p className="rounded-md bg-red-500/15 px-3 py-2 text-sm text-red-300">
              {error}
            </p>
          )}

          <button
            type="submit"
            disabled={loading}
            className="mt-2 rounded-md bg-accent px-4 py-3 font-semibold text-black transition hover:bg-accent-dark disabled:opacity-60"
          >
            {loading ? "Memproses…" : isRegister ? "Daftar" : "Masuk"}
          </button>
        </form>

        <p className="mt-5 text-center text-sm text-white/60">
          {isRegister ? (
            <>
              Sudah punya akun?{" "}
              <Link href="/masuk" className="text-accent hover:underline">
                Masuk
              </Link>
            </>
          ) : (
            <>
              Belum punya akun?{" "}
              <Link href="/daftar" className="text-accent hover:underline">
                Daftar
              </Link>
            </>
          )}
        </p>
      </div>
    </div>
  );
}
