"use client";

import { useEffect, useRef, useState } from "react";
import Link from "next/link";
import type { HeroSlide } from "@/lib/tmdb";
import { useTranslation } from "@/lib/i18n";
import {
  backdropUrl,
  itemTitle,
  itemYear,
  ratingText,
  detailHref,
  isTv,
} from "@/lib/helpers";
import { StarIcon, PlayIcon } from "./Icons";

const IMAGE_MS = 5000; // video main di belakang gambar dulu, baru gambar fade-out
const IMAGE_ONLY_MS = 10000;

function VolumeOn() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" width="18" height="18">
      <path d="M11 5 6 9H2v6h4l5 4V5Z" /><path d="M15.5 8.5a5 5 0 0 1 0 7M19 5a9 9 0 0 1 0 14" />
    </svg>
  );
}
function VolumeOff() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" width="18" height="18">
      <path d="M11 5 6 9H2v6h4l5 4V5Z" /><path d="m23 9-6 6M17 9l6 6" />
    </svg>
  );
}

export default function HeroCarousel({ slides }: { slides: HeroSlide[] }) {
  const { t } = useTranslation();
  const [active, setActive] = useState(0);
  const [revealed, setRevealed] = useState(false); // gambar sudah fade-out?
  const [muted, setMuted] = useState(true);
  const [inView, setInView] = useState(true);
  const sectionRef = useRef<HTMLElement>(null);
  const iframeRef = useRef<HTMLIFrameElement>(null);

  const total = slides.length;
  const current = slides[active];
  const playVideo = inView && !!current?.trailer;

  useEffect(() => {
    const el = sectionRef.current;
    if (!el) return;
    const io = new IntersectionObserver(([e]) => setInView(e.isIntersecting), { threshold: 0.35 });
    io.observe(el);
    return () => io.disconnect();
  }, []);

  // Auto-geser slide (hanya jika TIDAK ada trailer).
  // Jika ada trailer, perpindahan slide di-handle oleh event "infoDelivery" dari YouTube saat video selesai (state = 0).
  useEffect(() => {
    if (total <= 1) return;
    if (slides[active]?.trailer) return; // tunggu video selesai

    const t = setTimeout(() => setActive((i) => (i + 1) % total), IMAGE_ONLY_MS);
    return () => clearTimeout(t);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [active, total]);

  // Dengarkan progress video menggunakan YouTube IFrame API resmi
  useEffect(() => {
    if (!playVideo) return;
    let player: any;
    let timer: NodeJS.Timeout;
    let skipped = false;

    const attachAPI = () => {
      if (!iframeRef.current || !(window as any).YT) return;
      player = new (window as any).YT.Player(iframeRef.current, {
        events: {
          onStateChange: (e: any) => {
            // 0 = Ended
            if (e.data === 0 && !skipped) {
              skipped = true;
              setRevealed(false);
              setTimeout(() => {
                setActive((prev) => (prev + 1) % total);
              }, 10000);
            }
          }
        }
      });

      // Polling waktu setiap 250ms (dijalankan di luar onReady karena onReady sering terlewat jika iframe sudah loading)
      timer = setInterval(() => {
        if (skipped || !player || typeof player.getCurrentTime !== "function") return;
        try {
          const current = player.getCurrentTime();
          const duration = player.getDuration();
          // Skip 1.5 detik sebelum habis
          if (duration > 0 && duration - current < 1.5) {
            skipped = true;
            setRevealed(false); // Kembali ke foto
            setTimeout(() => {
              setActive((prev) => (prev + 1) % total);
            }, 10000); // Jeda 10 detik
          }
        } catch (err) {}
      }, 250);
    };

    if (!(window as any).YT) {
      const tag = document.createElement("script");
      tag.src = "https://www.youtube.com/iframe_api";
      const firstScriptTag = document.getElementsByTagName("script")[0];
      firstScriptTag.parentNode?.insertBefore(tag, firstScriptTag);
      (window as any).onYouTubeIframeAPIReady = attachAPI;
    } else if (!(window as any).YT.Player) {
      // Script loaded but API not ready yet, wait for it
      (window as any).onYouTubeIframeAPIReady = attachAPI;
    } else {
      attachAPI();
    }

    return () => {
      if (timer) clearInterval(timer);
    };
  }, [active, playVideo, total]);

  // Reveal: video main dulu 5 detik di belakang gambar, lalu gambar fade-out.
  useEffect(() => {
    setRevealed(false);
    if (!playVideo) return;
    const t = setTimeout(() => setRevealed(true), IMAGE_MS);
    return () => clearTimeout(t);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [active, playVideo]);

  const sendCmd = (func: string, args: any[] = []) =>
    iframeRef.current?.contentWindow?.postMessage(
      JSON.stringify({ event: "command", func, args }),
      "*"
    );
  const toggleMute = () => {
    const next = !muted;
    setMuted(next);
    sendCmd(next ? "mute" : "unMute");
  };

  if (total === 0) return null;

  const embed =
    playVideo && current
      ? `https://www.youtube.com/embed/${current.trailer}?autoplay=1&mute=1&controls=0&disablekb=1&fs=0&modestbranding=1&rel=0&enablejsapi=1&playsinline=1&iv_load_policy=3&cc_load_policy=0&vq=hd1080`
      : "";

  return (
    <section ref={sectionRef} className="relative h-[86vh] min-h-[560px] w-full overflow-hidden bg-black">
      {/* Video (base) — main di belakang gambar */}
      {playVideo && current && (
        <div key={current.item.id} className="absolute inset-0 overflow-hidden">
          <iframe
            ref={iframeRef}
            src={embed}
            title="Trailer"
            allow="autoplay; encrypted-media"
            onLoad={() => {
              if (!muted) sendCmd("unMute");
              
              // Matikan subtitle secara paksa via JS API (Fallback)
              sendCmd("unloadModule", ["captions"]);
              sendCmd("setOption", ["captions", "track", {}]);
            }}
            className="pointer-events-none absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2"
            style={{ width: "max(100%, 177.78vh)", height: "max(100%, 56.25vw)" }}
          />
        </div>
      )}

      {/* Gambar per-slide (di atas video, fade-out untuk mengungkap video) */}
      {slides.map((s, i) => (
        <div
          key={s.item.id}
          className="absolute inset-0 bg-cover bg-top transition-opacity duration-[1200ms]"
          style={{
            backgroundImage: `url('${backdropUrl(s.item)}')`,
            opacity: i === active ? (revealed && playVideo ? 0 : 1) : 0,
          }}
        />
      ))}

      {/* Redup + fade bawah (di atas video & gambar) */}
      <div className="pointer-events-none absolute inset-0 bg-black/35" />
      <div className="pointer-events-none absolute inset-x-0 bottom-0 h-48 bg-gradient-to-t from-bg to-transparent" />

      {/* Mute/unmute (saat trailer terlihat) */}
      {revealed && playVideo && (
        <button
          onClick={toggleMute}
          aria-label={muted ? "Unmute" : "Mute"}
          className="absolute bottom-[10vh] right-[4%] z-[3] grid h-11 w-11 place-items-center rounded-full border border-white/40 bg-black/40 text-white/90 backdrop-blur transition hover:border-white hover:text-white"
        >
          {muted ? <VolumeOff /> : <VolumeOn />}
        </button>
      )}

      {/* Konten slide aktif */}
      <div className="relative z-[2] flex h-full items-end">
        <div className="w-full max-w-2xl px-[4%] pb-[8vh]">
          {slides.map((s, i) => {
            const tv = isTv(s.item);
            const desc = s.overview.length > 220 ? s.overview.slice(0, 220) + "…" : s.overview;
            return (
              <div key={s.item.id} className={i === active ? "block animate-[fadeIn_.5s_ease]" : "hidden"}>
                <span className="mb-4 inline-block rounded bg-accent px-2.5 py-1 text-xs font-bold uppercase tracking-wider text-black">
                  {tv ? t("ui.tvSeries") : t("ui.movie")}
                </span>

                {s.tagline && (
                  <p className="mb-2 text-sm italic text-white/70 drop-shadow md:text-base">{s.tagline}</p>
                )}

                {s.logo ? (
                  // eslint-disable-next-line @next/next/no-img-element
                  <img src={s.logo} alt={itemTitle(s.item)} className="mb-4 max-h-16 w-auto max-w-[220px] object-contain object-left drop-shadow-logo md:max-h-24 md:max-w-[280px]" />
                ) : (
                  <h1 className="mb-4 max-w-xl text-3xl font-bold leading-tight drop-shadow-lg md:text-4xl">
                    {itemTitle(s.item)}
                  </h1>
                )}

                <div className="mb-4 flex flex-wrap items-center gap-x-3 gap-y-2 text-sm text-white/90 drop-shadow-meta">
                  <span className="flex items-center gap-1 font-semibold text-[#f5c518]"><StarIcon /> {ratingText(s.item)}</span>
                  <span className="text-white/40">&bull;</span>
                  <span>{itemYear(s.item)}</span>
                  {s.duration && (<><span className="text-white/40">&bull;</span><span>{s.duration}</span></>)}
                  {s.genres.length > 0 && (<><span className="text-white/40">&bull;</span><span>{s.genres.join(", ")}</span></>)}
                  {s.status && (
                    <span className="rounded border border-emerald-500/70 px-2 py-0.5 text-xs font-bold text-emerald-400">{s.status}</span>
                  )}
                </div>

                {desc && (
                  <p className="mb-6 max-w-lg text-xs leading-relaxed text-white/75 drop-shadow md:text-sm">{desc}</p>
                )}

                <Link href={detailHref(s.item)} className="inline-flex items-center gap-2 rounded-md bg-accent px-7 py-3 text-base font-semibold text-black transition hover:scale-105 hover:bg-accent-dark">
                  <PlayIcon className="text-black" /> {t("ui.watchNow")}
                </Link>
              </div>
            );
          })}

          {total > 1 && (
            <div className="mt-8 flex items-center gap-2">
              {slides.map((s, i) => (
                <button
                  key={s.item.id}
                  onClick={() => setActive(i)}
                  aria-label={`Slide ${i + 1}`}
                  className={`h-2 rounded-full transition-all ${i === active ? "w-7 bg-accent" : "w-2 bg-white/35 hover:bg-white/60"}`}
                />
              ))}
            </div>
          )}
        </div>
      </div>
    </section>
  );
}
