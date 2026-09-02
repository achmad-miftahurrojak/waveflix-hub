import { getTrendingGlobal } from "@/lib/tmdb";
import Carousel from "@/components/Carousel";
import EmailForm from "@/components/EmailForm";
import PosterWall from "@/components/landing/PosterWall";
import Faq from "@/components/landing/Faq";

const reasons = [
  {
    title: "Nikmati di TV-mu",
    desc: "Tonton di smart TV, PlayStation, Xbox, Chromecast, Apple TV, dan banyak lagi.",
    icon: (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" className="h-10 w-10">
        <rect x="2" y="4" width="20" height="13" rx="2" />
        <path strokeLinecap="round" d="M8 21h8M12 17v4" />
      </svg>
    ),
  },
  {
    title: "Tonton di mana saja",
    desc: "Streaming film dan serial tanpa batas di ponsel, tablet, laptop, dan TV-mu.",
    icon: (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" className="h-10 w-10">
        <rect x="2" y="5" width="13" height="11" rx="1.5" />
        <rect x="16" y="9" width="6" height="10" rx="1.5" />
      </svg>
    ),
  },
  {
    title: "Buat profil untuk anak",
    desc: "Kirim anak-anak untuk bertualang bersama karakter favorit mereka di dunia yang dibuat khusus untuk mereka.",
    icon: (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" className="h-10 w-10">
        <circle cx="12" cy="8" r="4" />
        <path strokeLinecap="round" d="M5 20c0-3.5 3-6 7-6s7 2.5 7 6" />
      </svg>
    ),
  },
];

const footerColumns: { label: string; href: string }[][] = [
  [
    { label: "FAQ", href: "#faq" },
    { label: "Film & Serial", href: "/browse" },
    { label: "Reality Show", href: "/reality" },
  ],
  [
    { label: "Genre", href: "/genres" },
    { label: "Negara", href: "/countries" },
    { label: "Jaringan TV", href: "/networks" },
  ],
  [
    { label: "Tahun Rilis", href: "/years" },
    { label: "Privasi", href: "/privasi" },
    { label: "Syarat Penggunaan", href: "/syarat" },
  ],
  [
    { label: "Akun Saya", href: "/account" },
    { label: "Daftar Saya", href: "/daftar-saya" },
    { label: "Riwayat Tontonan", href: "/riwayat" },
  ],
];

export default async function LandingPage() {

  const trending = await getTrendingGlobal();

  return (
    <main className="min-h-screen bg-black">
      {}
      <section className="relative flex min-h-[100dvh] items-center justify-center overflow-hidden">
        <PosterWall items={trending} />

        {}
        <div className="relative z-10 mx-auto w-full max-w-[800px] px-6 text-center">
          <h1 className="text-4xl font-bold leading-tight text-white sm:text-5xl md:text-6xl">
            Film dan serial TV tanpa batas, dan lebih banyak lagi
          </h1>
          <p className="mt-5 text-lg font-bold text-white md:text-2xl">
            Ribuan film dan serial TV — tonton kapan saja, di mana saja.
          </p>
          <p className="mt-6 text-base text-white/90 md:text-lg">
            Siap menonton? Masukkan email untuk mulai menonton.
          </p>
          <EmailForm />
        </div>
      </section>

      <section className="relative z-10 bg-black pt-24 pb-16">
        <div
          className="absolute left-0 right-0 top-0 h-16 bg-black"
          style={{ borderRadius: "50% 50% 0 0 / 100% 100% 0 0" }}
          aria-hidden
        />
        <h2 className="mx-auto mb-4 max-w-[1080px] text-2xl font-bold md:text-3xl">Sedang Tren Sekarang</h2>
        <Carousel items={trending} small quickView isLanding={true} />
      </section>

      {}
      <section className="mx-auto max-w-6xl px-6 py-16">
        <h2 className="mb-8 text-2xl font-bold text-white md:text-3xl">Alasan Lebih Banyak untuk Bergabung</h2>
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {reasons.map((r) => (
            <div
              key={r.title}
              className="flex min-h-56 flex-col justify-between rounded-2xl bg-gradient-to-br from-cyan-950/80 via-slate-900 to-slate-900 p-6 ring-1 ring-accent/15 transition hover:ring-accent/40"
            >
              <div>
                <h3 className="text-lg font-bold text-white">{r.title}</h3>
                <p className="mt-2 text-sm leading-relaxed text-white/70">{r.desc}</p>
              </div>
              <div className="mt-6 self-end text-accent/70">{r.icon}</div>
            </div>
          ))}
        </div>
      </section>

      {}
      <Faq />

      {}
      <footer className="mx-auto max-w-6xl px-6 pb-14 pt-10">
        <p className="text-white/60">
          Ada pertanyaan? Lihat{" "}
          <a href="#faq" className="underline hover:text-white">
            FAQ
          </a>
          .
        </p>
        <div className="mt-8 grid grid-cols-2 gap-x-6 gap-y-3 sm:grid-cols-4">
          {footerColumns.map((col, i) => (
            <ul key={i} className="space-y-3">
              {col.map((link) => (
                <li key={link.label}>
                  <a
                    href={link.href}
                    className="text-sm text-white/60 underline-offset-2 hover:underline"
                  >
                    {link.label}
                  </a>
                </li>
              ))}
            </ul>
          ))}
        </div>
      </footer>
    </main>
  );
}
