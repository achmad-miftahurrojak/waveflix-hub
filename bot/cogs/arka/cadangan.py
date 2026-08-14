import asyncio
import os
from datetime import datetime
import discord
from discord import app_commands
from discord.ext import commands, tasks
from . import core
from .config import *

def _berkas_cadangan():
    siap, hilang = ([], [])
    for nama in BACKUP_BERKAS:
        jalur = os.path.join(SINI, nama)
        if os.path.exists(jalur):
            siap.append((nama, jalur, os.path.getsize(jalur)))
        else:
            hilang.append(nama)
    return (siap, hilang)

async def kirim_cadangan(tujuan, manual=False):
    siap, hilang = _berkas_cadangan()
    if not siap:
        return False
    total = sum((u for _n, _j, u in siap))
    isi = discord.Embed(color=WARNA, title='💾  Cadangan data', description='Simpan file ini. Kalau hosting bermasalah, tinggal upload balik ke folder bot.')
    isi.add_field(name=f'{len(siap)} file · {total / 1024:.0f} KB', value='\n'.join((f'`{n}` — {u / 1024:.0f} KB' for n, _j, u in siap)), inline=False)
    if hilang:
        isi.add_field(name='Ga ketemu', value=', '.join((f'`{n}`' for n in hilang)), inline=False)
    isi.set_footer(text=('Dikirim manual' if manual else 'Cadangan harian otomatis') + ' · .env sengaja ga pernah ikut')
    berkas = [discord.File(j, filename=n) for n, j, _u in siap]
    await tujuan.send(embed=isi, files=berkas)
    return True

class Cadangan(commands.Cog):

    def __init__(self, bot):
        self.bot = bot
        self.jaga_cadangan.start()

    def cog_unload(self):
        self.jaga_cadangan.cancel()

    @tasks.loop(minutes=10)
    async def jaga_cadangan(self):
        try:
            hari = core.hari_ini()
            jam = datetime.now(WIB).hour
            if core.data.get('backup_terakhir') == hari or jam < JAM_BACKUP:
                return
            tujuan = None
            if CHANNEL_CADANGAN:
                tujuan = self.bot.get_channel(CHANNEL_CADANGAN)
            if tujuan is None:
                for guild in self.bot.guilds:
                    if guild.owner:
                        tujuan = guild.owner
                        break
            if tujuan and await kirim_cadangan(tujuan):
                core.data['backup_terakhir'] = hari
                core.simpan()
                print(f'[arka] cadangan harian kekirim ke {tujuan}')
        except discord.Forbidden:
            print('[arka] cadangan gagal kekirim. Cek izin Attach Files di channel cadangan, atau DM pemilik ketutup.')
            core.data['backup_terakhir'] = core.hari_ini()
            core.simpan()
        except Exception as e:
            print(f'[arka] cadangan error: {e}')

    @jaga_cadangan.before_loop
    async def before_jaga_cadangan(self):
        await self.bot.wait_until_ready()

    @app_commands.command(name='backup', description='Kirim cadangan data sekarang juga')
    async def cmd_backup(self, inter: discord.Interaction):
        if not await core.di_arena(inter):
            return
        if ROLE_SOAL:
            punya = {r.name for r in getattr(inter.user, 'roles', [])}
            if not punya & set(ROLE_SOAL):
                await inter.response.send_message('Yang boleh minta cadangan cuma ' + ' atau '.join(ROLE_SOAL) + '.', ephemeral=True)
                return
        await inter.response.send_message('Bentar, gua bungkus dulu...', ephemeral=True)
        tujuan = self.bot.get_channel(CHANNEL_CADANGAN) if CHANNEL_CADANGAN else None
        ke_dm = tujuan is None
        if ke_dm:
            tujuan = inter.user
        try:
            if await kirim_cadangan(tujuan, manual=True):
                kemana = 'ke DM lo' if ke_dm else f'ke <#{CHANNEL_CADANGAN}>'
                await inter.followup.send(f'Udah gua kirim {kemana}.', ephemeral=True)
            else:
                await inter.followup.send('Ga ada file data yang ketemu.', ephemeral=True)
        except discord.Forbidden:
            await inter.followup.send('Gua ga punya izin ngirim ke situ. Cek izin Attach Files di channel cadangan.', ephemeral=True)

async def setup(bot):
    await bot.add_cog(Cadangan(bot))