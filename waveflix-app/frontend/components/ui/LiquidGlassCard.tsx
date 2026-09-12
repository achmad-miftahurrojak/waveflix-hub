"use client";

import React, { useRef } from "react";
import { Canvas, useFrame, useThree } from "@react-three/fiber";
import { MeshTransmissionMaterial, RoundedBox } from "@react-three/drei";
import * as THREE from "three";

interface GlassMeshProps {
  thickness?: number;
  roughness?: number;
  transmission?: number;
  ior?: number;
  chromaticAberration?: number;
  distortion?: number;
  distortionScale?: number;
  animated?: boolean;
}

function GlassMesh({
  thickness = 1.5,
  roughness = 0.05,
  transmission = 1.0,
  ior = 1.5,
  chromaticAberration = 0.04,
  distortion = 0.5,
  distortionScale = 0.5,
  animated = true,
}: GlassMeshProps) {
  const meshRef = useRef<THREE.Mesh>(null);
  const { viewport } = useThree();

  useFrame((state) => {
    if (animated && meshRef.current) {
      
      meshRef.current.rotation.x = Math.sin(state.clock.elapsedTime / 2) * 0.05;
      meshRef.current.rotation.y = Math.sin(state.clock.elapsedTime / 3) * 0.05;
      meshRef.current.position.y = Math.sin(state.clock.elapsedTime) * 0.05;
    }
  });

  return (
    <RoundedBox
      ref={meshRef}
      
      args={[viewport.width * 0.95, viewport.height * 0.95, 0.4]}
      radius={0.3}
      smoothness={16}
    >
      {/* @ts-ignore - MeshTransmissionMaterial typings can sometimes be tricky */}
      <MeshTransmissionMaterial
        backside={true}
        backsideThickness={1}
        samples={8}
        thickness={thickness}
        roughness={roughness}
        transmission={transmission}
        ior={ior}
        chromaticAberration={chromaticAberration}
        distortion={distortion}
        distortionScale={distortionScale}
        temporalDistortion={0.1}
        color="#ffffff"
      />
    </RoundedBox>
  );
}

interface LiquidGlassCardProps extends GlassMeshProps {
  children?: React.ReactNode;
  className?: string;
  canvasClassName?: string;
  containerStyle?: React.CSSProperties;
}

export function LiquidGlassCard({
  children,
  className = "",
  canvasClassName = "",
  containerStyle = {},
  ...glassProps
}: LiquidGlassCardProps) {
  return (
    <div 
      className={`relative flex items-center justify-center rounded-3xl ${className}`}
      style={containerStyle}
    >
      {}
      <div className={`absolute inset-0 -z-10 pointer-events-none rounded-3xl overflow-hidden ${canvasClassName}`}>
        <Canvas camera={{ position: [0, 0, 5], fov: 45 }}>
          <ambientLight intensity={1.5} />
          <directionalLight position={[10, 10, 10]} intensity={1} />
          <GlassMesh {...glassProps} />
        </Canvas>
      </div>

      {}
      <div className="relative z-10 w-full h-full p-8 flex flex-col border border-white/20 shadow-[0_4px_30px_rgba(0,0,0,0.1),inset_0_1px_0_rgba(255,255,255,0.3)] bg-white/5 rounded-[inherit]">
        {children}
      </div>
    </div>
  );
}
