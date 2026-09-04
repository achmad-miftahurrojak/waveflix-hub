import AuthForm from "@/components/AuthForm";
import { getTrendingGlobal } from "@/lib/tmdb";

export default async function MasukPage() {
  const trending = await getTrendingGlobal();
  return <AuthForm mode="login" trending={trending} />;
}
