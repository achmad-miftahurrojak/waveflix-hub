"use client";

import { useRouter, useSearchParams, usePathname } from "next/navigation";
import { cn } from "@/lib/utils";
import { useCallback } from "react";

const FILTER_TYPES = [
  { id: "movie", label: "Film" },
  { id: "tv", label: "Series" },
  { id: "animation", label: "Animation" },
];

const FILTER_COUNTRIES = [
  { id: "US", label: "US" },
  { id: "GB", label: "UK" },
  { id: "JP", label: "Japan" },
  { id: "KR", label: "Korea" },
  { id: "MY", label: "Malaysia" },
];

export default function CollectionFilters() {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();

  const activeTypes = searchParams.get("type")?.split(",").filter(Boolean) || [];
  const activeCountries = searchParams.get("country")?.split(",").filter(Boolean) || [];

  const toggleFilter = useCallback((category: "type" | "country", value: string) => {
    const params = new URLSearchParams(searchParams.toString());
    const current = params.get(category)?.split(",").filter(Boolean) || [];
    
    let next;
    if (current.includes(value)) {
      next = current.filter(v => v !== value);
    } else {
      next = [...current, value];
    }

    if (next.length > 0) {
      params.set(category, next.join(","));
    } else {
      params.delete(category);
    }

    router.push(`${pathname}?${params.toString()}`, { scroll: false });
  }, [searchParams, pathname, router]);

  const FilterButton = ({ active, label, onClick }: { active: boolean, label: string, onClick: () => void }) => (
    <button
      onClick={onClick}
      className={cn(
        "px-4 py-1.5 rounded-full text-sm font-semibold transition-all duration-200 border",
        active 
          ? "bg-white text-black border-white shadow-[0_0_10px_rgba(255,255,255,0.3)]" 
          : "bg-black/50 text-white/70 border-white/20 hover:border-white/50 hover:text-white"
      )}
    >
      {label}
    </button>
  );

  return (
    <div className="flex flex-wrap items-center gap-6">
      <div className="flex items-center gap-2">
        <span className="text-xs font-bold text-white/50 uppercase tracking-wider mr-2">Type</span>
        {FILTER_TYPES.map(t => (
          <FilterButton 
            key={t.id} 
            active={activeTypes.includes(t.id)} 
            label={t.label} 
            onClick={() => toggleFilter("type", t.id)} 
          />
        ))}
      </div>
      
      <div className="w-px h-6 bg-white/10 hidden md:block"></div>

      <div className="flex items-center gap-2">
        <span className="text-xs font-bold text-white/50 uppercase tracking-wider mr-2">Negara</span>
        {FILTER_COUNTRIES.map(c => (
          <FilterButton 
            key={c.id} 
            active={activeCountries.includes(c.id)} 
            label={c.label} 
            onClick={() => toggleFilter("country", c.id)} 
          />
        ))}
      </div>
    </div>
  );
}
