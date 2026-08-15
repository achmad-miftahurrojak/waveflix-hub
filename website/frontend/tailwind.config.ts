import type { Config } from "tailwindcss";

const config: Config = {
  content: [
    "./app/**/*.{ts,tsx}",
    "./components/**/*.{ts,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        // Ganti nilai `accent` di sini kalau mau warna brand lain (mis. merah IDLIX).
        bg: "#0b0c10",
        surface: "#1f2833",
        accent: {
          DEFAULT: "#0EFFFF",
          dark: "#0BCCCC",
        },
      },
      fontFamily: {
        sans: ["var(--font-poppins)", "Segoe UI", "system-ui", "sans-serif"],
        body: ["var(--font-poppins)", "Segoe UI", "system-ui", "sans-serif"],
        logo: ["var(--font-logo)", "Impact", "Arial Narrow", "sans-serif"],
      },
      boxShadow: {
        card: "0 20px 30px rgba(0,0,0,0.7)",
      },
    },
  },
  plugins: [],
};

export default config;
