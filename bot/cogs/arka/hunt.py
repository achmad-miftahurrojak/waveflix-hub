import time
import random
from datetime import datetime, timedelta
import discord
from discord import app_commands
from discord.ext import commands
from . import core
from .config import *
from .voice import suara

class Hunt(commands.Cog):

    def __init__(self, bot):
        self.bot = bot
        self._hunt_terakhir = {}

    @app_commands.command(name='hunt', description='Jelajah pantai, siapa tau nemu sesuatu')
    async def cmd_hunt(self, inter: discord.Interaction):
        if not await core.di_arena(inter):
            return
        catat = core.catatan(inter.user.id)
        efek = catat.setdefault('efek', {})
        tas = catat.setdefault('barang', {})
        sampai = efek.get('cepat_sampai')
        masih_cepat = sampai and datetime.fromisoformat(sampai) > datetime.now(WIB)
        if not masih_cepat and tas.get('cepat', 0) > 0:
            tas['cepat'] -= 1
            if not tas['cepat']:
                del tas['cepat']
            efek['cepat_sampai'] = (datetime.now(WIB) + timedelta(minutes=CEPAT_LAMA)).isoformat()
            masih_cepat = True
            core.simpan()
        jeda = CEPAT_JEDA if masih_cepat else HUNT_JEDA
        sekarang = time.time()
        lalu = self._hunt_terakhir.get(inter.user.id, 0)
        sisa = jeda - (sekarang - lalu)
        if sisa > 0:
            await inter.response.send_message(f'Sabar, pantainya baru lo obrak abrik. Tunggu {int(sisa)} detik lagi.', ephemeral=True)
            return
        self._hunt_terakhir[inter.user.id] = sekarang
        if not efek.get('kail', 0) and tas.get('kail', 0) > 0:
            tas['kail'] -= 1
            if not tas['kail']:
                del tas['kail']
            efek['kail'] = 2
        bobot = [t[2] for t in TANGKAPAN]
        pakai_kail = efek.get('kail', 0) > 0
        if pakai_kail:
            bobot = [b * KAIL_PENGALI if t[3] >= 90 else b for b, t in zip(bobot, TANGKAPAN)]
            efek['kail'] -= 1
            if not efek['kail']:
                del efek['kail']
        nama, emoji, _bobot, xp = random.choices(TANGKAPAN, weights=bobot)[0]
        tambahan = ''
        if not efek.get('umpan', 0) and tas.get('umpan', 0) > 0:
            tas['umpan'] -= 1
            if not tas['umpan']:
                del tas['umpan']
            efek['umpan'] = 3
        if efek.get('umpan', 0) > 0:
            efek['umpan'] -= 1
            xp *= 2
            tambahan = f"\n🪝 Umpan Emas: XP dobel. Sisa {efek['umpan']} jelajah."
            if not efek['umpan']:
                del efek['umpan']
        if pakai_kail:
            tambahan += f"\n🎣 Kail Perak kepakai, peluang langka naik. Sisa {efek.get('kail', 0)} jelajah."
        koleksi = catat.setdefault('koleksi', {})
        koleksi[nama] = koleksi.get(nama, 0) + 1
        langka = xp >= 150
        core.catat_main(inter.user.id, 'hunt', menang=langka)
        misi = self.bot.get_cog('Misi')
        if misi:
            misi.maju_misi(inter.user.id, 'hunt')
        await core.tambah_xp(self.bot, inter.user, xp)
        kunci = 'hunt_langka' if langka else 'hunt_sampah' if xp <= 10 else 'hunt_dapat'
        await inter.response.send_message(suara(kunci, e=emoji, n=nama, x=xp) + tambahan)

    @app_commands.command(name='koleksi', description='Liat semua yang pernah lo temuin')
    async def cmd_koleksi(self, inter: discord.Interaction, orang: discord.Member=None):
        if not await core.di_arena(inter):
            return
        orang = orang or inter.user
        koleksi = core.catatan(orang.id).get('koleksi', {})
        if not koleksi:
            await inter.response.send_message(f'{orang.display_name} belum pernah jelajah pantai. Coba `/hunt`.', ephemeral=True)
            return
        baris = []
        for nama, emoji, _b, _x in TANGKAPAN:
            jumlah = koleksi.get(nama, 0)
            if jumlah:
                baris.append(f'{emoji}  **{nama}** × {jumlah}')
            else:
                baris.append(f'⬛  ~~{nama}~~')
        isi = discord.Embed(color=WARNA, title=f'🏖️  Koleksi {orang.display_name}', description='\n'.join(baris))
        kurang = len(TANGKAPAN) - len(koleksi)
        isi.set_footer(text=f'{len(koleksi)} dari {len(TANGKAPAN)} jenis ketemu · total {sum(koleksi.values())} kali dapet\n' + ('Lengkap. Lencananya nempel di /level, pamer sana.' if kurang == 0 else f'Kurang {kurang} jenis lagi buat dapet lencana di /level'))
        await inter.response.send_message(embed=isi, ephemeral=True)

async def setup(bot):
    await bot.add_cog(Hunt(bot))