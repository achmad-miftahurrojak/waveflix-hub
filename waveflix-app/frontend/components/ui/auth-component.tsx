"use client";

import React, { useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { Button } from './button';
import { Input } from './input';
import { cn } from '@/lib/utils';
import {
	AtSignIcon,
	ChevronLeftIcon,
	UserIcon,
	KeyRoundIcon,
	Loader2Icon,
    AlertCircleIcon,
    FilmIcon,
    EyeIcon,
    EyeOffIcon
} from 'lucide-react';
import Link from 'next/link';

export interface AuthComponentProps {
  initialMode?: "login" | "register";
  logo?: React.ReactNode;
  brandName?: string;
  trendingItems?: any[];
  onLogin: (email: string, pass: string) => Promise<string | null>;
  onRegisterSendCode: (email: string) => Promise<{ error?: string; test_code?: string }>;
  onRegisterVerify: (email: string, username: string, pass: string, code: string) => Promise<string | null>;
  onSuccess: () => void;
}

export const AuthComponent = ({
  initialMode = "register",
  logo = <FilmIcon className="size-6 text-primary" />,
  brandName = "Waveflix",
  trendingItems = [],
  onLogin,
  onRegisterSendCode,
  onRegisterVerify,
  onSuccess
}: AuthComponentProps) => {
  const [mode, setMode] = useState<"login" | "register">(initialMode);
  
  // Registration is 2 steps: form -> verification code
  const [step, setStep] = useState<"form" | "code">("form");

  const [email, setEmail] = useState("");
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [code, setCode] = useState("");
  const [showPassword, setShowPassword] = useState(false);

  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmitLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setLoading(true);
    const err = await onLogin(email, password);
    if (err) {
      setError(err);
      setLoading(false);
    } else {
      // Success is handled by AuthProvider (which redirects), but just in case:
      onSuccess();
    }
  };

  const handleSubmitRegister = async (e: React.FormEvent) => {
    e.preventDefault();
    if (password.length < 6) {
        setError("Password must be at least 6 characters.");
        return;
    }
    setError(null);
    setLoading(true);
    const res = await onRegisterSendCode(email);
    setLoading(false);
    if (res.error) {
      setError(res.error);
    } else {
      setStep("code");
    }
  };

  const handleSubmitVerify = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setLoading(true);
    const err = await onRegisterVerify(email, username, password, code);
    if (err) {
      setError(err);
      setLoading(false);
    } else {
      onSuccess();
    }
  };

  return (
		<main className="relative md:h-screen md:overflow-hidden lg:grid lg:grid-cols-2">
			<div className="bg-muted/60 relative hidden h-full flex-col border-r p-10 lg:flex overflow-hidden">
				<div className="from-background absolute inset-0 z-10 bg-gradient-to-t to-transparent" />
				<div className="z-10 flex items-center gap-2">
					{logo}
					<p className="text-xl font-semibold">{brandName}</p>
				</div>
				<TrendingMarquee items={trendingItems} />
			</div>
			<div className="relative flex min-h-screen flex-col justify-center p-4">
				<div
					aria-hidden
					className="absolute inset-0 isolate contain-strict -z-10 opacity-60"
				>
					<div className="bg-[radial-gradient(68.54%_68.72%_at_55.02%_31.46%,--theme(--color-foreground/.06)_0,hsla(0,0%,55%,.02)_50%,--theme(--color-foreground/.01)_80%)] absolute top-0 right-0 h-320 w-140 -translate-y-87.5 rounded-full" />
					<div className="bg-[radial-gradient(50%_50%_at_50%_50%,--theme(--color-foreground/.04)_0,--theme(--color-foreground/.01)_80%,transparent_100%)] absolute top-0 right-0 h-320 w-60 [translate:5%_-50%] rounded-full" />
					<div className="bg-[radial-gradient(50%_50%_at_50%_50%,--theme(--color-foreground/.04)_0,--theme(--color-foreground/.01)_80%,transparent_100%)] absolute top-0 right-0 h-320 w-60 -translate-y-87.5 rounded-full" />
				</div>
				<Button variant="ghost" className="absolute top-7 left-5" asChild>
					<Link href="/">
						<ChevronLeftIcon className='size-4 me-2' />
						Home
					</Link>
				</Button>
				<div className="mx-auto space-y-6 sm:w-[350px]">
					<div className="flex items-center gap-2 lg:hidden">
						{logo}
						<p className="text-xl font-semibold">{brandName}</p>
					</div>
					<div className="flex flex-col space-y-2">
						<h1 className="font-heading text-2xl font-bold tracking-wide">
							{mode === "login" ? "Welcome Back" : step === "code" ? "Verify Email" : "Create an Account"}
						</h1>
						<p className="text-muted-foreground text-sm">
							{mode === "login" 
                                ? "Enter your email and password to sign in to your account." 
                                : step === "code"
                                ? "We've sent a 6-digit verification code to your email."
                                : "Enter your details below to create your account."}
						</p>
					</div>
					
                    {/* Google OAuth Button Placeholder */}
                    {mode === "login" && (
                        <>
                            <div className="space-y-2">
                                <Button type="button" size="lg" className="w-full" variant="outline" onClick={() => alert("Google Login Not Implemented yet")}>
                                    <GoogleIcon className='size-4 me-2' />
                                    Continue with Google
                                </Button>
                            </div>

                            <AuthSeparator />
                        </>
                    )}

                    <AnimatePresence mode="wait">
                    {error && (
                        <motion.div initial={{ opacity: 0, y: -10 }} animate={{ opacity: 1, y: 0 }} exit={{ opacity: 0, y: -10 }} className="p-3 bg-destructive/10 text-destructive text-sm rounded-md flex items-start gap-2">
                            <AlertCircleIcon className="w-4 h-4 shrink-0 mt-0.5" />
                            <span>{error}</span>
                        </motion.div>
                    )}
                    </AnimatePresence>

                    {mode === "login" ? (
                        <form className="space-y-4" onSubmit={handleSubmitLogin}>
                            <div className="space-y-2">
                                <div className="relative h-max">
                                    <Input
                                        placeholder="your.email@example.com"
                                        className="peer ps-9"
                                        type="email"
                                        value={email}
                                        onChange={e => setEmail(e.target.value)}
                                        required
                                        disabled={loading}
                                    />
                                    <div className="text-muted-foreground pointer-events-none absolute inset-y-0 start-0 flex items-center justify-center ps-3 peer-disabled:opacity-50">
                                        <AtSignIcon className="size-4" aria-hidden="true" />
                                    </div>
                                </div>
                                <div className="relative h-max">
                                    <Input
                                        placeholder="password"
                                        className="peer ps-9 pe-9"
                                        type={showPassword ? "text" : "password"}
                                        value={password}
                                        onChange={e => setPassword(e.target.value)}
                                        required
                                        disabled={loading}
                                    />
                                    <div className="text-muted-foreground pointer-events-none absolute inset-y-0 start-0 flex items-center justify-center ps-3 peer-disabled:opacity-50">
                                        <KeyRoundIcon className="size-4" aria-hidden="true" />
                                    </div>
                                    <button
                                        type="button"
                                        onClick={() => setShowPassword(!showPassword)}
                                        className="text-muted-foreground hover:text-foreground absolute inset-y-0 end-0 flex items-center justify-center pe-3 peer-disabled:opacity-50"
                                        aria-label={showPassword ? "Hide password" : "Show password"}
                                    >
                                        {showPassword ? <EyeOffIcon className="size-4" /> : <EyeIcon className="size-4" />}
                                    </button>
                                </div>
                            </div>
                            
                            <Button type="submit" className="w-full" disabled={loading}>
                                {loading ? <Loader2Icon className="w-4 h-4 animate-spin" /> : <span>Sign In</span>}
                            </Button>
                        </form>
                    ) : step === "form" ? (
                        <form className="space-y-4" onSubmit={handleSubmitRegister}>
                            <div className="space-y-2">
                                <div className="relative h-max">
                                    <Input
                                        placeholder="your.email@example.com"
                                        className="peer ps-9"
                                        type="email"
                                        value={email}
                                        onChange={e => setEmail(e.target.value)}
                                        required
                                        disabled={loading}
                                    />
                                    <div className="text-muted-foreground pointer-events-none absolute inset-y-0 start-0 flex items-center justify-center ps-3 peer-disabled:opacity-50">
                                        <AtSignIcon className="size-4" aria-hidden="true" />
                                    </div>
                                </div>
                                <div className="relative h-max">
                                    <Input
                                        placeholder="username"
                                        className="peer ps-9"
                                        type="text"
                                        minLength={3}
                                        value={username}
                                        onChange={e => setUsername(e.target.value)}
                                        required
                                        disabled={loading}
                                    />
                                    <div className="text-muted-foreground pointer-events-none absolute inset-y-0 start-0 flex items-center justify-center ps-3 peer-disabled:opacity-50">
                                        <UserIcon className="size-4" aria-hidden="true" />
                                    </div>
                                </div>
                                <div className="relative h-max">
                                    <Input
                                        placeholder="password (min 6 characters)"
                                        className="peer ps-9 pe-9"
                                        type={showPassword ? "text" : "password"}
                                        minLength={6}
                                        value={password}
                                        onChange={e => setPassword(e.target.value)}
                                        required
                                        disabled={loading}
                                    />
                                    <div className="text-muted-foreground pointer-events-none absolute inset-y-0 start-0 flex items-center justify-center ps-3 peer-disabled:opacity-50">
                                        <KeyRoundIcon className="size-4" aria-hidden="true" />
                                    </div>
                                    <button
                                        type="button"
                                        onClick={() => setShowPassword(!showPassword)}
                                        className="text-muted-foreground hover:text-foreground absolute inset-y-0 end-0 flex items-center justify-center pe-3 peer-disabled:opacity-50"
                                        aria-label={showPassword ? "Hide password" : "Show password"}
                                    >
                                        {showPassword ? <EyeOffIcon className="size-4" /> : <EyeIcon className="size-4" />}
                                    </button>
                                </div>
                            </div>

                            <Button type="submit" className="w-full" disabled={loading}>
                                {loading ? <Loader2Icon className="w-4 h-4 animate-spin" /> : <span>Continue to Verification</span>}
                            </Button>
                        </form>
                    ) : (
                        <form className="space-y-4" onSubmit={handleSubmitVerify}>
                            <div className="space-y-2">
                                <div className="relative h-max">
                                    <Input
                                        placeholder="6-digit code"
                                        className="peer ps-9 text-center tracking-widest text-lg"
                                        type="text"
                                        maxLength={6}
                                        value={code}
                                        onChange={e => setCode(e.target.value.replace(/\D/g, ''))}
                                        required
                                        disabled={loading}
                                    />
                                </div>
                            </div>

                            <Button type="submit" className="w-full" disabled={loading || code.length !== 6}>
                                {loading ? <Loader2Icon className="w-4 h-4 animate-spin" /> : <span>Verify & Create Account</span>}
                            </Button>
                            
                            <div className="text-center mt-2">
                                <button type="button" onClick={() => setStep("form")} className="text-xs text-muted-foreground hover:text-primary underline">
                                    Back to details
                                </button>
                            </div>
                        </form>
                    )}

                    <div className="text-center text-sm">
                        {mode === "login" ? (
                            <p className="text-muted-foreground">
                                Don't have an account?{" "}
                                <button type="button" onClick={() => { setMode("register"); setError(null); }} className="hover:text-primary underline underline-offset-4 text-foreground">
                                    Sign Up
                                </button>
                            </p>
                        ) : (
                            <p className="text-muted-foreground">
                                Already have an account?{" "}
                                <button type="button" onClick={() => { setMode("login"); setStep("form"); setError(null); }} className="hover:text-primary underline underline-offset-4 text-foreground">
                                    Sign In
                                </button>
                            </p>
                        )}
                    </div>

				</div>
			</div>
		</main>
	);
}

const MarqueeCard = ({ item }: { item: any }) => (
  <div className="mx-1.5 shrink-0 overflow-hidden rounded-lg shadow-lg hover:shadow-xl transition-all duration-300">
    <img
      src={`https://image.tmdb.org/t/p/w500${item.poster_path}`}
      alt={item.title || item.name}
      className="h-32 w-20 object-cover sm:h-40 sm:w-28"
      loading="lazy"
    />
  </div>
);

const MarqueeRow = React.memo(function MarqueeRow({
  data,
  reverse = false,
  speed = 40,
}: {
  data: any[];
  reverse?: boolean;
  speed?: number;
}) {
  // Multiply data so a single set is wide enough to cover most screens
  const singleSet = React.useMemo(() => [...data, ...data, ...data, ...data], [data]);
  
  return (
    <div className="relative w-full max-w-full overflow-hidden isolation-isolate flex">
      <div className="pointer-events-none absolute left-0 top-0 h-full w-12 z-10 bg-gradient-to-r from-background to-transparent" />
      
      <div
        className="flex w-max pt-1 pb-1 transform-gpu"
        style={{
          animation: `marqueeScroll ${speed}s linear infinite`,
          animationDirection: reverse ? "reverse" : "normal",
        }}
      >
        {/* Set 1 */}
        <div className="flex shrink-0">
          {singleSet.map((item, i) => (
            <MarqueeCard key={`set1-${item.id}-${i}`} item={item} />
          ))}
        </div>
        {/* Set 2 (Duplicate for seamless loop) */}
        <div className="flex shrink-0">
          {singleSet.map((item, i) => (
            <MarqueeCard key={`set2-${item.id}-${i}`} item={item} />
          ))}
        </div>
      </div>
      
      <div className="pointer-events-none absolute right-0 top-0 h-full w-12 z-10 bg-gradient-to-l from-background to-transparent" />
    </div>
  );
});

const TrendingMarquee = React.memo(function TrendingMarquee({ items }: { items: any[] }) {
  if (!items || items.length === 0) return null;
  const row1 = items.slice(0, 4);
  const row2 = items.slice(4, 8);
  const row3 = items.slice(8, 12);
  const row4 = items.slice(12, 16);
  const row5 = items.slice(16, 20);

  return (
    <div className="absolute inset-0 flex flex-col justify-center gap-1 overflow-hidden opacity-70 rotate-[-4deg] scale-110">
      <style>{`
        @keyframes marqueeScroll {
          0% { transform: translateX(0%); }
          100% { transform: translateX(-50%); }
        }
      `}</style>
      <MarqueeRow data={row1} reverse={false} speed={30} />
      <MarqueeRow data={row2} reverse={true} speed={35} />
      <MarqueeRow data={row3} reverse={false} speed={40} />
      <MarqueeRow data={row4} reverse={true} speed={45} />
      <MarqueeRow data={row5} reverse={false} speed={50} />
    </div>
  );
});

const GoogleIcon = (props: React.ComponentProps<'svg'>) => (
	<svg
		xmlns="http://www.w3.org/2000/svg"
		viewBox="0 0 24 24"
		fill="currentColor"
		{...props}
	>
		<g>
			<path d="M12.479,14.265v-3.279h11.049c0.108,0.571,0.164,1.247,0.164,1.979c0,2.46-0.672,5.502-2.84,7.669   C18.744,22.829,16.051,24,12.483,24C5.869,24,0.308,18.613,0.308,12S5.869,0,12.483,0c3.659,0,6.265,1.436,8.223,3.307L18.392,5.62   c-1.404-1.317-3.307-2.341-5.913-2.341C7.65,3.279,3.873,7.171,3.873,12s3.777,8.721,8.606,8.721c3.132,0,4.916-1.258,6.059-2.401   c0.927-0.927,1.537-2.251,1.777-4.059L12.479,14.265z" />
		</g>
	</svg>
);

const AuthSeparator = () => {
	return (
		<div className="flex w-full items-center justify-center">
			<div className="bg-border h-px w-full" />
			<span className="text-muted-foreground px-2 text-xs">OR</span>
			<div className="bg-border h-px w-full" />
		</div>
	);
};
