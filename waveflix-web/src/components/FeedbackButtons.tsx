"use client";

import { useState } from "react";

export default function FeedbackButtons() {
  const [submitted, setSubmitted] = useState(false);

  if (submitted) {
    return (
      <div className="mt-16 pt-8 border-t border-netflix-divider">
        <h3 className="font-bold text-lg mb-4 text-green-700">Terima kasih atas tanggapan Anda.</h3>
      </div>
    );
  }

  return (
    <div className="mt-16 pt-8 border-t border-netflix-divider">
      <h3 className="font-bold text-lg mb-4">Apakah artikel ini membantu?</h3>
      <div className="flex gap-4">
        <button 
          onClick={() => setSubmitted(true)}
          className="px-8 py-3 border border-gray-400 rounded hover:bg-gray-50 font-semibold transition"
        >
          Ya
        </button>
        <button 
          onClick={() => setSubmitted(true)}
          className="px-8 py-3 border border-gray-400 rounded hover:bg-gray-50 font-semibold transition"
        >
          Tidak
        </button>
      </div>
    </div>
  );
}
