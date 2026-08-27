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
import { itemTitle, mediaTypeOf, BACKEND } from "@/lib/helpers";
import { TOKEN_KEY, migrateStorage } from "@/lib/storage";

export interface AuthUser {
  id: number;
  email: string;
  username: string;
  avatar?: string;
  banner?: string;
  bio?: string;
  name_font?: string;
  language?: string;
  joined?: string;
}

export interface ProfileFields {
  username?: string;
  email?: string;
  bio?: string;
  name_font?: string;
  language?: string;
}

export interface ActiveProfile {
  id: number;
  name?: string;
  avatar?: string;
  banner?: string;
  bio?: string;
}

interface AuthContextValue {
  user: AuthUser | null;
  token: string | null;
  ready: boolean;
  login: (email: string, password: string) => Promise<string | null>;
  register: (email: string, username: string, password: string, code: string) => Promise<string | null>;
  logout: () => void;
  authFetch: (path: string, init?: RequestInit) => Promise<Response>;
  recordHistory: (item: TmdbItem, season?: number, episode?: number, currentProgressSeconds?: number) => void;
  updateProfile: (fields: ProfileFields) => Promise<string | null>;
  changePassword: (current: string, next: string) => Promise<string | null>;
  uploadImage: (
    field: "avatar" | "banner",
    dataUrl: string
  ) => Promise<string | null>;
  activeProfile: ActiveProfile | null;
  setActiveProfile: (profile: ActiveProfile | null) => void;
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
  const [activeProfile, setActiveProfile] = useState<ActiveProfile | null>(null);
  const [ready, setReady] = useState(false);

  // Cookie HttpOnly is the normal session. A legacy token is read once for migration.
  useEffect(() => {
    migrateStorage();
    const saved = localStorage.getItem(TOKEN_KEY);
    fetch(`${BACKEND}/api/auth/me`, {
      credentials: "include",
      headers: saved ? { Authorization: `Bearer ${saved}` } : undefined,
    })
      .then((r) => {
        if (!r.ok) return Promise.reject({ status: r.status });
        return r.json();
      })
      .then((u) => {
        setToken(saved || "cookie");
        setUser(u);
        if (u.language) document.cookie = `waveflix_lang=${u.language}; path=/; max-age=31536000`;
        const storedProfileId = localStorage.getItem("activeProfileId");
        if (storedProfileId) {
          fetch(`${BACKEND}/api/profiles/${storedProfileId}`, {
            credentials: "include",
            headers: saved ? { Authorization: `Bearer ${saved}` } : undefined,
          })
            .then((r) => r.ok ? r.json() : null)
            .then((p) => {
              if (p) setActiveProfile(p);
              else setActiveProfile({ id: parseInt(storedProfileId, 10) });
            })
            .catch(() => setActiveProfile({ id: parseInt(storedProfileId, 10) }));
        }
      })
      .catch((err) => {
        // Hanya hapus token jika server benar-benar menolaknya (401/403)
        // Jangan hapus token saat network error (backend mungkin sedang restart)
        if (err?.status === 401 || err?.status === 403) {
          localStorage.removeItem(TOKEN_KEY);
          setToken(null);
        }
        // Jika network error: token tetap ada, user tetap "logged in"
      })
      .finally(() => setReady(true));
  }, []);

  const persist = (tok: string, u: AuthUser) => {
    // The backend sets the HttpOnly cookie. Keep the token in memory only for legacy fallback.
    setToken(tok || null);
    localStorage.removeItem(TOKEN_KEY);
    setUser(u);
    if (u.language) document.cookie = `waveflix_lang=${u.language}; path=/; max-age=31536000`;
  };

  const login = useCallback(async (email: string, password: string) => {
    try {
      const res = await fetch(`${BACKEND}/api/auth/login`, {
        method: "POST",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email, password }),
      });
      const data = await res.json();
      if (!res.ok) return data.error || "Gagal masuk";
      sessionStorage.removeItem("profileSelected"); // wajib pilih profil setelah login
      persist(data.token, data.user);
      return null;
    } catch {
      return "Tidak bisa terhubung ke server";
    }
  }, []);

  const register = useCallback(
    async (email: string, username: string, password: string, code: string) => {
      try {
        const res = await fetch(`${BACKEND}/api/auth/register`, {
          method: "POST",
          credentials: "include",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ email, username, password, code }),
        });
        const data = await res.json();
        if (!res.ok) return data.error || "Gagal daftar";
        sessionStorage.removeItem("profileSelected"); // wajib pilih profil setelah daftar
        persist(data.token, data.user);
        return null;
      } catch {
        return "Tidak bisa terhubung ke server";
      }
    },
    []
  );

  const logout = useCallback(() => {
    fetch(`${BACKEND}/api/auth/logout`, { method: "POST", credentials: "include" }).catch(() => {});
    localStorage.removeItem(TOKEN_KEY);
    localStorage.removeItem("activeProfileId");
    sessionStorage.removeItem("profileSelected");
    setToken(null);
    setUser(null);
    setActiveProfile(null);
  }, []);

  const authFetch = useCallback(
    (path: string, init: RequestInit = {}) => {
      const headers: Record<string, string> = {
        ...(init.headers as Record<string, string> || {}),
        "Content-Type": "application/json",
      };
      
      if (token && token !== "cookie") {
        headers["Authorization"] = `Bearer ${token}`;
      }
      
      const profileId = localStorage.getItem("activeProfileId");
      if (profileId) {
        headers["X-Profile-ID"] = profileId;
      }

      return fetch(`${BACKEND}${path}`, {
        ...init,
        credentials: "include",
        headers,
      });
    },
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
    (item: TmdbItem, season?: number, episode?: number, currentProgressSeconds?: number) => {
      if (!token) return;
      
      const detail = item as TmdbItem & { runtime?: number; episode_run_time?: number[] };
      let runtime = detail.runtime || 0;
      if (mediaTypeOf(item) === "tv" && (!runtime || runtime === 0)) {
        runtime = detail.episode_run_time?.[0] || 45;
      }
      if (runtime === 0) runtime = 120; // default for movie
      
      const progress = currentProgressSeconds !== undefined 
        ? Math.floor(currentProgressSeconds / 60) 
        : Math.floor(runtime * 0.7);

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
          runtime,
          progress,
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
        activeProfile,
        setActiveProfile: (profile: ActiveProfile | null) => {
          if (profile) {
            localStorage.setItem("activeProfileId", profile.id.toString());
          } else {
            localStorage.removeItem("activeProfileId");
          }
          setActiveProfile(profile);
        }
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}
