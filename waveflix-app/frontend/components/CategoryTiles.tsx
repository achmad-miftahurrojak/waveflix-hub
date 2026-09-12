"use client";

import Link from "next/link";
import { ChevronRight } from "./Icons";
import { motion } from "framer-motion";

interface Tile {
  label: string;
  href: string;
  logo?: string;
}

export default function CategoryTiles({
  title,
  tiles,
  compact,
  disableWrapper = false,
}: {
  title: string;
  tiles: Tile[];
  compact?: boolean;
  disableWrapper?: boolean;
}) {
  const content = (
    <>
      <h1 className="mb-8 text-3xl font-bold">{title}</h1>
      <div
        className={`grid gap-3 ${
          compact
            ? "grid-cols-3 sm:grid-cols-4 md:grid-cols-6 lg:grid-cols-8"
            : "grid-cols-2 sm:grid-cols-3 lg:grid-cols-5"
        }`}
      >
        {tiles.map((t) => (
          <motion.div
            key={t.href}
            whileHover={{ scale: 1.05 }}
            transition={{ type: "spring", stiffness: 300, damping: 20 }}
          >
            <Link
              href={t.href}
              className="group flex h-full items-center justify-between rounded-xl bg-white/[0.04] px-5 py-6 ring-1 ring-white/10 transition-colors duration-300 hover:bg-white/[0.08] hover:ring-accent/40 shadow-lg hover:shadow-accent/20"
            >
              {t.logo ? (
                <img src={t.logo} alt={t.label} className="h-24 object-contain grayscale transition group-hover:grayscale-0 brightness-200 group-hover:brightness-100" />
              ) : (
                <span className="font-semibold text-white/90 group-hover:text-white">
                  {t.label}
                </span>
              )}
              <ChevronRight className="text-white/30 transition group-hover:text-accent" />
            </Link>
          </motion.div>
        ))}
      </div>
    </>
  );

  if (disableWrapper) {
    return <div className="px-[4%]">{content}</div>;
  }

  return (
    <main className="min-h-screen px-[4%] pb-16 pt-28">
      {content}
    </main>
  );
}
