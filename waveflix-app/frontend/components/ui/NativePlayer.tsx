"use client";

import { useEffect, useRef, useState } from "react";
import Hls from "hls.js";
import { Play, Pause, Volume2, VolumeX, Maximize, Settings, RotateCcw, RotateCw, Subtitles } from "lucide-react";
import { cn } from "@/lib/utils";

interface NativePlayerProps {
  mediaType: "movie" | "tv";
  tmdbId: string;
  season?: string;
  episode?: string;
  title?: string;
}

const BACKEND_URL = process.env.NEXT_PUBLIC_BACKEND_URL || "http://localhost:8081";

function proxyUrl(url: string): string {
  // HLS M3U8 streams have IP-bound tokens — load directly without proxy
  if (url.includes(".m3u8")) return url;
  // Direct stream for Torrents
  if (url.includes("localhost:8000")) return url;
  const encoded = btoa(url).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
  return `${BACKEND_URL}/api/media-proxy?url=${encoded}`;
}

function formatTime(seconds: number): string {
  if (!seconds || isNaN(seconds)) return "00:00";
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const s = Math.floor(seconds % 60);
  if (h > 0) {
    return `${h}:${m.toString().padStart(2, '0')}:${s.toString().padStart(2, '0')}`;
  }
  return `${m.toString().padStart(2, '0')}:${s.toString().padStart(2, '0')}`;
}

export function NativePlayer({ mediaType, tmdbId, season, episode, title }: NativePlayerProps) {
  const videoRef = useRef<HTMLVideoElement>(null);
  const [isPlaying, setIsPlaying] = useState(false);
  const [isMuted, setIsMuted] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [qualityLevels, setQualityLevels] = useState<{file?: string; type: string; label: string; index?: number}[]>([]);
  const [currentQuality, setCurrentQuality] = useState(0);
  const [showSettings, setShowSettings] = useState(false);
  const [captions, setCaptions] = useState<{id: string; language: string; url: string}[]>([]);
  const [showSubSettings, setShowSubSettings] = useState(false);
  const [currentSubtitle, setCurrentSubtitle] = useState<number>(-1);
  const [allSources, setAllSources] = useState<{file: string; type: string; label: string}[]>([]);
  
  // Seekbar state
  const [currentTime, setCurrentTime] = useState(0);
  const [duration, setDuration] = useState(0);
  const [isDragging, setIsDragging] = useState(false);

  const hlsRef = useRef<Hls | null>(null);

  useEffect(() => {
    let query = `?media=${mediaType}&id=${tmdbId}`;
    if (mediaType === "tv" && season && episode) {
      query += `&s=${season}&e=${episode}`;
    }

    const bridgeUrl = `${BACKEND_URL}/api/stream${query}`;
    let cancelled = false;

    const fetchStream = async (attempt = 1): Promise<void> => {
      setIsLoading(true);
      setError(null);

      try {
        const res = await fetch(bridgeUrl);
        if (cancelled) return;

        if (!res.ok) throw new Error("Video belum tersedia atau provider offline.");

        const data = await res.json();
        if (cancelled) return;

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
          const fetchedCaps = data.tracks.map((t: any, i: number) => ({
            id: String(i),
            language: t.label,
            url: proxyUrl(t.file)
          }));
          setCaptions(fetchedCaps);
          const defaultIdx = fetchedCaps.findIndex((c: any) => c.language.toLowerCase().includes("indonesia"));
          setCurrentSubtitle(defaultIdx >= 0 ? defaultIdx : -1);
        } else if (data?.captions) {
          setCaptions(data.captions);
          const defaultIdx = data.captions.findIndex((c: any) => c.language.toLowerCase().includes("indonesia"));
          setCurrentSubtitle(defaultIdx >= 0 ? defaultIdx : -1);
        }

        if (!sources.length) throw new Error("Link stream tidak valid dari provider.");

        setAllSources(sources);
        if (!sources[0].file.includes(".m3u8")) {
          setQualityLevels(sources);
        }
        setCurrentQuality(0);
        initializePlayer(sources[0].file, sources[0].type);
      } catch (err: any) {
        if (cancelled) return;
        if (attempt < 3) {
          await new Promise(r => setTimeout(r, 3000));
          if (!cancelled) return fetchStream(attempt + 1);
        }
        setError(err.message);
        setIsLoading(false);
      }
    };

    fetchStream();

    return () => {
      cancelled = true;
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

    const isHls = src.includes(".m3u8") || type === "hls";
    const finalSrc = isHls ? src : proxyUrl(src);

    if (!isHls) {
      video.src = finalSrc;
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
        xhrSetup: (xhr) => {
          xhr.withCredentials = false;
        },
      });

      let networkRetries = 0;

      // 30s timeout failsafe
      const loadTimeout = setTimeout(() => {
        setError("Stream timeout. Silakan reload.");
        setIsLoading(false);
        hls.destroy();
      }, 30000);

      hls.loadSource(finalSrc);
      hls.attachMedia(video);

      hls.on(Hls.Events.MANIFEST_PARSED, (_event, data) => {
        clearTimeout(loadTimeout);
        
        if (data.levels && data.levels.length > 0) {
          const levels = data.levels.map((l: any, i: number) => ({
            type: "hls",
            label: l.height ? `${l.height}p` : `Level ${i}`,
            index: i
          })).sort((a, b) => parseInt(b.label) - parseInt(a.label));
          
          levels.unshift({ type: "hls", label: "Auto", index: -1 });
          setQualityLevels(levels);
          setCurrentQuality(0);
          hls.currentLevel = -1;
        }
        
        setIsLoading(false);
        video.play().catch(() => {});
      });

      hls.on(Hls.Events.ERROR, (_event, data) => {
        if (data.fatal) {
          switch (data.type) {
            case Hls.ErrorTypes.NETWORK_ERROR:
              networkRetries++;
              if (networkRetries <= 3) {
                hls.startLoad();
              } else {
                clearTimeout(loadTimeout);
                setError("Gagal memuat stream. Silakan reload.");
                setIsLoading(false);
                hls.destroy();
              }
              break;
            case Hls.ErrorTypes.MEDIA_ERROR:
              hls.recoverMediaError();
              break;
            default:
              clearTimeout(loadTimeout);
              setError("Terjadi kesalahan fatal saat memutar video.");
              setIsLoading(false);
              hls.destroy();
              break;
          }
        }
      });

      hlsRef.current = hls;
    } else if (video.canPlayType("application/vnd.apple.mpegurl")) {
      // Safari native HLS
      video.src = finalSrc;
      video.addEventListener("loadedmetadata", () => {
        setIsLoading(false);
        video.play().catch(() => {});
      });
    } else {
      setError("Browser tidak mendukung pemutaran HLS.");
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

  const skipBackward = () => {
    if (videoRef.current) {
      videoRef.current.currentTime -= 10;
    }
  };

  const skipForward = () => {
    if (videoRef.current) {
      videoRef.current.currentTime += 10;
    }
  };

  const toggleMute = () => {
    if (videoRef.current) {
      videoRef.current.muted = !isMuted;
      setIsMuted(!isMuted);
    }
  };

  const handleTimeUpdate = () => {
    if (!isDragging && videoRef.current) {
      setCurrentTime(videoRef.current.currentTime);
    }
  };

  const handleLoadedMetadata = () => {
    if (videoRef.current) {
      setDuration(videoRef.current.duration);
    }
  };

  const handleSeek = (e: React.ChangeEvent<HTMLInputElement>) => {
    const newTime = parseFloat(e.target.value);
    setCurrentTime(newTime);
    if (videoRef.current) {
      videoRef.current.currentTime = newTime;
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

  const changeSubtitle = (index: number) => {
    if (videoRef.current) {
      const tracks = videoRef.current.textTracks;
      for (let i = 0; i < tracks.length; i++) {
        tracks[i].mode = i === index ? "showing" : "disabled";
      }
    }
    setCurrentSubtitle(index);
    setShowSubSettings(false);
  };

  const handleSubtitleUpload = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      const url = URL.createObjectURL(file);
      const newIndex = captions.length;
      setCaptions(prev => [...prev, {
        id: `custom-${Date.now()}`,
        language: `Custom: ${file.name}`,
        url: url
      }]);
      // Wait for state to update and track to mount, then select it
      setTimeout(() => {
        changeSubtitle(newIndex);
        setShowSubSettings(true); // Keep open to show it's selected
      }, 100);
    }
  };

  const changeQuality = (idx: number) => {
    const q = qualityLevels[idx];
    if (!q) return;

    setCurrentQuality(idx);
    setShowSettings(false);

    if (hlsRef.current && q.index !== undefined) {
      hlsRef.current.currentLevel = q.index;
      return;
    }

    if (q.file && allSources[idx]) {
      const currentTime = videoRef.current?.currentTime ?? 0;
      const wasPlaying = isPlaying;
      setIsLoading(true);

      const src = allSources[idx];
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

      {/* Top Banner (NOW PLAYING) */}
      <div className="absolute top-0 left-0 right-0 p-6 bg-gradient-to-b from-black/80 to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-300 z-10 pointer-events-none">
        {title && (
          <div className="text-white">
            <p className="text-xs text-accent font-semibold tracking-wider uppercase mb-1">Now Playing</p>
            <h2 className="text-xl font-bold truncate">{title}</h2>
          </div>
        )}
      </div>

      <video
        ref={videoRef}
        className="w-full h-full object-contain"
        onClick={togglePlay}
        onPlay={() => setIsPlaying(true)}
        onPause={() => setIsPlaying(false)}
        onTimeUpdate={handleTimeUpdate}
        onLoadedMetadata={handleLoadedMetadata}
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
        
        {/* Seekbar */}
        <div className="flex items-center gap-3 w-full px-2 mb-2">
          <span className="text-xs text-white/80 font-medium font-mono min-w-[40px] text-right">
            {formatTime(currentTime)}
          </span>
          <div className="relative flex-1 group/seek h-5 flex items-center cursor-pointer">
            <input
              type="range"
              min="0"
              max={duration || 100}
              value={currentTime}
              onMouseDown={() => setIsDragging(true)}
              onMouseUp={() => setIsDragging(false)}
              onChange={handleSeek}
              className="absolute inset-0 w-full h-full opacity-0 cursor-pointer z-20"
            />
            {/* Custom track */}
            <div className="w-full h-1 bg-white/20 rounded-full overflow-hidden relative z-10 group-hover/seek:h-1.5 transition-all">
              <div 
                className="h-full bg-accent relative"
                style={{ width: `${(currentTime / (duration || 1)) * 100}%` }}
              >
                <div className="absolute right-0 top-1/2 -translate-y-1/2 w-3 h-3 bg-white rounded-full shadow-lg opacity-0 group-hover/seek:opacity-100 transition-opacity transform translate-x-1/2" />
              </div>
            </div>
          </div>
          <span className="text-xs text-white/80 font-medium font-mono min-w-[40px]">
            {formatTime(duration)}
          </span>
        </div>

        <div className="flex items-center justify-between mt-2 gap-4">
          <div className="flex items-center gap-5">
            <button onClick={togglePlay} className="text-white hover:text-accent transition">
              {isPlaying ? <Pause size={24} /> : <Play size={24} />}
            </button>
            <button onClick={skipBackward} className="text-white/80 hover:text-white transition" title="Mundur 10 detik">
              <RotateCcw size={20} />
            </button>
            <button onClick={skipForward} className="text-white/80 hover:text-white transition" title="Maju 10 detik">
              <RotateCw size={20} />
            </button>
            
            <div className="flex items-center gap-2 group/volume">
              <button onClick={toggleMute} className="text-white/80 hover:text-white transition">
                {isMuted ? <VolumeX size={20} /> : <Volume2 size={20} />}
              </button>
              <input 
                type="range" 
                min="0" 
                max="1" 
                step="0.05" 
                defaultValue="1"
                onChange={(e) => {
                  if (videoRef.current) {
                    videoRef.current.volume = parseFloat(e.target.value);
                    setIsMuted(e.target.value === "0");
                  }
                }}
                className="w-0 opacity-0 origin-left group-hover/volume:w-20 group-hover/volume:opacity-100 transition-all duration-300 accent-accent cursor-pointer"
              />
            </div>
          </div>

          <div className="flex items-center gap-4 relative">
            {captions.length > 0 && (
              <div className="relative">
                <button
                  onClick={() => {
                    setShowSubSettings(!showSubSettings);
                    setShowSettings(false);
                  }}
                  className={cn(
                    "text-white hover:text-accent transition flex items-center gap-1",
                    currentSubtitle !== -1 && "text-accent"
                  )}
                  title="Subtitles"
                >
                  <Subtitles size={20} />
                </button>
                
                {showSubSettings && (
                  <div className="absolute bottom-full right-0 mb-4 bg-zinc-900 border border-zinc-800 rounded-lg p-2 min-w-[120px] shadow-xl max-h-48 overflow-y-auto">
                    <button
                      onClick={() => changeSubtitle(-1)}
                      className={cn(
                        "w-full text-left px-3 py-2 text-sm rounded-md hover:bg-zinc-800 transition",
                        currentSubtitle === -1 ? "text-accent font-semibold" : "text-gray-300"
                      )}
                    >
                      Off
                    </button>
                    {captions.map((cap, index) => (
                      <button
                        key={cap.id}
                        onClick={() => changeSubtitle(index)}
                        className={cn(
                          "w-full text-left px-3 py-2 text-sm rounded-md hover:bg-zinc-800 transition",
                          currentSubtitle === index ? "text-accent font-semibold" : "text-gray-300"
                        )}
                      >
                        {cap.language}
                      </button>
                    ))}
                    
                    <div className="mt-2 pt-2 border-t border-zinc-800">
                      <label className="w-full block text-center px-3 py-2 text-xs rounded-md bg-zinc-800 hover:bg-zinc-700 text-white cursor-pointer transition">
                        Upload Subtitle
                        <input 
                          type="file" 
                          accept=".srt,.vtt" 
                          className="hidden" 
                          onChange={handleSubtitleUpload}
                        />
                      </label>
                    </div>
                  </div>
                )}
              </div>
            )}

            {qualityLevels.length > 1 && (
              <div className="relative">
                <button
                  onClick={() => {
                    setShowSettings(!showSettings);
                    setShowSubSettings(false);
                  }}
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
