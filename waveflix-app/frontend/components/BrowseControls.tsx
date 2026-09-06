"use client";

import { useState, useRef, useEffect } from "react";
import { useRouter, useSearchParams, usePathname } from "next/navigation";

const OPTIONS = [
  { value: "terpopuler", label: "Terpopuler" },
  { value: "terbaru", label: "Terbaru" },
  { value: "terlama", label: "Terlama" },
];

export default function BrowseControls({ defaultSort }: { defaultSort: string }) {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();

  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  const current = OPTIONS.find((o) => o.value === defaultSort) ?? OPTIONS[0];

  const handleSelect = (val: string) => {
    setOpen(false);
    const params = new URLSearchParams(searchParams.toString());
    params.set("sort_by", val);
    router.push(`${pathname}?${params.toString()}`);
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
    <div className="flex items-center gap-2" ref={ref}>
      <span className="text-sm text-white/60">Urutkan:</span>

      {}
      <div className="relative">
        <button
          onClick={() => setOpen((v) => !v)}
          aria-haspopup="listbox"
          aria-expanded={open}
          className="flex items-center gap-2 rounded-full border border-white/10 bg-white/5 px-4 py-2 text-sm font-semibold text-white shadow-lg backdrop-blur-xl transition-all hover:border-white/20 hover:bg-white/10"
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
