"use client";

import { useRouter } from "next/navigation";
import { useAuth } from "./AuthProvider";
import { AuthComponent } from "./ui/auth-component";
import { BACKEND } from "@/lib/helpers";

export default function AuthForm({ mode }: { mode: "login" | "register" }) {
  const router = useRouter();
  const { login, register } = useAuth();

  const handleLogin = async (email: string, pass: string): Promise<string | null> => {
    const err = await login(email, pass);
    return err || null;
  };

  const handleRegisterSendCode = async (email: string): Promise<{ error?: string; test_code?: string }> => {
    try {
      const res = await fetch(`${BACKEND}/api/auth/send-code`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email }),
      });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) {
        return { error: data.error || "Gagal mengirim kode verifikasi." };
      }
      return { test_code: data.test_code };
    } catch {
      return { error: "Tidak bisa terhubung ke server." };
    }
  };

  const handleRegisterVerify = async (email: string, username: string, pass: string, code: string): Promise<string | null> => {
    const err = await register(email, username, pass, code);
    return err || null;
  };

  const handleSuccess = () => {
    router.push("/home");
  };

  return (
    <AuthComponent
      initialMode={mode}
      onLogin={handleLogin}
      onRegisterSendCode={handleRegisterSendCode}
      onRegisterVerify={handleRegisterVerify}
      onSuccess={handleSuccess}
    />
  );
}
