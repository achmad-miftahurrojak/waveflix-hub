import UserLibrary from "@/components/UserLibrary";

export default function DaftarSayaPage() {
  return (
    <UserLibrary
      title="My List"
      endpoint="/api/watchlist"
      localFallbackKey="waveflix_mylist"
    />
  );
}
