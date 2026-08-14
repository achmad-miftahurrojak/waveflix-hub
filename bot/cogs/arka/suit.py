import asyncio
import random
import discord
from discord import app_commands
from discord.ext import commands
from . import core
from .config import *
from .voice import suara

class SuitPapan(discord.ui.View):
    GAMBAR = {'batu': '🪨', 'kertas': '📄', 'gunting': '✂️'}
    KALAH_DARI = {'batu': 'gunting', 'kertas': 'batu', 'gunting': 'kertas'}
    PUTAR = ['🪨 . . .', '📄 . . .', '✂️ . . .']

    def __init__(self, pemain, bot_instance):
        super().__init__(timeout=90)
        self.pemain = pemain
        self.bot_instance = bot_instance
        self.ronde = 1
        self.skor_kamu = 0
        self.skor_aku = 0
        self.riwayat = []

    @staticmethod
    def hadiah_ke(beruntun):
        return 60 + min(max(beruntun, 1) - 1, 5) * 30

    def papan(self, judul, bawah=''):
        beruntun = core.catatan(self.pemain.id).get('suit_beruntun', 0)
        embed = discord.Embed(color=WARNA, title=judul, description=bawah)
        embed.add_field(name='Skor pertandingan ini', value=f'Lo **{self.skor_kamu}**  —  **{self.skor_aku}** gua\nmenang 2 duluan', inline=True)
        embed.add_field(name='Kalau menang', value=f'**{self.hadiah_ke(beruntun + 1):,} XP**' + (f'\nberuntun jadi {beruntun + 1}x' if beruntun else ''), inline=True)
        embed.add_field(name='Beruntun sekarang', value=('🔥 ' if beruntun else '') + (f'**{beruntun}** pertandingan' if beruntun else 'belum ada, ini yang pertama'), inline=True)
        if self.riwayat:
            embed.add_field(name='Ronde yang udah jalan', value='  '.join(self.riwayat) + '   🟢 lo menang · 🔴 gua menang · 🟡 seri', inline=False)
        return embed

    async def main(self, inter, kamu):
        if inter.user.id != self.pemain.id:
            await inter.response.send_message('Ini suitnya orang lain. Ketik `/suit` buat mulai punya lo.', ephemeral=True)
            return
        for tombol in self.children:
            tombol.disabled = True
        await inter.response.edit_message(embed=self.papan(f'{self.GAMBAR[kamu]}  lawan  {self.PUTAR[0]}'), view=self)
        for bingkai in self.PUTAR[1:]:
            await asyncio.sleep(0.7)
            await inter.edit_original_response(embed=self.papan(f'{self.GAMBAR[kamu]}  lawan  {bingkai}'), view=self)
        aku = random.choice(list(self.GAMBAR))
        await asyncio.sleep(0.5)
        if kamu == aku:
            hasil, tanda = ('Seri, ronde ini ga dihitung.', '🟡')
        elif self.KALAH_DARI[kamu] == aku:
            self.skor_kamu += 1
            hasil, tanda = ('Ronde ini lo yang menang.', '🟢')
        else:
            self.skor_aku += 1
            hasil, tanda = ('Ronde ini gua yang menang.', '🔴')
        self.riwayat.append(tanda)
        judul = f'{self.GAMBAR[kamu]}  lawan  {self.GAMBAR[aku]}'
        selesai = self.skor_kamu == 2 or self.skor_aku == 2
        if not selesai:
            if tanda != '🟡':
                self.ronde += 1
            for tombol in self.children:
                tombol.disabled = False
            await inter.edit_original_response(embed=self.papan(judul, hasil + ' Lanjut, pilih lagi.'), view=self)
            return
        await self.tutup(inter, judul, hasil)

    async def tutup(self, inter, judul, hasil):
        catat = core.catatan(self.pemain.id)
        beruntun = catat.get('suit_beruntun', 0)
        if self.skor_kamu > self.skor_aku:
            beruntun += 1
            catat['suit_beruntun'] = beruntun
            catat['menang'] += 1
            core.catat_main(self.pemain.id, 'suit', menang=True)
            hadiah = self.hadiah_ke(beruntun)
            akhir = f'**Pertandingan selesai {self.skor_kamu}–{self.skor_aku}, lo yang menang.** Ambil **{hadiah:,} XP**.'
            if beruntun >= 2:
                lanjut = self.hadiah_ke(beruntun + 1)
                akhir += f'\n🔥 Ini kemenangan beruntun ke-**{beruntun}**. Menang lagi dapet **{lanjut:,} XP**.'
            else:
                akhir += f'\nMenang lagi tanpa kalah, hadiahnya naik jadi **{self.hadiah_ke(2):,} XP**.'
            misi = self.bot_instance.get_cog('Misi')
            if misi:
                misi.maju_misi(self.pemain.id, 'suit')
            await core.tambah_xp(self.bot_instance, self.pemain, hadiah)
        else:
            catat['kalah'] += 1
            core.catat_main(self.pemain.id, 'suit')
            akhir = f'**Pertandingan selesai {self.skor_kamu}–{self.skor_aku}, gua yang menang.**'
            tas = catat.setdefault('barang', {})
            if beruntun >= 1 and tas.get('perisai', 0) > 0:
                tas['perisai'] -= 1
                if not tas['perisai']:
                    del tas['perisai']
                akhir += f"\n🛡️ **Perisai Beruntun kepakai.** Runtutan **{beruntun}** lo selamat. Sisa perisai: {tas.get('perisai', 0)}."
            else:
                catat['suit_beruntun'] = 0
                if beruntun >= 2:
                    akhir += f'\n💀 Runtutan **{beruntun} pertandingan** lo putus di sini. Balik ke 60 XP lagi.'
            core.simpan()
        for tombol in self.children:
            tombol.disabled = True
        await inter.edit_original_response(embed=self.papan(judul, hasil + '\n\n' + akhir), view=self)
        self.stop()

    @discord.ui.button(label='Batu', emoji='🪨', style=discord.ButtonStyle.secondary)
    async def batu(self, inter: discord.Interaction, tombol: discord.ui.Button):
        await self.main(inter, 'batu')

    @discord.ui.button(label='Kertas', emoji='📄', style=discord.ButtonStyle.secondary)
    async def kertas(self, inter: discord.Interaction, tombol: discord.ui.Button):
        await self.main(inter, 'kertas')

    @discord.ui.button(label='Gunting', emoji='✂️', style=discord.ButtonStyle.secondary)
    async def gunting(self, inter: discord.Interaction, tombol: discord.ui.Button):
        await self.main(inter, 'gunting')

class DuelTerima(discord.ui.View):

    def __init__(self, penantang, lawan):
        super().__init__(timeout=60)
        self.penantang = penantang
        self.lawan = lawan
        self.jawaban = None

    async def _cek(self, inter):
        if inter.user.id != self.lawan.id:
            await inter.response.send_message('Bukan lo yang ditantang.', ephemeral=True)
            return False
        return True

    async def _tutup(self, inter, jawaban):
        self.jawaban = jawaban
        for t in self.children:
            t.disabled = True
        await inter.response.edit_message(view=self)
        self.stop()

    @discord.ui.button(label='Terima', emoji='⚔️', style=discord.ButtonStyle.success)
    async def terima(self, inter: discord.Interaction, tombol: discord.ui.Button):
        if await self._cek(inter):
            await self._tutup(inter, 'terima')

    @discord.ui.button(label='Kabur', emoji='🏃', style=discord.ButtonStyle.secondary)
    async def kabur(self, inter: discord.Interaction, tombol: discord.ui.Button):
        if await self._cek(inter):
            await self._tutup(inter, 'kabur')

class DuelSerang(discord.ui.View):

    def __init__(self, penantang, lawan):
        super().__init__(timeout=20)
        self.boleh = {penantang.id, lawan.id}
        self.pemenang = None

    @discord.ui.button(label='SERANG!', emoji='🌊', style=discord.ButtonStyle.danger)
    async def serang(self, inter: discord.Interaction, tombol: discord.ui.Button):
        if inter.user.id not in self.boleh:
            await inter.response.send_message('Ini duel orang lain, jangan ikut campur.', ephemeral=True)
            return
        if self.pemenang:
            await inter.response.send_message('Telat.', ephemeral=True)
            return
        self.pemenang = inter.user
        tombol.disabled = True
        tombol.label = f'Direbut {inter.user.display_name}'
        tombol.style = discord.ButtonStyle.secondary
        await inter.response.edit_message(view=self)
        self.stop()

class BalonAir(discord.ui.View):

    def __init__(self):
        super().__init__(timeout=25)
        self.pemenang = None

    @discord.ui.button(label='LEMPAR!', emoji='💦', style=discord.ButtonStyle.primary)
    async def lempar(self, inter: discord.Interaction, tombol: discord.ui.Button):
        if self.pemenang:
            await inter.response.send_message(f'Telat, {self.pemenang.display_name} udah duluan.', ephemeral=True)
            return
        self.pemenang = inter.user
        tombol.disabled = True
        tombol.label = f'Diambil {inter.user.display_name}'
        tombol.style = discord.ButtonStyle.secondary
        await inter.response.edit_message(view=self)
        self.stop()

class Suit(commands.Cog):

    def __init__(self, bot):
        self.bot = bot
        self._balon_jalan = set()

    @app_commands.command(name='suit', description='Suit best of 3 lawan Arka')
    async def cmd_suit(self, inter: discord.Interaction):
        if not await core.di_arena(inter):
            return
        papan = SuitPapan(inter.user, self.bot)
        beruntun = core.catatan(inter.user.id).get('suit_beruntun', 0)
        pembuka = 'Siapa yang **menang 2 ronde duluan**, dia menang pertandingan. Kalau 2–0, ronde ketiga ga usah dimainin.'
        if beruntun >= 2:
            pembuka += f'\n\n🔥 Lo lagi beruntun **{beruntun} pertandingan**. Kalah sekali, hangus semua.'
        await inter.response.send_message(embed=papan.papan('✂️  Suit — Ronde 1', pembuka), view=papan)

    @app_commands.command(name='duel', description='Tantang member lain, adu cepat rebut XP')
    @app_commands.describe(lawan='Siapa yang mau ditantang', taruhan=f'XP yang dipertaruhin, {DUEL_MIN} sampai {DUEL_MAKS}')
    async def cmd_duel(self, inter: discord.Interaction, lawan: discord.Member, taruhan: int=DUEL_TARUHAN):
        if not await core.di_arena(inter):
            return
        if lawan.bot or lawan.id == inter.user.id:
            await inter.response.send_message('Cari lawan yang bener.', ephemeral=True)
            return
        if not DUEL_MIN <= taruhan <= DUEL_MAKS:
            await inter.response.send_message(f'Taruhannya antara **{DUEL_MIN}** sampai **{DUEL_MAKS:,} XP** aja. Jangan ngeruk abis punya orang.', ephemeral=True)
            return
        punya_kamu = core.catatan(inter.user.id)['xp']
        punya_lawan = core.catatan(lawan.id)['xp']
        if punya_kamu < taruhan or punya_lawan < taruhan:
            kurang = inter.user.display_name if punya_kamu < taruhan else lawan.display_name
            await inter.response.send_message(f'Taruhan {taruhan:,} XP kegedean. **{kurang}** ga punya sebanyak itu.', ephemeral=True)
            return
        tantang = DuelTerima(inter.user, lawan)
        await inter.response.send_message(suara('duel_mulai', a=inter.user.mention, b=lawan.mention, x=f'{taruhan:,}') + f'\n{lawan.mention}, berani ga?', view=tantang)
        pesan = await inter.original_response()
        if DUEL_DM:
            try:
                kabar = discord.Embed(color=WARNA, title='⚔️  Lo ditantang duel', description=f'**{inter.user.display_name}** nantangin lo di **{inter.guild.name}**.\nTaruhannya **{taruhan:,} XP**.')
                kabar.add_field(name='Buruan', value=f'Tantangannya cuma nunggu **60 detik**.\n[Buka tantangannya]({pesan.jump_url})')
                await lawan.send(embed=kabar)
            except discord.Forbidden:
                await inter.followup.send(f'DM {lawan.display_name} ketutup, jadi dia cuma kekabarin leakar mention di sini.', ephemeral=True)
            except Exception as e:
                print(f'[arka] DM duel gagal: {e}')
        await tantang.wait()
        if tantang.jawaban == 'kabur':
            await inter.followup.send(f'🏃 {lawan.display_name} kabur. {inter.user.display_name} menang tanpa keringetan, tapi ga dapet XP.')
            return
        if tantang.jawaban is None:
            await inter.followup.send(f'{lawan.display_name} ga nyaut. Duelnya batal.')
            return
        for angka in ['3', '2', '1']:
            await pesan.edit(content=f'⚔️ Duel diterima. **{angka}...**', view=None)
            await asyncio.sleep(1)
        await pesan.edit(content='⚔️ Siap siap...', view=None)
        await asyncio.sleep(random.uniform(1.5, 5))
        serang = DuelSerang(inter.user, lawan)
        await pesan.edit(content='🌊 **SERANG SEKARANG!**', view=serang)
        await serang.wait()
        if not serang.pemenang:
            await inter.followup.send('Dua duanya bengong, ga ada yang mencet. Duel batal, XP aman.')
            return
        menang = serang.pemenang
        kalah = lawan if menang.id == inter.user.id else inter.user
        core.catatan(menang.id)['xp'] += taruhan
        core.catatan(menang.id)['menang'] += 1
        core.catatan(kalah.id)['xp'] = max(0, core.catatan(kalah.id)['xp'] - taruhan)
        core.catatan(kalah.id)['kalah'] += 1
        core.catat_main(menang.id, 'duel', menang=True)
        core.catat_main(kalah.id, 'duel')
        misi = self.bot.get_cog('Misi')
        if misi:
            misi.maju_misi(menang.id, 'duel')
        core.simpan()
        await inter.followup.send(suara('duel_selesai', a=menang.display_name, b=kalah.display_name, x=f'{taruhan:,}'))

    @app_commands.command(name='balon', description='Perang balon air, adu cepet mencet')
    async def cmd_balon(self, inter: discord.Interaction):
        if not await core.di_arena(inter):
            return
        if inter.channel_id in self._balon_jalan:
            await inter.response.send_message('Masih ada ronde jalan, tunggu bentar.', ephemeral=True)
            return
        self._balon_jalan.add(inter.channel_id)
        try:
            await inter.response.send_message(suara('balon_siap'))
            pesan = await inter.original_response()
            await asyncio.sleep(random.uniform(*BALON_TUNGGU))
            papan = BalonAir()
            await pesan.edit(content='💦 **SEKARANG!** Cepetan mencet!', view=papan)
            await papan.wait()
            if papan.pemenang:
                core.catatan(papan.pemenang.id)['menang'] += 1
                misi = self.bot.get_cog('Misi')
                if misi:
                    misi.maju_misi(papan.pemenang.id, 'balon')
                await core.tambah_xp(self.bot, papan.pemenang, BALON_HADIAH)
                await inter.followup.send(suara('balon_menang', a=papan.pemenang.display_name, x=BALON_HADIAH))
            else:
                for anak in papan.children:
                    anak.disabled = True
                await pesan.edit(view=papan)
                await inter.followup.send(suara('balon_sepi'))
        finally:
            self._balon_jalan.discard(inter.channel_id)

async def setup(bot):
    await bot.add_cog(Suit(bot))