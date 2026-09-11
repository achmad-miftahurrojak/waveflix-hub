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
  FolderIcon,
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

  const [mounted, setMounted] = useState(false);

  useEffect(() => setMounted(true), []);

  useEffect(() => {
    const onScroll = () => setScrolled(window.scrollY > 40);
    onScroll();
    window.addEventListener("scroll", onScroll, { passive: true });
    return () => window.removeEventListener("scroll", onScroll);
  }, []);

  const activeMedia = searchParams.get("media") === "tv" ? "tv" : searchParams.get("media") === "movie" ? "movie" : "all";

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

  if (activeMedia === "movie" || activeMedia === "all") {
    moreMenu.push({ label: t("nav.collection"), href: `/collections?media=movie`, icon: FolderIcon });
  }


  const isActive = (href: string) => {
    const [path, query] = href.split("?");
    if (path === "/home") return pathname === "/home";
    if (!pathname.startsWith(path)) return false;

    const hrefMedia = new URLSearchParams(query ?? "").get("media");
    if (hrefMedia) {

      if (onCategory) return false;
      return searchParams.get("media") === hrefMedia;
    }
    return true;
  };

  const isLandingPage = pathname === "/";
  if (pathname === "/profiles") return null;

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
          {}
          <div 
            className={`absolute inset-0 pointer-events-none transition-all duration-300 ${
              scrolled && !isLandingPage
                ? "rounded-full border border-white/20 bg-white/5 backdrop-blur-[15px] backdrop-saturate-200 shadow-[0_4px_30px_rgba(0,0,0,0.1),inset_0_1px_0_rgba(255,255,255,0.3)]"
                : "border-0 border-transparent bg-gradient-to-b from-black/70 via-black/30 to-transparent shadow-none backdrop-blur-none"
            }`}
          />
          <nav
            className={`relative flex items-center justify-between gap-4 transition-all duration-300 ${
              scrolled && !isLandingPage ? "px-8 py-2.5" : "px-[4%] py-4"
            }`}
          >
          <div className="flex items-center gap-6">
            <Link href={user ? "/home" : "/"}>
              <img src="/1.png" alt="Waveflix" className="h-8 w-auto object-contain scale-[2.5] origin-left mr-20" />
            </Link>
            {}
            {user && (
              <ul className="flex items-center gap-0 md:gap-1">
                {NAV.map(({ label, href }) => (
                  <li key={label}>
                    <Link
                      href={href}
                      title={label}
                      className={`relative flex items-center px-2 md:px-3 py-1.5 text-sm font-semibold transition-colors duration-200 ${
                        isActive(href) ? "text-accent" : "text-white/70 hover:text-white"
                      }`}
                    >
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
                {}
                <li
                  className="relative"
                  onMouseEnter={() => setOpenMore(true)}
                  onMouseLeave={() => setOpenMore(false)}
                >
                  <button
                    title={t("nav.more")}
                    className={`relative flex items-center gap-1.5 px-2 md:px-2 pb-2 pt-1 text-sm font-semibold transition ${
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
                    <span>{t("nav.more")}</span>
                    <ChevronRight className={`transition ${openMore ? "rotate-90" : ""}`} />
                  </button>
                  {openMore && (
                    <div className="absolute left-0 top-full pt-4">
                      <div className="w-56 rounded-xl border border-white/20 bg-white/5 p-2 backdrop-blur-[15px] backdrop-saturate-200 shadow-[0_4px_30px_rgba(0,0,0,0.1),inset_0_1px_0_rgba(255,255,255,0.3)]">
                        {moreMenu.map(({ label, href }) => (
                          <Link
                            key={label}
                            href={href}
                            className="flex items-center rounded-lg px-3 py-2.5 text-sm font-semibold text-white/75 transition hover:bg-white/10 hover:text-white"
                          >
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

                      <img
                        src={(activeProfile as any).avatar}
                        alt=""
                        className="h-full w-full object-cover"
                      />
                    ) : (
                      ((activeProfile as any).name || "?").charAt(0).toUpperCase()
                    )
                  ) : user.avatar ? (

                    <img src={user.avatar} alt="" className="h-full w-full object-cover" />
                  ) : (
                    (user.username || "?").charAt(0).toUpperCase()
                  )}
                </button>
                {openAccount && (
                  <div className="absolute right-0 top-full pt-4">
                    <div className="w-56 rounded-xl border border-white/20 bg-white/5 p-2 backdrop-blur-[15px] backdrop-saturate-200 shadow-[0_4px_30px_rgba(0,0,0,0.1),inset_0_1px_0_rgba(255,255,255,0.3)]">
                      <div className="px-3 py-2">
                        <div className="truncate text-sm font-semibold">
                          {activeProfile ? (activeProfile as any).name : user.username}
                        </div>
                        <div className="truncate text-xs text-white/50">{user.email}</div>
                      </div>
                      <div className="my-1 h-px bg-white/10" />
                      {}
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

          </div>
          </nav>
        </div>
      </header>

    </>
  );
}
