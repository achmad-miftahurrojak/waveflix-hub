import asyncio
import os
import discord
from discord.ext import commands, tasks
import random
from dotenv import load_dotenv
import cogs.arka.core as core
load_dotenv()
TOKEN = os.getenv('ARKA_TOKEN')
intents = discord.Intents.default()
intents.message_content = True
intents.members = True
intents.voice_states = True

class ArkaBot(commands.Bot):

    def __init__(self):
        super().__init__(command_prefix='!', intents=intents)
        self.cogs_loaded = False

    async def setup_hook(self):
        cogs = ['cogs.arka.xp', 'cogs.arka.cuaca', 'cogs.arka.absen', 'cogs.arka.tebak', 'cogs.arka.suit', 'cogs.arka.trivia', 'cogs.arka.lagu', 'cogs.arka.akinator', 'cogs.arka.turnamen', 'cogs.arka.hunt', 'cogs.arka.toko', 'cogs.arka.misi', 'cogs.arka.lomba_acara', 'cogs.arka.sosial', 'cogs.arka.cadangan', 'cogs.arka.panduan']
        core.muat()
        for cog in cogs:
            try:
                await self.load_extension(cog)
                print(f'[arka] Dimuat: {cog}')
            except Exception as e:
                print(f'[arka] Gagal memuat {cog}: {e}')
        await self.tree.sync()
        print('[arka] Slash commands disinkronkan.')
        self.cogs_loaded = True

    async def on_ready(self):
        print(f'[arka] Masuk sebagai {self.user.display_name}')
        if not ganti_status.is_running():
            ganti_status.start()
bot = ArkaBot()
status_list = ['Berani nantangin gua?', 'Lagi nunggu lawan sepadan...', 'Jangan lupa absen woy!', 'Ayo mabar grinding level!']

@tasks.loop(seconds=10)
async def ganti_status():
    if bot.is_closed():
        return
    status_terpilih = random.choice(status_list)
    try:
        await bot.change_presence(activity=discord.CustomActivity(name=status_terpilih))
    except Exception as e:
        if 'closing transport' not in str(e):
            print(f'[arka] gagal dipasang: {e}')

@ganti_status.before_loop
async def sebelum_ganti_status():
    await bot.wait_until_ready()
if __name__ == '__main__':
    if not TOKEN:
        print('[arka] ARKA_TOKEN belum diisi di .env')
    else:
        bot.run(TOKEN)
