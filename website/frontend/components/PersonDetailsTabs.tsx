"use client";

import { useState } from "react";
import { motion } from "framer-motion";
import { useTranslation } from "@/lib/i18n";

interface Props {
  biography: string;
  knownForDept: string;
  birthday: string | null;
  placeOfBirth: string | null;
}

export default function PersonDetailsTabs({ biography, knownForDept, birthday, placeOfBirth }: Props) {
  const [activeTab, setActiveTab] = useState<"bio" | "info">("bio");
  const { t } = useTranslation();

  return (
    <div>
      {}
      <div className="inline-flex bg-white/5 backdrop-blur-md border border-white/10 rounded-full p-1 mb-6 relative">
        {(["bio", "info"] as const).map((tab) => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab)}
            className={`relative px-5 py-1.5 rounded-full text-xs font-bold transition-colors ${
              activeTab === tab ? "text-bg" : "text-white/70 hover:text-white"
            }`}
          >
            {activeTab === tab && (
              <motion.div
                layoutId="personTabMorph"
                className="absolute inset-0 bg-accent rounded-full shadow-[0_0_10px_rgba(14,255,255,0.3)]"
                transition={{ type: "spring", bounce: 0.2, duration: 0.6 }}
              />
            )}
            <span className="relative z-10">
              {tab === "bio" ? t("person.biography") : t("person.personalInfo")}
            </span>
          </button>
        ))}
      </div>

      {}
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
