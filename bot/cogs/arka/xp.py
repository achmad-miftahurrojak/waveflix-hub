import asyncio
import time
from datetime import datetime, timedelta
import discord
from discord import app_commands
from discord.ext import commands, tasks
from . import core
from .config import *
from .voice import suara

class Xp(commands.Cog):

    def __init__(self, bot):
        self.bot = bot
        self.pantau_voice.start()
        self.rekap_mingguan.start()
        self.jaga_arsip.start()

    def cog_unload(self):
        self.pantau_voice.cancel()
        self.rekap_mingguan.cancel()
        self.jaga_arsip.cancel()

    @commands.Cog.listener()
    async def on_message(self, pesan):
        if pesan.author.bot or not pesan.guild:
            return
        trivia = self.bot.get_cog('Trivia')
        if trivia and await trivia.cek_jawaban(pesan):
            return
        lagu = self.bot.get_cog('Lagu')
        if lagu and await lagu.cek_jawaban(pesan):
            return
        lomba = self.bot.get_cog('LombaAcara')
        if lomba and await lomba.cek_kiriman(pesan):
            pass
        rantai = self.bot.get_cog('Sosial')
        if rantai and await rantai.cek_rantai(pesan):
            return
        if len(pesan.content.strip()) < MINIMAL_HURUF:
            return
        kunci = pesan.author.id
        sekarang = time.time()
        if sekarang - core.jeda_terakhir.get(kunci, 0) < JEDA_CHAT:
            return
        core.jeda_terakhir[kunci] = sekarang
        if PET_AKTIF:
            pet = core.kondisi_pet()
            pet['kenyang'] = min(100.0, pet['kenyang'] + PET_KENYANG_PER_CHAT)
        core.catatan(pesan.author.id)['pesan'] += 1
        misi = self.bot.get_cog('Misi')
        if misi:
            misi.maju_misi(pesan.author.id, 'chat')
        import random
        await core.tambah_xp(self.bot, pesan.author, random.randint(*XP_CHAT))

    @commands.Cog.listener()
    async def on_member_remove(self, anggota):
        if anggota.bot:
            return
        catat = core.data['orang'].get(str(anggota.id))
        if catat is None:
            return
        catat['keluar'] = core.hari_ini()
        core.simpan()
        print(f'[arka] {anggota.display_name} keluar, datanya diarsipin {ARSIP_HARI} hari')

    @commands.Cog.listener()
    async def on_member_join(self, anggota):
        if anggota.bot:
            return
        catat = core.data['orang'].get(str(anggota.id))
        if catat and catat.pop('keluar', None):
            core.simpan()
            print(f'[arka] {anggota.display_name} balik lagi, datanya dipulihin')

    @tasks.loop(seconds=21600)
    async def jaga_arsip(self):
        try:
            sekarang = datetime.now(WIB)
            batas = (sekarang - timedelta(days=ARSIP_HARI)).strftime('%Y-%m-%d')
            buang = []
            for uid, catat in core.data['orang'].items():
                keluar = catat.get('keluar')
                if keluar and keluar <= batas:
                    buang.append(uid)
            if buang:
                for uid in buang:
                    core.data['orang'].pop(uid, None)
                core.simpan()
                print(f'[arka] {len(buang)} data orang lama dibuang (keluar lebih dari {ARSIP_HARI} hari lalu)')
        except Exception as e:
            print(f'[arka] bersih bersih arsip error: {e}')

    @jaga_arsip.before_loop
    async def before_jaga_arsip(self):
        await self.bot.wait_until_ready()

    @tasks.loop(seconds=JEDA_VOICE)
    async def pantau_voice(self):
        try:
            for guild in self.bot.guilds:
                for channel in guild.voice_channels:
                    if channel == guild.afk_channel:
                        continue
                    orang = [m for m in channel.members if not m.bot and m.voice and (not m.voice.self_deaf) and (not m.voice.self_mute) and (not core._dipindah_afk.get(m.id))]
                    if len(orang) < 2:
                        continue
                    for m in orang:
                        core.catatan(m.id)['menit'] += 5
                        await core.tambah_xp(self.bot, m, XP_VOICE)
        except Exception as e:
            print(f'[arka] pantau voice error: {e}')

    @pantau_voice.before_loop
    async def before_pantau_voice(self):
        await self.bot.wait_until_ready()

    def kategori_giliran(self):
        daftar = list(KATEGORI_GAME)
        urutan = core.data.get('rekap_giliran', 0) % len(daftar)
        return daftar[urutan]

    def papan_game(self, guild, game, batas=3):
        isi = []
        for uid, catat in core.data['orang'].items():
            angka = catat.get('game', {}).get(game, {}).get('minggu', 0)
            if angka > 0 and core.masih_anggota(guild, uid):
                isi.append((uid, angka))
        isi.sort(key=lambda x: x[1], reverse=True)
        baris = []
        for i, (uid, angka) in enumerate(isi[:batas]):
            anggota = guild.get_member(int(uid))
            nama = anggota.display_name if anggota else 'entah siapa'
            tanda = ['🥇', '🥈', '🥉'][i]
            baris.append(f'{tanda} {nama} — {angka}x')
        return baris

    async def pindahin_juara(self, guild, juara):
        if not ROLE_JUARA:
            return
        role = discord.utils.get(guild.roles, name=ROLE_JUARA)
        if not role:
            print(f"[arka] role '{ROLE_JUARA}' ga ketemu, gelarnya dilewat")
            return
        try:
            for lama in list(role.members):
                if lama != juara:
                    await lama.remove_roles(role, reason='Wave Champion minggu lalu')
            if juara and role not in juara.roles:
                await juara.add_roles(role, reason='Wave Champion minggu ini')
        except discord.Forbidden:
            print('[arka] ga bisa atur Wave Champion. Arka butuh izin Manage Roles, dan role Arka harus di atas Wave Champion')
        except Exception as e:
            print(f'[arka] Wave Champion bermasalah: {e}')

    async def kirim_rekap(self, channel, resmi=False):
        hidup = [(uid, catat) for uid, catat in core.data['orang'].items() if catat.get('xp_minggu', 0) > 0 and core.masih_anggota(channel.guild, uid)]
        urut = sorted(hidup, key=lambda x: x[1].get('xp_minggu', 0), reverse=True)[:3]
        if not urut:
            await channel.send('Minggu ini sepi banget, ga ada yang ngumpulin XP sama sekali. Minggu depan jangan gitu ya.')
            return
        baris = []
        for i, (uid, catat) in enumerate(urut):
            anggota = channel.guild.get_member(int(uid))
            nama = anggota.display_name if anggota else 'entah siapa'
            tanda = ['🥇', '🥈', '🥉'][i]
            baris.append(f"{tanda}  **{nama}** — {catat['xp_minggu']:,} XP")
        juara_uid = urut[0][0]
        core.catatan(juara_uid)['juara'] += 1
        juara = channel.guild.get_member(int(juara_uid))
        nama_juara = juara.display_name if juara else 'entah siapa'
        kali = core.catatan(juara_uid)['juara']
        if resmi:
            await self.pindahin_juara(channel.guild, juara)
        isi = discord.Embed(color=WARNA, title='🏆  Rekap Minggu Ini', description='\n'.join(baris))
        isi.add_field(name='Juara minggu ini', value=f'**{nama_juara}**' + (f' — gelar ke-{kali}' if kali > 1 else ''), inline=False)
        sorot = self.kategori_giliran()
        potongan = []
        for game in KATEGORI_GAME[sorot]:
            baris_game = self.papan_game(channel.guild, game)
            judul = NAMA_GAME.get(game, game)
            potongan.append(f'**{judul}**\n' + ('\n'.join(baris_game) if baris_game else 'belum ada yang main'))
        isi.add_field(name=f'🔦 Sorotan minggu ini — {sorot}', value='\n\n'.join(potongan), inline=False)
        if resmi and ROLE_JUARA:
            isi.set_footer(text=f'Gelar {ROLE_JUARA} pindah ke {nama_juara}. Hitungan mingguan direset sekarang.')
        else:
            isi.set_footer(text='Hitungan mingguan direset tiap Senin. Rebutan lagi dari nol.')
        await channel.send(embed=isi)
        await channel.send(f'Papan mingguan gua kosongin. {nama_juara} udah di depan, yang lain masa mau kalah terus.')

    @tasks.loop(seconds=900)
    async def rekap_mingguan(self):
        try:
            sekarang = datetime.now(WIB)
            minggu_ini = sekarang.strftime('%Y-W%W')
            waktunya = sekarang.weekday() == HARI_REKAP and sekarang.hour >= JAM_REKAP
            tujuan = CHANNEL_REKAP or CHANNEL_ARENA
            if tujuan and waktunya and (core.data.get('rekap_terakhir') != minggu_ini):
                channel = self.bot.get_channel(tujuan)
                if channel:
                    await self.kirim_rekap(channel, resmi=True)
                    core.data['rekap_terakhir'] = minggu_ini
                    core.data['rekap_giliran'] = core.data.get('rekap_giliran', 0) + 1
                    for catat in core.data['orang'].values():
                        catat['xp_minggu'] = 0
                        for isi_game in catat.get('game', {}).values():
                            isi_game['minggu'] = 0
                    core.simpan()
        except Exception as e:
            print(f'[arka] rekap mingguan error: {e}')

    @rekap_mingguan.before_loop
    async def before_rekap_mingguan(self):
        await self.bot.wait_until_ready()

    @app_commands.command(name='level', description='Liat kedalaman lo udah sampai mana')
    @app_commands.describe(orang='Kosongin kalau mau liat punya sendiri')
    async def cmd_level(self, inter: discord.Interaction, orang: discord.Member=None):
        if not await core.di_arena(inter):
            return
        orang = orang or inter.user
        catat = core.catatan(orang.id)
        xp = catat['xp']
        depan = core.berikutnya(xp)
        if core.kartu_level is not None:
            await inter.response.defer()
            try:
                berkas = await core.rakit_kartu_level(inter, orang, catat)
                await inter.followup.send(file=discord.File(berkas, filename='level.png'))
                return
            except Exception as e:
                import traceback
                print(f'[arka] kartu level gagal, pakai embed: {e}')
                traceback.print_exc()
        isi = discord.Embed(color=WARNA, title=f'🌊  {core.tingkat(xp)[1]}')
        isi.set_thumbnail(url=orang.display_avatar.url)
        isi.add_field(name='XP', value=f'{xp:,}', inline=True)
        isi.add_field(name='Pesan', value=f"{catat['pesan']:,}", inline=True)
        isi.add_field(name='Voice', value=f"{catat['menit']:,} menit", inline=True)
        isi.add_field(name='Absen beruntun', value=f"{catat['absen']} hari", inline=True)
        isi.add_field(name='Menang', value=f"{catat['menang']}", inline=True)
        isi.add_field(name='Kalah', value=f"{catat['kalah']}", inline=True)
        isi.add_field(name='XP minggu ini', value=f"{catat['xp_minggu']:,}", inline=True)
        if catat['juara']:
            isi.add_field(name='Gelar juara mingguan', value='🏆 ' * min(catat['juara'], 5) + f" {catat['juara']}x", inline=True)
        if core.koleksi_lengkap(catat):
            isi.add_field(name='Koleksi pantai lengkap', value=f'{core.deret_tangkapan()}\nSemua {len(TANGKAPAN)} jenis udah pernah ketemu', inline=False)
        if depan:
            isi.add_field(name=f'Menuju {depan[1]}', value=f'`{core.batang(xp)}`\nkurang {depan[0] - xp:,} XP', inline=False)
        else:
            isi.add_field(name='Sudah paling dalam', value='`' + core.batang(xp) + '`', inline=False)
        isi.set_footer(text=orang.display_name)
        if inter.response.is_done():
            await inter.followup.send(embed=isi)
        else:
            await inter.response.send_message(embed=isi)

    @app_commands.command(name='papan', description='Papan peringkat SUMMER TIDE')
    @app_commands.describe(urutan='Mau diurutin berdasarkan apa', game='Atau liat papan satu game tertentu')
    @app_commands.choices(urutan=[app_commands.Choice(name='XP minggu ini', value='xp_minggu'), app_commands.Choice(name='XP sepanjang masa', value='xp'), app_commands.Choice(name='Absen terpanjang', value='absen_panjang'), app_commands.Choice(name='Menang terbanyak', value='menang')])
    @app_commands.choices(game=[app_commands.Choice(name='Trivia', value='trivia'), app_commands.Choice(name='Duel', value='duel'), app_commands.Choice(name='Suit', value='suit'), app_commands.Choice(name='Tebak Angka', value='tebak'), app_commands.Choice(name='Perang Balon', value='balon'), app_commands.Choice(name='Jelajah Pantai', value='hunt'), app_commands.Choice(name='Rantai Kata', value='rantai'), app_commands.Choice(name='Tebak Lagu', value='lagu'), app_commands.Choice(name='Turnamen Suit', value='turnamen')])
    async def cmd_papan(self, inter: discord.Interaction, urutan: app_commands.Choice[str]=None, game: app_commands.Choice[str]=None):
        if not await core.di_arena(inter):
            return
        if game is not None:
            isi = []
            for uid, catat in core.data['orang'].items():
                per = catat.get('game', {}).get(game.value, {})
                if per.get('menang', 0) > 0 and core.masih_anggota(inter.guild, uid):
                    isi.append((uid, per['menang'], per.get('main', 0)))
            isi.sort(key=lambda x: x[1], reverse=True)
            if not isi:
                await inter.response.send_message(f'Belum ada yang menang di **{NAMA_GAME.get(game.value, game.value)}**. Jadi yang pertama gih.')
                return
            baris = []
            for i, (uid, menang, main) in enumerate(isi[:10], 1):
                anggota = inter.guild.get_member(int(uid))
                nama = anggota.display_name if anggota else 'entah siapa'
                tanda = ['🥇', '🥈', '🥉'][i - 1] if i <= 3 else f'`{i}.`'
                rasio = f' ({menang / main * 100:.0f}%)' if main else ''
                baris.append(f'{tanda} **{nama}** — {menang} menang dari {main} main{rasio}')
            embed = discord.Embed(color=WARNA, title=f'🏖️  Paling jago {NAMA_GAME.get(game.value, game.value)}', description='\n'.join(baris))
            await inter.response.send_message(embed=embed)
            return
        kunci = urutan.value if urutan else 'xp'
        judul = {'xp': 'Paling dalam sepanjang masa', 'xp_minggu': 'Paling ngotot minggu ini', 'absen_panjang': 'Paling rajin', 'menang': 'Paling sering menang'}[kunci]
        hidup = [(uid, catat) for uid, catat in core.data['orang'].items() if catat.get(kunci, 0) > 0 and core.masih_anggota(inter.guild, uid)]
        urut = sorted(hidup, key=lambda x: x[1].get(kunci, 0), reverse=True)[:10]
        if not urut:
            await inter.response.send_message('Belum ada yang nyemplung.')
            return
        baris = []
        for i, (uid, catat) in enumerate(urut, 1):
            anggota = inter.guild.get_member(int(uid))
            nama = anggota.display_name if anggota else 'entah siapa'
            tanda = ['🥇', '🥈', '🥉'][i - 1] if i <= 3 else f'`{i}.`'
            if kunci == 'xp':
                nilai = f"{catat['xp']:,} XP · {core.tingkat(catat['xp'])[1]}"
            elif kunci == 'xp_minggu':
                nilai = f"{catat['xp_minggu']:,} XP minggu ini"
            elif kunci == 'absen_panjang':
                nilai = f"{catat['absen_panjang']} hari beruntun"
            else:
                nilai = f"{catat['menang']} menang"
            baris.append(f'{tanda} **{nama}** — {nilai}')
        isi = discord.Embed(color=WARNA, title=f'🏖️  {judul}', description='\n'.join(baris))
        await inter.response.send_message(embed=isi)

    @app_commands.command(name='statistik', description='Ringkasan seluruh server')
    async def cmd_statistik(self, inter: discord.Interaction):
        if not await core.di_arena(inter):
            return
        semua = core.data['orang'].values()
        if not semua:
            await inter.response.send_message('Belum ada data.')
            return
        isi = discord.Embed(color=WARNA, title='📊  Statistik SUMMER TIDE')
        isi.add_field(name='Orang tercatat', value=f"{len(core.data['orang'])}", inline=True)
        isi.add_field(name='Total XP', value=f"{sum((o['xp'] for o in semua)):,}", inline=True)
        isi.add_field(name='Total pesan', value=f"{sum((o['pesan'] for o in semua)):,}", inline=True)
        isi.add_field(name='Jam di voice', value=f"{sum((o['menit'] for o in semua)) // 60:,}", inline=True)
        isi.add_field(name='Duel & suit', value=f"{sum((o['menang'] for o in semua))}", inline=True)
        c = core.cuaca_hari_ini()
        isi.add_field(name='Cuaca', value=f"{c['emoji']} {c['nama']}", inline=True)
        await inter.response.send_message(embed=isi)

    @app_commands.command(name='rekap', description='Papan mingguan sekarang juga')
    async def cmd_rekap(self, inter: discord.Interaction):
        if not await core.di_arena(inter):
            return
        await self.kirim_rekap(inter.channel, resmi=False)
        if not inter.response.is_done():
            await inter.response.send_message('Papan dipanggil manual.', ephemeral=True)

async def setup(bot):
    await bot.add_cog(Xp(bot))