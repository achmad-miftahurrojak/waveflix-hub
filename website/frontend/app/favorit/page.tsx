import UserLibrary from "@/components/UserLibrary";

export default function FavoritesPage() {
  return (
    <UserLibrary
      title="Favorites"
      endpoint="/api/favorites"
      localFallbackKey="waveflix_favorites"
    />
  );
}
