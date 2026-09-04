import React from "react";
import Link from "next/link";

export type Article = {
  slug: string;
  title: string;
  content: React.ReactNode;
};

export const articles: Article[] = [
  {
    slug: "apa-itu-waveflix",
    title: "Apa itu Waveflix?",
    content: (
      <div className="space-y-6 text-[16px] leading-[1.6]">
        <p>
          Waveflix adalah layanan streaming berbasis komunitas yang memungkinkan anggota kami menonton serial TV dan film di perangkat yang terhubung ke internet tanpa iklan pop-up yang mengganggu.
        </p>
        <p>
          Berbeda dengan platform gratisan lainnya, Waveflix menggunakan arsitektur modern yang menjamin kecepatan muat super tinggi karena dukungan Redis Cache layer.
        </p>

        <hr className="border-netflix-divider my-8" />

        <h2 className="text-2xl font-bold mb-4">Serial TV & Film</h2>
        <p>
          Konten Waveflix bersumber dari pustaka global open-source yang terus diperbarui secara otomatis. Semakin banyak yang kamu tonton, semakin baik Waveflix merekomendasikan acara untukmu di aplikasi utama.
        </p>

        <hr className="border-netflix-divider my-8" />

        <h2 className="text-2xl font-bold mb-4">Perangkat yang Didukung</h2>
        <p>
          Kamu dapat menonton Waveflix melalui perangkat yang terhubung ke internet yang memiliki browser web modern, termasuk laptop, PC, smartphone, dan tablet.
        </p>

        <div className="bg-netflix-note p-5 rounded-sm my-6 text-sm text-[#000000DE]">
          <strong className="block mb-1 font-bold text-black">Catatan:</strong> 
          Pastikan browser web kamu (Google Chrome, Safari, atau Firefox) selalu diperbarui ke versi terbaru untuk performa pemutaran video yang lancar dan optimal.
        </div>

        <hr className="border-netflix-divider my-8" />

        <h2 className="text-2xl font-bold mb-4">Paket dan Biaya</h2>
        <p>
          Waveflix <strong>100% gratis selamanya</strong>. Platform ini dibangun murni untuk memajukan komunitas open-source tanpa afiliasi komersial berbayar. Tidak ada tagihan bulanan atau paket berbayar (tiering). Semua pengguna mendapatkan kualitas pemutaran tertinggi yang tersedia.
        </p>

        <hr className="border-netflix-divider my-8" />

        <h2 className="text-2xl font-bold mb-4">Mulai Menonton</h2>
        <ol className="list-decimal pl-6 space-y-3">
          <li>
            Kunjungi <a href="http://localhost:3000" className="text-netflix-link hover:underline font-semibold">Beranda Waveflix App</a>.
          </li>
          <li>Buat akun menggunakan email dan kata sandi pilihanmu. (Tidak memerlukan kartu kredit)</li>
          <li>Mulai jelajahi perpustakaan film tak terbatas!</li>
        </ol>
      </div>
    )
  },
  {
    slug: "cara-menonton",
    title: "Cara menonton Waveflix di TV",
    content: (
      <div className="space-y-6 text-[16px] leading-[1.6]">
        <p>
          Meskipun Waveflix belum memiliki aplikasi TV khusus (seperti Tizen atau WebOS), kamu tetap bisa menikmati film dan serial kami di layar lebar TV-mu.
        </p>

        <hr className="border-netflix-divider my-8" />

        <h2 className="text-2xl font-bold mb-4">Menggunakan Kabel HDMI</h2>
        <p>
          Cara paling andal adalah menghubungkan laptop atau komputer langsung ke TV menggunakan kabel HDMI. Ini menjamin resolusi penuh tanpa penundaan (lag).
        </p>

        <h2 className="text-2xl font-bold mb-4">Menggunakan Chromecast atau AirPlay</h2>
        <p>
          Buka Waveflix di browser Google Chrome pada komputer atau ponsel pintar kamu, lalu gunakan fitur "Cast" untuk memantulkan (mirror) tab tersebut ke perangkat yang kompatibel dengan Chromecast. Jika menggunakan ekosistem Apple, kamu bisa memakai fitur AirPlay.
        </p>
        
        <h2 className="text-2xl font-bold mb-4">Melalui Browser Smart TV</h2>
        <p>
          Beberapa Smart TV model baru memiliki browser web terintegrasi yang cukup kuat untuk memutar video HTML5. Cukup buka aplikasi Web Browser di TV kamu dan kunjungi alamat Waveflix App.
        </p>
        
        <div className="bg-netflix-note p-5 rounded-sm my-6 text-sm text-[#000000DE]">
          <strong className="block mb-1 font-bold text-black">Catatan:</strong> 
          Pemutaran lewat browser bawaan TV terkadang kurang optimal karena batasan memori TV. Kami sangat menyarankan penggunaan HDMI.
        </div>
      </div>
    )
  },
  {
    slug: "keamanan-privasi",
    title: "Informasi Keamanan & Privasi Akun",
    content: (
      <div className="space-y-6 text-[16px] leading-[1.6]">
        <p>
          Di Waveflix, kami menanggapi privasi dengan sangat serius. Sebagai platform nirlaba, kami berjanji untuk tidak pernah menjual data pribadimu.
        </p>

        <hr className="border-netflix-divider my-8" />

        <h2 className="text-2xl font-bold mb-4">Data apa yang kami kumpulkan?</h2>
        <ul className="list-disc pl-6 space-y-3">
          <li>Alamat email untuk keperluan login dan reset kata sandi.</li>
          <li>Riwayat tontonan kamu agar sistem bisa merekomendasikan film dengan tepat.</li>
          <li>Katalog "Daftar Saya" untuk menyimpan acara favorit.</li>
        </ul>

        <hr className="border-netflix-divider my-8" />

        <h2 className="text-2xl font-bold mb-4">Bagaimana data diamankan?</h2>
        <p>
          Sandi kamu dienkripsi menggunakan algoritma *Bcrypt* sebelum disimpan di basis data kami, artinya staf administrator sekalipun tidak bisa melihat kata sandi aslimu. API backend kami ditulis di Golang yang ketat, terlindungi di balik sistem pertahanan dan CORS yang ketat.
        </p>
      </div>
    )
  },
  {
    slug: "troubleshooting",
    title: "Pesan error saat memutar video",
    content: (
      <div className="space-y-6 text-[16px] leading-[1.6]">
        <p>
          Jika kamu mengalami masalah saat pemutaran, layar macet, atau film tidak bisa dimuat, hal ini biasanya disebabkan oleh masalah jaringan atau ekstensi pemblokir iklan (Ad-blocker).
        </p>

        <hr className="border-netflix-divider my-8" />

        <h2 className="text-2xl font-bold mb-4">Langkah 1: Matikan Ekstensi Ad-Blocker</h2>
        <p>
          Pemutar video Waveflix bebas iklan, namun beberapa ekstensi pihak ketiga (seperti AdBlock Plus, uBlock Origin) terkadang memblokir *script* esensial untuk memutar *stream* video. Cobalah menonaktifkan sementara dan *refresh* halaman.
        </p>

        <h2 className="text-2xl font-bold mb-4">Langkah 2: Bersihkan Cache Browser</h2>
        <p>
          Sistem kami sangat mengandalkan penyimpanan sementara (cache) agar situs terasa cepat. Jika kamu mengalami *glitch* aneh, hapus cache situs di browser-mu atau coba buka dari jendela penyamaran (Incognito).
        </p>
        
        <h2 className="text-2xl font-bold mb-4">Langkah 3: Coba Server atau Film Lain</h2>
        <p>
          Terkadang, sebuah video spesifik mungkin terhapus dari *provider* utama kami karena alasan teknis. Silakan coba judul lain untuk memastikan apakah gangguan hanya terjadi pada satu video atau keseluruhan sistem.
        </p>
      </div>
    )
  }
];

export function getArticleBySlug(slug: string): Article | undefined {
  return articles.find(a => a.slug === slug);
}
