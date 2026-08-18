import { useEffect } from "react";
import type { TmdbItem } from "@/lib/types";
import { itemTitle, itemYear, isTv, backdropUrl } from "@/lib/helpers";

export default function QuickViewModal({
  isOpen,
  onClose,
  item,
}: {
  isOpen: boolean;
  onClose: () => void;
  item: TmdbItem | null;
}) {
  useEffect(() => {
    if (isOpen) {
      document.body.style.overflow = "hidden";
    } else {
      document.body.style.overflow = "";
    }
    return () => {
      document.body.style.overflow = "";
    };
  }, [isOpen]);

  if (!isOpen || !item) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      {/* Backdrop */}
      <div 
        className="absolute inset-0 bg-black/80 backdrop-blur-sm transition-opacity"
        onClick={onClose}
      />
      
      {/* Modal */}
      <div className="relative bg-[#181818] w-full max-w-3xl rounded-xl overflow-hidden shadow-2xl animate-in fade-in zoom-in-95 duration-200">
        <button 
          onClick={onClose}
          className="absolute top-4 right-4 z-20 p-2 bg-black/50 hover:bg-black/80 rounded-full text-white transition"
        >
          <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
             <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>

        {/* Header/Backdrop Image */}
        <div className="relative aspect-video w-full">
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img 
            src={backdropUrl(item)} 
            alt={itemTitle(item)}
            className="w-full h-full object-cover"
          />
          <div className="absolute inset-0 bg-gradient-to-t from-[#181818] via-black/20 to-transparent" />
          
          <div className="absolute bottom-6 left-8 right-8">
            <h2 className="text-4xl md:text-5xl font-bold text-white mb-2 drop-shadow-lg">
              {itemTitle(item)}
            </h2>
          </div>
        </div>

        {/* Content */}
        <div className="p-8 pt-2">
          <div className="flex flex-wrap gap-2 mb-6 text-sm font-semibold text-gray-300">
            {itemYear(item) && (
              <span className="px-2 py-1 bg-white/10 rounded">{itemYear(item)}</span>
            )}
            <span className="px-2 py-1 bg-white/10 rounded">18+</span>
            <span className="px-2 py-1 bg-white/10 rounded">{isTv(item) ? "Serial" : "Film"}</span>
          </div>
          
          <p className="text-gray-200 text-lg leading-relaxed">
            {item.overview || "Tidak ada deskripsi tersedia untuk judul ini."}
          </p>
        </div>
      </div>
    </div>
  );
}
