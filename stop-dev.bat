@echo off
echo ========================================================
echo         Mematikan Semua Servis Waveflix...
echo ========================================================
echo.

echo Mematikan proses di Port 8000 (Python Vidlink Decryptor)...
FOR /F "tokens=5" %%T IN ('netstat -a -n -o ^| findstr :8000') DO taskkill /F /PID %%T 2>NUL

echo Mematikan proses di Port 8080 (Go Backend)...
FOR /F "tokens=5" %%T IN ('netstat -a -n -o ^| findstr :8080') DO taskkill /F /PID %%T 2>NUL

echo Mematikan proses di Port 3000 (Next.js App Frontend)...
FOR /F "tokens=5" %%T IN ('netstat -a -n -o ^| findstr :3000') DO taskkill /F /PID %%T 2>NUL

echo Mematikan proses di Port 3001 (Next.js Web)...
FOR /F "tokens=5" %%T IN ('netstat -a -n -o ^| findstr :3001') DO taskkill /F /PID %%T 2>NUL

echo.
echo ========================================================
echo  Semua servis di port tersebut telah dimatikan!
echo ========================================================
pause
