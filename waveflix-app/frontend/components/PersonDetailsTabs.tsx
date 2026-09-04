"use client";

import { useState } from "react";
import { motion } from "framer-motion";
import { useTranslation } from "@/lib/i18n";
import type { PersonCredit } from "@/lib/types";
import MovieCard from "./MovieCard";

interface Props {
  biography: string;
  knownForDept: string;
  birthday: string | null;
  placeOfBirth: string | null;
  filmography?: PersonCredit[];
}

export default function PersonDetailsTabs({ biography, knownForDept, birthday, placeOfBirth, filmography }: Props) {
  const [activeTab, setActiveTab] = useState<"bio" | "info" | "films">("bio");
  const { t } = useTranslation();

  const tabs = [
    { key: "bio" as const, label: t("person.biography") },
    { key: "films" as const, label: "Filmography" },
    { key: "info" as const, label: t("person.personalInfo") },
  ];

  return (
    <div>
      <div className="inline-flex bg-white/5 backdrop-blur-md border border-white/10 rounded-full p-1 mb-6 relative">
        {tabs.map((tab) => (
          <button
            key={tab.key}
            onClick={() => setActiveTab(tab.key)}
            className={`relative px-5 py-1.5 rounded-full text-xs font-bold transition-colors ${
              activeTab === tab.key ? "text-bg" : "text-white/70 hover:text-white"
            }`}
          >
            {activeTab === tab.key && (
              <motion.div
                layoutId="personTabMorph"
                className="absolute inset-0 bg-accent rounded-full shadow-[0_0_10px_rgba(14,255,255,0.3)]"
                transition={{ type: "spring", bounce: 0.2, duration: 0.6 }}
              />
            )}
            <span className="relative z-10">{tab.label}</span>
          </button>
        ))}
      </div>

      <div className="text-white/80">
        {activeTab === "bio" ? (
          <div>
            {biography ? (
              <div className="leading-relaxed text-sm whitespace-pre-wrap">
                {biography}
              </div>
            ) : (
              <p className="text-sm italic text-white/50">{t("person.noBiography")}</p>
            )}
          </div>
        ) : activeTab === "films" ? (
          <div>
            {filmography && filmography.length > 0 ? (
              <div className="no-scrollbar flex snap-x snap-mandatory gap-3 overflow-x-auto py-2">
                {filmography.map((credit, i) => (
                  <div key={`${credit.media_type}-${credit.id}-${i}`} className="w-[150px] shrink-0 snap-start">
                    <MovieCard item={credit} />
                  </div>
                ))}
              </div>
            ) : (
              <p className="text-sm italic text-white/50">No filmography available.</p>
            )}
          </div>
        ) : (
          <div className="space-y-4">
            <div>
              <p className="font-bold text-white text-xs mb-0.5 tracking-wider">{t("person.knownFor")}</p>
              <p className="text-sm">{knownForDept || "-"}</p>
            </div>
            {birthday && (
              <div>
                <p className="font-bold text-white text-xs mb-0.5 tracking-wider">{t("person.birthdate")}</p>
                <p className="text-sm">{birthday}</p>
              </div>
            )}
            {placeOfBirth && (
              <div>
                <p className="font-bold text-white text-xs mb-0.5 tracking-wider">{t("person.placeOfBirth")}</p>
                <p className="text-sm">{placeOfBirth}</p>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}