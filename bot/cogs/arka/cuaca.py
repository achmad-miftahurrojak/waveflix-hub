import asyncio
from datetime import datetime
import discord
from discord import app_commands
from discord.ext import commands, tasks
from . import core
from .config import *
from .voice import suara

class Cuaca(commands.Cog):

    def __init__(self, bot):
        self.bot = bot
        self.umumin_cuaca.start()

    def cog_unload(self):
        self.umumin_cuaca.cancel()

    @app_commands.command(name='cuaca', description='Cuaca hari ini gimana?')
    async def cmd_cuaca(self, inter: discord.Interaction):
        if not await core.di_arena(inter):
            return
        c = core.cuaca_hari_ini()
        isi = discord.Embed(color=WARNA, title=f"{c['emoji']}  {c['nama']}", description=c['kata'])
        isi.add_field(name='Pengali XP', value=f"**x{c['kali']}**")
        isi.set_footer(text=f'Cuaca diundi tiap hari, diumumin jam {JAM_CUACA} WIB')
        await inter.response.send_message(embed=isi)

    @tasks.loop(seconds=600)
    async def umumin_cuaca(self):
        try:
            c = core.cuaca_hari_ini()
            jam = datetime.now(WIB).hour
            tujuan = CHANNEL_CUACA or CHANNEL_ARENA
            if tujuan and (not c.get('diumumkan')) and (jam >= JAM_CUACA):
                channel = self.bot.get_channel(tujuan)
                if channel:
                    isi = discord.Embed(color=WARNA, title=f"{c['emoji']}  Cuaca hari ini: {c['nama']}", description=c['kata'])
                    isi.add_field(name='Pengali XP', value=f"**x{c['kali']}**")
                    isi.set_footer(text='Jangan lupa /absen')
                    c['diumumkan'] = True
                    core.simpan()
                    try:
                        await channel.send(embed=isi)
                    except Exception:
                        c['diumumkan'] = False
                        core.simpan()
                        raise
        except Exception as e:
            print(f'[arka] pengumuman cuaca error: {e}')

    @umumin_cuaca.before_loop
    async def before_umumin_cuaca(self):
        await self.bot.wait_until_ready()

async def setup(bot):
    await bot.add_cog(Cuaca(bot))