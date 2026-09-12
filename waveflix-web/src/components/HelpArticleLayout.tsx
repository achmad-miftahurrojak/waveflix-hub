import Link from "next/link";
import PrintButton from "./PrintButton";
import FeedbackButtons from "./FeedbackButtons";
import { Article } from "@/lib/articles";

export default function HelpArticleLayout({ article }: { article: Article }) {
  return (
    <div className="flex flex-col lg:flex-row gap-16 py-12 items-start">
      {}
      <main className="w-full lg:w-[65%] shrink-0">
        <h1 className="text-4xl font-bold mb-8">{article.title}</h1>
        
        {}
        {article.content}
        
        {}
        <FeedbackButtons />
      </main>

      {}
      <aside className="w-full lg:w-[35%] lg:pl-6">
        <div className="flex items-center justify-between mb-6">
          <h3 className="text-xs font-bold text-gray-500 uppercase tracking-widest">
            Artikel Terkait
          </h3>
          <PrintButton />
        </div>
        
        <ul className="flex flex-col border-t border-netflix-divider">
          <li className="border-b border-netflix-divider">
            <Link href="/article/cara-menonton" className="flex items-start gap-4 py-4 hover:underline text-netflix-text group">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" className="w-6 h-6 text-gray-400 shrink-0 mt-0.5" strokeWidth="1.5">
                <path strokeLinecap="round" strokeLinejoin="round" d="M19 20H5a2 2 0 01-2-2V6a2 2 0 012-2h10a2 2 0 012 2v1m2 13a2 2 0 01-2-2V7m2 13a2 2 0 002-2V9.5a2.5 2.5 0 00-2.5-2.5H15" />
              </svg>
              <span className="font-semibold text-[15px] group-hover:text-netflix-link transition-colors">Cara menonton Waveflix di TV</span>
            </Link>
          </li>
          <li className="border-b border-netflix-divider">
            <Link href="/article/keamanan-privasi" className="flex items-start gap-4 py-4 hover:underline text-netflix-text group">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" className="w-6 h-6 text-gray-400 shrink-0 mt-0.5" strokeWidth="1.5">
                <path strokeLinecap="round" strokeLinejoin="round" d="M3 10h18M7 15h1m4 0h1m-7 4h12a3 3 0 003-3V8a3 3 0 00-3-3H6a3 3 0 00-3 3v8a3 3 0 003 3z" />
              </svg>
              <span className="font-semibold text-[15px] group-hover:text-netflix-link transition-colors">Informasi Keamanan & Privasi Akun</span>
            </Link>
          </li>
          <li className="border-b border-netflix-divider">
            <Link href="/article/troubleshooting" className="flex items-start gap-4 py-4 hover:underline text-netflix-text group">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" className="w-6 h-6 text-gray-400 shrink-0 mt-0.5" strokeWidth="1.5">
                <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
              </svg>
              <span className="font-semibold text-[15px] group-hover:text-netflix-link transition-colors">Pesan error saat memutar video</span>
            </Link>
          </li>
        </ul>
      </aside>
    </div>
  );
}
