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

export function NativePlayer({ mediaType, tmdbId, season, episode }: NativePlayerProps) {
  const videoRef = useRef<HTMLVideoElement>(null);
  const [isPlaying, setIsPlaying] = useState(false);
  const [isMuted, setIsMuted] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [qualityLevels, setQualityLevels] = useState<any[]>([]);
  const [currentQuality, setCurrentQuality] = useState<number>(-1); // -1 is Auto
  const [showSettings, setShowSettings] = useState(false);
  const [captions, setCaptions] = useState<{id: string; language: string; url: string}[]>([]);

  const hlsRef = useRef<Hls | null>(null);

  useEffect(() => {
    let query = `?media=${mediaType}&id=${tmdbId}`;
    if (mediaType === "tv" && season && episode) {
      query += `&s=${season}&e=${episode}`;
    }

    // Call our local bridge API which fetches from the python decryptor
    const bridgeUrl = `http://localhost:8080/api/stream${query}`;

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
        let streamUrl = "";
        if (data?.stream?.qualities) {
          if (data.stream.qualities.auto) {
            streamUrl = data.stream.qualities.auto.url;
          } else if (data.stream.qualities["1080"]) {
            streamUrl = data.stream.qualities["1080"].url;
          } else if (data.stream.qualities["720"]) {
            streamUrl = data.stream.qualities["720"].url;
          } else if (data.stream.qualities["480"]) {
            streamUrl = data.stream.qualities["480"].url;
          } else {
            const firstKey = Object.keys(data.stream.qualities)[0];
            streamUrl = data.stream.qualities[firstKey]?.url || "";
          }
        }

        if (data?.captions) {
          setCaptions(data.captions);
        }

        if (!streamUrl) {
          throw new Error("Link stream tidak valid dari provider.");
        }
        initializePlayer(streamUrl);
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

  const initializePlayer = (src: string) => {
    const video = videoRef.current;
    if (!video) return;

    if (Hls.isSupported()) {
      const hls = new Hls({
        maxBufferLength: 30,
        maxMaxBufferLength: 60,
      });

      hls.loadSource(src);
      hls.attachMedia(video);

      hls.on(Hls.Events.MANIFEST_PARSED, (event, data) => {
        setQualityLevels(data.levels);
        setIsLoading(false);
      });

      hls.on(Hls.Events.ERROR, (event, data) => {
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
      // For Safari which has native HLS support
      video.src = src;
      video.addEventListener("loadedmetadata", () => {
        setIsLoading(false);
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

  const changeQuality = (levelIndex: number) => {
    if (hlsRef.current) {
      hlsRef.current.currentLevel = levelIndex;
      setCurrentQuality(levelIndex);
      setShowSettings(false);
    }
  };

  if (error) {
    return (
      <div className="w-full aspect-video bg-black/90 flex flex-col items-center justify-center rounded-xl border border-red-500/20 shadow-lg relative overflow-hidden">
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
    <div className="relative w-full aspect-video bg-black rounded-xl overflow-hidden group">
      {isLoading && (
        <div className="absolute inset-0 z-20 flex items-center justify-center bg-black/80 backdrop-blur-sm">
          <div className="w-10 h-10 border-4 border-red-500 border-t-transparent rounded-full animate-spin" />
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

      {/* Custom Controls Overlay */}
      <div className="absolute bottom-0 left-0 right-0 p-4 bg-gradient-to-t from-black/90 via-black/40 to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-300 z-10">
        <div className="flex items-center justify-between mt-4 gap-4">
          <div className="flex items-center gap-4">
            <button onClick={togglePlay} className="text-white hover:text-red-500 transition">
              {isPlaying ? <Pause size={24} /> : <Play size={24} />}
            </button>
            <button onClick={toggleMute} className="text-white hover:text-red-500 transition">
              {isMuted ? <VolumeX size={20} /> : <Volume2 size={20} />}
            </button>
          </div>

          <div className="flex items-center gap-4 relative">
            {qualityLevels.length > 0 && (
              <div className="relative">
                <button
                  onClick={() => setShowSettings(!showSettings)}
                  className="text-white hover:text-red-500 transition"
                >
                  <Settings size={20} />
                </button>
                
                {showSettings && (
                  <div className="absolute bottom-full right-0 mb-4 bg-zinc-900 border border-zinc-800 rounded-lg p-2 min-w-[120px] shadow-xl">
                    <button
                      onClick={() => changeQuality(-1)}
                      className={cn(
                        "w-full text-left px-3 py-2 text-sm rounded-md hover:bg-zinc-800 transition",
                        currentQuality === -1 ? "text-red-500 font-semibold" : "text-gray-300"
                      )}
                    >
                      Auto
                    </button>
                    {qualityLevels.map((level, index) => (
                      <button
                        key={index}
                        onClick={() => changeQuality(index)}
                        className={cn(
                          "w-full text-left px-3 py-2 text-sm rounded-md hover:bg-zinc-800 transition",
                          currentQuality === index ? "text-red-500 font-semibold" : "text-gray-300"
                        )}
                      >
                        {level.height}p
                      </button>
                    ))}
                  </div>
                )}
              </div>
            )}
            <button onClick={toggleFullscreen} className="text-white hover:text-red-500 transition">
              <Maximize size={20} />
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
