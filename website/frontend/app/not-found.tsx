import Link from "next/link";

export default function NotFound() {
  return (
    <main className="flex min-h-screen flex-col items-center justify-center bg-black px-6 text-center">
      <p className="text-7xl font-bold text-accent md:text-8xl">404</p>
      <h1 className="mt-4 text-2xl font-bold text-white md:text-3xl">
        Halaman tidak ditemukan
      </h1>
      <p className="mt-3 max-w-md text-white/60">
        Halaman yang kamu cari hilang, dipindahkan, atau memang tidak pernah
        ada.
      </p>
      <Link
        href="/"
        className="mt-8 rounded bg-accent px-6 py-3 font-semibold text-black transition hover:bg-accent-dark"
      >
        Kembali ke Beranda
      </Link>
    </main>
  );
}
