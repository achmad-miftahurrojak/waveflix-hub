import os, asyncio, discord
from dotenv import load_dotenv
SINI = os.path.dirname(os.path.abspath(__file__))
load_dotenv(os.path.join(SINI, '.env'))
FOLDER = os.path.join(SINI, 'icon-banner-app')
DAFTAR = [('Julian', os.getenv('JULIAN_TOKEN'), 'banner-bot.gif'),
           ('Dirga', os.getenv('DIRGA_TOKEN'), 'banner-bot.gif'),
           ('Arka', os.getenv('ARKA_TOKEN'), 'banner-bot.gif'),
           ('Kevin', os.getenv('KEVIN_TOKEN'), 'banner-bot.gif'),
           ('WaveFlix', os.getenv('WAVEFLIX_TOKEN'), 'banner-app.gif'),
           ('TideTunes', os.getenv('TIDETUNES_TOKEN'), 'banner-app.gif'),
           ]

async def ganti(nama, token, berkas):
    if not token:
        print(f'[{nama}] tokennya kosong di .env, dilewat')
        return
    jalur = os.path.join(FOLDER, berkas)
    if not os.path.exists(jalur):
        print(f"[{nama}] file '{jalur}' ga ketemu")
        return
    with open(jalur, 'rb') as f:
        data = f.read()
    print(f'[{nama}] ukuran file: {len(data) / 1024 / 1024:.1f} MB')
    bot = discord.Client(intents=discord.Intents.default())

    @bot.event
    async def on_ready():
        try:
            await bot.user.edit(banner=data)
            print(f'[{nama}] banner diganti pakai {berkas}')
        except Exception as e:
            print(f'[{nama}] gagal: {e}')
        await bot.close()
    await bot.start(token)

async def main():
    for nama, token, berkas in DAFTAR:
        await ganti(nama, token, berkas)
asyncio.run(main())