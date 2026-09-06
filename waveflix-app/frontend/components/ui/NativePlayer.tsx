"use client";

import { useEffect, useRef, useState } from "react";
import Hls from "hls.js";
import { Play, Pause, Volume2, VolumeX, Maximize, Settings } from "lucide-react";
import { cn } from "@/lib/utils";

interface NativePlayerProps {
  mediaType: "movie" | "tv";
  tmdbId: string;
  season?: string;
  episode?: string;
}

const BACKEND_URL = process.env.NEXT_PUBLIC_BACKEND_URL || "http://localhost:8081";

function proxyUrl(url: string): string {
  if (url.includes("localhost:8000")) return url; // Direct stream for Torrents
  const encoded = btoa(url).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
  return `${BACKEND_URL}/api/media-proxy?url=${encoded}`;
}

export function NativePlayer({ mediaType, tmdbId, season, episode }: NativePlayerProps) {
  const videoRef = useRef<HTMLVideoElement>(null);
  const [isPlaying, setIsPlaying] = useState(false);
  const [isMuted, setIsMuted] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [qualityLevels, setQualityLevels] = useState<{file: string; type: string; label: string}[]>([]);
  const [currentQuality, setCurrentQuality] = useState(0);
  const [showSettings, setShowSettings] = useState(false);
  const [captions, setCaptions] = useState<{id: string; language: string; url: string}[]>([]);
  const [allSources, setAllSources] = useState<{file: string; type: string; label: string}[]>([]);

  const hlsRef = useRef<Hls | null>(null);

  useEffect(() => {
    let query = `?media=${mediaType}&id=${tmdbId}`;
    if (mediaType === "tv" && season && episode) {
      query += `&s=${season}&e=${episode}`;
    }

    const bridgeUrl = `${BACKEND_URL}/api/stream${query}`;

    setIsLoading(true);
    setError(null);

    fetch(bridgeUrl)
      .then((res) => {
        if (!res.ok) {
          throw new Error("Video belum tersedia atau provider offline.");
        }
        return res.json();
      })
      .then((data) => {
        let sources: {file: string; type: string; label: string}[] = [];

        if (data?.sources?.length) {
          sources = data.sources;
        } else if (data?.stream?.qualities) {
          const q = data.stream.qualities;
          const preferred = ["1080", "720", "480", "360"];
          for (const p of preferred) {
            if (q[p]?.url) {
              sources.push({ file: q[p].url, type: "mp4", label: `${p}p` });
            }
          }
        }

        if (data?.tracks) {
          setCaptions(data.tracks.map((t: any, i: number) => ({
            id: String(i),
            language: t.label,
            url: proxyUrl(t.file)
          })));
        } else if (data?.captions) {
          setCaptions(data.captions);
        }

        if (!sources.length) {
          throw new Error("Link stream tidak valid dari provider.");
        }

        setAllSources(sources);
        setQualityLevels(sources);
        setCurrentQuality(0);
        initializePlayer(sources[0].file, sources[0].type);
      })
      .catch((err) => {
        setError(err.message);
        setIsLoading(false);
      });

    return () => {
      if (hlsRef.current) {
        hlsRef.current.destroy();
      }
    };
  }, [mediaType, tmdbId, season, episode]);

  const initializePlayer = (src: string, type: string = "hls") => {
    const video = videoRef.current;
    if (!video) return;

    if (hlsRef.current) {
      hlsRef.current.destroy();
      hlsRef.current = null;
    }

    const proxiedSrc = proxyUrl(src);

    if (type === "mp4" || (!src.includes(".m3u8") && !src.endsWith("/hls"))) {
      video.src = proxiedSrc;
      video.load();
      video.addEventListener("loadedmetadata", () => {
        setIsLoading(false);
        video.play().catch(() => {});
      }, { once: true });
      video.addEventListener("error", () => setError("Gagal memutar video. Silakan coba kualitas lain."), { once: true });
      return;
    }

    if (Hls.isSupported()) {
      const hls = new Hls({
        maxBufferLength: 30,
        maxMaxBufferLength: 60,
      });

      hls.loadSource(proxiedSrc);
      hls.attachMedia(video);

      hls.on(Hls.Events.MANIFEST_PARSED, () => {
        setIsLoading(false);
        video.play().catch(() => {});
      });

      hls.on(Hls.Events.ERROR, (_event, data) => {
        if (data.fatal) {
          switch (data.type) {
            case Hls.ErrorTypes.NETWORK_ERROR:
              hls.startLoad();
              break;
            case Hls.ErrorTypes.MEDIA_ERROR:
              hls.recoverMediaError();
              break;
            default:
              setError("Terjadi kesalahan fatal saat memutar video.");
              hls.destroy();
              break;
          }
        }
      });

      hlsRef.current = hls;
    } else if (video.canPlayType("application/vnd.apple.mpegurl")) {
      video.src = proxiedSrc;
      video.addEventListener("loadedmetadata", () => {
        setIsLoading(false);
        video.play().catch(() => {});
      });
    }
  };


  const togglePlay = () => {
    if (videoRef.current) {
      if (isPlaying) {
        videoRef.current.pause();
      } else {
        videoRef.current.play();
      }
      setIsPlaying(!isPlaying);
    }
  };

  const toggleMute = () => {
    if (videoRef.current) {
      videoRef.current.muted = !isMuted;
      setIsMuted(!isMuted);
    }
  };

  const toggleFullscreen = () => {
    if (videoRef.current) {
      if (document.fullscreenElement) {
        document.exitFullscreen();
      } else {
        videoRef.current.requestFullscreen();
      }
    }
  };

  const changeQuality = (index: number) => {
    if (allSources[index]) {
      const currentTime = videoRef.current?.currentTime ?? 0;
      const wasPlaying = isPlaying;
      setCurrentQuality(index);
      setShowSettings(false);
      setIsLoading(true);

      const src = allSources[index];
      initializePlayer(src.file, src.type);

      const video = videoRef.current;
      if (video) {
        const onLoaded = () => {
          video.currentTime = currentTime;
          if (wasPlaying) video.play().catch(() => {});
          video.removeEventListener("loadedmetadata", onLoaded);
        };
        video.addEventListener("loadedmetadata", onLoaded);
      }
    }
  };

  if (error) {
    return (
      <div className="w-full h-full bg-black/90 flex flex-col items-center justify-center relative overflow-hidden">
        <div className="absolute inset-0 bg-red-500/5 backdrop-blur-3xl" />
        <div className="relative z-10 flex flex-col items-center p-8 text-center max-w-md">
          <div className="w-16 h-16 rounded-full bg-red-500/10 flex items-center justify-center mb-4">
            <span className="text-red-500 text-3xl">!</span>
          </div>
          <h3 className="text-xl font-bold text-white mb-2">Konten Tidak Tersedia</h3>
          <p className="text-gray-400 text-sm">
            {error} <br />
            Silakan coba lagi nanti atau pilih resolusi lain.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="relative w-full h-full bg-black overflow-hidden group">
      {isLoading && (
        <div className="absolute inset-0 z-20 flex items-center justify-center bg-black/80 backdrop-blur-sm">
          <div className="flex flex-col items-center gap-3">
            <div className="w-10 h-10 border-4 border-accent border-t-transparent rounded-full animate-spin" />
            <p className="text-sm text-white/60">Memuat video...</p>
          </div>
        </div>
      )}

      <video
        ref={videoRef}
        className="w-full h-full object-contain"
        onClick={togglePlay}
        onPlay={() => setIsPlaying(true)}
        onPause={() => setIsPlaying(false)}
        crossOrigin="anonymous"
      >
        {captions.map((cap) => (
          <track
            key={cap.id}
            kind="captions"
            label={cap.language}
            srcLang={cap.language.substring(0, 2).toLowerCase()}
            src={cap.url}
            default={cap.language.toLowerCase() === "indonesian" || cap.language.toLowerCase() === "indonesia"}
          />
        ))}
      </video>

      <div className="absolute bottom-0 left-0 right-0 p-4 bg-gradient-to-t from-black/90 via-black/40 to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-300 z-10">
        <div className="flex items-center justify-between mt-4 gap-4">
          <div className="flex items-center gap-4">
            <button onClick={togglePlay} className="text-white hover:text-accent transition">
              {isPlaying ? <Pause size={24} /> : <Play size={24} />}
            </button>
            <button onClick={toggleMute} className="text-white hover:text-accent transition">
              {isMuted ? <VolumeX size={20} /> : <Volume2 size={20} />}
            </button>
          </div>

          <div className="flex items-center gap-4 relative">
            {qualityLevels.length > 1 && (
              <div className="relative">
                <button
                  onClick={() => setShowSettings(!showSettings)}
                  className="text-white hover:text-accent transition flex items-center gap-1"
                >
                  <Settings size={20} />
                  <span className="text-xs font-medium">{qualityLevels[currentQuality]?.label}</span>
                </button>
                
                {showSettings && (
                  <div className="absolute bottom-full right-0 mb-4 bg-zinc-900 border border-zinc-800 rounded-lg p-2 min-w-[120px] shadow-xl">
                    {qualityLevels.map((level, index) => (
                      <button
                        key={index}
                        onClick={() => changeQuality(index)}
                        className={cn(
                          "w-full text-left px-3 py-2 text-sm rounded-md hover:bg-zinc-800 transition",
                          currentQuality === index ? "text-accent font-semibold" : "text-gray-300"
                        )}
                      >
                        {level.label}
                      </button>
                    ))}
                  </div>
                )}
              </div>
            )}
            <button onClick={toggleFullscreen} className="text-white hover:text-accent transition">
              <Maximize size={20} />
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
