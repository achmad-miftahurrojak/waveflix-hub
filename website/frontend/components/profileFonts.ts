import {
  Playfair_Display,
  Bebas_Neue,
  Pacifico,
  Caveat,
  Lobster,
} from "next/font/google";
import localFont from "next/font/local";

const serif = Playfair_Display({ subsets: ["latin"], weight: ["600", "700"], variable: "--pf-serif", display: "swap" });
const bebas = Bebas_Neue({ subsets: ["latin"], weight: "400", variable: "--pf-bebas", display: "swap" });
const pacifico = Pacifico({ subsets: ["latin"], weight: "400", variable: "--pf-pacifico", display: "swap" });
const caveat = Caveat({ subsets: ["latin"], weight: ["600", "700"], variable: "--pf-caveat", display: "swap" });
const lobster = Lobster({ subsets: ["latin"], weight: "400", variable: "--pf-lobster", display: "swap" });

// Font custom lokal (.ttf) untuk nama profil.
const birds = localFont({ src: "./fonts/birds.ttf", variable: "--pf-birds", display: "swap" });
const poti = localFont({ src: "./fonts/poti.ttf", variable: "--pf-poti", display: "swap" });
const somelist = localFont({ src: "./fonts/somelist.ttf", variable: "--pf-somelist", display: "swap" });
const porkys = localFont({ src: "./fonts/porkys.ttf", variable: "--pf-porkys", display: "swap" });
const floozy = localFont({ src: "./fonts/floozy.ttf", variable: "--pf-floozy", display: "swap" });

/** Pilihan font untuk nama profil (formal → cute → custom). key disimpan di DB. */
export const NAME_FONTS: { key: string; label: string; css: string }[] = [
  { key: "", label: "Default", css: "inherit" },
  { key: "serif", label: "Elegant", css: "var(--pf-serif)" },
  { key: "bebas", label: "Bold", css: "var(--pf-bebas)" },
  { key: "caveat", label: "Handwritten", css: "var(--pf-caveat)" },
  { key: "lobster", label: "Playful", css: "var(--pf-lobster)" },
  { key: "pacifico", label: "Cute", css: "var(--pf-pacifico)" },
  { key: "birds", label: "Birds of Paradise", css: "var(--pf-birds)" },
  { key: "poti", label: "Pretty Inside", css: "var(--pf-poti)" },
  { key: "somelist", label: "Somelist", css: "var(--pf-somelist)" },
  { key: "porkys", label: "Porky's", css: "var(--pf-porkys)" },
  { key: "floozy", label: "Floozy", css: "var(--pf-floozy)" },
];

export const nameFontCss = (key?: string) =>
  NAME_FONTS.find((f) => f.key === (key || ""))?.css ?? "inherit";

export const profileFontVars = [
  serif.variable,
  bebas.variable,
  pacifico.variable,
  caveat.variable,
  lobster.variable,
  birds.variable,
  poti.variable,
  somelist.variable,
  porkys.variable,
  floozy.variable,
].join(" ");
