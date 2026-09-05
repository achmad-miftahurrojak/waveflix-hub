import Link from "next/link";

export default function GlobalHeader() {
  return (
    <header className="bg-black text-white h-[72px] flex items-center justify-between px-6 sticky top-0 z-50">
      <div className="flex items-center gap-6">
        <Link href="/" className="flex items-center">
          <img src="/1.png" alt="WAVEFLIX" className="h-8 w-auto object-contain scale-[2.5] origin-left mr-12" />
        </Link>
        <span className="w-px h-8 bg-gray-600 hidden sm:block"></span>
        <span className="font-semibold text-lg hidden sm:block">Pusat Bantuan</span>
      </div>
      <div className="flex items-center gap-4">
        <a
          href="http://localhost:3000"
          className="px-4 py-2 bg-transparent hover:underline font-semibold text-sm transition"
        >
          Masuk Waveflix
        </a>
      </div>
    </header>
  );
}
