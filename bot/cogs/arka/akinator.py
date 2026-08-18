import random
import discord
from discord import app_commands
from discord.ext import commands
from . import core
from .config import *

def aki_kandidat(jawaban):
    hasil = []
    for nama, sifat in AKI_SUBJEK.items():
        punya = set(sifat)
        if all(((s in punya) == nilai for s, nilai in jawaban.items())):
            hasil.append(nama)
    return hasil

def aki_tanya_terbaik(kandidat, sudah):
    terbaik, skor_terbaik = (None, None)
    for sifat in AKI_SIFAT:
        if sifat in sudah:
            continue
        ya = sum((1 for n in kandidat if sifat in AKI_SUBJEK[n]))
        if ya == 0 or ya == len(kandidat):
            continue
        skor = abs(ya - len(kandidat) / 2)
        if skor_terbaik is None or skor < skor_terbaik:
            terbaik, skor_terbaik = (sifat, skor)
    return terbaik

class AkiBener(discord.ui.Button):

    def __init__(self, induk):
        super().__init__(label='Bener!', emoji='🎯', style=discord.ButtonStyle.success)
        self.induk = induk

    async def callback(self, inter: discord.Interaction):
        self.induk.stop()
        isi = discord.Embed(color=WARNA, title='🧞  Gua menang', description=f'**{self.induk.tebakan}**. Cuma butuh **{self.induk.jumlah} pertanyaan**.\nCoba lagi, bikin yang susah.')
        await inter.response.edit_message(embed=isi, view=None)
        core.catat_main(self.induk.pemain.id, 'akinator')

class AkiSalah(discord.ui.Button):

    def __init__(self, induk):
        super().__init__(label='Salah', emoji='🙅', style=discord.ButtonStyle.danger)
        self.induk = induk

    async def callback(self, inter: discord.Interaction):
        self.induk.stop()
        isi = discord.Embed(color=WARNA, title='🧞  Yah, meleset', description=f'Gua kira **{self.induk.tebakan}**. Ternyata bukan.\nNih **{AKI_HADIAH_MENANG} XP** buat lo, lo lebih pinter dari gua kali ini.')
        await inter.response.edit_message(embed=isi, view=None)
        core.catat_main(self.induk.pemain.id, 'akinator', menang=True)
        await core.tambah_xp(self.induk.bot_instance, self.induk.pemain, AKI_HADIAH_MENANG)

class AkiTanya(discord.ui.View):

    def __init__(self, pemain, bot_instance):
        super().__init__(timeout=AKI_WAKTU)
        self.pemain = pemain
        self.bot_instance = bot_instance
        self.jawaban = {}
        self.sudah = set()
        self.jumlah = 0
        self.tebakan = None
        self.pesan = None

    async def interaction_check(self, inter: discord.Interaction):
        if inter.user.id != self.pemain.id:
            await inter.response.send_message('Ini sesi orang lain.', ephemeral=True)
            return False
        return True

    def embed_tanya(self, sifat, kandidat):
        isi = discord.Embed(color=WARNA, title=f'🧞  Pertanyaan ke-{self.jumlah + 1}', description=f'## {AKI_SIFAT[sifat]}')
        isi.set_footer(text=f'Sisa {len(kandidat)} kemungkinan di kepala gua')
        return isi

    async def lanjut(self, inter):
        kandidat = aki_kandidat(self.jawaban)
        if len(kandidat) == 1:
            await self.tebak(inter, kandidat[0])
            return
        if not kandidat:
            await self.nyerah(inter, 'Ga ada yang cocok sama jawaban lo. Lo mikirin sesuatu yang belum gua kenal.')
            return
        sifat = aki_tanya_terbaik(kandidat, self.sudah)
        if sifat is None or self.jumlah >= AKI_MAKS_TANYA:
            await self.tebak(inter, random.choice(kandidat), kandidat)
            return
        self.sifat_sekarang = sifat
        self.jumlah += 1
        await inter.response.edit_message(embed=self.embed_tanya(sifat, kandidat), view=self)

    async def jawab(self, inter, nilai):
        if nilai is not None:
            self.jawaban[self.sifat_sekarang] = nilai
        self.sudah.add(self.sifat_sekarang)
        await self.lanjut(inter)

    @discord.ui.button(label='Ya', emoji='✅', style=discord.ButtonStyle.success)
    async def ya(self, inter, tombol):
        await self.jawab(inter, True)

    @discord.ui.button(label='Engga', emoji='❌', style=discord.ButtonStyle.danger)
    async def engga(self, inter, tombol):
        await self.jawab(inter, False)

    @discord.ui.button(label='Ga tau', emoji='🤷', style=discord.ButtonStyle.secondary)
    async def ga_tau(self, inter, tombol):
        await self.jawab(inter, None)

    async def tebak(self, inter, nama, kandidat=None):
        self.tebakan = nama
        self.clear_items()
        self.add_item(AkiBener(self))
        self.add_item(AkiSalah(self))
        isi = discord.Embed(color=WARNA, title='🧞  Gua tau!', description=f'Yang lo pikirin itu... **{nama}**?')
        isi.set_footer(text=f'Ketebak dalam {self.jumlah} pertanyaan' + (f' · sempet ada {len(kandidat)} kemungkinan' if kandidat and len(kandidat) > 1 else ''))
        await inter.response.edit_message(embed=isi, view=self)

    async def nyerah(self, inter, alasan):
        self.clear_items()
        self.stop()
        isi = discord.Embed(color=WARNA, title='🧞  Gua nyerah', description=alasan)
        isi.set_footer(text='Menang lo. Ajarin gua lain kali.')
        await inter.response.edit_message(embed=isi, view=self)
        await core.tambah_xp(self.bot_instance, self.pemain, AKI_HADIAH_MENANG)

    async def on_timeout(self):
        if self.pesan:
            try:
                await self.pesan.edit(view=None)
            except discord.HTTPException:
                pass

class Akinator(commands.Cog):

    def __init__(self, bot):
        self.bot = bot

    @app_commands.command(name='akinator', description='Pikirin satu benda atau hewan, gua tebak')
    async def cmd_akinator(self, inter: discord.Interaction):
        if not await core.di_arena(inter):
            return
        tampilan = AkiTanya(inter.user, self.bot)
        kandidat = list(AKI_SUBJEK)
        sifat = aki_tanya_terbaik(kandidat, set())
        # [BUG FIX] aki_tanya_terbaik bisa balikin None kalau semua sifat udah
        # terpakai/ga ngebantu. Jangan di-KEYError-kan di AKI_SIFAT[sifat].
        if sifat is None or sifat not in AKI_SIFAT:
            await inter.response.send_message('Sifat buat nebak lagi abis. Coba lagi lain kali.', ephemeral=True)
            return
        tampilan.sifat_sekarang = sifat
        tampilan.jumlah = 1
        pembuka = discord.Embed(color=WARNA, title='🧞  Pertanyaan ke-1', description=f'## {AKI_SIFAT[sifat]}')
        pembuka.set_footer(text=f'Sisa {len(kandidat)} kemungkinan di kepala gua')
        await inter.response.send_message(f'{inter.user.mention} pikirin satu **hewan atau benda sehari hari**, jangan dikasih tau. Gua tebak dalam {AKI_MAKS_TANYA} pertanyaan.', embed=pembuka, view=tampilan)
        tampilan.pesan = await inter.original_response()

async def setup(bot):
    await bot.add_cog(Akinator(bot))