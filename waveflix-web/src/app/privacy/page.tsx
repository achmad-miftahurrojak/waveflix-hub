import Link from "next/link";
import { Metadata } from "next";

export const metadata: Metadata = {
  title: "Kebijakan Privasi - Waveflix",
  description: "Kebijakan Privasi Waveflix",
};

export default function PrivacyPage() {
  return (
    <div className="max-w-[800px] mx-auto w-full px-6 md:px-12 bg-white pb-24">
      {/* Breadcrumb Row */}
      <div className="py-5 flex justify-between items-center border-b border-netflix-divider mb-12">
        <Link href="/" className="flex items-center gap-2 text-sm font-semibold hover:underline">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" className="w-4 h-4" strokeWidth="2">
            <path d="M15 18l-6-6 6-6" strokeLinecap="round" strokeLinejoin="round" />
          </svg>
          Kembali Ke Beranda Bantuan
        </Link>
      </div>

      <h1 className="text-4xl font-bold mb-8">Kebijakan Privasi</h1>
      
      <div className="space-y-6 text-[16px] leading-[1.6]">
        <p>
          Selamat datang di Waveflix. Kami sangat menghargai privasi Anda. Halaman ini menjelaskan bagaimana kami mengumpulkan, menggunakan, dan melindungi informasi pribadi Anda saat menggunakan layanan kami.
        </p>

        <hr className="border-netflix-divider my-8" />

        <h2 className="text-2xl font-bold mb-4">1. Data yang Kami Kumpulkan</h2>
        <p>
          Kami hanya mengumpulkan data yang diperlukan untuk memberikan pengalaman terbaik kepada Anda. Ini termasuk:
        </p>
        <ul className="list-disc pl-6 space-y-2">
          <li>Informasi akun dasar (email) saat Anda mendaftar.</li>
          <li>Aktivitas penggunaan seperti film yang ditambahkan ke Watchlist dan riwayat tontonan (History).</li>
        </ul>
        
        <h2 className="text-2xl font-bold mb-4 mt-8">2. Penggunaan Data</h2>
        <p>
          Data yang kami kumpulkan semata-mata digunakan untuk:
        </p>
        <ul className="list-disc pl-6 space-y-2">
          <li>Menyimpan preferensi tontonan Anda.</li>
          <li>Mengamankan akun Anda dari akses tidak sah.</li>
        </ul>

        <h2 className="text-2xl font-bold mb-4 mt-8">3. Row Level Security</h2>
        <p>
          Semua data aktivitas Anda (seperti Watchlist) diisolasi dengan ketat menggunakan teknologi <em>Row Level Security</em> (RLS) di database PostgreSQL kami. Artinya, pengguna lain tidak akan pernah bisa mengakses riwayat atau daftar simpanan Anda.
        </p>

        <h2 className="text-2xl font-bold mb-4 mt-8">4. Pihak Ketiga</h2>
        <p>
          Kami menggunakan <em>The Movie Database (TMDB) API</em> untuk menampilkan informasi film. Kami tidak membagikan informasi pribadi Anda dengan TMDB.
        </p>
      </div>
    </div>
  );
}
