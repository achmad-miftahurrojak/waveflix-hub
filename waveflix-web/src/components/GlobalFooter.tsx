import Link from "next/link";

export default function GlobalFooter() {
  return (
    <footer className="bg-[#F3F3F3] border-t border-netflix-divider mt-auto py-16">
      <div className="max-w-[1140px] mx-auto px-6 md:px-12 text-gray-600 text-sm flex flex-col md:flex-row justify-between items-center">
        <div className="flex gap-4">
          <Link href="/terms" className="hover:underline">Syarat Ketentuan</Link>
          <span>|</span>
          <Link href="/privacy" className="hover:underline">Privasi</Link>
        </div>
        <div className="mt-4 md:mt-0">
          &copy; {new Date().getFullYear()} Pusat Bantuan Waveflix.
        </div>
      </div>
    </footer>
  );
}
