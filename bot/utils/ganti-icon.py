import os, asyncio, discord
from dotenv import load_dotenv
SINI = os.path.dirname(os.path.abspath(__file__))
load_dotenv(os.path.join(SINI, '.env'))
FOLDER = os.path.join(SINI, 'icon-banner-app')
DAFTAR = [('Julian', os.getenv('JULIAN_TOKEN'), 'icon-julian.jpg'),
          ('Dirga', os.getenv('DIRGA_TOKEN'), 'icon-dirga.jpg'),
          ('Arka', os.getenv('ARKA_TOKEN'), 'icon-arka.jpg'),
          ('Kevin', os.getenv('KEVIN_TOKEN'), 'icon-kevin.jpg'),
          ('WaveFlix', os.getenv('WAVEFLIX_TOKEN'), 'icon-waveflix.png'),
          ('TideTunes', os.getenv('TIDETUNES_TOKEN'), 'icon-tidetunes.jpg'),
          ]

MAX_RETRY = 2
TIMEOUT = 30

async def ganti(nama, token, berkas):
    """Ganti avatar bot dengan retry dan timeout."""
    if not token:
        print(f'[{nama}] tokennya kosong di .env, dilewat')
        return False
    jalur = os.path.join(FOLDER, berkas)
    # Validasi file
    if not os.path.exists(jalur):
        print(f"[{nama}] file '{jalur}' ga ketemu")
        return False
    if not os.path.isfile(jalur):
        print(f"[{nama}] '{jalur}' bukan file")
        return False
    if os.path.getsize(jalur) == 0:
        print(f"[{nama}] file '{berkas}' kosong")
        return False
    try:
        with open(jalur, 'rb') as f:
            data = f.read()
    except (IOError, OSError) as e:
        print(f"[{nama}] gagal baca file: {e}")
        return False
    # Retry loop
    for attempt in range(MAX_RETRY):
        try:
            bot = discord.Client(intents=discord.Intents.default())
            berhasil = False

            @bot.event
            async def on_ready():
                nonlocal berhasil
                try:
                    await bot.user.edit(avatar=data)
                    berhasil = True
                    print(f'[{nama}] avatar diganti pakai {berkas}')
                except Exception as e:
                    print(f'[{nama}] gagal: {e}')
                await bot.close()

            await asyncio.wait_for(bot.start(token), timeout=TIMEOUT)
            return berhasil
        except asyncio.TimeoutError:
            try:
                await bot.close()
            except Exception:
                pass
            if attempt < MAX_RETRY - 1:
                tunggu = 2 ** attempt
                print(f'[{nama}] timeout (percobaan {attempt + 1}/{MAX_RETRY}), coba lagi dalam {tunggu}s...')
                await asyncio.sleep(tunggu)
            else:
                print(f'[{nama}] timeout setelah {MAX_RETRY} percobaan')
        except discord.LoginFailure:
            print(f'[{nama}] login gagal, token invalid')
            return False
        except Exception as e:
            try:
                await bot.close()
            except Exception:
                pass
            if attempt < MAX_RETRY - 1:
                print(f'[{nama}] error (percobaan {attempt + 1}/{MAX_RETRY}): {e}')
                await asyncio.sleep(1)
            else:
                print(f'[{nama}] error setelah {MAX_RETRY} percobaan: {e}')
    return False

async def main():
    hasil = []
    for nama, token, berkas in DAFTAR:
        sukses = await ganti(nama, token, berkas)
        hasil.append((nama, sukses))
    # Laporan ringkas
    berhasil = sum(1 for _, s in hasil if s)
    print(f'\n[ganti-icon] Ringkas: {berhasil}/{len(hasil)} berhasil')
    if berhasil < len(hasil):
        for nama, sukses in hasil:
            if not sukses:
                print(f'  ❌ {nama}')
    return all(s for _, s in hasil)

if __name__ == '__main__':
    try:
        sukses = asyncio.run(main())
        exit(0 if sukses else 1)
    except KeyboardInterrupt:
        print('\n[ganti-icon] dibatalkan manual')
        exit(130)