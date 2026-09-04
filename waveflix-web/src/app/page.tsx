import { Metadata } from "next";
import HelpArticleLayout from "@/components/HelpArticleLayout";
import { getArticleBySlug } from "@/lib/articles";
import Link from "next/link";

export const metadata: Metadata = {
  title: "Pusat Bantuan - Waveflix",
  description: "Pusat bantuan dan FAQ Waveflix",
};

export default function HelpCenter() {
  const defaultArticle = getArticleBySlug("apa-itu-waveflix");

  if (!defaultArticle) {
    return null;
  }

  return (
    <div className="max-w-[1140px] mx-auto w-full px-6 md:px-12 bg-white">
      {/* Breadcrumb Row */}
      <div className="py-5 flex justify-between items-center border-b border-netflix-divider">
        <Link href="/" className="flex items-center gap-2 text-sm font-semibold hover:underline">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" className="w-4 h-4" strokeWidth="2">
            <path d="M15 18l-6-6 6-6" strokeLinecap="round" strokeLinejoin="round" />
          </svg>
          Kembali Ke Beranda Bantuan
        </Link>
      </div>

      {/* Render the article layout component */}
      <HelpArticleLayout article={defaultArticle} />
    </div>
  );
}
