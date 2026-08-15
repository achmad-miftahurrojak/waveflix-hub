import Link from "next/link";
import type { TmdbItem } from "@/lib/types";
import Carousel from "./Carousel";

interface Props {
  title: string;
  items: TmdbItem[];
  viewAll?: boolean;
  href?: string;
}

/** Baris standar: judul (+ "Lihat semua") lalu carousel. Dipakai untuk provider & recently added. */
export default function MovieRow({ title, items, viewAll, href }: Props) {
  if (items.length === 0) return null;
  return (
    <section className="mb-6">
      <div className="mb-1 flex items-center justify-between px-[4%]">
        <h2 className="text-xl font-bold">{title}</h2>
        {viewAll &&
          (href ? (
            <Link
              href={href}
              className="text-sm text-white/50 transition hover:text-white"
            >
              View all &rsaquo;
            </Link>
          ) : (
            <span className="text-sm text-white/40">View all &rsaquo;</span>
          ))}
      </div>
      <Carousel items={items} />
    </section>
  );
}
