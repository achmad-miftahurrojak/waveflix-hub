import UserLibrary from "@/components/UserLibrary";

export default function DaftarSayaPage() {
  return (
    <main className="min-h-screen pb-16">
      <UserLibrary
        title="My List"
        endpoint="/api/watchlist"
        localFallbackKey="waveflix_mylist"
      />
    </main>
  );
}
