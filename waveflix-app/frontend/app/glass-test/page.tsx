import React from "react";
import { LiquidGlassCard } from "@/components/ui/LiquidGlassCard";

export const metadata = {
  title: "Liquid Glass Test",
};

export default function GlassTestPage() {
  return (
    <div className="relative min-h-screen w-full flex items-center justify-center bg-zinc-950 overflow-hidden text-white font-sans">
      {/* Background elements to refract */}
      <div className="absolute inset-0 z-0 overflow-hidden pointer-events-none">
        <div className="absolute top-1/4 left-1/4 w-[40vw] h-[40vw] bg-pink-500 rounded-full mix-blend-screen filter blur-[100px] opacity-60 animate-pulse" />
        <div className="absolute top-1/3 right-1/4 w-[30vw] h-[30vw] bg-blue-500 rounded-full mix-blend-screen filter blur-[100px] opacity-60" style={{ animation: "pulse 4s cubic-bezier(0.4, 0, 0.6, 1) infinite" }} />
        <div className="absolute bottom-1/4 left-1/3 w-[35vw] h-[35vw] bg-purple-500 rounded-full mix-blend-screen filter blur-[100px] opacity-60" style={{ animation: "pulse 3s cubic-bezier(0.4, 0, 0.6, 1) infinite" }} />
        
        {/* Some text behind to see the distortion */}
        <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 text-center pointer-events-none opacity-20 whitespace-nowrap">
          <h1 className="text-[15vw] font-bold leading-none tracking-tighter">WAVEFLIX</h1>
        </div>
      </div>

      {/* Glass Card */}
      <div className="z-10 w-full max-w-md p-6">
        <LiquidGlassCard 
          className="h-[400px]" 
          thickness={1.5}
          roughness={0.05}
          distortion={1.0}
        >
          <div className="flex flex-col h-full justify-between">
            <div className="flex justify-between items-start">
              <div>
                <h2 className="text-2xl font-semibold tracking-tight text-white drop-shadow-md">Premium</h2>
                <p className="text-white/70 text-sm mt-1">Waveflix Subscription</p>
              </div>
              <div className="w-10 h-10 rounded-full bg-white/20 flex items-center justify-center backdrop-blur-md">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="text-white">
                  <polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2" />
                </svg>
              </div>
            </div>
            
            <div className="mt-8">
              <div className="flex items-baseline gap-1">
                <span className="text-4xl font-bold tracking-tight">Rp 99.000</span>
                <span className="text-white/70">/bulan</span>
              </div>
              
              <ul className="mt-6 space-y-3">
                {['Kualitas 4K HDR', 'Tanpa iklan', 'Download untuk offline'].map((feature, i) => (
                  <li key={i} className="flex items-center gap-3 text-sm text-white/90">
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="text-green-400">
                      <polyline points="20 6 9 17 4 12" />
                    </svg>
                    {feature}
                  </li>
                ))}
              </ul>
            </div>
            
            <button className="mt-8 w-full py-3 bg-white text-black font-medium rounded-xl shadow-lg hover:bg-white/90 transition-colors">
              Pilih Paket
            </button>
          </div>
        </LiquidGlassCard>
      </div>
    </div>
  );
}
