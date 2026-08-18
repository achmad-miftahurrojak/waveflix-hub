"use client";

import { useRouter } from "next/navigation";
import { type FormEvent, useRef, useState } from "react";
import { SearchIcon } from "./Icons";

export default function SearchBox() {
  const router = useRouter();
  const [open, setOpen] = useState(false);
  const [value, setValue] = useState("");
  const inputRef = useRef<HTMLInputElement>(null);

  const submit = (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const q = value.trim();
    if (q) router.push(`/search?q=${encodeURIComponent(q)}`);
  };

  return (
    <form onSubmit={submit} className="flex items-center">
      <button
        type="button"
        aria-label="Cari"
        onClick={() => {
          setOpen((o) => !o);
          setTimeout(() => inputRef.current?.focus(), 50);
        }}
        className="text-white/80 transition hover:text-accent"
      >
        <SearchIcon />
      </button>
      <input
        ref={inputRef}
        value={value}
        onChange={(e) => setValue(e.target.value)}
        placeholder="Cari film atau series…"
        className={`ml-2 rounded-full bg-white/10 text-sm text-white outline-none transition-all duration-300 focus:ring-1 focus:ring-accent ${
          open ? "w-44 px-4 py-1.5 md:w-56" : "w-0 px-0 py-0"
        }`}
      />
    </form>
  );
}
