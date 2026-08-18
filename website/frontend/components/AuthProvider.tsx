"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
  type ReactNode,
} from "react";
import type { TmdbItem } from "@/lib/types";
import { itemTitle, mediaTypeOf } from "@/lib/helpers";
import { TOKEN_KEY, migrateStorage } from "@/lib/storage";

const BACKEND = process.env.NEXT_PUBLIC_BACKEND_URL ?? "http://localhost:8080";

export interface AuthUser {
  id: number;
  email: string;
  username: string;
  avatar?: string;
  banner?: string;
  bio?: string;
  name_font?: string;
  joined?: string;
}

export interface ProfileFields {
  username?: string;
  email?: string;
  bio?: string;
  name_font?: string;
}

interface AuthContextValue {
  user: AuthUser | null;
  token: string | null;
  ready: boolean;
  login: (email: string, password: string) => Promise<string | null>;
  register: (email: string, username: string, password: string) => Promise<string | null>;
  logout: () => void;
  authFetch: (path: string, init?: RequestInit) => Promise<Response>;
  recordHistory: (item: TmdbItem, season?: number, episode?: number) => void;
  updateProfile: (fields: ProfileFields) => Promise<string | null>;
  changePassword: (current: string, next: string) => Promise<string | null>;
  uploadImage: (
    field: "avatar" | "banner",
    dataUrl: string
  ) => Promise<string | null>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth harus dipakai di dalam <AuthProvider>");
  return ctx;
}

export default function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<AuthUser | null>(null);
  const [token, setToken] = useState<string | null>(null);
  const [ready, setReady] = useState(false);

  // Muat token dari sessionStorage saat pertama render.
  // sessionStorage = sesi hilang saat browser ditutup → user harus login ulang.
  useEffect(() => {
    migrateStorage(); // pindahkan kunci lama (summertide_*) → baru sekali saja
    const saved = sessionStorage.getItem(TOKEN_KEY);
    if (!saved) {
      setReady(true);
      return;
    }
    setToken(saved);
    fetch(`${BACKEND}/api/auth/me`, {
      headers: { Authorization: `Bearer ${saved}` },
    })
      .then((r) => (r.ok ? r.json() : Promise.reject()))
      .then((u) => setUser(u))
      .catch(() => {
        sessionStorage.removeItem(TOKEN_KEY);
        setToken(null);
      })
      .finally(() => setReady(true));
  }, []);

  const persist = (tok: string, u: AuthUser) => {
    sessionStorage.setItem(TOKEN_KEY, tok);
    setToken(tok);
    setUser(u);
  };

  const login = useCallback(async (email: string, password: string) => {
    try {
      const res = await fetch(`${BACKEND}/api/auth/login`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email, password }),
      });
      const data = await res.json();
      if (!res.ok) return data.error || "Gagal masuk";
      persist(data.token, data.user);
      return null;
    } catch {
      return "Tidak bisa terhubung ke server";
    }
  }, []);

  const register = useCallback(
    async (email: string, username: string, password: string) => {
      try {
        const res = await fetch(`${BACKEND}/api/auth/register`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ email, username, password }),
        });
        const data = await res.json();
        if (!res.ok) return data.error || "Gagal daftar";
        persist(data.token, data.user);
        return null;
      } catch {
        return "Tidak bisa terhubung ke server";
      }
    },
    []
  );

  const logout = useCallback(() => {
    sessionStorage.removeItem(TOKEN_KEY);
    setToken(null);
    setUser(null);
  }, []);

  const authFetch = useCallback(
    (path: string, init: RequestInit = {}) =>
      fetch(`${BACKEND}${path}`, {
        ...init,
        headers: {
          ...(init.headers || {}),
          "Content-Type": "application/json",
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
        },
      }),
    [token]
  );

  const updateProfile = useCallback(
    async (fields: ProfileFields) => {
      try {
        const res = await authFetch("/api/auth/profile", {
          method: "PATCH",
          body: JSON.stringify(fields),
        });
        const data = await res.json();
        if (!res.ok) return data.error || "Failed to update profile";
        setUser(data);
        return null;
      } catch {
        return "Cannot reach the server";
      }
    },
    [authFetch]
  );

  const changePassword = useCallback(
    async (current: string, next: string) => {
      try {
        const res = await authFetch("/api/auth/password", {
          method: "POST",
          body: JSON.stringify({ current_password: current, new_password: next }),
        });
        const data = await res.json();
        if (!res.ok) return data.error || "Failed to change password";
        return null;
      } catch {
        return "Cannot reach the server";
      }
    },
    [authFetch]
  );

  const uploadImage = useCallback(
    async (field: "avatar" | "banner", dataUrl: string) => {
      try {
        const res = await authFetch(`/api/auth/${field}`, {
          method: "POST",
          body: JSON.stringify({ image: dataUrl }),
        });
        const data = await res.json();
        if (!res.ok) return data.error || "Failed to upload image";
        setUser(data);
        return null;
      } catch {
        return "Cannot reach the server";
      }
    },
    [authFetch]
  );

  const recordHistory = useCallback(
    (item: TmdbItem, season?: number, episode?: number) => {
      if (!token) return;
      authFetch("/api/history", {
        method: "POST",
        body: JSON.stringify({
          tmdb_id: item.id,
          media_type: mediaTypeOf(item),
          title: itemTitle(item),
          poster_path: item.poster_path || "",
          vote_average: item.vote_average || 0,
          season,
          episode,
        }),
      }).catch(() => {});
    },
    [token, authFetch]
  );

  return (
    <AuthContext.Provider
      value={{
        user,
        token,
        ready,
        login,
        register,
        logout,
        authFetch,
        recordHistory,
        updateProfile,
        changePassword,
        uploadImage,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}
