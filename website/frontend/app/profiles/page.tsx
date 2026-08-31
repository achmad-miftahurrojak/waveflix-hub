"use client";

import { useAuth } from "@/components/AuthProvider";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import Image from "next/image";

interface Profile {
  id: number;
  name: string;
  avatar: string;
}

export default function ProfilesPage() {
  const { user, authFetch, setActiveProfile, ready } = useAuth();
  const router = useRouter();

  const [profiles, setProfiles] = useState<Profile[]>([]);
  const [loading, setLoading] = useState(true);
  const [isManaging, setIsManaging] = useState(false);
  const [showAddModal, setShowAddModal] = useState(false);
  const [showEditModal, setShowEditModal] = useState(false);
  const [saveError, setSaveError] = useState("");

  const [newProfileName, setNewProfileName] = useState("");
  const [editingProfile, setEditingProfile] = useState<Profile | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (!ready) return;
    if (!user) {
      router.push("/masuk");
      return;
    }

    fetchProfiles();
  }, [ready, user, router]);

  const fetchProfiles = async () => {
    setLoading(true);
    try {
      const res = await authFetch("/api/profiles");
      if (res.ok) {
        const data = await res.json();
        setProfiles(Array.isArray(data) ? data : []);
      }
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  const handleSelectProfile = (profile: Profile) => {
    if (isManaging) {
      setEditingProfile(profile);
      setNewProfileName(profile.name);
      setShowEditModal(true);
    } else {
      setActiveProfile(profile);
      sessionStorage.setItem("profileSelected", "true");
      router.push("/home");
    }
  };

  const handleAddProfile = async () => {
    if (!newProfileName.trim() || saving) return;
    setSaveError("");
    setSaving(true);

    try {
      const res = await authFetch("/api/profiles", {
        method: "POST",
        body: JSON.stringify({ name: newProfileName, avatar: "" })
      });

      if (res.ok) {
        setShowAddModal(false);
        setNewProfileName("");
        fetchProfiles();
      } else {
        const data = await res.json().catch(() => ({}));
        setSaveError(data.error || "Gagal menyimpan profil. Coba lagi.");
      }
    } catch (err) {
      console.error(err);
      setSaveError("Tidak bisa terhubung ke server.");
    } finally {
      setSaving(false);
    }
  };

  const handleUpdateProfile = async () => {
    if (!editingProfile || !newProfileName.trim() || saving) return;
    setSaving(true);

    try {
      const res = await authFetch(`/api/profiles/${editingProfile.id}`, {
        method: "PUT",
        body: JSON.stringify({ name: newProfileName, avatar: editingProfile.avatar })
      });

      if (res.ok) {
        setShowEditModal(false);
        setEditingProfile(null);
        setNewProfileName("");
        fetchProfiles();
      } else {
        setSaveError("Gagal menyimpan profil.");
      }
    } catch (err) {
      console.error(err);
      setSaveError("Tidak bisa terhubung ke server.");
    } finally {
      setSaving(false);
    }
  };

  const handleDeleteProfile = async () => {
    if (!editingProfile || saving) return;

    if (!confirm("Yakin ingin menghapus profil ini? Semua data profil akan ikut terhapus.")) return;
    setSaving(true);

    try {
      const res = await authFetch(`/api/profiles/${editingProfile.id}`, {
        method: "DELETE"
      });

      if (res.ok) {
        setShowEditModal(false);
        setEditingProfile(null);
        fetchProfiles();
      } else {
        setSaveError("Gagal menghapus profil.");
      }
    } catch (err) {
      console.error(err);
      setSaveError("Tidak bisa terhubung ke server.");
    } finally {
      setSaving(false);
    }
  };

  if (!ready || loading) {
    return (
      <div className="min-h-screen bg-bg flex items-center justify-center text-white">
        <div className="w-12 h-12 border-4 border-white/20 border-t-white rounded-full animate-spin"></div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-bg flex flex-col items-center justify-center text-white p-4">
      <h1 className="text-4xl md:text-5xl font-semibold mb-12">
        {isManaging ? "Kelola Profil" : "Siapa yang menonton?"}
      </h1>

      <div className="flex flex-wrap justify-center gap-6">
        {profiles.map((profile) => (
          <button 
            key={profile.id}
            onClick={() => handleSelectProfile(profile)}
            className="group flex flex-col items-center cursor-pointer transition-transform hover:scale-105 focus-visible:scale-105 relative"
          >
            <div className="w-32 h-32 md:w-40 md:h-40 rounded-md overflow-hidden border-2 border-transparent group-hover:border-white transition-colors duration-200 bg-surface-overlay flex items-center justify-center relative">
              {profile.avatar ? (

                <img
                  src={profile.avatar}
                  alt={profile.name}
                  className={`w-full h-full object-cover ${isManaging ? "opacity-50" : ""}`}
                />
              ) : (
                <span className={`text-4xl font-bold text-white/60 group-hover:text-white ${isManaging ? "opacity-50" : ""}`}>
                  {profile.name.charAt(0).toUpperCase()}
                </span>
              )}

              {isManaging && (
                <div className="absolute inset-0 flex items-center justify-center">
                  <svg xmlns="http://www.w3.org/2000/svg" className="h-10 w-10 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" />
                  </svg>
                </div>
              )}
            </div>
            <span className="mt-4 text-white/60 group-hover:text-white transition-colors text-lg md:text-xl">
              {profile.name}
            </span>
          </button>
        ))}

        {profiles.length < 4 && (
          <button 
            onClick={() => setShowAddModal(true)}
            className="group flex flex-col items-center cursor-pointer transition-transform hover:scale-105 focus-visible:scale-105 opacity-60 hover:opacity-100"
          >
            <div className="w-32 h-32 md:w-40 md:h-40 rounded-md border-2 border-transparent group-hover:border-white flex items-center justify-center bg-surface-overlay transition-colors duration-200">
              <svg xmlns="http://www.w3.org/2000/svg" className="h-16 w-16 text-white/60 group-hover:text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
              </svg>
            </div>
            <span className="mt-4 text-white/60 group-hover:text-white transition-colors text-lg md:text-xl">
              Tambah Profil
            </span>
          </button>
        )}
      </div>

      <button 
        onClick={() => setIsManaging(!isManaging)}
        className="mt-16 px-6 py-2 border border-white/30 text-white/60 hover:text-white hover:border-white uppercase tracking-widest text-sm transition-colors"
      >
        {isManaging ? "Selesai" : "Kelola Profil"}
      </button>

      {}
      {showAddModal && (
        <div className="fixed inset-0 bg-black/80 flex items-center justify-center z-50 p-4">
          <div className="bg-surface-raised border border-white/10 rounded-lg p-6 w-full max-w-md">
            <h2 className="text-2xl font-bold mb-4">Tambah Profil</h2>
            <div className="mb-4">
              <label className="block text-sm font-semibold text-white/60 mb-1">Nama Profil</label>
              <input 
                type="text" 
                value={newProfileName}
                onChange={(e) => setNewProfileName(e.target.value)}
                className="w-full bg-surface-overlay border border-white/10 rounded p-2 text-white outline-none focus:border-accent"
                placeholder="Nama"
                autoFocus
              />
            </div>
            {saveError && (
              <p className="text-red-400 text-sm mt-2">{saveError}</p>
            )}
            <div className="flex justify-end gap-3 mt-4">
              <button 
                onClick={() => {
                  setShowAddModal(false);
                  setNewProfileName("");
                  setSaveError("");
                }}
                className="px-4 py-2 border border-white/20 rounded text-white/75 hover:text-white hover:border-white"
              >
                Batal
              </button>
              <button 
                onClick={handleAddProfile}
                disabled={!newProfileName.trim() || saving}
                className="px-4 py-2 bg-accent text-black rounded hover:bg-accent-dark disabled:cursor-not-allowed disabled:opacity-50"
              >
                {saving ? "Menyimpan…" : "Simpan"}
              </button>
            </div>
          </div>
        </div>
      )}

      {}
      {showEditModal && editingProfile && (
        <div className="fixed inset-0 bg-black/80 flex items-center justify-center z-50 p-4">
          <div className="bg-surface-raised border border-white/10 rounded-lg p-6 w-full max-w-md">
            <h2 className="text-2xl font-bold mb-4">Edit Profil</h2>
            <div className="mb-4">
              <label className="block text-sm font-semibold text-white/60 mb-1">Nama Profil</label>
              <input 
                type="text" 
                value={newProfileName}
                onChange={(e) => setNewProfileName(e.target.value)}
                className="w-full bg-surface-overlay border border-white/10 rounded p-2 text-white outline-none focus:border-accent"
                placeholder="Nama"
                autoFocus
              />
            </div>
            <div className="flex justify-between mt-6">
              <button 
                onClick={handleDeleteProfile}
                disabled={saving}
                className="px-4 py-2 border border-red-600 text-red-500 rounded hover:bg-red-600/10 disabled:cursor-not-allowed disabled:opacity-50"
              >
                Hapus
              </button>
              <div className="flex gap-3">
                <button 
                  onClick={() => {
                    setShowEditModal(false);
                    setEditingProfile(null);
                    setNewProfileName("");
                  }}
                  className="px-4 py-2 border border-white/20 rounded text-white/75 hover:text-white hover:border-white"
                >
                  Batal
                </button>
                <button 
                  onClick={handleUpdateProfile}
                  disabled={!newProfileName.trim() || newProfileName === editingProfile.name || saving}
                  className="px-4 py-2 bg-accent text-black rounded hover:bg-accent-dark disabled:cursor-not-allowed disabled:opacity-50"
                >
                  {saving ? "Menyimpan…" : "Simpan"}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
