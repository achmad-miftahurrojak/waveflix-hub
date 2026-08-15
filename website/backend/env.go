package main

import (
	"bufio"
	"log"
	"os"
	"strings"
)

// loadDotEnv membaca file .env sederhana (KEY=VALUE per baris) dan meng-set
// variabel yang BELUM ada di environment. Tanpa dependency eksternal.
// Baris kosong & yang diawali '#' diabaikan. Tanda kutip di value dilepas.
func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		// Bukan error fatal: env var bisa juga di-set langsung dari shell.
		log.Printf("[env] %s tidak ditemukan, pakai environment shell saja", path)
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Dukung "export KEY=VALUE".
		line = strings.TrimPrefix(line, "export ")

		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		val = strings.Trim(val, `"'`)
		if key == "" {
			continue
		}
		// Env var dari shell menang atas file .env.
		if _, exists := os.LookupEnv(key); !exists {
			os.Setenv(key, val)
		}
	}
	if err := scanner.Err(); err != nil {
		log.Printf("[env] gagal baca %s: %v", path, err)
	}
}

// getenv mengambil env var dengan nilai default.
func getenv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
