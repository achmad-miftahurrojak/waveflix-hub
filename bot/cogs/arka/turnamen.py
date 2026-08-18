import asyncio
import random
import discord
from discord import app_commands
from discord.ext import commands
from . import core
from .config import *
KALAHIN = {'batu': 'gunting', 'gunting': 'kertas', 'kertas': 'batu'}
EMOJI_SUIT = {'batu': '🪨', 'gunting': '✂️', 'kertas': '📄'}
turnamen_jalan = {}

class DaftarTurnamen(discord.ui.View):

    def __init__(self, pembuat, label='Suit'):
        super().__init__(timeout=TURNAMEN_DAFTAR_MENIT * 60)
        self.pembuat = pembuat
        self.label = label
        self.peserta = []
        self.mulai = asyncio.Event()
        self.pesan = None

    @discord.ui.button(label='Ikut', emoji='✋', style=discord.ButtonStyle.primary)
    async def ikut(self, inter: discord.Interaction, tombol: discord.ui.Button):
        if inter.user in self.peserta:
            self.peserta.remove(inter.user)
            await inter.response.send_message('Oke, lo gua coret.', ephemeral=True)
        elif len(self.peserta) >= TURNAMEN_MAKS:
            await inter.response.send_message(f'Udah penuh, {TURNAMEN_MAKS} orang.', ephemeral=True)
            return
        else:
            self.peserta.append(inter.user)
            await inter.response.send_message('Kedaftar. Pencet lagi kalau mau batal.', ephemeral=True)
        if self.pesan:
            try:
                await self.pesan.edit(embed=self.embed())
            except discord.HTTPException:
                pass

    @discord.ui.button(label='Mulai Sekarang', emoji='▶️', style=discord.ButtonStyle.success)
    async def mulai_sekarang(self, inter: discord.Interaction, tombol: discord.ui.Button):
        if inter.user.id != self.pembuat.id:
            await inter.response.send_message('Cuma yang buka turnamen yang bisa mulai.', ephemeral=True)
            return
        if len(self.peserta) < TURNAMEN_MIN:
            await inter.response.send_message(f'Minimal {TURNAMEN_MIN} peserta.', ephemeral=True)
            return
        await inter.response.defer()
        self.mulai.set()
        self.stop()

    def embed(self):
        daftar = '\n'.join((f'`{i}.` {p.display_name}' for i, p in enumerate(self.peserta, 1)))
        isi = discord.Embed(color=WARNA, title=f'🏆  Turnamen {self.label} — pendaftaran dibuka', description=f'Pencet **Ikut** buat daftar.\nSistem gugur, sekali kalah langsung pulang.\n\n**Peserta ({len(self.peserta)}/{TURNAMEN_MAKS})**\n' + (daftar or '_belum ada yang berani_'))
        isi.add_field(name='Hadiah', value=f'🥇 {TURNAMEN_HADIAH[0]} XP · 🥈 {TURNAMEN_HADIAH[1]} XP · 🥉 {TURNAMEN_HADIAH[2]} XP', inline=False)
        isi.set_footer(text=f'Nutup sendiri {TURNAMEN_DAFTAR_MENIT} menit lagi')
        return isi

class PilihSuit(discord.ui.View):

    def __init__(self, a, b):
        super().__init__(timeout=TURNAMEN_PILIH_DETIK)
        self.a, self.b = (a, b)
        self.pilihan = {}
        self.pesan = None

    async def interaction_check(self, inter: discord.Interaction):
        if inter.user.id not in (self.a.id, self.b.id):
            await inter.response.send_message('Lo bukan yang lagi tanding. Nonton aja.', ephemeral=True)
            return False
        return True

    async def catat(self, inter, pilih):
        if inter.user.id in self.pilihan:
            await inter.response.send_message('Lo udah milih.', ephemeral=True)
            return
        self.pilihan[inter.user.id] = pilih
        await inter.response.send_message(f'Oke, **{pilih}**. Jangan bocorin.', ephemeral=True)
        if len(self.pilihan) == 2:
            self.stop()
        elif self.pesan:
            nunggu = self.b if inter.user.id == self.a.id else self.a
            try:
                await self.pesan.edit(content=f'⚔️ **{self.a.display_name}** vs **{self.b.display_name}**\nNunggu {nunggu.display_name}...')
            except discord.HTTPException:
                pass

    @discord.ui.button(label='Batu', emoji='🪨')
    async def batu(self, inter, tombol):
        await self.catat(inter, 'batu')

    @discord.ui.button(label='Gunting', emoji='✂️')
    async def gunting(self, inter, tombol):
        await self.catat(inter, 'gunting')

    @discord.ui.button(label='Kertas', emoji='📄')
    async def kertas(self, inter, tombol):
        await self.catat(inter, 'kertas')

class Turnamen(commands.Cog):

    def __init__(self, bot):
        self.bot = bot
        self.JENIS_TURNAMEN = {'suit': ('Suit', self.main_satu_ronde), 'trivia': ('Trivia', self.ronde_trivia), 'lagu': ('Tebak Lagu', self.ronde_lagu)}

    async def main_satu_ronde(self, channel, a, b):
        for percobaan in range(3):
            tampilan = PilihSuit(a, b)
            pesan = await channel.send(f'⚔️ **{a.display_name}** vs **{b.display_name}**\nPilih dalam {TURNAMEN_PILIH_DETIK} detik.' + ('\n_Seri, ulang._' if percobaan else ''), view=tampilan)
            tampilan.pesan = pesan
            await tampilan.wait()
            for anak in tampilan.children:
                anak.disabled = True
            try:
                await pesan.edit(view=tampilan)
            except discord.HTTPException:
                pass
            pa = tampilan.pilihan.get(a.id)
            pb = tampilan.pilihan.get(b.id)
            if pa is None and pb is None:
                menang = random.choice([a, b])
                await channel.send(f'Dua duanya diem aja. Gua lempar koin — **{menang.display_name}** yang lanjut.')
                return (menang, b if menang == a else a)
            if pa is None:
                await channel.send(f'**{a.display_name}** ga milih. **{b.display_name}** lanjut.')
                return (b, a)
            if pb is None:
                await channel.send(f'**{b.display_name}** ga milih. **{a.display_name}** lanjut.')
                return (a, b)
            hasil = f'{EMOJI_SUIT[pa]} **{pa}** vs **{pb}** {EMOJI_SUIT[pb]}'
            if pa == pb:
                await channel.send(f'{hasil} — seri.')
                continue
            if KALAHIN[pa] == pb:
                await channel.send(f'{hasil}\n🌊 **{a.display_name}** menang.')
                return (a, b)
            await channel.send(f'{hasil}\n🌊 **{b.display_name}** menang.')
            return (b, a)
        menang = random.choice([a, b])
        await channel.send(f'Seri terus tiga kali. **{menang.display_name}** lanjut karena gua males.')
        return (menang, b if menang == a else a)

    async def ronde_trivia(self, channel, a, b):
        from .trivia import soal, _cocok
        if not soal:
            # [BUG FIX] 'b if a else a' selalu ngasilin b (a selalu truthy).
            # Dikoin sama kayak timeout, kalahnya pasangan yang satunya.
            menang = random.choice([a, b])
            return (menang, b if menang == a else a)
        pilih = random.choice(soal)
        jawaban = pilih['jawaban']
        isi = discord.Embed(color=WARNA, title=f'❓ {a.display_name} vs {b.display_name}', description=pilih['soal'])
        isi.set_footer(text='Ketik jawabannya di chat. Cuma dua orang ini yang dihitung. 40 detik.')
        await channel.send(embed=isi)

        def bener(pesan):
            return pesan.channel.id == channel.id and pesan.author.id in (a.id, b.id) and _cocok(pesan.content, jawaban)
        try:
            pesan = await self.bot.wait_for('message', check=bener, timeout=40)
        except asyncio.TimeoutError:
            menang = random.choice([a, b])
            await channel.send(f'⏰ Waktu habis, ga ada yang bener. Jawabannya **{jawaban[0]}**.\nGua lempar koin — **{menang.display_name}** yang lanjut.')
            return (menang, b if menang == a else a)
        menang = a if pesan.author.id == a.id else b
        kalah = b if menang == a else a
        await channel.send(f'✅ **{menang.display_name}** duluan! Jawabannya **{jawaban[0]}**.')
        return (menang, kalah)

    async def ronde_lagu(self, channel, a, b):
        from .lagu import lagu_bank
        from .trivia import _cocok
        if not lagu_bank:
            # [BUG FIX] Sama kayak ronde_trivia: 'b if a else a' selalu b.
            menang = random.choice([a, b])
            return (menang, b if menang == a else a)
        pilih = random.choice(lagu_bank)
        jawaban = [x.strip() for x in pilih['judul'].split(',') if x.strip()]
        isi = discord.Embed(color=WARNA, title=f'🎵 {a.display_name} vs {b.display_name}', description=f"# {pilih['emoji']}\n{pilih['tahun']} · {pilih['genre']} · {pilih['penyanyi']}")
        isi.set_footer(text='Tebak judulnya di chat. 40 detik.')
        await channel.send(embed=isi)

        def bener(pesan):
            return pesan.channel.id == channel.id and pesan.author.id in (a.id, b.id) and _cocok(pesan.content, jawaban)
        try:
            pesan = await self.bot.wait_for('message', check=bener, timeout=40)
        except asyncio.TimeoutError:
            menang = random.choice([a, b])
            await channel.send(f'⏰ Ga ada yang tau. Jawabannya **{jawaban[0]}**.\n**{menang.display_name}** lanjut lewat koin.')
            return (menang, b if menang == a else a)
        menang = a if pesan.author.id == a.id else b
        kalah = b if menang == a else a
        await channel.send(f'🎵 **{menang.display_name}** bener! **{jawaban[0]}**.')
        return (menang, kalah)

    async def jalanin_bracket(self, channel, peserta, fungsi_ronde=None, label='Suit'):
        fungsi_ronde = fungsi_ronde or self.main_satu_ronde
        random.shuffle(peserta)
        await channel.send(f'🏆 **Turnamen {label} dimulai.** {len(peserta)} peserta:\n' + ', '.join((p.display_name for p in peserta)))
        ronde = 1
        semifinalis = []
        while len(peserta) > 1:
            nama_ronde = {2: 'FINAL', 4: 'SEMIFINAL'}.get(len(peserta), f'Ronde {ronde}')
            await channel.send(f'\n**── {nama_ronde} ──**')
            if len(peserta) == 4:
                semifinalis = list(peserta)
            lolos = []
            if len(peserta) % 2:
                beruntung = peserta.pop()
                lolos.append(beruntung)
                await channel.send(f'🎟️ **{beruntung.display_name}** dapet bye, langsung lolos.')
            for i in range(0, len(peserta), 2):
                a, b = (peserta[i], peserta[i + 1])
                menang, kalah = await fungsi_ronde(channel, a, b)
                lolos.append(menang)
                core.catat_main(menang.id, 'turnamen', menang=True)
                core.catat_main(kalah.id, 'turnamen')
                core.catatan(menang.id)['menang'] += 1
                core.catatan(kalah.id)['kalah'] += 1
                core.simpan()
                await asyncio.sleep(2)
            peserta = lolos
            ronde += 1
        juara = peserta[0]
        await core.tambah_xp(self.bot, juara, TURNAMEN_HADIAH[0])
        isi = discord.Embed(color=WARNA, title=f'🏆  JUARA TURNAMEN {label.upper()}', description=f'# {juara.mention}\nDapet **{TURNAMEN_HADIAH[0]} XP**.')
        isi.set_thumbnail(url=juara.display_avatar.url)
        hiburan = [p for p in semifinalis if p.id != juara.id]
        if hiburan:
            for p in hiburan:
                await core.tambah_xp(self.bot, p, TURNAMEN_HADIAH[2])
            isi.add_field(name='Sampai semifinal', value=', '.join((p.display_name for p in hiburan)) + f' — masing masing {TURNAMEN_HADIAH[2]} XP', inline=False)
        await channel.send(embed=isi)

    @app_commands.command(name='turnamen', description='Bikin turnamen sistem gugur')
    @app_commands.describe(jenis='Turnamen apa')
    @app_commands.choices(jenis=[app_commands.Choice(name='Suit', value='suit'), app_commands.Choice(name='Trivia', value='trivia'), app_commands.Choice(name='Tebak Lagu', value='lagu')])
    async def cmd_turnamen(self, inter: discord.Interaction, jenis: app_commands.Choice[str]=None):
        if not await core.di_arena(inter):
            return
        if turnamen_jalan.get(inter.channel_id):
            await inter.response.send_message('Masih ada turnamen yang jalan di sini.', ephemeral=True)
            return
        kode = jenis.value if jenis else 'suit'
        label, fungsi_ronde = self.JENIS_TURNAMEN[kode]
        turnamen_jalan[inter.channel_id] = True
        try:
            daftar = DaftarTurnamen(inter.user, label)
            await inter.response.send_message(embed=daftar.embed(), view=daftar)
            daftar.pesan = await inter.original_response()
            tunggu = asyncio.create_task(daftar.mulai.wait())
            selesai = asyncio.create_task(daftar.wait())
            await asyncio.wait([tunggu, selesai], return_when=asyncio.FIRST_COMPLETED)
            tunggu.cancel()
            selesai.cancel()
            for anak in daftar.children:
                anak.disabled = True
            try:
                await daftar.pesan.edit(view=daftar)
            except discord.HTTPException:
                pass
            peserta = list(daftar.peserta)
            if len(peserta) < TURNAMEN_MIN:
                await inter.channel.send(f'Cuma {len(peserta)} yang daftar, turnamennya batal. Butuh minimal {TURNAMEN_MIN}.')
                return
            await self.jalanin_bracket(inter.channel, peserta, fungsi_ronde, label)
        finally:
            turnamen_jalan.pop(inter.channel_id, None)

async def setup(bot):
    await bot.add_cog(Turnamen(bot))