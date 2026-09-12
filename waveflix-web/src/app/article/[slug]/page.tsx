import { Metadata } from "next";
import HelpArticleLayout from "@/components/HelpArticleLayout";
import { getArticleBySlug, articles } from "@/lib/articles";
import Link from "next/link";
import { notFound } from "next/navigation";

type Props = {
  params: Promise<{ slug: string }>;
};

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const p = await params;
  const article = getArticleBySlug(p.slug);
  
  if (!article) {
    return {
      title: "Artikel Tidak Ditemukan - Waveflix",
    };
  }

  return {
    title: `${article.title} - Pusat Bantuan Waveflix`,
  };
}

export async function generateStaticParams() {
  return articles.map((article) => ({
    slug: article.slug,
  }));
}

export default async function ArticlePage({ params }: Props) {
  const p = await params;
  const article = getArticleBySlug(p.slug);

  if (!article) {
    notFound();
  }

  return (
    <div className="max-w-[1140px] mx-auto w-full px-6 md:px-12 bg-white">
      {}
      <div className="py-5 flex justify-between items-center border-b border-netflix-divider">
        <Link href="/" className="flex items-center gap-2 text-sm font-semibold hover:underline">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" className="w-4 h-4" strokeWidth="2">
            <path d="M15 18l-6-6 6-6" strokeLinecap="round" strokeLinejoin="round" />
          </svg>
          Kembali Ke Beranda Bantuan
        </Link>
      </div>

      <HelpArticleLayout article={article} />
    </div>
  );
}
