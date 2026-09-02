import AuthForm from "@/components/AuthForm";
import { getTrendingGlobal } from "@/lib/tmdb";

export default async function DaftarPage() {
  const trending = await getTrendingGlobal();
  return <AuthForm mode="register" trending={trending} />;
}
