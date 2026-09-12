@echo off
echo ========================================================
echo         Memulai Lingkungan Server Waveflix...
echo ========================================================
echo.

echo [1/4] Menjalankan Vidlink Decryptor (Port 8000)...
start "Vidlink Decryptor (Python)" cmd /k "cd /d .\Vidlink.pro-Decryptor && title Vidlink Decryptor && echo Menjalankan Decryptor... && python main.py"

timeout /t 2 >nul

echo [2/4] Menjalankan Backend Proxy (Port 8080)...
start "Waveflix Backend (Go)" cmd /k "cd /d .\waveflix-app\backend && title Waveflix Backend && echo Menjalankan Backend... && go run ."

timeout /t 2 >nul

echo [3/4] Menjalankan App Frontend (Port 3000)...
start "Waveflix App Frontend" cmd /k "cd /d .\waveflix-app\frontend && title Waveflix App Frontend && echo Menjalankan App Frontend... && npm run dev"

timeout /t 2 >nul

echo [4/4] Menjalankan Web (Port 3001)...
start "Waveflix Web" cmd /k "cd /d .\waveflix-web && title Waveflix Web && echo Menjalankan Web... && npm run dev -- -p 3001"

echo.
echo ========================================================
echo  Semua servis (4 Terminal) sedang berjalan di jendela baru!
echo  Silakan tutup jendela masing-masing jika ingin stop.
echo ========================================================
pause
