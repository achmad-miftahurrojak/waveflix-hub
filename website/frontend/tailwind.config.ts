import type { Config } from "tailwindcss";

const config: Config = {
  content: [
    "./app/**/*.{ts,tsx}",
    "./components/**/*.{ts,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        // Ganti nilai variabel di globals.css (`:root`) kalau mau ubah warna brand.
        bg: "rgb(var(--color-bg) / <alpha-value>)",
        surface: "rgb(var(--color-surface) / <alpha-value>)",
        // Hierarki surface: bg (dasar) → surface (card) → surface-raised (modal)
        // → surface-overlay (dropdown/search). Semua near-black wajib lewat token ini.
        "surface-raised": "rgb(var(--color-surface-raised) / <alpha-value>)",
        "surface-overlay": "rgb(var(--color-surface-overlay) / <alpha-value>)",
        accent: {
          DEFAULT: "rgb(var(--color-accent) / <alpha-value>)",
          dark: "rgb(var(--color-accent-dark) / <alpha-value>)",
        },
      },
      fontFamily: {
        sans: ["var(--font-poppins)", "Segoe UI", "system-ui", "sans-serif"],
        body: ["var(--font-poppins)", "Segoe UI", "system-ui", "sans-serif"],
        logo: ["var(--font-logo)", "Impact", "Arial Narrow", "sans-serif"],
      },
      boxShadow: {
        // Shadow card dibuat subtle — depth pendukung, bukan pesaing konten.
        card: "0 10px 20px rgba(0,0,0,0.5)",
        glass: "0 12px 40px rgba(0,0,0,0.7)",
      },
      dropShadow: {
        logo: "0 2px 10px rgba(0,0,0,0.8)",
        title: "0 2px 10px rgba(0,0,0,0.9)",
        meta: "0 1px 5px rgba(0,0,0,0.9)",
      },
    },
  },
  plugins: [],
};

export default config;
