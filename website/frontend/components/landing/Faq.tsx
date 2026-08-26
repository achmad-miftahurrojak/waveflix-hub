"use client";

import { useState } from "react";
import EmailForm from "@/components/EmailForm";

const faqs = [
  {
    q: "Apa itu Waveflix?",
    a: "Waveflix adalah layanan streaming dengan ribuan film dan serial TV yang bisa kamu tonton kapan saja. Tonton di ponsel, tablet, laptop, atau TV dengan satu akun.",
  },
  {
    q: "Di mana saya bisa menonton?",
    a: "Tonton di mana saja, kapan saja. Masuk lewat web, atau di aplikasi iOS, Android, smart TV, dan konsol game.",
  },
  {
    q: "Apa yang bisa saya tonton di Waveflix?",
    a: "Koleksi film fitur, dokumenter, serial, anime, dan banyak lagi yang terus bertambah. Judul baru hadir setiap minggu.",
  },
];

export default function Faq() {
  const [open, setOpen] = useState<number | null>(0);

  return (
    <section id="faq" className="mx-auto max-w-3xl px-6 py-16">
      <h2 className="mb-6 text-center text-2xl font-bold text-white md:text-3xl">
        Pertanyaan yang Sering Diajukan
      </h2>
      <div className="space-y-2">
        {faqs.map((f, i) => {
          const isOpen = open === i;
          return (
            <div
              key={i}
              className={`rounded border-none px-5 transition ${
                isOpen ? "bg-white/10" : "bg-surface"
              }`}
            >
              <button
                onClick={() => setOpen(isOpen ? null : i)}
                aria-expanded={isOpen}
                className="flex w-full items-center justify-between gap-4 py-5 text-left text-base font-semibold text-white md:text-lg"
              >
                {f.q}
                <svg
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2.5"
                  className={`h-5 w-5 shrink-0 transition-transform duration-300 ${
                    isOpen ? "rotate-90" : ""
                  }`}
                >
                  <path strokeLinecap="round" strokeLinejoin="round" d="M9 5l7 7-7 7" />
                </svg>
              </button>
              <div
                className={`grid transition-all duration-300 ease-out ${
                  isOpen ? "grid-rows-[1fr] pb-5 opacity-100" : "grid-rows-[0fr] opacity-0"
                }`}
              >
                <p className="overflow-hidden text-sm leading-relaxed text-white/80 md:text-base">
                  {f.a}
                </p>
              </div>
            </div>
          );
        })}
      </div>

      <div className="mt-12 text-center">
        <p className="text-white/80">
          Siap menonton? Masukkan email untuk mulai menonton.
        </p>
        <EmailForm />
      </div>
    </section>
  );
}
