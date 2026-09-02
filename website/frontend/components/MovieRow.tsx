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
    <section className="mb-6">
      <div className={`mb-3 flex items-center justify-between ${noPadding ? '' : 'px-[4%]'}`}>
        <h2 className="text-xl font-bold text-gray-100">{t(title)}</h2>
        <Link
          href={href || "#"}
          className="rounded bg-white/10 px-3 py-1 text-xs font-semibold text-white transition-colors hover:bg-white/20"
        >
          {t("ui.viewAll")} &rsaquo;
        </Link>
      </div>
      <Carousel items={items} noPadding={noPadding} />
    </section>
  );
}