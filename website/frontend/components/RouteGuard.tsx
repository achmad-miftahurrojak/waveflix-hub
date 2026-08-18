"use client";

import { useEffect, type ReactNode } from "react";
import { usePathname, useRouter } from "next/navigation";
import { useAuth } from "@/components/AuthProvider";

// Rute yang bebas diakses tanpa login
const PUBLIC_ROUTES = ["/", "/masuk", "/daftar"];

export default function RouteGuard({ children }: { children: ReactNode }) {
  const { ready, token } = useAuth();
  const pathname = usePathname();
  const router = useRouter();
  
  // Karena pathname bisa undefined di awal (meski jarang), fallback ke "/"
  const currentPath = pathname || "/";
  const isPublic = PUBLIC_ROUTES.includes(currentPath);

  useEffect(() => {
    if (!ready) return;
    
    if (!token && !isPublic) {
      // Belum login tapi akses halaman private -> lempar ke login
      router.replace("/masuk");
    } else if (token && isPublic) {
      // Sudah login tapi akses halaman public -> lempar ke /home
      router.replace("/home");
    }
  }, [ready, token, isPublic, router]);

  // Saat belum ready, tampilkan layar kosong supaya tidak flickering (kedap-kedip) konten private
  if (!ready) {
    return <div className="min-h-screen bg-bg" />;
  }

  // Jika perlu diredirect, tunggu sebentar jangan render children agar tidak bocor
  if (!token && !isPublic) return <div className="min-h-screen bg-bg" />;
  if (token && isPublic) return <div className="min-h-screen bg-bg" />;

  return <>{children}</>;
}
