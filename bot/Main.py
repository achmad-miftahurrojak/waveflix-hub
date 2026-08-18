import asyncio
import atexit
import os
import sys
import importlib  # [REFACTOR] Standar industri untuk import dinamis
import math       # [REFACTOR] Digunakan untuk validasi latency (NaN)

# Tambahkan subfolder ke sys.path agar modul yang sudah dipindah tetap bisa diimpor
_BASE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, os.path.join(_BASE, 'bots'))
sys.path.insert(0, os.path.join(_BASE, 'utils'))
sys.path.insert(0, _BASE)

KUNCI = os.path.join(os.path.dirname(os.path.abspath(__file__)), '.arka.lock')

def _hidup_windows(pid):
    """Check apakah process masih hidup di Windows via ctypes."""
    import ctypes
    from ctypes import wintypes
    PROCESS_QUERY_LIMITED_INFORMATION = 4096
    STILL_ACTIVE = 259
    ERROR_ACCESS_DENIED = 5
    k32 = ctypes.WinDLL('kernel32', use_last_error=True)
    k32.OpenProcess.restype = wintypes.HANDLE
    k32.OpenProcess.argtypes = [wintypes.DWORD, wintypes.BOOL, wintypes.DWORD]
    k32.GetExitCodeProcess.restype = wintypes.BOOL
    k32.GetExitCodeProcess.argtypes = [wintypes.HANDLE, ctypes.POINTER(wintypes.DWORD)]
    k32.CloseHandle.argtypes = [wintypes.HANDLE]
    pegangan = k32.OpenProcess(PROCESS_QUERY_LIMITED_INFORMATION, False, pid)
    if not pegangan:
        return ctypes.get_last_error() == ERROR_ACCESS_DENIED
    try:
        kode = wintypes.DWORD()
        if not k32.GetExitCodeProcess(pegangan, ctypes.byref(kode)):
            return False
        return kode.value == STILL_ACTIVE
    finally:
        k32.CloseHandle(pegangan)

def _masih_hidup(pid):
    """Platform-agnostic process liveness check."""
    if pid <= 0:
        return False
    if os.name == 'nt':
        try:
            return _hidup_windows(pid)
        except Exception as e:
            print(f'[main] pengecekan pid gagal ({e}), dianggap masih hidup')
            return True
    try:
        os.kill(pid, 0)
    except ProcessLookupError:
        return False
    except PermissionError:
        return True
    except OSError:
        return False
    return True

def _tulis_kunci():
    with open(KUNCI, 'w') as f:
        f.write(str(os.getpid()))
    atexit.register(lepas_kunci)

def pasang_kunci():
    """Acquire process lock, takeover jika proses lama sudah mati."""
    try:
        fd = os.open(KUNCI, os.O_CREAT | os.O_EXCL | os.O_WRONLY)
        os.write(fd, str(os.getpid()).encode())
        os.close(fd)
        atexit.register(lepas_kunci)
        return
    except FileExistsError:
        pass
    try:
        with open(KUNCI) as f:
            isi = f.read().strip()
        pid = int(isi)
    except (ValueError, OSError):
        print('[main] file kunci rusak atau kosong, gua ambil alih')
        _tulis_kunci()
        return
    if pid == os.getpid():
        return
    if _masih_hidup(pid):
        print(f'[main] udah ada proses lain jalan (pid {pid}). keluar.')
        print(f'[main] kalau yakin ga ada, hapus dulu {KUNCI}')
        sys.exit(1)
    print(f'[main] kunci sisa dari proses {pid} yang udah mati. gua ambil alih.')
    _tulis_kunci()

def lepas_kunci():
    try:
        with open(KUNCI) as f:
            if f.read().strip() != str(os.getpid()):
                return
    except OSError:
        return
    try:
        os.remove(KUNCI)
    except OSError:
        pass

pasang_kunci()

# ── Bot Loading ──────────────────────────────────────────────────────────────

import penyimpanan

# Bot wajib (gagal = exit)
import botDirga
import botJulian

# Bot opsional (gagal = dilewat)
_OPSIONAL = [
    ('botArka', 'Arka'),
    ('botKevin', 'Kevin'),
    ('botWaveFlix', 'WaveFlix'),
    ('botTideTunes', 'TideTunes'),
]

def muatkan_bot(nama_modul, nama_tampil):
    """Load bot module, return None jika gagal."""
    try:
        # [REFACTOR] Diganti dari usangnya __import__ ke importlib 
        return importlib.import_module(nama_modul)
    except Exception as e:
        print(f'[main] {nama_tampil} dilewat: {e}')
        return None

# Dict: nama_tampil -> module (atau None)
bot_mods = {}
for _modul, _nama in _OPSIONAL:
    bot_mods[_nama] = muatkan_bot(_modul, _nama)

# ── Lifecycle ────────────────────────────────────────────────────────────────

async def nyalain(nama, bot, token):
    """Start satu bot. Berjalan selamanya sampai bot mati."""
    if not token:
        print(f'[main] {nama} dilewat, tokennya belum diisi di .env')
        return
    try:
        await bot.start(token)
    except Exception as e:
        print(f'[main] {nama} berhenti: {e}')

def _daftar_bot():
    """Kumpulkan semua bot yang aktif (wajib + opsional)."""
    hasil = [('Dirga', botDirga.bot), ('Julian', botJulian.bot)]
    for nama, mod in bot_mods.items():
        if mod and hasattr(mod, 'bot'):
            hasil.append((nama, mod.bot))
    return hasil

async def denyut():
    """Health check tiap 30 menit, lapor ke log kalau ada yang mati."""
    # [REFACTOR] Mengubah magic number 120 dan 1800 ke variabel
    JEDA_AWAL = 120
    JEDA_DENYUT = 1800
    
    await asyncio.sleep(JEDA_AWAL)
    while True:
        try:
            # [CRITICAL REFACTOR] File I/O bisa mem-blocking event loop. Kita lempar ke async thread.
            await asyncio.to_thread(penyimpanan.sapu_yang_rusak)
            
            daftar = _daftar_bot()
            baris, ada_yang_mati = ([], False)
            for nama, bot in daftar:
                hidup = bot.is_ready() and (not bot.is_closed())
                if not hidup:
                    ada_yang_mati = True
                
                jeda = bot.latency
                # [REFACTOR] Mengecek nan/None secara matematis logis.
                if jeda is not None and not math.isnan(jeda):
                    ping = f'{jeda * 1000:.0f} ms'
                else:
                    ping = '?'
                    
                baris.append(f"{('🟢' if hidup else '🔴')} **{nama}** — " + (f'nyambung, {ping}' if hidup else 'MATI'))
            print('[denyut] ' + ' | '.join((b.replace('**', '') for b in baris)))
            if ada_yang_mati:
                await lapor_denyut(baris)
        except Exception as e:
            print(f'[denyut] error: {e}')
        
        await asyncio.sleep(JEDA_DENYUT)

async def lapor_denyut(baris):
    import discord
    ch_id = getattr(botJulian, 'CHANNEL_LOG_ID', 0)
    if not ch_id:
        return
    channel = botJulian.bot.get_channel(ch_id)
    if channel is None:
        return
    embed = discord.Embed(title='⚠️ Ada bot yang mati', description='\n'.join(baris), color=discord.Color.red())
    embed.set_footer(text='Dicek tiap 30 menit. Restart Main.py buat nyalain lagi.')
    try:
        await channel.send(embed=embed)
    except Exception as e:
        print(f'[denyut] gagal lapor: {e}')

async def graceful_shutdown():
    """Tutup semua bot secara bersih."""
    print('\n[main] menutup semua bot dan membersihkan sesi (graceful shutdown)...')
    for nama, bot in _daftar_bot():
        try:
            if not bot.is_closed():
                await bot.close()
                print(f'[main] {nama} berhasil ditutup.')
        except Exception as e:
            print(f'[main] error saat menutup {nama}: {e}')

async def main():
    # [CRITICAL FIX] Membungkus ke try/finally untuk mencegah wavelink aiohttp memory leak
    try:
        # Jalankan clean up awal tanpa mem-block asyncio main loop
        await asyncio.to_thread(penyimpanan.sapu_yang_rusak)
        
        tugas = [nyalain('Dirga', botDirga.bot, botDirga.TOKEN),
                 nyalain('Julian', botJulian.bot, botJulian.TOKEN)]
        for nama, mod in bot_mods.items():
            if mod:
                tugas.append(nyalain(nama, mod.bot, mod.TOKEN))
                
        tugas.append(denyut())
        
        # return_exceptions=True agar crash 1 bot nggak menjatuhkan task bot lainnya (Cascading fail)
        await asyncio.gather(*tugas, return_exceptions=True)
    except asyncio.CancelledError:
        print('[main] Loop dibatalkan oleh sinyal.')
    finally:
        # [CRITICAL FIX] Pastikan fungsi ini dipanggil kapanpun bot exit!
        await graceful_shutdown()

if __name__ == '__main__':
    jumlah_bot = 2 + sum(1 for m in bot_mods.values() if m)
    print(f'[main] nyalain {jumlah_bot} bot sekaligus...')
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        print('\n[main] dimatiin manual (CTRL+C)')
    finally:
        # Pastikan lock file dibersihkan
        lepas_kunci()