import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Syarat Penggunaan | Waveflix",
  description: "Syarat dan ketentuan penggunaan layanan Waveflix.",
};

export default function SyaratPage() {
  return (
    <main className="mx-auto max-w-3xl px-6 py-16">
      <h1 className="text-3xl font-bold text-white md:text-4xl">Syarat Penggunaan</h1>
      <p className="mt-2 text-sm text-white/40">Terakhir diperbarui: Agustus 2026</p>

      <div className="mt-10 space-y-8 text-white/75 leading-relaxed">
        <section>
          <h2 className="text-xl font-bold text-white">1. Penerimaan Syarat</h2>
          <p className="mt-3">
            Dengan mendaftar dan menggunakan Waveflix, kamu menyetujui syarat
            dan ketentuan ini. Jika kamu tidak setuju, mohon untuk tidak
            menggunakan layanan ini.
          </p>
        </section>

        <section>
          <h2 className="text-xl font-bold text-white">2. Akun Pengguna</h2>
          <p className="mt-3">
            Kamu bertanggung jawab menjaga kerahasiaan kredensial akun kamu dan
            untuk semua aktivitas yang terjadi di bawah akun kamu. Kami sarankan
            menggunakan kata sandi yang kuat dan unik.
          </p>
        </section>

        <section>
          <h2 className="text-xl font-bold text-white">3. Penggunaan yang Diperbolehkan</h2>
          <p className="mt-3">
            Layanan ini hanya untuk penggunaan pribadi dan non-komersial. Kamu
            dilarang melakukan scraping, reverse engineering, mengganggu
            ketersediaan layanan, atau menyalahgunakan API kami.
          </p>
        </section>

        <section>
          <h2 className="text-xl font-bold text-white">4. Konten Pihak Ketiga</h2>
          <p className="mt-3">
            Waveflix tidak meng-host, menyimpan, atau mendistribusikan file
            media apa pun. Semua konten diambil secara otomatis dari penyedia
            pihak ketiga di internet. Kami tidak bertanggung jawab atas
            ketersediaan maupun keabsahan konten dari penyedia tersebut.
          </p>
        </section>

        <section>
          <h2 className="text-xl font-bold text-white">5. Penghentian Layanan</h2>
          <p className="mt-3">
            Kami berhak menangguhkan atau menghentikan akun yang melanggar
            syarat ini, atau yang digunakan untuk aktivitas yang melanggar
            hukum.
          </p>
        </section>

        <section>
          <h2 className="text-xl font-bold text-white">6. Perubahan Syarat</h2>
          <p className="mt-3">
            Syarat ini dapat diperbarui dari waktu ke waktu. Perubahan
            penting akan diumumkan melalui layanan. Penggunaan berkelanjutan
            setelah perubahan berarti kamu menyetujuinya.
          </p>
        </section>
      </div>
    </main>
  );
}
