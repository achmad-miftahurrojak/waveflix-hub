const { execSync } = require('child_process');

const ports = [3000, 3001, 3005, 8000, 8081];

for (const port of ports) {
  try {
    const result = execSync(`netstat -ano | findstr :${port}`, { encoding: 'utf8', stdio: ['pipe', 'pipe', 'pipe'] });
    const lines = result.trim().split('\n');
    for (const line of lines) {
      const parts = line.trim().split(/\s+/);
      const pid = parts[parts.length - 1];
      if (pid && /^\d+$/.test(pid) && pid !== '0') {
        try {
          execSync(`taskkill /F /PID ${pid}`, { stdio: 'ignore' });
          console.log(`Killed PID ${pid} on port ${port}`);
        } catch (_) {}
      }
    }
  } catch (_) {}
}
console.log('Ports cleared.');
