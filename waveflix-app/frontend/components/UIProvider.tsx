"use client";

import {
  createContext,
  useCallback,
  useContext,
  useState,
  type ReactNode,
} from "react";
import type { TmdbItem } from "@/lib/types";
import { useAuth } from "./AuthProvider";

interface PlayerState {
  item: TmdbItem;
  season?: number;
  episode?: number;
  progress?: number;
}

interface UIContextValue {
  play: (m: TmdbItem, season?: number, episode?: number, progress?: number) => void;
  stop: () => void;
  player: PlayerState | null;
}

const UIContext = createContext<UIContextValue | null>(null);

export function useUI(): UIContextValue {
  const ctx = useContext(UIContext);
  if (!ctx) throw new Error("useUI harus dipakai di dalam <UIProvider>");
  return ctx;
}

export default function UIProvider({ children }: { children: ReactNode }) {
  const [player, setPlayer] = useState<PlayerState | null>(null);
  const { recordHistory } = useAuth();

  const play = useCallback(
    (m: TmdbItem, season?: number, episode?: number, progress?: number) => {
      setPlayer({ item: m, season, episode, progress });
    },
    []
  );

  const stop = useCallback(() => setPlayer(null), []);

  return (
    <UIContext.Provider value={{ play, stop, player }}>
      {children}
    </UIContext.Provider>
  );
}
