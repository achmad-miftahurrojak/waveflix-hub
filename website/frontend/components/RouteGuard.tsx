"use client";

import { useEffect, useState, type ReactNode } from "react";
import { usePathname, useRouter } from "next/navigation";
import { useAuth } from "@/components/AuthProvider";

// Rute yang bisa diakses tanpa login
const PUBLIC_ROUTES = ["/", "/masuk", "/daftar"];

export default function RouteGuard({ children }: { children: ReactNode }) {
  const { ready, token } = useAuth();
  const pathname = usePathname();
  const router = useRouter();
  const [profileChecked, setProfileChecked] = useState(false);
  const [hasProfileSelected, setHasProfileSelected] = useState(false);

  const currentPath = pathname || "/";
  const isPublic = PUBLIC_ROUTES.includes(currentPath);
  const isProfilesPage = currentPath === "/profiles";

  useEffect(() => {
    if (!ready) return;

    if (!token) {
      // Belum login
      if (!isPublic) {
        // Coba akses halaman private → lempar ke landing page
        router.replace("/");
      }
      setProfileChecked(true);
      return;
    }

    // Sudah login
    if (isPublic) {
      // Login tapi buka /, /masuk, /daftar → lempar ke /profiles
      router.replace("/profiles");
      setProfileChecked(true);
      return;
    }

    // Sudah login, halaman private
    const profileSelected = sessionStorage.getItem("profileSelected") === "true";
    setHasProfileSelected(profileSelected);
    setProfileChecked(true);

    if (!profileSelected && !isProfilesPage) {
      // Belum pilih profil dan bukan di /profiles → lempar ke /profiles
      router.replace("/profiles");
    }
    // Jika di /profiles → SELALU boleh (baik untuk pilih maupun ganti profil)
    // Jika profileSelected && !isProfilesPage → BOLEH navigasi normal
  }, [ready, token, isPublic, currentPath, isProfilesPage, router]);

  // Tampilkan layar kosong saat sedang load atau belum selesai cek
  if (!ready || !profileChecked) {
    return <div className="min-h-screen bg-bg" />;
  }

  // Blok render saat sedang proses redirect
  if (!token && !isPublic) {
    // Belum login di halaman private → redirect ke / sedang terjadi
    return <div className="min-h-screen bg-bg" />;
  }
  if (token && isPublic) {
    // Sudah login di halaman public → redirect ke /profiles sedang terjadi
    return <div className="min-h-screen bg-bg" />;
  }
  if (token && !isPublic && !hasProfileSelected && !isProfilesPage) {
    // Sudah login tapi belum pilih profil dan bukan di /profiles → redirect sedang terjadi
    return <div className="min-h-screen bg-bg" />;
  }

  return <>{children}</>;
}
