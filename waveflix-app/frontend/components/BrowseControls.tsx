"use client";

import { useState, useRef, useEffect } from "react";
import { useRouter, useSearchParams, usePathname } from "next/navigation";

const OPTIONS = [
  { value: "terpopuler", label: "Terpopuler" },
  { value: "terbaru", label: "Terbaru" },
  { value: "terlama", label: "Terlama" },
];

export default function BrowseControls({ defaultSort, showMediaTabs = true }: { defaultSort: string; showMediaTabs?: boolean }) {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();

  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  const current = OPTIONS.find((o) => o.value === defaultSort) ?? OPTIONS[0];

  const media = searchParams.get("media") || "all";

  const handleSelect = (val: string) => {
    setOpen(false);
    const params = new URLSearchParams(searchParams.toString());
    params.set("sort_by", val);
    router.push(`${pathname}?${params.toString()}`, { scroll: false });
  };

  const handleMedia = (val: string) => {
    const params = new URLSearchParams(searchParams.toString());
    if (val === "all") {
      params.delete("media");
    } else {
      params.set("media", val);
    }
    params.delete("page");
    router.push(`${pathname}?${params.toString()}`, { scroll: false });
  };

  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false);
      }
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, []);

  return (
    <div 
      className="flex flex-wrap items-center gap-1 rounded-full border border-white/10 bg-white/5 p-1 shadow-lg backdrop-blur-xl" 
      ref={ref}
    >
      {showMediaTabs && (
        <>
          <div className="flex items-center gap-1">
            {(["all", "movie", "tv"] as const).map((m) => (
              <button
                key={m}
                onClick={() => handleMedia(m)}
                className={`rounded-full px-3.5 py-1.5 text-sm font-medium transition-all ${
                  media === m
                    ? "bg-white text-black shadow-sm"
                    : "text-white/70 hover:bg-white/10 hover:text-white"
                }`}
              >
                {m === "movie" ? "Film" : m === "tv" ? "Series" : "Semua"}
              </button>
            ))}
          </div>
          <div className="mx-1 h-4 w-[1px] bg-white/20" />
        </>
      )}

      <div className="relative">
        <button
          onClick={() => setOpen((v) => !v)}
          aria-haspopup="listbox"
          aria-expanded={open}
          className="flex items-center gap-2 rounded-full px-3.5 py-1.5 text-sm font-medium text-white/90 transition-all hover:bg-white/10 hover:text-white"
        >
          <span>{current.label}</span>
          <svg
            width="14"
            height="14"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2.5"
            strokeLinecap="round"
            strokeLinejoin="round"
            className={`text-white/50 transition-transform duration-200 ${open ? "rotate-180" : ""}`}
          >
            <polyline points="6 9 12 15 18 9" />
          </svg>
        </button>

        {}
        {open && (
          <ul
            role="listbox"
            className="absolute right-0 top-full mt-2 z-50 min-w-[150px] overflow-hidden rounded-2xl border border-white/10 bg-black/40 py-1.5 shadow-[0_8px_32px_rgba(0,0,0,0.37)] backdrop-blur-xl"
          >
            {OPTIONS.map((opt) => {
              const isActive = opt.value === current.value;
              return (
                <li
                  key={opt.value}
                  role="option"
                  aria-selected={isActive}
                  onClick={() => handleSelect(opt.value)}
                  className={`flex cursor-pointer items-center gap-2.5 px-4 py-2.5 text-sm transition-colors ${
                    isActive
                      ? "text-white bg-white/[0.06]"
                      : "text-white/60 hover:bg-white/[0.08] hover:text-white"
                  }`}
                >
                  {}
                  <span
                    className={`h-1.5 w-1.5 flex-none rounded-full transition-all ${
                      isActive ? "bg-cyan-400 scale-110" : "bg-transparent"
                    }`}
                  />
                  {opt.label}
                </li>
              );
            })}
          </ul>
        )}
      </div>
    </div>
  );
}
