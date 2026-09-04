import UserLibrary from "@/components/UserLibrary";

export default function FavoritesPage() {
  return (
    <main className="min-h-screen pb-16">
      <UserLibrary
        title="Favorites"
        endpoint="/api/favorites"
        localFallbackKey="waveflix_favorites"
      />
    </main>
  );
}
