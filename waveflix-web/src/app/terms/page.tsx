import Link from "next/link";
import { Metadata } from "next";

export const metadata: Metadata = {
  title: "Syarat dan Ketentuan - Waveflix",
  description: "Syarat dan Ketentuan Penggunaan Waveflix",
};

export default function TermsPage() {
  return (
    <div className="max-w-[800px] mx-auto w-full px-6 md:px-12 bg-white pb-24">
      {}
      <div className="py-5 flex justify-between items-center border-b border-netflix-divider mb-12">
        <Link href="/" className="flex items-center gap-2 text-sm font-semibold hover:underline">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" className="w-4 h-4" strokeWidth="2">
            <path d="M15 18l-6-6 6-6" strokeLinecap="round" strokeLinejoin="round" />
          </svg>
          Kembali Ke Beranda Bantuan
        </Link>
      </div>

      <h1 className="text-4xl font-bold mb-8">Syarat dan Ketentuan</h1>
      
      <div className="space-y-6 text-[16px] leading-[1.6]">
        <p>
          Dengan menggunakan layanan Waveflix, Anda menyetujui syarat dan ketentuan berikut. Harap baca dengan seksama.
        </p>
        
        <hr className="border-netflix-divider my-8" />

        <h2 className="text-2xl font-bold mb-4">1. Penggunaan Layanan</h2>
        <p>
          Waveflix adalah platform eksplorasi film dan serial televisi yang menggunakan data publik dari TMDB. Kami menyediakan antarmuka untuk menelusuri, memfilter, dan mengelola daftar tontonan (Watchlist).
        </p>
        
        <h2 className="text-2xl font-bold mb-4 mt-8">2. Filter Konten</h2>
        <p>
          Kami berkomitmen untuk menyediakan lingkungan penelusuran yang aman. Waveflix menerapkan filter ketat untuk hanya menampilkan karya dari platform distribusi resmi (seperti Netflix, HBO, Disney, dll). Namun, kami tidak bertanggung jawab atas metadata (poster, deskripsi) yang bersumber dari pihak ketiga (TMDB).
        </p>

        <h2 className="text-2xl font-bold mb-4 mt-8">3. Akun Pengguna</h2>
        <p>
          Anda bertanggung jawab untuk menjaga kerahasiaan kredensial akun Anda. Jika terdapat aktivitas mencurigakan, segera ganti kata sandi Anda.
        </p>

        <h2 className="text-2xl font-bold mb-4 mt-8">4. Pembatasan Tanggung Jawab</h2>
        <p>
          Waveflix disediakan "apa adanya" (as is) tanpa jaminan apapun. Kami tidak menjamin bahwa layanan akan bebas dari gangguan (downtime) atau error.
        </p>
      </div>
    </div>
  );
}
