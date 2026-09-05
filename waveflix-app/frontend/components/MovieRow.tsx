"use client";

import Link from "next/link";
import type { TmdbItem } from "@/lib/types";
import Carousel from "./Carousel";
import { useTranslation } from "@/lib/i18n";

interface Props {
  title: string;
  items: TmdbItem[];
  href?: string;
  noPadding?: boolean;
}

export default function MovieRow({ title, items, href, noPadding }: Props) {
  const { t } = useTranslation();
  if (items.length === 0) return null;
  return (
    <section className="mb-8">
      <div className={`mb-3 flex items-center justify-between ${noPadding ? '' : 'px-[4%]'}`}>
        <h2 className="text-xl font-bold text-gray-100">{t(title)}</h2>
        <Link
          href={href || "#"}
          className="group flex items-center text-sm font-semibold text-gray-400 transition-colors hover:text-white"
        >
          <span className="hidden sm:inline-block mr-1">{t("ui.viewAll")}</span>
          <span className="text-xl leading-none transition-transform group-hover:translate-x-1 text-[var(--color-accent)]">&rsaquo;</span>
        </Link>
      </div>
      <Carousel items={items} noPadding={noPadding} />
    </section>
  );
}