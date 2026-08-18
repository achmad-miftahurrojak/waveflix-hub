import { getTrendingIndonesia } from "@/lib/tmdb";
import { backdropUrl } from "@/lib/helpers";
import Top10Row from "@/components/Top10Row";
import EmailForm from "@/components/EmailForm";

export default async function LandingPage() {
  // Ambil data untuk "Sedang Tren Sekarang" (Trending Indonesia)
  const trending = await getTrendingIndonesia();
  
  // Ambil background dari film trending pertama
  const bgImage = trending.length > 0 ? backdropUrl(trending[0]) : "";

  return (
    <main className="min-h-screen bg-black">
      {/* Hero Section */}
      <section className="relative flex flex-col items-center justify-center min-h-[90vh] px-4 pt-24 pb-16 text-center border-b-[8px] border-[#232323]">
        {/* Background Image & Overlay */}
        {bgImage && (
          <div
            className="absolute inset-0 bg-cover bg-center"
            style={{ backgroundImage: `url('${bgImage}')` }}
          />
        )}
        <div className="absolute inset-0 bg-black/60 bg-gradient-to-t from-black via-black/40 to-black/80" />
        
        {/* Konten Hero */}
        <div className="relative z-10 max-w-4xl mx-auto space-y-6">
          <h1 className="text-4xl md:text-5xl lg:text-[4rem] font-extrabold leading-tight tracking-tight text-white drop-shadow-lg">
            Film dan serial TV tanpa batas, dan lebih banyak lagi
          </h1>
          <p className="text-lg md:text-2xl font-medium text-white drop-shadow-md">
            Harga mulai dari Rp54.000. Batalkan kapan pun.
          </p>
          <div className="pt-4">
            <p className="text-base md:text-lg text-white mb-4 drop-shadow-md">
              Siap menonton? Masukkan email untuk membuat atau memulai lagi keanggotaanmu.
            </p>
            <EmailForm />
          </div>
        </div>
      </section>

      {/* Section: Sedang Tren Sekarang */}
      <section className="bg-black py-12 border-b-[8px] border-[#232323]">
        <div className="max-w-[1400px] mx-auto">
          <Top10Row items={trending} />
        </div>
      </section>

      {/* Section Placeholder tambahan jika diperlukan (Sesuai gaya Netflix) */}
      <section className="bg-black py-20 px-[4%] text-center border-b-[8px] border-[#232323]">
        <div className="max-w-4xl mx-auto space-y-4">
          <h2 className="text-3xl md:text-5xl font-extrabold">Buat profil untuk anak</h2>
          <p className="text-lg md:text-2xl text-white/80">
            Kirim anak-anak untuk bertualang bersama karakter favorit mereka di dunia yang dibuat khusus untuk mereka—gratis dengan keanggotaanmu.
          </p>
        </div>
      </section>
    </main>
  );
}
