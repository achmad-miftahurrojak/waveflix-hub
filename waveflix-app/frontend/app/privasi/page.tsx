import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Kebijakan Privasi | Waveflix",
  description: "Bagaimana Waveflix mengumpulkan, menggunakan, dan melindungi data kamu.",
};

export default function PrivasiPage() {
  return (
    <main className="mx-auto max-w-3xl px-6 py-16">
      <h1 className="text-3xl font-bold text-white md:text-4xl">Kebijakan Privasi</h1>
      <p className="mt-2 text-sm text-white/40">Terakhir diperbarui: Agustus 2026</p>

      <div className="mt-10 space-y-8 text-white/75 leading-relaxed">
        <section>
          <h2 className="text-xl font-bold text-white">1. Data yang Kami Kumpulkan</h2>
          <p className="mt-3">
            Saat kamu mendaftar dan menggunakan Waveflix, kami mengumpulkan:
            alamat email, nama pengguna, kata sandi (dalam bentuk terenkripsi),
            gambar profil yang kamu unggah, serta aktivitas menonton seperti
            daftar tontonan, favorit, dan riwayat pemutaran.
          </p>
        </section>

        <section>
          <h2 className="text-xl font-bold text-white">2. Cara Kami Menggunakan Data</h2>
          <p className="mt-3">
            Data digunakan solely untuk mengoperasikan layanan: mengelola akun,
            menyimpan daftar tontonan dan riwayat nonton kamu, serta
            menampilkan konten yang relevan. Kami tidak menjual data pribadi
            kamu kepada pihak ketiga.
          </p>
        </section>

        <section>
          <h2 className="text-xl font-bold text-white">3. Penyimpanan dan Keamanan</h2>
          <p className="mt-3">
            Kata sandi dienkripsi menggunakan bcrypt dan tidak pernah disimpan
            dalam bentuk asli. Komunikasi antara perangkat kamu dan server kami
            dilindungi menggunakan enkripsi TLS.
          </p>
        </section>

        <section>
          <h2 className="text-xl font-bold text-white">4. Pihak Ketiga</h2>
          <p className="mt-3">
            Informasi konten film dan serial (poster, sinopsis, rating) bersumber
            dari penyedia metadata pihak ketiga. Waveflix tidak meng-host,
            menyimpan, atau mendistribusikan file media apa pun.
          </p>
        </section>

        <section>
          <h2 className="text-xl font-bold text-white">5. Hak Kamu</h2>
          <p className="mt-3">
            Kamu berhak mengakses, memperbaiki, atau menghapus data akun kamu
            kapan saja melalui halaman pengaturan akun, atau dengan menghubungi
            kami.
          </p>
        </section>

        <section>
          <h2 className="text-xl font-bold text-white">6. Kontak</h2>
          <p className="mt-3">
            Pertanyaan tentang kebijakan ini bisa dikirim melalui halaman
            Pusat Bantuan.
          </p>
        </section>
      </div>
    </main>
  );
}
