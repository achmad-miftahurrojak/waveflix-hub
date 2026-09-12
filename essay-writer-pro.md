---
name: essay-writer-pro
description: Use when drafting, structuring, atau menulis essay/personal statement akademik baru berdasarkan latar belakang, CV, atau draf awal milik user.
---

# Essay Writer Pro

## Overview
Skill ini mengubah informasi mentah, sejarah/latar belakang, dan draf awal milik user menjadi essay akademik yang sangat profesional dan tertarget. Skill ini mewajibkan pendekatan *research-first* (Deep Search) sebelum penulisan dimulai agar narasi selaras dengan detail spesifik dari program tujuan.

## When to Use
- User memberikan CV, latar belakang, atau ide mentah dan meminta dibuatkan essay.
- User memberikan draf essay yang masih lemah dan meminta perombakan total (rewrite).
- User menyebutkan target universitas/lab/profesor dan meminta latar belakangnya disesuaikan dengan target tersebut.

**Kapan TIDAK digunakan:**
- Untuk koreksi grammar ringan atau sekadar *fact-check* draf yang sudah final (gunakan skill `scholarship-essay-proofread`).

## Core Pattern: Research-Driven Drafting

**ATURAN KRITIS:** JANGAN PERNAH langsung menulis essay. Anda wajib mengikuti langkah-langkah ini secara berurutan.

### 1. Ingesti Konteks & Sejarah
Pelajari latar belakang, riwayat, atau draf essay saat ini yang diberikan user. Identifikasi pencapaian utama, *mindset*, serta target program dan universitas.

### 2. Mandatory Deep Search (Wajib)
Sebelum menyusun kerangka, Anda WAJIB menggunakan tool pencarian web untuk mencari detail spesifik dan terbaru tentang target:
- **Profesor:** Publikasi terbaru, minat riset saat ini.
- **Laboratorium:** Proyek yang sedang berjalan, fokus spesifik (misal: *Physical AI*, *Embedded Systems*).
- **Kurikulum/Jurusan:** Mata kuliah unik, fasilitas, atau visi misi program yang selaras dengan profil user.

### 3. Structural Alignment (Kerangka Narasi)
Sajikan kerangka singkat yang secara eksplisit menghubungkan sejarah user dengan temuan *Deep Search*.
- ❌ **Buruk:** "Saya akan menulis tentang proyek Arduino Anda dan alasan Anda menyukai universitas tersebut."
- ✅ **Baik:** "Saya akan menghubungkan proyek *embedded* Arduino Anda dengan paper terbaru Prof. X tentang *Edge AI* di lab Y, untuk menunjukkan kesiapan riset yang langsung relevan."

### 4. Eksekusi Profesional
Mulai tulis draf essay **HANYA** setelah user menyetujui kerangka narasi tersebut.

## Quick Reference: Standar Penulisan Profesional

| Aspek | Aturan Emas |
|---|---|
| **Bukti (Evidence)** | *Show, don't tell.* Gunakan metrik, angka, dan nama proyek spesifik (bukan sekadar kata sifat). |
| **Nada (Tone)** | Percaya diri, akademik, dan lugas. Bebas dari hiperbola, drama, dan kata-kata klise AI. |
| **Relevansi (Fit)** | Setiap kalimat harus membuktikan kecocokan user dengan program/lab spesifik yang dituju. |
| **Otentisitas** | Pertahankan suara asli user, namun tingkatkan level profesionalismenya. |

## Red Flags - STOP dan Ulangi

- Mulai menulis paragraf draf **sebelum** melakukan *deep search* terhadap universitas/lab/profesor target.
- Menggunakan frasa generik pengisi ruang kosong (contoh: "fasilitas kelas dunia", "dosen ternama") tanpa menyebut spesifikasinya.
- Mengarang (halusinasi) fokus riset profesor karena malas mencari di web.

**Jika Red Flags ini terjadi: Berhenti menulis. Kembali ke fase Deep Search.**
