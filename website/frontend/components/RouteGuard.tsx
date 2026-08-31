"use client";

import { useEffect, useState, type ReactNode } from "react";
import { usePathname, useRouter } from "next/navigation";
import { useAuth } from "@/components/AuthProvider";

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

      if (!isPublic) {

        router.replace("/");
      }
      setProfileChecked(true);
      return;
    }

    if (isPublic) {

      router.replace("/profiles");
      setProfileChecked(true);
      return;
    }

    const profileSelected = sessionStorage.getItem("profileSelected") === "true";
    setHasProfileSelected(profileSelected);
    setProfileChecked(true);

    if (!profileSelected && !isProfilesPage) {

      router.replace("/profiles");
    }

  }, [ready, token, isPublic, currentPath, isProfilesPage, router]);

  if (!ready || !profileChecked) {
    return <div className="min-h-screen bg-bg" />;
  }

  if (!token && !isPublic) {

    return <div className="min-h-screen bg-bg" />;
  }
  if (token && isPublic) {

    return <div className="min-h-screen bg-bg" />;
  }
  if (token && !isPublic && !hasProfileSelected && !isProfilesPage) {

    return <div className="min-h-screen bg-bg" />;
  }

  return <>{children}</>;
}
