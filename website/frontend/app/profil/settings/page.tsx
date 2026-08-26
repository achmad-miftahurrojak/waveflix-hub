"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { useAuth } from "@/components/AuthProvider";

interface ProfileData {
  id: number;
  name: string;
  avatar: string;
  banner: string;
  bio: string;
}

export default function ProfileSettingsPage() {
  const { user, ready, authFetch, activeProfile, setActiveProfile } = useAuth();
  const router = useRouter();

  const [profile, setProfile] = useState<ProfileData | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);
  const [error, setError] = useState("");

  const [name, setName] = useState("");
  const [avatar, setAvatar] = useState("");
  const [banner, setBanner] = useState("");
  const [bio, setBio] = useState("");

  const profileId = (activeProfile as any)?.id;

  useEffect(() => {
    if (!ready) return;
    if (!user) { router.replace("/masuk"); return; }
    if (!profileId) { router.replace("/profiles"); return; }

    authFetch(`/api/profiles/${profileId}`)
      .then((r) => r.json())
      .then((data) => {
        setProfile(data);
        setName(data.name || "");
        setAvatar(data.avatar || "");
        setBanner(data.banner || "");
        setBio(data.bio || "");
      })
      .catch(() => setError("Gagal memuat data profil"))
      .finally(() => setLoading(false));
  }, [ready, user, profileId, authFetch, router]);

  const handleSave = async () => {
    if (!profileId) return;
    setSaving(true);
    setError("");
    setSaved(false);

    try {
      const res = await authFetch(`/api/profiles/${profileId}`, {
        method: "PUT",
        body: JSON.stringify({ name, avatar, banner, bio }),
      });

      if (res.ok) {
        const updated = await res.json();
        setProfile(updated);
        // Update active profile in context so navbar reflects changes immediately
        setActiveProfile({ ...(activeProfile as any), ...updated });
        setSaved(true);
        setTimeout(() => setSaved(false), 3000);
      } else {
        const d = await res.json().catch(() => ({}));
        setError(d.error || "Gagal menyimpan");
      }
    } catch {
      setError("Tidak bisa terhubung ke server");
    } finally {
      setSaving(false);
    }
  };

  if (!ready || loading) {
    return (
      <div className="min-h-screen bg-bg flex items-center justify-center">
        <div className="w-12 h-12 border-4 border-white/20 border-t-white rounded-full animate-spin" />
      </div>
    );
  }

  return (
    <main className="min-h-screen bg-bg text-white pt-24 pb-16 px-4">
      <div className="max-w-2xl mx-auto">
        {/* Header */}
        <div className="flex items-center gap-4 mb-10">
          <button
            onClick={() => router.back()}
            className="text-white/50 hover:text-white transition"
          >
            <svg xmlns="http://www.w3.org/2000/svg" className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
            </svg>
          </button>
          <h1 className="text-2xl font-bold">Pengaturan Profil</h1>
        </div>

        {/* Banner Preview */}
        <div className="relative w-full h-40 rounded-xl overflow-hidden mb-6 bg-surface-overlay">
          {banner ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img src={banner} alt="Banner" className="w-full h-full object-cover" />
          ) : (
            <div className="w-full h-full bg-gradient-to-br from-accent/30 via-surface-overlay to-surface-raised" />
          )}

          {/* Avatar overlay */}
          <div className="absolute bottom-0 left-6 translate-y-1/2">
            <div className="w-20 h-20 rounded-full border-4 border-bg bg-surface overflow-hidden">
              {avatar ? (
                // eslint-disable-next-line @next/next/no-img-element
                <img src={avatar} alt="Avatar" className="w-full h-full object-cover" />
              ) : (
                <span className="w-full h-full flex items-center justify-center text-2xl font-bold text-white/60">
                  {name.charAt(0).toUpperCase()}
                </span>
              )}
            </div>
          </div>
        </div>

        {/* Form */}
        <div className="mt-12 space-y-5">
          {/* Nama Profil */}
          <div>
            <label className="block text-sm font-semibold text-white/60 mb-1">Nama Profil</label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="w-full bg-surface-overlay border border-white/10 rounded-lg px-4 py-2.5 text-white outline-none focus:border-accent transition"
              placeholder="Nama profil"
            />
          </div>

          {/* Avatar URL */}
          <div>
            <label className="block text-sm font-semibold text-white/60 mb-1">
              URL Foto Profil
            </label>
            <input
              type="url"
              value={avatar}
              onChange={(e) => setAvatar(e.target.value)}
              className="w-full bg-surface-overlay border border-white/10 rounded-lg px-4 py-2.5 text-white outline-none focus:border-accent transition"
              placeholder="https://example.com/avatar.jpg"
            />
            <p className="mt-1 text-xs text-white/40">Masukkan URL gambar untuk foto profil</p>
          </div>

          {/* Banner URL */}
          <div>
            <label className="block text-sm font-semibold text-white/60 mb-1">
              URL Banner
            </label>
            <input
              type="url"
              value={banner}
              onChange={(e) => setBanner(e.target.value)}
              className="w-full bg-surface-overlay border border-white/10 rounded-lg px-4 py-2.5 text-white outline-none focus:border-accent transition"
              placeholder="https://example.com/banner.jpg"
            />
            <p className="mt-1 text-xs text-white/40">Gambar lebar untuk banner profil (16:9 atau lebih lebar)</p>
          </div>

          {/* Bio */}
          <div>
            <label className="block text-sm font-semibold text-white/60 mb-1">Bio</label>
            <textarea
              value={bio}
              onChange={(e) => setBio(e.target.value)}
              rows={3}
              className="w-full bg-surface-overlay border border-white/10 rounded-lg px-4 py-2.5 text-white outline-none focus:border-accent transition resize-none"
              placeholder="Tulis sesuatu tentang dirimu..."
              maxLength={200}
            />
            <p className="text-right text-xs text-white/50">{bio.length}/200</p>
          </div>

          {/* Error / Success */}
          {error && (
            <p className="text-red-400 text-sm">{error}</p>
          )}
          {saved && (
            <p className="text-green-400 text-sm">✓ Perubahan berhasil disimpan</p>
          )}

          {/* Actions */}
          <div className="flex justify-end gap-3 pt-2">
            <button
              onClick={() => router.back()}
              className="px-5 py-2.5 rounded-lg border border-white/20 text-white/75 hover:border-white hover:text-white transition"
            >
              Batal
            </button>
            <button
              onClick={handleSave}
              disabled={saving || !name.trim()}
              className="px-6 py-2.5 rounded-lg bg-accent text-black font-semibold hover:bg-accent/80 disabled:opacity-50 transition"
            >
              {saving ? "Menyimpan..." : "Simpan"}
            </button>
          </div>
        </div>
      </div>
    </main>
  );
}
