"use client";

import Link from "next/link";
import { usePathname, useSearchParams } from "next/navigation";
import { useEffect, useState } from "react";
import {
  HomeIcon,
  FilmIcon,
  TvIcon,
  BookmarkIcon,
  ChevronRight,
  MoreIcon,
  TagIcon,
  GlobeIcon,
  CalendarIcon,
  NetworkIcon,
} from "./Icons";
import SearchBox from "./SearchBox";
import { useAuth } from "./AuthProvider";

const NAV = [
  { label: "Home", href: "/", icon: HomeIcon },
  { label: "Movies", href: "/browse?media=movie", icon: FilmIcon },
  { label: "Series", href: "/browse?media=tv", icon: TvIcon },
  { label: "My List", href: "/daftar-saya", icon: BookmarkIcon },
];

export default function Navbar() {
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const { user, ready, logout } = useAuth();
  const [scrolled, setScrolled] = useState(false);
  const [openMore, setOpenMore] = useState(false);
  const [openAccount, setOpenAccount] = useState(false);
  // Cegah mismatch hydration: UI akun baru dirender setelah mount di client.
  const [mounted, setMounted] = useState(false);

  useEffect(() => setMounted(true), []);

  useEffect(() => {
    const onScroll = () => setScrolled(window.scrollY > 40);
    onScroll();
    window.addEventListener("scroll", onScroll, { passive: true });
    return () => window.removeEventListener("scroll", onScroll);
  }, []);

  // Pertahankan media yang sedang aktif (movie/tv) di dropdown filter.
  const activeMedia = searchParams.get("media") === "tv" ? "tv" : "movie";

  // Halaman kategori (via "More") → dianggap konteks "More", bukan Movies/Series.
  const onCategory =
    pathname === "/genres" ||
    pathname === "/countries" ||
    pathname === "/years" ||
    pathname === "/networks" ||
    (pathname.startsWith("/browse") &&
      ["genre", "country", "year", "provider"].some((k) =>
        searchParams.get(k)
      ));

  const moreMenu = [
    { label: "Genres", href: `/genres?media=${activeMedia}`, icon: TagIcon },
    { label: "Country", href: `/countries?media=${activeMedia}`, icon: GlobeIcon },
    { label: "Year", href: `/years?media=${activeMedia}`, icon: CalendarIcon },
    { label: "Network", href: `/networks?media=${activeMedia}`, icon: NetworkIcon },
  ];

  const isActive = (href: string) => {
    const [path, query] = href.split("?");
    if (path === "/") return pathname === "/";
    if (!pathname.startsWith(path)) return false;
    // Movies & Series berbagi path /browse — bedakan lewat query media.
    const hrefMedia = new URLSearchParams(query ?? "").get("media");
    if (hrefMedia) {
      // Movies/Series aktif hanya di listing polos, bukan halaman kategori.
      if (onCategory) return false;
      return (searchParams.get("media") ?? "movie") === hrefMedia;
    }
    return true;
  };

  return (
    <header
      className={`fixed inset-x-0 top-0 z-[1000] transition-all duration-300 ${
        scrolled ? "px-[3%] pt-3" : ""
      }`}
    >
      <nav
        className={`flex items-center justify-between gap-4 transition-all duration-300 ${
          scrolled
            ? "rounded-full bg-[#050507]/95 px-8 py-3 shadow-lg ring-1 ring-white/10 backdrop-blur"
            : "bg-gradient-to-b from-black/70 via-black/30 to-transparent px-[4%] py-4"
        }`}
      >
        <div className="flex items-center gap-8">
          <Link
            href="/"
            className="text-4xl uppercase leading-none tracking-[-0.03em] text-accent"
            style={{ fontFamily: "var(--font-logo)" }}
          >
            Waveflix
          </Link>
          <ul className="hidden items-center gap-1 lg:flex">
            {NAV.map(({ label, href, icon: Icon }) => (
              <li key={label}>
                <Link
                  href={href}
                  className={`flex items-center gap-2 border-b-2 px-2 py-1 text-sm font-semibold transition ${
                    isActive(href)
                      ? "border-accent text-accent"
                      : "border-transparent text-white/70 hover:text-white"
                  }`}
                >
                  <Icon />
                  <span>{label}</span>
                </Link>
              </li>
            ))}
            {/* Dropdown "More" — Genres / Country / Year jadi satu (ala IDLIX) */}
            <li
              className="relative"
              onMouseEnter={() => setOpenMore(true)}
              onMouseLeave={() => setOpenMore(false)}
            >
              <button
                className={`flex items-center gap-2 border-b-2 px-2 py-1 text-sm font-semibold transition ${
                  onCategory
                    ? "border-accent text-accent"
                    : "border-transparent text-white/70 hover:text-white"
                }`}
              >
                <MoreIcon />
                <span>More</span>
                <ChevronRight className={`transition ${openMore ? "rotate-90" : ""}`} />
              </button>
              {openMore && (
                <div className="absolute left-0 top-full w-56 rounded-xl bg-[#0d0f14] p-2 shadow-xl ring-1 ring-white/10">
                  {moreMenu.map(({ label, href, icon: Icon }) => (
                    <Link
                      key={label}
                      href={href}
                      className="flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium text-white/75 transition hover:bg-white/10 hover:text-white"
                    >
                      <span className="text-white/60">
                        <Icon />
                      </span>
                      {label}
                    </Link>
                  ))}
                </div>
              )}
            </li>
          </ul>
        </div>

        <div className="flex items-center gap-4">
          <SearchBox />
          {!mounted || !ready ? (
            <div className="h-9 w-9 shrink-0 rounded-full bg-white/10" />
          ) : user ? (
            <div
              className="relative"
              onMouseEnter={() => setOpenAccount(true)}
              onMouseLeave={() => setOpenAccount(false)}
            >
              <button
                aria-label="Account"
                className="grid h-9 w-9 shrink-0 place-items-center rounded-full bg-accent font-bold text-black"
              >
                {user.username.charAt(0).toUpperCase()}
              </button>
              {openAccount && (
                <div className="absolute right-0 top-full w-52 rounded-xl bg-[#0d0f14] p-2 shadow-xl ring-1 ring-white/10">
                  <div className="px-3 py-2">
                    <div className="truncate text-sm font-semibold">{user.username}</div>
                    <div className="truncate text-xs text-white/50">{user.email}</div>
                  </div>
                  <div className="my-1 h-px bg-white/10" />
                  <Link href="/daftar-saya" className="block rounded-md px-3 py-2 text-sm text-white/70 hover:bg-white/10 hover:text-white">
                    My List
                  </Link>
                  <Link href="/favorit" className="block rounded-md px-3 py-2 text-sm text-white/70 hover:bg-white/10 hover:text-white">
                    Favorites
                  </Link>
                  <Link href="/riwayat" className="block rounded-md px-3 py-2 text-sm text-white/70 hover:bg-white/10 hover:text-white">
                    Watch History
                  </Link>
                  <button
                    onClick={logout}
                    className="block w-full rounded-md px-3 py-2 text-left text-sm text-red-300 hover:bg-white/10"
                  >
                    Log Out
                  </button>
                </div>
              )}
            </div>
          ) : (
            <Link
              href="/masuk"
              className="rounded-full bg-accent px-5 py-2 text-sm font-semibold text-black transition hover:bg-accent-dark"
            >
              Sign In
            </Link>
          )}
        </div>
      </nav>
    </header>
  );
}
