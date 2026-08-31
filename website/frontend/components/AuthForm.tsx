"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { type FormEvent, useState } from "react";
import { useAuth } from "./AuthProvider";
import { CheckIcon } from "./Icons";
import { BACKEND } from "@/lib/helpers";

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

  const [step, setStep] = useState<"form" | "code">("form");
  const [code, setCode] = useState("");
  const [sendingCode, setSendingCode] = useState(false);

  const isRegister = mode === "register";

  const sendCode = async (): Promise<boolean> => {
    setSendingCode(true);
    try {
      const res = await fetch(`${BACKEND}/api/auth/send-code`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email }),
      });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) {
        setError(data.error || "Gagal mengirim kode verifikasi.");
        return false;
      }
      setError(null);
      setStep("code");
      if (data.test_code) setCode(data.test_code);
      return true;
    } catch {
      setError("Tidak bisa terhubung ke server.");
      return false;
    } finally {
      setSendingCode(false);
    }
  };

  const submit = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setError(null);
    if (isRegister && step === "form") {
      await sendCode();
      return;
    }
    if (isRegister && !/^\d{6}$/.test(code)) {
      setError("Masukkan 6 digit kode verifikasi.");
      return;
    }
    setLoading(true);
    const err = isRegister
      ? await register(email, username, password, code)
      : await login(email, password);
    setLoading(false);
    if (err) setError(err);
    else router.push("/home");
  };

  const inputCls =
    "w-full rounded-lg border border-white/15 bg-white/5 px-4 py-3 text-sm outline-none transition focus:border-accent focus:bg-white/[0.07]";

  const topBar = (
    <header className="absolute inset-x-0 top-0 z-10 flex items-center justify-between px-6 py-5 md:px-12">
      <Link
        href="/"
        className="text-2xl uppercase tracking-[-0.03em] text-accent"
        style={{ fontFamily: "var(--font-logo)" }}
      >
        Waveflix
      </Link>
      <Link
        href={isRegister ? "/masuk" : "/daftar"}
        className="rounded bg-accent px-5 py-2 text-sm font-semibold text-black transition hover:bg-accent-dark"
      >
        {isRegister ? "Sign In" : "Sign Up"}
      </Link>
    </header>
  );

  if (!isRegister) {
    return (
      <div className="relative min-h-screen bg-bg">
        {topBar}
        <div className="flex min-h-screen items-center justify-center px-6 pb-16 pt-28">
          <div className="w-full max-w-md">
            <h1 className="text-3xl font-bold md:text-4xl">Welcome back</h1>
            <p className="mb-8 mt-2 text-white/60">
              Don&apos;t have an account?{" "}
              <Link href="/daftar" className="font-semibold text-accent hover:underline">
                Sign up
              </Link>
            </p>

            <form onSubmit={submit} className="flex flex-col gap-3">
              <input
                type="email"
                required
                placeholder="Email"
                aria-label="Email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                className={inputCls}
              />
              <input
                type="password"
                required
                placeholder="Password"
                aria-label="Password"
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
                className="mt-2 rounded-lg bg-accent px-4 py-3 font-semibold text-black transition hover:bg-accent-dark disabled:opacity-60"
              >
                {loading ? "Please wait…" : "Sign In"}
              </button>
            </form>

            <p className="mt-10 text-sm text-white/50">
              This page is protected to keep your account safe.
            </p>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="relative grid min-h-screen lg:grid-cols-2">
      {topBar}
      {}
      <div className="relative hidden flex-col justify-center overflow-hidden px-[10%] lg:flex">
        <div className="absolute inset-0 bg-gradient-to-br from-accent/15 via-bg to-bg" />
        <div className="absolute -left-24 top-1/3 h-96 w-96 rounded-full bg-accent/20 blur-[120px]" />
        <div className="relative z-[2] max-w-md">
          <h2 className="text-4xl font-bold leading-tight">
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

      {}
      <div className="flex items-center justify-center px-6 py-16">
        <div className="w-full max-w-sm">
          <h1 className="mb-1 text-3xl font-bold">
            {step === "code" ? "Check your email" : "Create your account"}
          </h1>
          <p className="mb-7 text-sm text-white/50">
            {step === "code"
              ? `We sent a 6-digit code to ${email}. Valid for 10 minutes.`
              : "Free forever. We just need a few basics."}
          </p>

          <form onSubmit={submit} className="flex flex-col gap-3">
            {step === "form" ? (
              <>
                <div>
                  <label htmlFor="reg-email" className="mb-1 block text-xs font-semibold uppercase tracking-wide text-white/50">
                    Email
                  </label>
                  <input
                    id="reg-email"
                    type="email"
                    required
                    placeholder="you@example.com"
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    className={inputCls}
                  />
                </div>

                <div>
                  <label htmlFor="reg-name" className="mb-1 block text-xs font-semibold uppercase tracking-wide text-white/50">
                    Display name
                  </label>
                  <input
                    id="reg-name"
                    type="text"
                    required
                    placeholder="Your name"
                    value={username}
                    onChange={(e) => setUsername(e.target.value)}
                    className={inputCls}
                  />
                </div>

                <div>
                  <label htmlFor="reg-password" className="mb-1 block text-xs font-semibold uppercase tracking-wide text-white/50">
                    Password
                  </label>
                  <input
                    id="reg-password"
                    type="password"
                    required
                    minLength={8}
                    placeholder="At least 8 characters"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    className={inputCls}
                  />
                </div>
              </>
            ) : (
              <div>
                <label htmlFor="reg-code" className="mb-1 block text-xs font-semibold uppercase tracking-wide text-white/50">
                  Verification code
                </label>
                <input
                  id="reg-code"
                  type="text"
                  inputMode="numeric"
                  autoComplete="one-time-code"
                  required
                  maxLength={6}
                  placeholder="••••••"
                  value={code}
                  onChange={(e) => setCode(e.target.value.replace(/\D/g, ""))}
                  className={`${inputCls} text-center text-2xl font-bold tracking-[0.4em]`}
                  autoFocus
                />
              </div>
            )}

            {error && (
              <p className="rounded-md bg-red-500/15 px-3 py-2 text-sm text-red-300">
                {error}
              </p>
            )}

            <button
              type="submit"
              disabled={loading || sendingCode}
              className="mt-2 rounded-lg bg-accent px-4 py-3 font-semibold text-black transition hover:bg-accent-dark disabled:opacity-60"
            >
              {sendingCode
                ? "Sending code…"
                : loading
                ? "Please wait…"
                : step === "form"
                ? "Send Verification Code"
                : "Verify & Create Account"}
            </button>
          </form>

          {step === "code" && (
            <div className="mt-4 flex items-center justify-between text-sm text-white/60">
              <button
                onClick={() => {
                  setStep("form");
                  setCode("");
                  setError(null);
                }}
                className="hover:text-white"
              >
                ← Edit details
              </button>
              <button
                onClick={sendCode}
                disabled={sendingCode}
                className="font-semibold text-accent hover:underline disabled:opacity-60"
              >
                {sendingCode ? "Sending…" : "Resend code"}
              </button>
            </div>
          )}

          <p className="mt-6 text-center text-sm text-white/60">
            Already have an account?{" "}
            <Link href="/masuk" className="font-semibold text-accent hover:underline">
              Sign in
            </Link>
          </p>
        </div>
      </div>
    </div>
  );
}
