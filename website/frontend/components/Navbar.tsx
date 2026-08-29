"use client";

import Link from "next/link";
import { usePathname, useSearchParams, useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { useTranslation } from "@/lib/i18n";
import { GlassButton } from "@/components/ui/glass-button";

const underlineSpring = { type: "spring", stiffness: 380, damping: 30 } as const;
import {
  HomeIcon,
  FilmIcon,
  TvIcon,
  BookmarkIcon,
  SparklesIcon,
  ChevronRight,
  MoreIcon,
  TagIcon,
  GlobeIcon,
  CalendarIcon,
  NetworkIcon,
} from "./Icons";
import SearchBox from "./SearchBox";
import { useAuth } from "./AuthProvider";

export default function Navbar() {
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const { user, ready, logout, activeProfile } = useAuth();
  const router = useRouter();
  const { t } = useTranslation();

  const NAV = [
    { label: t("nav.home"), href: "/home", icon: HomeIcon },
    { label: t("nav.movies"), href: "/browse?media=movie", icon: FilmIcon },
    { label: t("nav.tv"), href: "/browse?media=tv", icon: TvIcon },
    { label: t("nav.reality"), href: "/reality", icon: SparklesIcon },
    { label: t("nav.myList"), href: "/daftar-saya", icon: BookmarkIcon },
  ];

  const handleSwitchProfile = () => {
    sessionStorage.removeItem("profileSelected");
    router.push("/profiles");
  };
  const [scrolled, setScrolled] = useState(false);
  const [openMore, setOpenMore] = useState(false);
  const [openAccount, setOpenAccount] = useState(false);
  const [openMobileMenu, setOpenMobileMenu] = useState(false);
  // Cegah mismatch hydration: UI akun baru dirender setelah mount di client.
  const [mounted, setMounted] = useState(false);

  useEffect(() => setMounted(true), []);

  useEffect(() => {
    const onScroll = () => setScrolled(window.scrollY > 40);
    onScroll();
    window.addEventListener("scroll", onScroll, { passive: true });
    return () => window.removeEventListener("scroll", onScroll);
  }, []);

  // Tutup mobile menu saat navigasi
  useEffect(() => {
    setOpenMobileMenu(false);
  }, [pathname, searchParams]);

  // Kunci scroll body saat mobile menu terbuka
  useEffect(() => {
    if (openMobileMenu) {
      document.body.style.overflow = "hidden";
    } else {
      document.body.style.overflow = "";
    }
    return () => { document.body.style.overflow = ""; };
  }, [openMobileMenu]);

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
    { label: t("nav.genres"), href: `/genres?media=${activeMedia}`, icon: TagIcon },
    { label: t("nav.country"), href: `/countries?media=${activeMedia}`, icon: GlobeIcon },
    { label: t("nav.year"), href: `/years?media=${activeMedia}`, icon: CalendarIcon },
    { label: t("nav.network"), href: `/networks?media=${activeMedia}`, icon: NetworkIcon },
  ];

  const isActive = (href: string) => {
    const [path, query] = href.split("?");
    if (path === "/home") return pathname === "/home";
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

  const isLandingPage = pathname === "/";
  if (pathname === "/profiles") return null;
  // Halaman auth punya bar atas sendiri (logo + tombol pindah mode).
  if (pathname === "/masuk" || pathname === "/daftar") return null;

  return (
    <>
      <header
        className={`${
          isLandingPage ? "absolute" : "fixed"
        } inset-x-0 top-0 z-[1000] transition-all duration-300 ${
          scrolled && !isLandingPage ? "px-[3%] pt-3" : "px-0 pt-0"
        }`}
      >
        <div className="relative">
          {/* Glass Background isolated to prevent backdrop-filter CSS bug on dropdowns */}
          <div 
            className={`absolute inset-0 pointer-events-none transition-all duration-300 ${
              scrolled && !isLandingPage
                ? "rounded-full border border-white/20 bg-black/50 shadow-glass backdrop-blur-[35px]"
                : "border-0 border-transparent bg-gradient-to-b from-black/70 via-black/30 to-transparent shadow-none backdrop-blur-none"
            }`}
          />
          <nav
            className={`relative flex items-center justify-between gap-4 transition-all duration-300 ${
              scrolled && !isLandingPage ? "px-8 py-2.5" : "px-[4%] py-4"
            }`}
          >
          <div className="flex items-center gap-6">
            <Link
              href={user ? "/home" : "/"}
              className="text-4xl uppercase leading-none tracking-[-0.03em] text-accent"
              style={{ fontFamily: "var(--font-logo)" }}
            >
              Waveflix
            </Link>
            {/* Desktop nav tabs — hanya muncul di lg ke atas */}
            {user && (
              <ul className="hidden items-center gap-1.5 lg:flex">
                {NAV.map(({ label, href, icon: Icon }) => (
                  <li key={label}>
                    <Link
                      href={href}
                      className={`relative flex items-center gap-2 px-3 py-1.5 text-sm font-semibold transition-colors duration-200 ${
                        isActive(href) ? "text-accent" : "text-white/70 hover:text-white"
                      }`}
                    >
                      <Icon className="w-4 h-4" />
                      <span>{label}</span>
                      {isActive(href) && (
                        <motion.span
                          layoutId="navUnderline"
                          transition={underlineSpring}
                          className="absolute inset-0 -z-10 rounded-full bg-white/[0.08] shadow-[inset_0_1px_1px_rgba(255,255,255,0.15)] ring-1 ring-white/10"
                        />
                      )}
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
                    className={`relative flex items-center gap-2 px-2 pb-2 pt-1 text-sm font-semibold transition ${
                      onCategory ? "text-accent" : "text-white/70 hover:text-white"
                    }`}
                  >
                    {onCategory && (
                      <motion.span
                        layoutId="navUnderline"
                        transition={underlineSpring}
                        className="absolute inset-x-1 bottom-0 h-0.5 rounded bg-accent"
                      />
                    )}
                    <MoreIcon />
                    <span>{t("nav.more")}</span>
                    <ChevronRight className={`transition ${openMore ? "rotate-90" : ""}`} />
                  </button>
                  {openMore && (
                    <div className="absolute left-0 top-full pt-4">
                      <div className="w-56 rounded-xl border border-white/20 bg-black/50 p-2 shadow-glass backdrop-blur-[35px]">
                        {moreMenu.map(({ label, href, icon: Icon }) => (
                          <Link
                            key={label}
                            href={href}
                            className="flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-semibold text-white/75 transition hover:bg-white/10 hover:text-white"
                          >
                            <span className="text-white/60">
                              <Icon />
                            </span>
                            {label}
                          </Link>
                        ))}
                      </div>
                    </div>
                  )}
                </li>
              </ul>
            )}
          </div>

          <div className="flex items-center gap-3">
            {user && <SearchBox />}
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
                  className="grid h-9 w-9 shrink-0 place-items-center overflow-hidden rounded-full bg-accent font-bold text-black"
                >
                  {activeProfile ? (
                    (activeProfile as any).avatar ? (
                      // eslint-disable-next-line @next/next/no-img-element
                      <img
                        src={(activeProfile as any).avatar}
                        alt=""
                        className="h-full w-full object-cover"
                      />
                    ) : (
                      ((activeProfile as any).name || "?").charAt(0).toUpperCase()
                    )
                  ) : user.avatar ? (
                    // eslint-disable-next-line @next/next/no-img-element
                    <img src={user.avatar} alt="" className="h-full w-full object-cover" />
                  ) : (
                    (user.username || "?").charAt(0).toUpperCase()
                  )}
                </button>
                {openAccount && (
                  <div className="absolute right-0 top-full pt-4">
                    <div className="w-56 rounded-xl border border-white/20 bg-black/50 p-2 shadow-glass backdrop-blur-[35px]">
                      <div className="px-3 py-2">
                        <div className="truncate text-sm font-semibold">
                          {activeProfile ? (activeProfile as any).name : user.username}
                        </div>
                        <div className="truncate text-xs text-white/50">{user.email}</div>
                      </div>
                      <div className="my-1 h-px bg-white/10" />
                      {/* Ganti Profil */}
                      <button
                        onClick={handleSwitchProfile}
                        className="flex w-full items-center gap-2 rounded-md px-3 py-2 text-left text-sm text-white/70 hover:bg-white/10 hover:text-white"
                      >
                        <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
                        </svg>
                        {t("nav.switchProfile")}
                      </button>
                      <div className="my-1 h-px bg-white/10" />
                      <Link href="/account/settings" className="block rounded-md px-3 py-2 text-sm text-white/70 hover:bg-white/10 hover:text-white">
                        {t("nav.settings")}
                      </Link>
                      <Link href="/daftar-saya" className="block rounded-md px-3 py-2 text-sm text-white/70 hover:bg-white/10 hover:text-white">
                        {t("nav.myList")}
                      </Link>
                      <Link href="/favorit" className="block rounded-md px-3 py-2 text-sm text-white/70 hover:bg-white/10 hover:text-white">
                        {t("nav.favorites")}
                      </Link>
                      <Link href="/riwayat" className="block rounded-md px-3 py-2 text-sm text-white/70 hover:bg-white/10 hover:text-white">
                        {t("nav.history")}
                      </Link>
                      <div className="my-1 h-px bg-white/10" />
                      <button
                        onClick={logout}
                        className="block w-full rounded-md px-3 py-2 text-left text-sm text-red-400 hover:bg-white/10"
                      >
                        {t("nav.logout")}
                      </button>
                    </div>
                  </div>
                )}
              </div>
            ) : (
              <Link href="/masuk">
                <GlassButton size="sm" contentClassName="px-2">
                  {t("nav.login")}
                </GlassButton>
              </Link>
            )}

            {/* Hamburger button — hanya muncul di bawah lg, hanya jika user login */}
            {user && (
              <button
                aria-label="Open menu"
                onClick={() => setOpenMobileMenu((v) => !v)}
                className="relative flex lg:hidden h-9 w-9 shrink-0 flex-col items-center justify-center gap-[5px] rounded-full transition hover:bg-white/10"
              >
                <motion.span
                  animate={openMobileMenu ? { rotate: 45, y: 7 } : { rotate: 0, y: 0 }}
                  transition={{ duration: 0.2 }}
                  className="block h-0.5 w-5 rounded-full bg-white"
                />
                <motion.span
                  animate={openMobileMenu ? { opacity: 0, scaleX: 0 } : { opacity: 1, scaleX: 1 }}
                  transition={{ duration: 0.15 }}
                  className="block h-0.5 w-5 rounded-full bg-white"
                />
                <motion.span
                  animate={openMobileMenu ? { rotate: -45, y: -7 } : { rotate: 0, y: 0 }}
                  transition={{ duration: 0.2 }}
                  className="block h-0.5 w-5 rounded-full bg-white"
                />
              </button>
            )}
          </div>
          </nav>
        </div>
      </header>

      {/* Mobile Drawer Menu */}
      <AnimatePresence>
        {openMobileMenu && user && (
          <>
            {/* Backdrop */}
            <motion.div
              key="mobile-backdrop"
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              transition={{ duration: 0.2 }}
              onClick={() => setOpenMobileMenu(false)}
              className="fixed inset-0 z-[999] bg-black/60 backdrop-blur-sm lg:hidden"
            />
            {/* Drawer */}
            <motion.div
              key="mobile-drawer"
              initial={{ x: "-100%" }}
              animate={{ x: 0 }}
              exit={{ x: "-100%" }}
              transition={{ type: "spring", stiffness: 350, damping: 35 }}
              className="fixed left-0 top-0 z-[1001] h-full w-72 bg-[#0a0a0f] border-r border-white/10 shadow-2xl lg:hidden flex flex-col"
            >
              {/* Header drawer */}
              <div className="flex items-center justify-between px-6 py-5 border-b border-white/10">
                <Link
                  href={user ? "/home" : "/"}
                  className="text-3xl uppercase leading-none tracking-[-0.03em] text-accent"
                  style={{ fontFamily: "var(--font-logo)" }}
                  onClick={() => setOpenMobileMenu(false)}
                >
                  Waveflix
                </Link>
                <button
                  onClick={() => setOpenMobileMenu(false)}
                  className="grid h-8 w-8 place-items-center rounded-full text-white/60 hover:bg-white/10 hover:text-white transition"
                  aria-label="Close menu"
                >
                  <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>

              {/* Profile info */}
              <div className="flex items-center gap-3 px-6 py-4 border-b border-white/10">
                <div className="grid h-10 w-10 shrink-0 place-items-center overflow-hidden rounded-full bg-accent font-bold text-black text-sm">
                  {activeProfile ? (
                    (activeProfile as any).avatar ? (
                      // eslint-disable-next-line @next/next/no-img-element
                      <img src={(activeProfile as any).avatar} alt="" className="h-full w-full object-cover" />
                    ) : (
                      ((activeProfile as any).name || "?").charAt(0).toUpperCase()
                    )
                  ) : user.avatar ? (
                    // eslint-disable-next-line @next/next/no-img-element
                    <img src={user.avatar} alt="" className="h-full w-full object-cover" />
                  ) : (
                    (user.username || "?").charAt(0).toUpperCase()
                  )}
                </div>
                <div className="min-w-0">
                  <div className="truncate text-sm font-semibold text-white">
                    {activeProfile ? (activeProfile as any).name : user.username}
                  </div>
                  <div className="truncate text-xs text-white/50">{user.email}</div>
                </div>
              </div>

              {/* Nav Links */}
              <nav className="flex-1 overflow-y-auto px-3 py-4 space-y-1">
                <p className="px-3 pb-2 text-[10px] font-semibold uppercase tracking-widest text-white/30">
                  Menu
                </p>
                {NAV.map(({ label, href, icon: Icon }) => (
                  <Link
                    key={label}
                    href={href}
                    className={`flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm font-semibold transition-all ${
                      isActive(href)
                        ? "bg-accent/15 text-accent"
                        : "text-white/70 hover:bg-white/8 hover:text-white"
                    }`}
                  >
                    <Icon className="w-4 h-4 shrink-0" />
                    {label}
                    {isActive(href) && (
                      <span className="ml-auto h-1.5 w-1.5 rounded-full bg-accent" />
                    )}
                  </Link>
                ))}

                {/* More section */}
                <p className="px-3 pb-2 pt-4 text-[10px] font-semibold uppercase tracking-widest text-white/30">
                  {t("nav.more")}
                </p>
                {moreMenu.map(({ label, href, icon: Icon }) => (
                  <Link
                    key={label}
                    href={href}
                    className="flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm font-semibold text-white/70 transition-all hover:bg-white/8 hover:text-white"
                  >
                    <Icon className="w-4 h-4 shrink-0" />
                    {label}
                  </Link>
                ))}
              </nav>

              {/* Footer drawer */}
              <div className="border-t border-white/10 px-3 py-4 space-y-1">
                <button
                  onClick={handleSwitchProfile}
                  className="flex w-full items-center gap-3 rounded-xl px-3 py-2.5 text-sm text-white/70 hover:bg-white/8 hover:text-white transition"
                >
                  <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
                  </svg>
                  {t("nav.switchProfile")}
                </button>
                <Link
                  href="/account/settings"
                  className="flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm text-white/70 hover:bg-white/8 hover:text-white transition"
                >
                  <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                  </svg>
                  {t("nav.settings")}
                </Link>
                <button
                  onClick={logout}
                  className="flex w-full items-center gap-3 rounded-xl px-3 py-2.5 text-sm text-red-400 hover:bg-red-500/10 transition"
                >
                  <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1" />
                  </svg>
                  {t("nav.logout")}
                </button>
              </div>
            </motion.div>
          </>
        )}
      </AnimatePresence>
    </>
  );
}
