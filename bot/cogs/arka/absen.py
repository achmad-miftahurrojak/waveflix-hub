import discord
from discord import app_commands
from discord.ext import commands
from datetime import datetime, timedelta
from . import core
from .config import *
from .voice import suara

class Absen(commands.Cog):

    def __init__(self, bot):
        self.bot = bot

    @app_commands.command(name='absen', description='Absen harian, makin beruntun makin banyak')
    async def cmd_absen(self, inter: discord.Interaction):
        if not await core.di_arena(inter):
            return
        catat = core.catatan(inter.user.id)
        hari = core.hari_ini()
        if catat['absen_terakhir'] == hari:
            await inter.response.send_message('Udah absen hari ini. Besok lagi.', ephemeral=True)
            return
        kemarin = (datetime.now(WIB) - timedelta(days=1)).strftime('%Y-%m-%d')
        putus = catat['absen_terakhir'] != kemarin and catat['absen'] > 0
        tiket_kepakai = False
        tas_absen = catat.setdefault('barang', {})
        if putus and tas_absen.get('tiket', 0) > 0:
            tas_absen['tiket'] -= 1
            if not tas_absen['tiket']:
                del tas_absen['tiket']
            putus = False
            tiket_kepakai = True
            catat['absen_terakhir'] = kemarin
        catat['absen'] = catat['absen'] + 1 if catat['absen_terakhir'] == kemarin else 1
        catat['absen_terakhir'] = hari
        catat['absen_panjang'] = max(catat['absen_panjang'], catat['absen'])
        riwayat_absen = catat.setdefault('absen_riwayat', [])
        if hari not in riwayat_absen:
            riwayat_absen.append(hari)
        if len(riwayat_absen) > 400:
            del riwayat_absen[:-400]
        hadiah = 25 + min(catat['absen'], 14) * 5
        misi = self.bot.get_cog('Misi')
        if misi:
            misi.maju_misi(inter.user.id, 'absen')
        await core.tambah_xp(self.bot, inter.user, hadiah)
        kata = '☀️ ' + suara('absen', x=catat['absen'], y=hadiah)
        if tiket_kepakai:
            kata += '\n⛵ **Tiket Kapal kepakai.** Bolong kemarin ga dihitung, runtutan lo aman.'
        if putus:
            kata = suara('absen_putus') + '\n' + kata
        if catat['absen'] == 14:
            kata += '\nDua minggu penuh. Hadiahnya mentok di sini, tapi gua respek.'
        await inter.response.send_message(kata)

async def setup(bot):
    await bot.add_cog(Absen(bot))