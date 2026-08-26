"use client";

import { useEffect, useState } from "react";
import { motion, AnimatePresence } from "framer-motion";
import type { TmdbItem } from "@/lib/types";
import { isTv, mediaTypeOf, itemTitle } from "@/lib/helpers";
import { LIST_KEY, FAV_KEY, migrateStorage } from "@/lib/storage";
import { useUI } from "./UIProvider";
import { useAuth } from "./AuthProvider";
import { useTranslation } from "@/lib/i18n";
import {
  PlayIcon,
  PlusIcon,
  CheckIcon,
  HeartIcon,
  HeartSolidIcon,
} from "./Icons";

function readList(key: string): TmdbItem[] {
  if (typeof window === "undefined") return [];
  try {
    migrateStorage();
    return JSON.parse(localStorage.getItem(key) || "[]");
  } catch {
    return [];
  }
}

function toggleLocal(key: string, item: TmdbItem): boolean {
  const list = readList(key);
  const idx = list.findIndex((m) => m.id === item.id);
  if (idx > -1) list.splice(idx, 1);
  else list.push(item);
  localStorage.setItem(key, JSON.stringify(list));
  return idx === -1;
}

export default function DetailActions({ item }: { item: TmdbItem }) {
  const { t } = useTranslation();
  const { play } = useUI();
  const { user, authFetch } = useAuth();
  const tv = isTv(item);
  const [saved, setSaved] = useState(false);
  const [faved, setFaved] = useState(false);
  const [pending, setPending] = useState<"save" | "fav" | null>(null);

  useEffect(() => {
    let alive = true;
    if (user) {
      authFetch("/api/watchlist")
        .then((r) => r.json())
        .then((d) => {
          if (alive)
            setSaved((d.results ?? []).some((m: any) => m.tmdb_id === item.id));
        })
        .catch(() => {});
      authFetch("/api/favorites")
        .then((r) => r.json())
        .then((d) => {
          if (alive)
            setFaved((d.results ?? []).some((m: any) => m.tmdb_id === item.id));
        })
        .catch(() => {});
    } else {
      setSaved(readList(LIST_KEY).some((m) => m.id === item.id));
      setFaved(readList(FAV_KEY).some((m) => m.id === item.id));
    }
    return () => {
      alive = false;
    };
  }, [user, item.id, authFetch]);

  // Sinkron ke backend saat login: POST untuk tambah, DELETE untuk hapus.
  const remoteToggle = async (endpoint: string, active: boolean) => {
    if (active) {
      await authFetch(
        `${endpoint}?tmdb_id=${item.id}&media_type=${mediaTypeOf(item)}`,
        { method: "DELETE" }
      );
    } else {
      await authFetch(endpoint, {
        method: "POST",
        body: JSON.stringify({
          tmdb_id: item.id,
          media_type: mediaTypeOf(item),
          title: itemTitle(item),
          poster_path: item.poster_path || "",
          vote_average: item.vote_average || 0,
        }),
      });
    }
  };

  const toggleSave = async () => {
    if (pending) return;
    if (user) {
      setPending("save");
      try {
        await remoteToggle("/api/watchlist", saved);
        setSaved(!saved);
      } finally {
        setPending(null);
      }
    } else {
      setSaved(toggleLocal(LIST_KEY, item));
    }
  };

  const toggleFav = async () => {
    if (pending) return;
    if (user) {
      setPending("fav");
      try {
        await remoteToggle("/api/favorites", faved);
        setFaved(!faved);
      } finally {
        setPending(null);
      }
    } else {
      setFaved(toggleLocal(FAV_KEY, item));
    }
  };

  return (
    <div className="flex flex-wrap gap-3">
      {/* Movie diputar langsung; series diputar dari halaman episode. */}
      {!tv && (
        <motion.button
          whileHover={{ scale: 1.05 }}
          whileTap={{ scale: 0.95 }}
          onClick={() => play(item)}
          className="flex items-center gap-2 rounded-md bg-accent px-7 py-3 font-semibold text-black transition hover:bg-accent-dark"
        >
          <PlayIcon className="text-black" /> {t("ui.play")}
        </motion.button>
      )}
      <motion.button
        whileHover={{ scale: 1.05 }}
        whileTap={{ scale: 0.95 }}
        onClick={toggleSave}
        disabled={pending !== null}
        className={`flex items-center gap-2 rounded-md border-2 px-6 py-3 font-semibold transition hover:bg-white/10 disabled:cursor-not-allowed disabled:opacity-50 ${
          saved ? "border-white text-white" : "border-white/40"
        }`}
      >
        <AnimatePresence mode="wait">
          <motion.div
            key={saved ? "saved" : "unsaved"}
            initial={{ scale: 0.5, opacity: 0 }}
            animate={{ scale: 1, opacity: 1 }}
            exit={{ scale: 0.5, opacity: 0 }}
            transition={{ duration: 0.15 }}
          >
            {saved ? <CheckIcon /> : <PlusIcon />}
          </motion.div>
        </AnimatePresence>
        {saved ? t("ui.saved") : t("ui.myList")}
      </motion.button>
      <motion.button
        whileHover={{ scale: 1.05 }}
        whileTap={{ scale: 0.95 }}
        onClick={toggleFav}
        disabled={pending !== null}
        className={`flex items-center gap-2 rounded-md border-2 px-6 py-3 font-semibold transition hover:bg-white/10 disabled:cursor-not-allowed disabled:opacity-50 ${
          faved ? "border-accent text-accent" : "border-white/40"
        }`}
      >
        <AnimatePresence mode="wait">
          <motion.div
            key={faved ? "faved" : "unfaved"}
            initial={{ scale: 0.5, opacity: 0 }}
            animate={{ scale: 1, opacity: 1 }}
            exit={{ scale: 0.5, opacity: 0 }}
            transition={{ duration: 0.15 }}
          >
            {faved ? <HeartSolidIcon /> : <HeartIcon />}
          </motion.div>
        </AnimatePresence>
        {faved ? t("ui.favorited") : t("ui.favorite")}
      </motion.button>
    </div>
  );
}
