"use client";

import {
  createContext,
  useCallback,
  useContext,
  useState,
  type ReactNode,
} from "react";
import type { TmdbItem } from "@/lib/types";
import PlayerModal from "./PlayerModal";
import { useAuth } from "./AuthProvider";

interface PlayerState {
  item: TmdbItem;
  season?: number;
  episode?: number;
}

interface UIContextValue {
  play: (m: TmdbItem, season?: number, episode?: number) => void;
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
    (m: TmdbItem, season?: number, episode?: number) => {
      setPlayer({ item: m, season, episode });
      recordHistory(m, season, episode); // dicatat kalau user login
    },
    [recordHistory]
  );

  return (
    <UIContext.Provider value={{ play }}>
      {children}
      <PlayerModal state={player} onClose={() => setPlayer(null)} />
    </UIContext.Provider>
  );
}
