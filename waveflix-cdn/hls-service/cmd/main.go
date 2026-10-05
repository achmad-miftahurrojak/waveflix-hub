package main

import (
"log"
"os"

"github.com/waveflix-hub/hls-service/internal/server"
)

func main() {
port := os.Getenv("PORT")
if port == "" {
port = "8080"
}

log.Println("[HLS] WaveFlix HLS Service starting...")

hlsServer := server.NewHLSServer(port)

if err := hlsServer.Start(); err != nil {
log.Fatalf("[HLS] Failed to start server: %v", err)
}
}
