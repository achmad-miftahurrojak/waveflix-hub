import UserLibrary from "@/components/UserLibrary";

export default function RiwayatPage() {
  return (
    <main className="min-h-screen pb-16">
      <UserLibrary title="Riwayat Tontonan" endpoint="/api/history" />
    </main>
  );
}
