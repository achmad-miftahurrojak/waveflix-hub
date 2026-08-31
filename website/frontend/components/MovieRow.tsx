"use client";

import Link from "next/link";
import type { TmdbItem } from "@/lib/types";
import Carousel from "./Carousel";
import { useTranslation } from "@/lib/i18n";

interface Props {
  title: string;
  items: TmdbItem[];
  viewAll?: boolean;
  href?: string;
  noPadding?: boolean;
}

export default function MovieRow({ title, items, viewAll, href, noPadding }: Props) {
  const { t } = useTranslation();
  if (items.length === 0) return null;
  return (
    <section className="mb-3">
      <div className={`mb-1 flex items-center justify-between ${noPadding ? '' : 'px-[4%]'}`}>
        <h2 className="text-xl font-bold">{t(title)}</h2>
        {viewAll &&
          (href ? (
            <Link
              href={href}
              className="text-sm text-white/50 transition hover:text-white"
            >
              {t("ui.viewAll")} &rsaquo;
            </Link>
          ) : (
            <span className="text-sm text-white/40">{t("ui.viewAll")} &rsaquo;</span>
          ))}
      </div>
      <Carousel items={items} noPadding={noPadding} />
    </section>
  );
}
