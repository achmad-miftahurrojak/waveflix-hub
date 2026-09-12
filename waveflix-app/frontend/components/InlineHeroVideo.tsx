"use client";

import { useEffect, useRef, useState, useCallback, type ReactNode } from "react";
import { NativePlayer } from "./ui/NativePlayer";
import { useUI } from "./UIProvider";
import { useAuth } from "./AuthProvider";
import { MaximizeIcon, MinimizeIcon, CloseIcon } from "./Icons";

interface Props {
  id: number;
  season?: number;
  episode?: number;
  backdrop: string;
  heightClass: string;
  children: ReactNode;
  trailer?: string | null;
}

export default function InlineHeroVideo({
  id,
  season,
  episode,
  backdrop,
  heightClass,
  children,
  trailer,
}: Props) {
  const { player, stop } = useUI();
  const { user, authFetch, recordHistory } = useAuth();

  const [savedProgressSeconds, setSavedProgressSeconds] = useState(0);
  const [lastProgressSaved, setLastProgressSaved] = useState(0);
  const active =
    !!player &&
    player.item.id === id &&
    player.season === season &&
    player.episode === episode;

  const wrapRef = useRef<HTMLDivElement>(null);
  const [isFs, setIsFs] = useState(false);
  const [controlsVisible, setControlsVisible] = useState(true);
  const hideTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const [showTrailer, setShowTrailer] = useState(false);
  const [trailerPlaying, setTrailerPlaying] = useState(false);
  const [playCount, setPlayCount] = useState(0);
  const trailerIframeRef = useRef<HTMLIFrameElement>(null);

  useEffect(() => {
    if (active || !trailer) {
      setShowTrailer(false);
      setTrailerPlaying(false);
      return;
    }
    const t = setTimeout(() => {
      setShowTrailer(true);
    }, 10000);
    return () => clearTimeout(t);
  }, [active, trailer, playCount]);

  useEffect(() => {
    if (!showTrailer || !trailer) return;
    let ytPlayer: any;
    const attachAPI = () => {
      if (!trailerIframeRef.current || !(window as any).YT) return;
      ytPlayer = new (window as any).YT.Player(trailerIframeRef.current, {
        events: {
          onReady: () => {},
          onStateChange: (e: any) => {
            if (e.data === 1) {
              setTrailerPlaying(true);
            }
            if (e.data === 0) {
              setShowTrailer(false);
              setTrailerPlaying(false);
              setPlayCount(c => c + 1);
            }
          },
          onError: () => {
            setShowTrailer(false);
            setTrailerPlaying(false);
            setPlayCount(c => c + 1);
          }
        }
      });
    };

    if (!(window as any).YT) {
      if (!document.querySelector('script[src="https://www.youtube.com/iframe_api"]')) {
        const tag = document.createElement("script");
        tag.src = "https://www.youtube.com/iframe_api";
        const firstScriptTag = document.getElementsByTagName("script")[0];
        firstScriptTag.parentNode?.insertBefore(tag, firstScriptTag);
      }
      const oldFn = (window as any).onYouTubeIframeAPIReady;
      (window as any).onYouTubeIframeAPIReady = () => {
        if (oldFn) oldFn();
        attachAPI();
      };
    } else {
      attachAPI();
    }
    return () => {
      if (ytPlayer && typeof ytPlayer.destroy === "function") {
        ytPlayer.destroy();
      }
    };
  }, [showTrailer, trailer]);

  useEffect(() => {
    const onFs = () => setIsFs(!!document.fullscreenElement);
    document.addEventListener("fullscreenchange", onFs);
    return () => document.removeEventListener("fullscreenchange", onFs);
  }, []);

  const revealControls = useCallback(() => {
    setControlsVisible(true);
    if (hideTimer.current) clearTimeout(hideTimer.current);
    hideTimer.current = setTimeout(() => setControlsVisible(false), 2600);
  }, []);

  useEffect(() => {
    if (active && user && player) {
      if (player.progress !== undefined) {
         setSavedProgressSeconds(player.progress * 60);
      } else {
        authFetch("/api/history")
          .then(r => r.json())
          .then(d => {
             const historyItem = (d.results ?? []).find((x: any) => 
               x.tmdb_id === player.item.id && 
               x.season == player.season && 
               x.episode == player.episode
             );
             if (historyItem && historyItem.progress) {
               setSavedProgressSeconds(historyItem.progress * 60);
             } else {
               setSavedProgressSeconds(0);
             }
          }).catch(() => setSavedProgressSeconds(0));
      }
    } else {
      setSavedProgressSeconds(0);
      setLastProgressSaved(0);
    }
  }, [active, user, player, authFetch]);

  useEffect(() => {
    if (!active || !player) return;

    const handleMessage = (e: MessageEvent) => {
      let currentTime = 0;
      if (e.data && e.data.type === 'video.timeupdate') {
        currentTime = e.data.time;
      } else if (e.data && e.data.event === 'timeupdate' && e.data.currentTime) {
        currentTime = e.data.currentTime;
      }

      if (currentTime > 0) {
        setLastProgressSaved((prev) => {
          if (Math.abs(currentTime - prev) > 10) {
             recordHistory(player.item, player.season, player.episode, currentTime);
             return currentTime;
          }
          return prev;
        });
      }
    };

    window.addEventListener('message', handleMessage);

    recordHistory(player.item, player.season, player.episode, savedProgressSeconds > 0 ? savedProgressSeconds : 0);

    return () => window.removeEventListener('message', handleMessage);
  }, [active, player, recordHistory, savedProgressSeconds]);

  useEffect(() => {
    if (!active) {
      if (hideTimer.current) clearTimeout(hideTimer.current);
      return;
    }
    revealControls();
  }, [active]);

  const toggleFullscreen = () => {
    const el = wrapRef.current;
    if (!el) return;
    if (document.fullscreenElement) document.exitFullscreen().catch(() => {});
    else el.requestFullscreen().catch(() => {});
  };

  if (active) {
    return (
      <div className="w-full bg-background pt-24 pb-12 px-[4%]">
        <div
          ref={wrapRef}
          onMouseMove={revealControls}
          onMouseLeave={() => setControlsVisible(false)}
          className={`relative w-full max-w-[1200px] mx-auto bg-black aspect-video rounded-xl overflow-hidden shadow-2xl ring-1 ring-white/10`}
        >
          <div className="absolute inset-0 z-10">
            <NativePlayer
              mediaType={player.item.media_type as "movie" | "tv"}
              tmdbId={player.item.id.toString()}
              season={player.season?.toString()}
              episode={player.episode?.toString()}
            />
          </div>

          <div
            className={`absolute right-4 top-4 z-30 flex items-center gap-3 transition-opacity duration-300 ${
              controlsVisible ? "opacity-100" : "pointer-events-none opacity-0"
            }`}
          >
            <button
              onClick={toggleFullscreen}
              aria-label="Toggle Fullscreen"
              className="grid h-10 w-10 place-items-center rounded-full bg-black/60 text-white/80 transition hover:bg-accent hover:text-black border border-white/20"
            >
              {isFs ? <MinimizeIcon /> : <MaximizeIcon />}
            </button>
            <button
              onClick={stop}
              aria-label="Close player"
              className="grid h-10 w-10 place-items-center rounded-full bg-black/60 text-white/80 transition hover:bg-accent hover:text-black border border-white/20"
            >
              <CloseIcon />
            </button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div
      className={`relative flex items-end overflow-hidden ${heightClass}`}
    >
      <div
        className={`absolute inset-0 transition-opacity duration-1000 ${
          trailerPlaying ? "opacity-0" : "opacity-100"
        }`}
        style={{
          backgroundImage: `url('${backdrop}')`,
          backgroundSize: "cover",
          backgroundPosition: "center top",
        }}
      />
      {!active && trailer && showTrailer && (
        <div className="absolute inset-0 overflow-hidden pointer-events-none z-0">
          <iframe
            ref={trailerIframeRef}
            src={`https://www.youtube.com/embed/${trailer}?enablejsapi=1&autoplay=1&mute=1&controls=0&disablekb=1&fs=0&modestbranding=1&rel=0&playsinline=1&iv_load_policy=3&cc_load_policy=0&vq=hd1080`}
            title="Trailer"
            allow="autoplay; encrypted-media"
            className="absolute top-1/2 left-1/2 h-[56.25vw] w-[177.77vh] min-h-full min-w-full -translate-x-1/2 -translate-y-1/2 object-cover pointer-events-none scale-125"
          />
        </div>
      )}
      <div className="pointer-events-none absolute inset-0 bg-black/45 z-[1]" />
      <div className="pointer-events-none absolute inset-x-0 bottom-0 h-48 bg-gradient-to-t from-bg to-transparent z-[1]" />
      <div className="relative z-10 w-full">
        {children}
      </div>
    </div>
  );
}
