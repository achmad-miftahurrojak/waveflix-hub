"use client";

import { motion } from "framer-motion";
import { useRouter } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import type { FormEvent } from "react";
import { useAuth } from "@/components/AuthProvider";
import { NAME_FONTS, nameFontCss, profileFontVars } from "@/lib/profileFonts";
import { useTranslation } from "@/lib/i18n";

const pillSpring = { type: "spring", stiffness: 380, damping: 30 } as const;

type Msg = { ok: boolean; text: string } | null;
type Tab = "profile" | "security" | "account";

const card = "rounded-2xl bg-white/[0.03] p-6 ring-1 ring-white/10";
const inputCls =
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
  const { user, ready, updateProfile, changePassword, uploadImage, authFetch, activeProfile, setActiveProfile } = useAuth();
  const { t } = useTranslation();
  const router = useRouter();
  const [tab, setTab] = useState<Tab>("profile");

  const [profName, setProfName] = useState("");
  const [profBio, setProfBio] = useState("");
  const [profFont, setProfFont] = useState("");
  const [profMsg, setProfMsg] = useState<Msg>(null);
  const [savingProf, setSavingProf] = useState(false);

  const [imgMsg, setImgMsg] = useState<Msg>(null);
  const [uploading, setUploading] = useState<"avatar" | "banner" | null>(null);
  const avatarInput = useRef<HTMLInputElement>(null);
  const bannerInput = useRef<HTMLInputElement>(null);

  const [cur, setCur] = useState("");
  const [nw, setNw] = useState("");
  const [confirm, setConfirm] = useState("");
  const [pwMsg, setPwMsg] = useState<Msg>(null);
  const [newEmail, setNewEmail] = useState("");
  const [emailMsg, setEmailMsg] = useState<Msg>(null);
  const [savingPw, setSavingPw] = useState(false);
  const [savingEmail, setSavingEmail] = useState(false);

  const [histMsg, setHistMsg] = useState<Msg>(null);
  const [clearingHistory, setClearingHistory] = useState(false);

  const [langCode, setLangCode] = useState("id");
  const [langMsg, setLangMsg] = useState<Msg>(null);
  const [savingLang, setSavingLang] = useState(false);

  const profileId = (activeProfile as any)?.id;

  useEffect(() => {
    if (ready && !user) router.replace("/masuk");
  }, [ready, user, router]);

  useEffect(() => {
    if (!activeProfile) return;
    const p = activeProfile as any;
    setProfName(p.name || "");
    setProfBio(p.bio || "");
    setProfFont(p.name_font || user?.name_font || "");
  }, [activeProfile, user]);

  useEffect(() => {
    if (user?.language) setLangCode(user.language);
  }, [user]);

  if (!ready || !user) {
    return <main className="min-h-screen pt-28 text-center text-white/50">Loading…</main>;
  }

  const currentAvatar = activeProfile ? (activeProfile as any).avatar : user.avatar || "";
  const currentBanner = activeProfile ? (activeProfile as any).banner : user.banner || "";
  const displayName = (activeProfile ? (activeProfile as any).name : user.username) || "?";

  const onPick = (field: "avatar" | "banner", file?: File | null) => {
    if (!file || !profileId) return;
    setImgMsg(null);
    if (!["image/png", "image/jpeg", "image/gif"].includes(file.type))
      return setImgMsg({ ok: false, text: "Gunakan gambar JPG, PNG, atau GIF." });
    if (file.size > MAX_BYTES)
      return setImgMsg({ ok: false, text: "Gambar terlalu besar (maks ~6MB)." });

    const reader = new FileReader();
    reader.onload = async () => {
      setUploading(field);
      try {
        const res = await authFetch(`/api/profiles/${profileId}`, {
          method: "PUT",
          body: JSON.stringify({ [field]: String(reader.result) }),
        });
        if (res.ok) {
          const updated = await res.json();
          setActiveProfile({ ...(activeProfile as any), ...updated });
          setImgMsg({ ok: true, text: `${field === "avatar" ? "Foto profil" : "Banner"} berhasil diperbarui.` });
        } else {
          setImgMsg({ ok: false, text: "Gagal upload gambar." });
        }
      } catch {
        setImgMsg({ ok: false, text: "Tidak bisa terhubung ke server." });
      } finally {
        setUploading(null);
      }
    };
    reader.readAsDataURL(file);
  };

  const removeImg = async (field: "avatar" | "banner") => {
    if (!profileId) return;
    setUploading(field);
    try {
      const res = await authFetch(`/api/profiles/${profileId}`, {
        method: "PUT",
        body: JSON.stringify({ [field]: "" }),
      });
      if (res.ok) {
        const updated = await res.json();
        setActiveProfile({ ...(activeProfile as any), ...updated });
        setImgMsg({ ok: true, text: `${field === "avatar" ? "Foto profil" : "Banner"} dihapus.` });
      }
    } catch {
      setImgMsg({ ok: false, text: "Gagal menghapus." });
    } finally {
      setUploading(null);
    }
  };

  const saveProfile = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    if (!profileId) return;
    setSavingProf(true);
    try {
      const res = await authFetch(`/api/profiles/${profileId}`, {
        method: "PUT",
        body: JSON.stringify({ name: profName.trim(), bio: profBio }),
      });
      if (res.ok) {
        const updated = await res.json();
        setActiveProfile({ ...(activeProfile as any), ...updated });
        setProfMsg({ ok: true, text: "Perubahan profil berhasil disimpan." });
      } else {
        setProfMsg({ ok: false, text: "Gagal menyimpan." });
      }
    } catch {
      setProfMsg({ ok: false, text: "Tidak bisa terhubung ke server." });
    } finally {
      setSavingProf(false);
    }
  };

  const savePw = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setPwMsg(null);
    if (nw !== confirm) return setPwMsg({ ok: false, text: "Password baru tidak cocok." });
    setSavingPw(true);
    const err = await changePassword(cur, nw);
    setSavingPw(false);
    if (err) setPwMsg({ ok: false, text: err });
    else { setPwMsg({ ok: true, text: "Password berhasil diperbarui." }); setCur(""); setNw(""); setConfirm(""); }
  };

  const saveEmail = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setEmailMsg(null);
    setSavingEmail(true);
    const err = await updateProfile({ email: newEmail.trim() });
    setSavingEmail(false);
    if (err) setEmailMsg({ ok: false, text: err });
    else { setEmailMsg({ ok: true, text: "Email berhasil diperbarui." }); setNewEmail(""); }
  };

  const clearHistory = async () => {
    setHistMsg(null);
    setClearingHistory(true);
    try {
      const r = await authFetch("/api/history", { method: "DELETE" });
      setHistMsg(r.ok
        ? { ok: true, text: "Riwayat nonton berhasil dihapus." }
        : { ok: false, text: "Gagal menghapus riwayat." });
    } catch {
      setHistMsg({ ok: false, text: "Tidak bisa terhubung ke server." });
    } finally {
      setClearingHistory(false);
    }
  };

  const saveLanguage = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setLangMsg(null);
    setSavingLang(true);
    const err = await updateProfile({ language: langCode });
    setSavingLang(false);
    if (err) {
      setLangMsg({ ok: false, text: err });
    } else {
      document.cookie = `waveflix_lang=${langCode}; path=/; max-age=31536000`;
      setLangMsg({ ok: true, text: "Berhasil/Success" });
      window.location.reload();
    }
  };

  const TABS: { key: Tab; label: string }[] = [
    { key: "profile", label: t("settings.tab.profile") },
    { key: "security", label: t("settings.tab.security") },
    { key: "account", label: t("settings.tab.account") },
  ];

  return (
    <main className={`mx-auto min-h-screen max-w-5xl px-[4%] pb-24 pt-20 md:pt-24 ${profileFontVars}`}>
      <h1 className="text-3xl font-bold">{t("settings.title")}</h1>
      <p className="mb-4 text-sm text-white/50">
        {t("settings.desc")}
      </p>

      <input ref={avatarInput} type="file" accept="image/png,image/jpeg,image/gif" className="hidden" onChange={(e) => onPick("avatar", e.target.files?.[0])} />
      <input ref={bannerInput} type="file" accept="image/png,image/jpeg,image/gif" className="hidden" onChange={(e) => onPick("banner", e.target.files?.[0])} />

      {}
      <section className="overflow-hidden rounded-2xl bg-black/50 backdrop-blur-[35px] ring-1 ring-white/10 shadow-xl">
        <div className="relative h-40 w-full">
          {currentBanner ? (

            <img src={currentBanner} alt="" className="h-full w-full object-cover" />
          ) : (
            <div className="h-full w-full bg-gradient-to-br from-accent/25 via-bg to-bg" />
          )}
          <button
            onClick={() => bannerInput.current?.click()}
            disabled={uploading === "banner"}
            className="absolute right-4 top-4 flex items-center gap-2 rounded-lg bg-black/60 px-3 py-1.5 text-xs font-semibold backdrop-blur transition hover:bg-black/80 disabled:opacity-60"
          >
            <CameraIcon /> {uploading === "banner" ? t("settings.profile.uploading") : t("settings.profile.changeBanner")}
          </button>
        </div>
        <div className="flex flex-wrap items-end justify-between gap-4 p-5">
          <div className="flex items-center gap-4">
            <div className="relative -mt-16 shrink-0">
              {currentAvatar ? (

                <img src={currentAvatar} alt="" className="h-24 w-24 rounded-full border-4 border-bg object-cover" />
              ) : (
                <span className="grid h-24 w-24 place-items-center rounded-full border-4 border-bg bg-accent text-3xl font-bold text-black">
                  {displayName.charAt(0).toUpperCase()}
                </span>
              )}
              <button
                onClick={() => avatarInput.current?.click()}
                disabled={uploading === "avatar"}
                aria-label="Ganti foto profil"
                className="absolute bottom-0 right-0 grid h-8 w-8 place-items-center rounded-full bg-accent text-black shadow-lg transition hover:bg-accent-dark disabled:opacity-60"
              >
                <CameraIcon />
              </button>
            </div>
            <div>
              <div className="text-lg font-bold" style={{ fontFamily: nameFontCss(profFont), lineHeight: 1.4 }}>
                {displayName}
              </div>
              <div className="text-xs text-white/40 mt-0.5">
                {activeProfile ? t("settings.account.activeProfile") : t("settings.tab.account")}
              </div>
              <div className="text-sm text-white/50">{user.email}</div>
            </div>
          </div>
          <div className="flex gap-4 text-sm text-white/60">
            <button onClick={() => removeImg("avatar")} className="hover:text-red-300 transition">{t("settings.profile.removeAvatar")}</button>
            <button onClick={() => removeImg("banner")} className="hover:text-red-300 transition">{t("settings.profile.removeBanner")}</button>
          </div>
        </div>
        <div className="px-5 pb-4"><Note msg={imgMsg} /></div>
      </section>

      {}
      <div className="mt-6 grid grid-cols-3 gap-2 rounded-2xl bg-black/50 backdrop-blur-[35px] p-2 ring-1 ring-white/10 shadow-lg">
        {TABS.map((t) => (
          <button key={t.key} onClick={() => setTab(t.key)} className={`relative rounded-xl py-3 text-sm font-semibold transition ${tab === t.key ? "text-accent" : "text-white/60 hover:text-white"}`}>
            {tab === t.key && <motion.span layoutId="settingsTab" transition={pillSpring} className="absolute inset-0 rounded-xl bg-white/10 shadow-sm" />}
            <span className="relative z-10">{t.label}</span>
          </button>
        ))}
      </div>

      {}
      {tab === "profile" && (
        <form onSubmit={saveProfile} className={`${card} mt-5`}>
          <h2 className="text-lg font-semibold">{t("settings.profile.info")}</h2>
          <p className="mb-5 text-sm text-white/50">
            {t("settings.profile.infoDesc")}
          </p>

          <div className="mb-4">
            <label className={label}>{t("settings.profile.name")}</label>
            <input className={inputCls} maxLength={25} value={profName} onChange={(e) => setProfName(e.target.value)} required />
            <div className="mt-1 text-right text-xs text-white/40">{profName.length}/25</div>
          </div>

          <div className="mb-4">
            <label className={label}>{t("settings.profile.nameStyle")}</label>
            <div className="flex flex-wrap gap-2">
              {NAME_FONTS.map((f) => (
                <button key={f.key} type="button" onClick={() => setProfFont(f.key)} style={{ fontFamily: f.css }}
                  className={`rounded-lg px-3 py-2 text-base ring-1 transition ${profFont === f.key ? "bg-accent text-black ring-accent" : "bg-white/5 ring-white/15 hover:bg-white/10"}`}>
                  {f.label}
                </button>
              ))}
            </div>
            <p className="mt-3 text-2xl" style={{ fontFamily: nameFontCss(profFont), lineHeight: 1.5 }}>{profName || t("settings.profile.preview")}</p>
          </div>

          <div className="mb-4">
            <label className={label}>{t("settings.profile.bio")}</label>
            <textarea className={`${inputCls} min-h-24 resize-y`} maxLength={250} value={profBio} onChange={(e) => setProfBio(e.target.value)} placeholder={t("settings.profile.bioPlaceholder")} />
            <div className="mt-1 text-right text-xs text-white/40">{profBio.length}/250</div>
          </div>

          {}
          <div className="mb-4">
            <label className={label}>{t("settings.profile.accountEmail")}</label>
            <input className={`${inputCls} cursor-not-allowed opacity-60`} value={user.email} disabled />
            <p className="mt-1 text-xs text-white/40">{t("settings.profile.accountEmailDesc")}</p>
          </div>

          <button type="submit" disabled={savingProf} className="rounded-lg bg-accent px-5 py-2.5 text-sm font-semibold text-black transition hover:bg-accent-dark disabled:opacity-60">
            {savingProf ? t("settings.profile.saving") : t("settings.profile.save")}
          </button>
          <Note msg={profMsg} />
        </form>
      )}

      {}
      {tab === "security" && (
        <div className="mt-5 grid gap-5 md:grid-cols-2">
          <form onSubmit={savePw} className={card}>
            <h2 className="text-lg font-semibold">{t("settings.security.changePw")}</h2>
            <p className="mb-5 text-sm text-white/50">{t("settings.security.changePwDesc")}</p>
            <div className="mb-4">
              <label className={label}>{t("settings.security.curPw")}</label>
              <input type="password" className={inputCls} value={cur} onChange={(e) => setCur(e.target.value)} required />
            </div>
            <div className="mb-4">
              <label className={label}>{t("settings.security.newPw")}</label>
              <input type="password" minLength={8} className={inputCls} value={nw} onChange={(e) => setNw(e.target.value)} required />
            </div>
            <div className="mb-4">
              <label className={label}>{t("settings.security.confirmPw")}</label>
              <input type="password" minLength={8} className={inputCls} value={confirm} onChange={(e) => setConfirm(e.target.value)} required />
            </div>
            <button type="submit" disabled={savingPw} className="rounded-lg bg-accent px-5 py-2.5 text-sm font-semibold text-black transition hover:bg-accent-dark disabled:opacity-60">
              {savingPw ? t("settings.security.updating") : t("settings.security.updatePw")}
            </button>
            <Note msg={pwMsg} />
          </form>

          <form onSubmit={saveEmail} className={card}>
            <h2 className="text-lg font-semibold">{t("settings.security.changeEmail")}</h2>
            <p className="mb-5 text-sm text-white/50">{t("settings.security.changeEmailDesc")}</p>
            <div className="mb-4 flex items-center justify-between text-sm">
              <span className="text-white/50">{t("settings.security.curEmail")}</span>
              <span className="font-semibold">{user.email}</span>
            </div>
            <div className="mb-4">
              <label className={label}>{t("settings.security.newEmail")}</label>
              <input type="email" className={inputCls} value={newEmail} onChange={(e) => setNewEmail(e.target.value)} placeholder={t("settings.security.emailPlaceholder")} required />
            </div>
            <button type="submit" disabled={savingEmail} className="rounded-lg bg-accent px-5 py-2.5 text-sm font-semibold text-black transition hover:bg-accent-dark disabled:opacity-60">
              {savingEmail ? t("settings.security.updating") : t("settings.security.updateEmail")}
            </button>
            <Note msg={emailMsg} />
          </form>
        </div>
      )}

      {}
      {tab === "account" && (
        <div className="mt-5 grid gap-5 md:grid-cols-2">
          <form onSubmit={saveLanguage} className={card}>
            <h2 className="text-lg font-semibold">{t("settings.account.language")}</h2>
            <p className="mb-5 text-sm text-white/50">{t("settings.account.languageDesc")}</p>
            <div className="mb-4">
              <label className={label}>{t("settings.account.language")}</label>
              <select className={`${inputCls} appearance-none`} value={langCode} onChange={(e) => setLangCode(e.target.value)}>
                <option value="id" className="bg-bg">Bahasa Indonesia</option>
                <option value="en" className="bg-bg">English</option>
                <option value="ja" className="bg-bg">日本語 (Japanese)</option>
              </select>
            </div>
            <button type="submit" disabled={savingLang} className="rounded-lg bg-accent px-5 py-2.5 text-sm font-semibold text-black transition hover:bg-accent-dark disabled:opacity-60">
              {savingLang ? "Menyimpan…" : t("settings.account.saveLanguage")}
            </button>
            <Note msg={langMsg} />
          </form>

          <div className={card}>
            <h2 className="mb-5 text-lg font-semibold">{t("settings.account.info")}</h2>
            <div className="flex items-center justify-between border-b border-white/10 py-3 text-sm">
              <span className="text-white/50">{t("settings.account.email")}</span>
              <span className="font-semibold">{user.email}</span>
            </div>
            <div className="flex items-center justify-between border-b border-white/10 py-3 text-sm">
              <span className="text-white/50">{t("settings.account.activeProfile")}</span>
              <span className="font-semibold">{(activeProfile as any)?.name || "-"}</span>
            </div>
            <div className="flex items-center justify-between py-3 text-sm">
              <span className="text-white/50">{t("settings.account.status")}</span>
              <span className="rounded-full bg-emerald-500/15 px-3 py-0.5 text-xs font-bold text-emerald-400">{t("settings.account.active")}</span>
            </div>
          </div>

          <div className={card}>
            <h2 className="text-lg font-semibold">{t("settings.account.history")}</h2>
            <p className="mb-5 text-sm text-white/50">{t("settings.account.historyDesc")}</p>
            <p className="mb-4 text-sm text-white/60">
              {t("settings.account.historyWarn")}
            </p>
            <button
              onClick={clearHistory}
              disabled={clearingHistory}
              className="rounded-lg border border-red-400/40 px-5 py-2.5 text-sm font-semibold text-red-300 transition hover:bg-red-500/10 disabled:cursor-not-allowed disabled:opacity-50"
            >
              {clearingHistory ? "Menghapus…" : t("settings.account.clearHistory")}
            </button>
            <Note msg={histMsg} />
          </div>
        </div>
      )}
    </main>
  );
}
