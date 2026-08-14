import random
import discord
from discord import app_commands
from discord.ext import commands
from . import core
from .config import *
from .voice import suara

class Tebak(commands.Cog):

    def __init__(self, bot):
        self.bot = bot

    @app_commands.command(name='tebak', description='Tebak angka rahasia Arka, ada petunjuknya')
    @app_commands.describe(angka=f'Antara 1 sampai {TEBAK_MAKS}')
    async def cmd_tebak(self, inter: discord.Interaction, angka: int):
        if not await core.di_arena(inter):
            return
        if not 1 <= angka <= TEBAK_MAKS:
            await inter.response.send_message(f'Antara 1 sampai {TEBAK_MAKS} aja.', ephemeral=True)
            return
        sesi = core.sesi_tebak.get(inter.user.id)
        if not sesi:
            sesi = {'angka': random.randint(1, TEBAK_MAKS), 'sisa': TEBAK_KESEMPATAN}
            core.sesi_tebak[inter.user.id] = sesi
        sesi['sisa'] -= 1
        kepakai = TEBAK_KESEMPATAN - sesi['sisa']
        if angka == sesi['angka']:
            hadiah = 60 + (TEBAK_KESEMPATAN - kepakai) * 25
            core.catatan(inter.user.id)['menang'] += 1
            del core.sesi_tebak[inter.user.id]
            misi = self.bot.get_cog('Misi')
            if misi:
                misi.maju_misi(inter.user.id, 'tebak')
            await core.tambah_xp(self.bot, inter.user, hadiah)
            await inter.response.send_message('🎯 ' + suara('tebak_kena', x=angka, y=kepakai) + f'\nAmbil **{hadiah} XP**.')
            return
        if sesi['sisa'] <= 0:
            jawaban = sesi['angka']
            core.catatan(inter.user.id)['kalah'] += 1
            del core.sesi_tebak[inter.user.id]
            core.simpan()
            await inter.response.send_message(suara('tebak_habis', x=jawaban))
            return
        arah = '**lebih besar** ⬆️' if angka < sesi['angka'] else '**lebih kecil** ⬇️'
        if abs(angka - sesi['angka']) <= 5:
            arah += '  — tapi deket banget, hampir!'
        await inter.response.send_message(suara('tebak_meleset', x=angka, arah=arah) + f"\nSisa {sesi['sisa']} kesempatan.")

async def setup(bot):
    await bot.add_cog(Tebak(bot))