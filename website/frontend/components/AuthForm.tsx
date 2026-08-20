"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { type FormEvent, useState } from "react";
import { useAuth } from "./AuthProvider";
import { CheckIcon } from "./Icons";

const PERKS = [
  "Personalized recommendations",
  "Sync your list across devices",
  "Save favorites & watch history",
];

export default function AuthForm({ mode }: { mode: "login" | "register" }) {
  const router = useRouter();
  const { login, register } = useAuth();
  const searchParams = useSearchParams();
  const [email, setEmail] = useState(searchParams.get("email") || "");
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const isRegister = mode === "register";

  const submit = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setError(null);
    setLoading(true);
    const err = isRegister
      ? await register(email, username, password)
      : await login(email, password);
    setLoading(false);
    if (err) setError(err);
    else router.push("/home");
  };

  const inputCls =
    "w-full rounded-lg border border-white/15 bg-white/5 px-4 py-3 text-sm outline-none transition focus:border-accent focus:bg-white/[0.07]";

  return (
    <div className="grid min-h-screen lg:grid-cols-2">
      {/* Branding */}
      <div className="relative hidden flex-col justify-center overflow-hidden px-[10%] lg:flex">
        <div className="absolute inset-0 bg-gradient-to-br from-accent/15 via-bg to-bg" />
        <div className="absolute -left-24 top-1/3 h-96 w-96 rounded-full bg-accent/20 blur-[120px]" />
        <div className="relative z-[2] max-w-md">
          <span
            className="text-5xl uppercase tracking-[-0.03em] text-accent"
            style={{ fontFamily: "var(--font-logo)" }}
          >
            Waveflix
          </span>
          <h2 className="mt-6 text-4xl font-extrabold leading-tight">
            Your favorites are waiting.
          </h2>
          <p className="mt-4 text-white/60">
            Thousands of movies and series, all in one place. Pick up your
            watchlist or find something new.
          </p>
          <ul className="mt-8 space-y-3">
            {PERKS.map((p) => (
              <li key={p} className="flex items-center gap-3 text-white/80">
                <span className="grid h-6 w-6 shrink-0 place-items-center rounded-full bg-accent/20 text-accent">
                  <CheckIcon className="h-4 w-4" />
                </span>
                {p}
              </li>
            ))}
          </ul>
        </div>
      </div>

      {/* Form */}
      <div className="flex items-center justify-center px-6 py-16">
        <div className="w-full max-w-sm">
          <h1 className="mb-1 text-3xl font-bold">
            {isRegister ? "Create your account" : "Welcome back"}
          </h1>
          <p className="mb-7 text-sm text-white/50">
            {isRegister
              ? "Free forever. We just need a few basics."
              : "Sign in to continue watching."}
          </p>

          <form onSubmit={submit} className="flex flex-col gap-3">
            <div>
              <label className="mb-1 block text-xs font-semibold uppercase tracking-wide text-white/50">
                Email
              </label>
              <input
                type="email"
                required
                placeholder="you@example.com"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                className={inputCls}
              />
            </div>

            {isRegister && (
              <div>
                <label className="mb-1 block text-xs font-semibold uppercase tracking-wide text-white/50">
                  Display name
                </label>
                <input
                  type="text"
                  required
                  placeholder="Your name"
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  className={inputCls}
                />
              </div>
            )}

            <div>
              <label className="mb-1 block text-xs font-semibold uppercase tracking-wide text-white/50">
                Password
              </label>
              <input
                type="password"
                required
                minLength={8}
                placeholder="At least 8 characters"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className={inputCls}
              />
            </div>

            {error && (
              <p className="rounded-md bg-red-500/15 px-3 py-2 text-sm text-red-300">
                {error}
              </p>
            )}

            <button
              type="submit"
              disabled={loading}
              className="mt-2 rounded-lg bg-accent px-4 py-3 font-semibold text-black transition hover:bg-accent-dark disabled:opacity-60"
            >
              {loading
                ? "Please wait…"
                : isRegister
                ? "Sign Up"
                : "Sign In"}
            </button>
          </form>

          <p className="mt-6 text-center text-sm text-white/60">
            {isRegister ? (
              <>
                Already have an account?{" "}
                <Link href="/masuk" className="font-semibold text-accent hover:underline">
                  Sign in
                </Link>
              </>
            ) : (
              <>
                New to Waveflix?{" "}
                <Link href="/daftar" className="font-semibold text-accent hover:underline">
                  Create account
                </Link>
              </>
            )}
          </p>
        </div>
      </div>
    </div>
  );
}
