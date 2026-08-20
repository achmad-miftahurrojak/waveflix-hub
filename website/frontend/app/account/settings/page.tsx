"use client";

import { motion } from "framer-motion";
import { useRouter } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import type { FormEvent } from "react";
import { useAuth } from "@/components/AuthProvider";
import { NAME_FONTS, nameFontCss, profileFontVars } from "@/lib/profileFonts";

const pillSpring = { type: "spring", stiffness: 380, damping: 30 } as const;

type Msg = { ok: boolean; text: string } | null;
type Tab = "profile" | "security" | "account";

const card = "rounded-2xl bg-white/[0.03] p-6 ring-1 ring-white/10";
const input =
  "w-full rounded-lg border border-white/15 bg-white/5 px-4 py-2.5 text-sm outline-none transition focus:border-accent";
const label = "mb-1 block text-xs font-semibold uppercase tracking-wide text-white/50";
const MAX_BYTES = 6 * 1024 * 1024;

function Note({ msg }: { msg: Msg }) {
  if (!msg) return null;
  return (
    <p className={`mt-2 rounded-md px-3 py-2 text-sm ${msg.ok ? "bg-accent/15 text-accent" : "bg-red-500/15 text-red-300"}`}>
      {msg.text}
    </p>
  );
}

function CameraIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" width="15" height="15">
      <path d="M23 19a2 2 0 0 1-2 2H3a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h4l2-3h6l2 3h4a2 2 0 0 1 2 2z" />
      <circle cx="12" cy="13" r="4" />
    </svg>
  );
}

export default function SettingsPage() {
  const { user, ready, updateProfile, changePassword, uploadImage, authFetch } = useAuth();
  const router = useRouter();
  const [tab, setTab] = useState<Tab>("profile");

  // profile
  const [name, setName] = useState("");
  const [font, setFont] = useState("");
  const [bio, setBio] = useState("");
  const [profMsg, setProfMsg] = useState<Msg>(null);
  const [savingProf, setSavingProf] = useState(false);

  // security
  const [cur, setCur] = useState("");
  const [nw, setNw] = useState("");
  const [confirm, setConfirm] = useState("");
  const [pwMsg, setPwMsg] = useState<Msg>(null);
  const [newEmail, setNewEmail] = useState("");
  const [emailMsg, setEmailMsg] = useState<Msg>(null);

  // images
  const [imgMsg, setImgMsg] = useState<Msg>(null);
  const [uploading, setUploading] = useState<"avatar" | "banner" | null>(null);
  const avatarInput = useRef<HTMLInputElement>(null);
  const bannerInput = useRef<HTMLInputElement>(null);

  // account
  const [histMsg, setHistMsg] = useState<Msg>(null);

  useEffect(() => {
    if (ready && !user) router.replace("/masuk");
  }, [ready, user, router]);
  useEffect(() => {
    if (!user) return;
    setName(user.username);
    setFont(user.name_font || "");
    setBio(user.bio || "");
  }, [user]);

  if (!ready || !user) {
    return <main className="min-h-screen pt-28 text-center text-white/50">Loading…</main>;
  }

  const onPick = (field: "avatar" | "banner", file?: File | null) => {
    if (!file) return;
    setImgMsg(null);
    if (!["image/png", "image/jpeg", "image/gif"].includes(file.type)) return setImgMsg({ ok: false, text: "Use a JPG, PNG, or GIF image." });
    if (file.size > MAX_BYTES) return setImgMsg({ ok: false, text: "Image is too large (max ~6MB)." });
    const reader = new FileReader();
    reader.onload = async () => {
      setUploading(field);
      const err = await uploadImage(field, String(reader.result));
      setUploading(null);
      setImgMsg(err ? { ok: false, text: err } : { ok: true, text: `${field} updated.` });
    };
    reader.readAsDataURL(file);
  };

  const removeImg = async (field: "avatar" | "banner") => {
    setUploading(field);
    const err = await uploadImage(field, "");
    setUploading(null);
    setImgMsg(err ? { ok: false, text: err } : { ok: true, text: `${field} removed.` });
  };

  const saveProfile = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setSavingProf(true);
    const err = await updateProfile({ username: name.trim(), bio, name_font: font });
    setSavingProf(false);
    setProfMsg(err ? { ok: false, text: err } : { ok: true, text: "Changes saved." });
  };

  const savePw = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setPwMsg(null);
    if (nw !== confirm) return setPwMsg({ ok: false, text: "New passwords don't match." });
    const err = await changePassword(cur, nw);
    if (err) setPwMsg({ ok: false, text: err });
    else {
      setPwMsg({ ok: true, text: "Password updated." });
      setCur(""); setNw(""); setConfirm("");
    }
  };

  const saveEmail = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const err = await updateProfile({ email: newEmail.trim() });
    if (err) setEmailMsg({ ok: false, text: err });
    else {
      setEmailMsg({ ok: true, text: "Email updated." });
      setNewEmail("");
    }
  };

  const clearHistory = async () => {
    setHistMsg(null);
    try {
      const r = await authFetch("/api/history", { method: "DELETE" });
      setHistMsg(r.ok ? { ok: true, text: "Watch history cleared." } : { ok: false, text: "Failed to clear history." });
    } catch {
      setHistMsg({ ok: false, text: "Cannot reach the server." });
    }
  };

  const TABS: { key: Tab; label: string }[] = [
    { key: "profile", label: "Profile" },
    { key: "security", label: "Security" },
    { key: "account", label: "Account" },
  ];

  return (
    <main className={`mx-auto min-h-screen max-w-5xl px-[4%] pb-16 pt-28 ${profileFontVars}`}>
      <h1 className="text-3xl font-bold">Settings</h1>
      <p className="mb-6 text-sm text-white/50">Manage your account preferences</p>

      <input ref={avatarInput} type="file" accept="image/png,image/jpeg,image/gif" className="hidden" onChange={(e) => onPick("avatar", e.target.files?.[0])} />
      <input ref={bannerInput} type="file" accept="image/png,image/jpeg,image/gif" className="hidden" onChange={(e) => onPick("banner", e.target.files?.[0])} />

      {/* Top card: avatar + banner */}
      <section className="overflow-hidden rounded-2xl ring-1 ring-white/10">
        <div className="relative h-40 w-full">
          {user.banner ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img src={user.banner} alt="" className="h-full w-full object-cover" />
          ) : (
            <div className="h-full w-full bg-gradient-to-br from-accent/25 via-bg to-bg" />
          )}
          <button onClick={() => bannerInput.current?.click()} disabled={uploading === "banner"} className="absolute right-4 top-4 flex items-center gap-2 rounded-lg bg-black/60 px-3 py-1.5 text-xs font-semibold backdrop-blur transition hover:bg-black/80 disabled:opacity-60">
            <CameraIcon /> {uploading === "banner" ? "Uploading…" : "Change cover"}
          </button>
        </div>
        <div className="flex flex-wrap items-end justify-between gap-4 bg-white/[0.02] p-5">
          <div className="flex items-center gap-4">
            <div className="relative -mt-16 shrink-0">
              {user.avatar ? (
                // eslint-disable-next-line @next/next/no-img-element
                <img src={user.avatar} alt="" className="h-24 w-24 rounded-full border-4 border-bg object-cover" />
              ) : (
                <span className="grid h-24 w-24 place-items-center rounded-full border-4 border-bg bg-accent text-3xl font-bold text-black">
                  {user.username.charAt(0).toUpperCase()}
                </span>
              )}
              <button onClick={() => avatarInput.current?.click()} disabled={uploading === "avatar"} aria-label="Change avatar" className="absolute bottom-0 right-0 grid h-8 w-8 place-items-center rounded-full bg-accent text-black shadow-lg transition hover:bg-accent-dark disabled:opacity-60">
                <CameraIcon />
              </button>
            </div>
            <div>
              <div className="text-lg font-bold" style={{ fontFamily: nameFontCss(user.name_font), lineHeight: 1.4 }}>{user.username}</div>
              <div className="text-sm text-white/50">@{user.username.toLowerCase().replace(/\s+/g, "")}</div>
            </div>
          </div>
          <div className="flex gap-4 text-sm text-white/60">
            <button onClick={() => removeImg("avatar")} className="hover:text-red-300">Remove avatar</button>
            <button onClick={() => removeImg("banner")} className="hover:text-red-300">Remove cover</button>
          </div>
        </div>
        <div className="px-5 pb-4"><Note msg={imgMsg} /></div>
      </section>

      {/* Tabs */}
      <div className="mt-6 grid grid-cols-3 gap-2 rounded-2xl bg-white/[0.03] p-2 ring-1 ring-white/10">
        {TABS.map((t) => (
          <button key={t.key} onClick={() => setTab(t.key)} className={`relative rounded-xl py-3 text-sm font-semibold transition ${tab === t.key ? "text-accent" : "text-white/60 hover:text-white"}`}>
            {tab === t.key && <motion.span layoutId="settingsTab" transition={pillSpring} className="absolute inset-0 rounded-xl bg-white/10" />}
            <span className="relative z-10">{t.label}</span>
          </button>
        ))}
      </div>

      {/* Profile tab */}
      {tab === "profile" && (
        <form onSubmit={saveProfile} className={`${card} mt-5`}>
          <h2 className="text-lg font-semibold">Profile Information</h2>
          <p className="mb-5 text-sm text-white/50">Visible on your public profile</p>

          <div className="mb-4">
            <label className={label}>Display name</label>
            <input className={input} maxLength={25} value={name} onChange={(e) => setName(e.target.value)} required />
            <div className="mt-1 text-right text-xs text-white/40">{name.length}/25</div>
          </div>

          <div className="mb-4">
            <label className={label}>Name font</label>
            <div className="flex flex-wrap gap-2">
              {NAME_FONTS.map((f) => (
                <button key={f.key} type="button" onClick={() => setFont(f.key)} style={{ fontFamily: f.css }} className={`rounded-lg px-3 py-2 text-base ring-1 transition ${font === f.key ? "bg-accent text-black ring-accent" : "bg-white/5 ring-white/15 hover:bg-white/10"}`}>
                  {f.label}
                </button>
              ))}
            </div>
            <p className="mt-3 text-2xl" style={{ fontFamily: nameFontCss(font), lineHeight: 1.5 }}>{name || "Preview"}</p>
          </div>

          <div className="mb-4">
            <label className={label}>Bio</label>
            <textarea className={`${input} min-h-[90px] resize-y`} maxLength={250} value={bio} onChange={(e) => setBio(e.target.value)} placeholder="Tell us about yourself" />
            <div className="mt-1 text-right text-xs text-white/40">{bio.length}/250</div>
          </div>

          <div className="mb-4">
            <label className={label}>Email</label>
            <input className={`${input} cursor-not-allowed opacity-60`} value={user.email} disabled />
            <p className="mt-1 text-xs text-white/40">Change your email in the Security tab.</p>
          </div>

          <button type="submit" disabled={savingProf} className="rounded-lg bg-accent px-5 py-2.5 text-sm font-semibold text-black transition hover:bg-accent-dark disabled:opacity-60">
            {savingProf ? "Saving…" : "Save Changes"}
          </button>
          <Note msg={profMsg} />
        </form>
      )}

      {/* Security tab */}
      {tab === "security" && (
        <div className="mt-5 grid gap-5 md:grid-cols-2">
          <form onSubmit={savePw} className={card}>
            <h2 className="text-lg font-semibold">Change Password</h2>
            <p className="mb-5 text-sm text-white/50">Keep your account secure</p>
            <div className="mb-4">
              <label className={label}>Current password</label>
              <input type="password" className={input} value={cur} onChange={(e) => setCur(e.target.value)} required />
            </div>
            <div className="mb-4">
              <label className={label}>New password</label>
              <input type="password" minLength={8} className={input} value={nw} onChange={(e) => setNw(e.target.value)} required />
            </div>
            <div className="mb-4">
              <label className={label}>Confirm new password</label>
              <input type="password" minLength={8} className={input} value={confirm} onChange={(e) => setConfirm(e.target.value)} required />
            </div>
            <button type="submit" className="rounded-lg bg-accent px-5 py-2.5 text-sm font-semibold text-black transition hover:bg-accent-dark">Update Password</button>
            <Note msg={pwMsg} />
          </form>

          <form onSubmit={saveEmail} className={card}>
            <h2 className="text-lg font-semibold">Change Email</h2>
            <p className="mb-5 text-sm text-white/50">Used for sign-in</p>
            <div className="mb-4 flex items-center justify-between text-sm">
              <span className="text-white/50">Current</span>
              <span className="font-semibold">{user.email}</span>
            </div>
            <div className="mb-4">
              <label className={label}>New email</label>
              <input type="email" className={input} value={newEmail} onChange={(e) => setNewEmail(e.target.value)} placeholder="you@example.com" required />
            </div>
            <button type="submit" className="rounded-lg bg-accent px-5 py-2.5 text-sm font-semibold text-black transition hover:bg-accent-dark">Update Email</button>
            <Note msg={emailMsg} />
          </form>
        </div>
      )}

      {/* Account tab */}
      {tab === "account" && (
        <div className="mt-5 grid gap-5 md:grid-cols-2">
          <div className={card}>
            <h2 className="text-lg font-semibold">Watch History</h2>
            <p className="mb-5 text-sm text-white/50">Manage your viewing activity</p>
            <p className="mb-4 text-sm text-white/60">
              Clears every movie and episode from your history. This cannot be undone.
            </p>
            <button onClick={clearHistory} className="rounded-lg border border-red-400/40 px-5 py-2.5 text-sm font-semibold text-red-300 transition hover:bg-red-500/10">
              Clear all history
            </button>
            <Note msg={histMsg} />
          </div>

          <div className={card}>
            <h2 className="mb-5 text-lg font-semibold">Account</h2>
            <div className="flex items-center justify-between border-b border-white/10 py-3 text-sm">
              <span className="text-white/50">Email</span>
              <span className="font-semibold">{user.email}</span>
            </div>
            <div className="flex items-center justify-between py-3 text-sm">
              <span className="text-white/50">Status</span>
              <span className="rounded-full bg-emerald-500/15 px-3 py-0.5 text-xs font-bold text-emerald-400">Active</span>
            </div>
          </div>
        </div>
      )}
    </main>
  );
}
