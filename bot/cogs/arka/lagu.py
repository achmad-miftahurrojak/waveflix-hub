import asyncio
import random
import time
import discord
from discord import app_commands
from discord.ext import commands
from . import core
from .config import *
from .voice import suara
LAGU_BAWAAN = [{'judul': 'Laskar Pelangi', 'penyanyi': 'Nidji', 'tahun': '2008', 'genre': 'pop rock', 'emoji': '🌈⚔️'}, {'judul': 'Bintang di Surga', 'penyanyi': 'Peterpan', 'tahun': '2004', 'genre': 'pop rock', 'emoji': '⭐🕊️'}, {'judul': 'Kau Adalah', 'penyanyi': 'Isyana Sarasvati', 'tahun': '2018', 'genre': 'pop', 'emoji': '💫🫵'}, {'judul': 'Lathi', 'penyanyi': 'Weird Genius', 'tahun': '2020', 'genre': 'EDM', 'emoji': '🩰🔥'}, {'judul': 'Hymne Guru', 'penyanyi': 'Sartono', 'tahun': '1980', 'genre': 'lagu wajib', 'emoji': '🎓🙏'}, {'judul': 'Berita Kepada Kawan', 'penyanyi': 'Ebiet G. Ade', 'tahun': '1979', 'genre': 'balada', 'emoji': '📰👥'}, {'judul': 'Kupu Kupu Malam', 'penyanyi': 'Titiek Puspa', 'tahun': '1977', 'genre': 'pop lawas', 'emoji': '🦋🌙'}, {'judul': 'Bengawan Solo', 'penyanyi': 'Gesang', 'tahun': '1940', 'genre': 'keroncong', 'emoji': '🌊🏞️'}, {'judul': 'Rungkad', 'penyanyi': 'Happy Asmara', 'tahun': '2022', 'genre': 'dangdut', 'emoji': '💔🏚️'}, {'judul': 'Ojo Dibandingke', 'penyanyi': 'Farel Prayoga', 'tahun': '2022', 'genre': 'campursari', 'emoji': '⚖️🙅'}, {'judul': 'Akad', 'penyanyi': 'Payung Teduh', 'tahun': '2017', 'genre': 'folk', 'emoji': '💍☂️'}, {'judul': 'Sewu Kutho', 'penyanyi': 'Didi Kempot', 'tahun': '1990', 'genre': 'campursari', 'emoji': '1️⃣0️⃣0️⃣0️⃣🏙️'}, {'judul': 'Zona Nyaman', 'penyanyi': 'Fourtwnty', 'tahun': '2017', 'genre': 'folk', 'emoji': '🛋️😌'}, {'judul': 'Hari Bersamanya', 'penyanyi': 'Sheila On 7', 'tahun': '2004', 'genre': 'pop rock', 'emoji': '📅🧑\u200d🤝\u200d🧑'}, {'judul': 'Kamulah Satu Satunya', 'penyanyi': 'Dewa 19', 'tahun': '2000', 'genre': 'rock', 'emoji': '☝️❤️'}, {'judul': 'Separuh Aku', 'penyanyi': 'NOAH', 'tahun': '2012', 'genre': 'pop rock', 'emoji': '➗👤'}, {'judul': 'Cinta Luar Biasa', 'penyanyi': 'Andmesh', 'tahun': '2019', 'genre': 'pop', 'emoji': '❤️🤯'}, {'judul': 'Melukis Senja', 'penyanyi': 'Budi Doremi', 'tahun': '2021', 'genre': 'pop', 'emoji': '🎨🌇'}, {'judul': 'Sial', 'penyanyi': 'Mahalini', 'tahun': '2021', 'genre': 'pop', 'emoji': '🎲😖'}, {'judul': 'Rumah Ke Rumah', 'penyanyi': 'Hindia', 'tahun': '2019', 'genre': 'indie', 'emoji': '🏠➡️🏠'}, {'judul': 'Bahaya Komunis', 'penyanyi': 'Jason Ranti', 'tahun': '2017', 'genre': 'folk', 'emoji': '⚠️🚩'}, {'judul': 'Bunga Terakhir', 'penyanyi': 'Naif', 'tahun': '1998', 'genre': 'pop rock', 'emoji': '🌺🔚'}, {'judul': 'Kimi No Toriko', 'penyanyi': 'Rich Brian', 'tahun': '2019', 'genre': 'hip hop', 'emoji': '🇯🇵🫶'}, {'judul': 'Lagu Untukmu', 'penyanyi': 'Gigi', 'tahun': '2003', 'genre': 'rock', 'emoji': '🎵➡️🫵'}, {'judul': 'Ku Menunggu', 'penyanyi': 'Rossa', 'tahun': '2009', 'genre': 'pop', 'emoji': '⏳👩'}, {'judul': 'Andai Aku Bisa', 'penyanyi': 'Chrisye', 'tahun': '2001', 'genre': 'pop', 'emoji': '🤔💪'}, {'judul': 'Ada Band Karena Wanita', 'penyanyi': 'Ada Band', 'tahun': '2004', 'genre': 'pop', 'emoji': '👩\u200d🦰🎸'}, {'judul': 'Kisah Klasik', 'penyanyi': 'Sheila On 7', 'tahun': '2000', 'genre': 'pop rock', 'emoji': '📖🏛️'}, {'judul': 'Peri Cintaku', 'penyanyi': 'Marcell', 'tahun': '2003', 'genre': 'pop', 'emoji': '🧚❤️'}, {'judul': 'Widuri', 'penyanyi': 'Bob Tutupoly', 'tahun': '1976', 'genre': 'pop lawas', 'emoji': '🌸👩'}]
lagu_bank = []
lagu_jalan = {}
lagu_jeda = {}

def muat_lagu():
    global lagu_bank
    import penyimpanan
    lagu_bank = penyimpanan.baca_atau_berhenti(BERKAS_LAGU, [], 'gudang soal lagu')
    if not lagu_bank:
        lagu_bank = list(LAGU_BAWAAN)
        simpan_lagu()
        print(f'[arka] soal-lagu.json dibikin, isi {len(lagu_bank)} lagu bawaan')
        return
    print(f'[arka] {len(lagu_bank)} soal lagu dimuat')

def simpan_lagu():
    import penyimpanan
    penyimpanan.tulis(BERKAS_LAGU, lagu_bank)

def _pola_judul(judul):
    keluar = []
    for kata in judul.split():
        if len(kata) <= 1:
            keluar.append(kata.upper())
        else:
            keluar.append(kata[0].upper() + '_' * (len(kata) - 1))
    return ' '.join(keluar)

def _embed_lagu(pilih, tahap):
    isi = discord.Embed(color=WARNA, title='🎵  Tebak Lagu', description=f"# {pilih['emoji']}\nTebak judul lagunya. Ketik langsung di chat.")
    if tahap >= 1:
        isi.add_field(name='Petunjuk 2 — kapan & apa', value=f"Rilis **{pilih['tahun']}** · {pilih['genre']}", inline=False)
    if tahap >= 2:
        isi.add_field(name='Petunjuk 3 — siapa', value=f"Dinyanyiin **{pilih['penyanyi']}**", inline=False)
    if tahap >= 3:
        judul_utama = pilih['judul'].split(',')[0].strip()
        isi.add_field(name='Petunjuk 4 — pola judul', value=f'`{_pola_judul(judul_utama)}`', inline=False)
    hadiah = LAGU_HADIAH[min(tahap, len(LAGU_HADIAH) - 1)]
    sisa = len(LAGU_HADIAH) - 1 - tahap
    isi.set_footer(text=f'Jawab sekarang dapet {hadiah} XP' + (f' · {sisa} petunjuk lagi, tiap petunjuk XP-nya turun' if sisa > 0 else ' · ini petunjuk terakhir'))
    return isi

class Lagu(commands.Cog):

    def __init__(self, bot):
        self.bot = bot

    @commands.Cog.listener()
    async def on_ready(self):
        muat_lagu()

    async def cek_jawaban(self, pesan):
        keadaan_lagu = lagu_jalan.get(pesan.channel.id)
        if keadaan_lagu:
            from .trivia import _cocok
            if _cocok(pesan.content, keadaan_lagu['jawaban']):
                del lagu_jalan[pesan.channel.id]
                tahap = keadaan_lagu['tahap']
                hadiah = LAGU_HADIAH[min(tahap, len(LAGU_HADIAH) - 1)]
                lagu = keadaan_lagu['lagu']
                core.catat_main(pesan.author.id, 'lagu', menang=True)
                core.catatan(pesan.author.id)['menang'] += 1
                await core.tambah_xp(self.bot, pesan.author, hadiah)
                try:
                    await pesan.add_reaction('🎵')
                except Exception:
                    pass
                cepat = '  Ketebak dari emoji doang, gila.' if tahap == 0 else ''
                await pesan.channel.send(f"🎵 **{pesan.author.display_name}** bener! **{keadaan_lagu['jawaban'][0]}** — {lagu['penyanyi']} ({lagu['tahun']}). Ambil **{hadiah} XP**.{cepat}")
                return True
        return False

    @app_commands.command(name='lagu', description='Tebak lagu dari emoji, petunjuknya nambah pelan pelan')
    async def cmd_lagu(self, inter: discord.Interaction):
        if not await core.di_arena(inter):
            return
        if not lagu_bank:
            await inter.response.send_message('Gudang lagunya kosong. Tambahin pakai `/tambahlagu`.', ephemeral=True)
            return
        if inter.channel_id in lagu_jalan:
            await inter.response.send_message('Masih ada lagu yang belum ketebak tuh.', ephemeral=True)
            return
        sisa = LAGU_JEDA_ORANG - (time.time() - lagu_jeda.get(inter.user.id, 0))
        if sisa > 0:
            await inter.response.send_message(f'Sabar, **{int(sisa) + 1} detik** lagi.', ephemeral=True)
            return
        lagu_jeda[inter.user.id] = time.time()
        ronde = object()
        pilih = random.choice(lagu_bank)
        jawaban = [x.strip() for x in pilih['judul'].split(',') if x.strip()]
        lagu_jalan[inter.channel_id] = {'id': ronde, 'jawaban': jawaban, 'tahap': 0, 'lagu': pilih}
        await inter.response.send_message(embed=_embed_lagu(pilih, 0))
        for tahap in range(1, len(LAGU_HADIAH)):
            await asyncio.sleep(LAGU_JEDA_TAHAP)
            keadaan = lagu_jalan.get(inter.channel_id)
            if not keadaan or keadaan['id'] is not ronde:
                return
            keadaan['tahap'] = tahap
            await inter.followup.send(embed=_embed_lagu(pilih, tahap))
        await asyncio.sleep(LAGU_JEDA_TAHAP)
        keadaan = lagu_jalan.get(inter.channel_id)
        if keadaan and keadaan['id'] is ronde:
            del lagu_jalan[inter.channel_id]
            await inter.followup.send(f"⏰ Ga ada yang tau. Jawabannya **{jawaban[0]}** — {pilih['penyanyi']} ({pilih['tahun']}).")

    @app_commands.command(name='tambahlagu', description='Nambah lagu ke gudang tebak lagu')
    @app_commands.describe(judul='Judul lagunya. Pisahin pakai koma kalau ada nama lain', penyanyi='Yang nyanyi', tahun='Tahun rilis', genre='Genrenya apa', emoji='Emoji yang nggambarin judulnya, 2 sampai 4 biji')
    async def cmd_tambahlagu(self, inter: discord.Interaction, judul: str, penyanyi: str, tahun: int, genre: str, emoji: str):
        if not await core.di_arena(inter):
            return
        if ROLE_SOAL:
            punya = {r.name for r in getattr(inter.user, 'roles', [])}
            if not punya & set(ROLE_SOAL):
                await inter.response.send_message('Nambah lagu cuma buat ' + ' atau '.join(ROLE_SOAL) + '.', ephemeral=True)
                return
        utama = judul.split(',')[0].strip().lower()
        if any((l['judul'].split(',')[0].strip().lower() == utama for l in lagu_bank)):
            await inter.response.send_message(f"**{judul.split(',')[0].strip()}** udah ada di gudang.", ephemeral=True)
            return
        from datetime import datetime
        if not 1900 <= tahun <= datetime.now(WIB).year:
            await inter.response.send_message('Tahunnya ga masuk akal.', ephemeral=True)
            return
        lagu_bank.append({'judul': judul, 'penyanyi': penyanyi, 'tahun': tahun, 'genre': genre, 'emoji': emoji})
        simpan_lagu()
        isi = discord.Embed(color=WARNA, title='🎵  Lagu ditambahin', description=f'# {emoji}\n**{judul}**')
        isi.add_field(name='Penyanyi', value=penyanyi, inline=True)
        isi.add_field(name='Tahun', value=str(tahun), inline=True)
        isi.add_field(name='Genre', value=genre, inline=True)
        isi.add_field(name='Pola judul', value=f"`{_pola_judul(judul.split(',')[0].strip())}`", inline=False)
        isi.set_footer(text=f'Gudang lagu sekarang {len(lagu_bank)} judul')
        await inter.response.send_message(embed=isi, ephemeral=True)

async def setup(bot):
    await bot.add_cog(Lagu(bot))