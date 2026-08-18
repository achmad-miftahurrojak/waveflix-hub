import asyncio
import io
import json
import os
import random
import re
from datetime import timedelta, datetime, timezone
import aiohttp
import discord
from discord import app_commands
from discord.ext import commands, tasks
from dotenv import load_dotenv
import acara
import penyimpanan
try:
    import kartu
    if not hasattr(kartu, 'buat_kartu'):
        print(f"[kartu] SALAH FILE. Python muat: {getattr(kartu, '__file__', 'folder, bukan file')}")
        print('[kartu] isinya: ' + ', '.join((a for a in dir(kartu) if not a.startswith('_'))) or '(kosong)')
        kartu = None
except Exception as e:
    print(f'[kartu] modul kartu ga kepakai: {e}')
    kartu = None
try:
    from botDirga import tanya_ai
except Exception as e:
    print(f'[ai] ga bisa pinjem otak botDirga: {e}')
    tanya_ai = None
load_dotenv()
_BOT_ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
TOKEN = os.getenv('JULIAN_TOKEN')
# [REFACTOR] Memindahkan Hardcoded ID ke .env untuk deployment fleksibel
CHANNEL_SAMBUTAN_ID = int(os.getenv('JULIAN_CHANNEL_SAMBUTAN_ID', '1535071902312169492'))
ROLE_OTOMATIS = os.getenv('JULIAN_ROLE_OTOMATIS', 'Tourist')
KARTU_SAMBUTAN = True
CHANNEL_LOG_ID = int(os.getenv('JULIAN_CHANNEL_LOG_ID', '1535070933583405169'))
LOG_PESAN_DIHAPUS = True
LOG_KELUAR_MASUK = True
REACTION_ROLE = {}
STATUS_LIST_JULIAN = ['Jangan rusuh, admin lagi pusing!', 'Ketahuan spam langsung gua ban', 'Baca rules sebelum nanya', 'Gua pantau lu dari pojokan']
CATATAN_WARN = 'warn.json'
WARN_KADALUARSA_HARI = 30
WARN_ESKALASI = {3: 60, 5: 1440}
ROLE_PENGUASA = ['Island Owner']
ANTI_RAID = True
RAID_JUMLAH = 5
RAID_DETIK = 60
AKUN_BARU_HARI = 7
RAID_AUTO_KUNCI = True
RAID_KUNCI_MENIT = 10
ROLE_SIAGA = os.getenv('JULIAN_ROLE_SIAGA', 'Island Owner')
ANTI_NUKE = True
NUKE_JUMLAH = 3
NUKE_DETIK = 30
NUKE_CABUT_ROLE = True
PANTAU_ROLE_ADMIN = True
LACAK_UNDANGAN = True
CATATAN_UNDANGAN = 'undangan.json'
STARBOARD_AKTIF = True
CHANNEL_STARBOARD_ID = int(os.getenv('JULIAN_CHANNEL_STARBOARD_ID', '1532236816986669056'))
STARBOARD_EMOJI = '⭐'
STARBOARD_AMBANG = 4
CATATAN_STARBOARD = 'starboard.json'
ROLE_PENJAGA_TIKET = os.getenv('JULIAN_ROLE_PENJAGA_TIKET', 'Island Owner')
CATATAN_TIKET = 'tiket.json'
TIKET_MAKS_PER_ORANG = 2
DM_SAMBUTAN = True
CHANNEL_ATURAN_ID = int(os.getenv('JULIAN_CHANNEL_ATURAN_ID', '1531955366269681784'))
AUTOMOD_AKTIF = True
AUTOMOD_BEBAS = []
FILTER_KATA = True
KATA_BERAT = ['kontol', 'memek', 'pepek', 'peler', 'titit', 'jembut', 'puki', 'pukimak', 'ngentot', 'ngentod', 'entot', 'kentot', 'ngewe', 'sange', 'coli', 'colmek', 'peju', 'pejuh', 'bokep', 'bangsat', 'bajingan', 'jancok', 'jancuk', 'kimak', 'asu', 'asw', 'anjing', 'anjg', 'njing', 'babi', 'tai', 'taik', 'bangke', 'bangkai', 'lonte', 'sundal', 'pelacur']
KATA_RINGAN = ['tolol', 'goblok', 'goblog', 'bego', 'bodoh', 'idiot', 'dungu', 'kampret', 'monyet', 'setan', 'sialan', 'brengsek', 'anjir', 'njir', 'anjay', 'bangsad', 'cok', 'cuk']
BLOKIR_UNDANGAN = True
ANTI_SPAM = True
SPAM_JUMLAH = 5
SPAM_DETIK = 7
AUTO_SLOWMODE = True
SLOWMODE_PEMICU = 3
SLOWMODE_JENDELA = 60
SLOWMODE_ORANG_BEDA = 2
SLOWMODE_DETIK = 15
SLOWMODE_LAMA_MENIT = 10
BATAS_KAPITAL = True
KAPITAL_PERSEN = 70
KAPITAL_MINIMAL = 12
MAX_MENTION = 5
STRIKE_SEBELUM_TIMEOUT = 3
TIMEOUT_OTOMATIS_MENIT = 10
CHANNEL_RESOURCES_ID = int(os.getenv('JULIAN_CHANNEL_RESOURCES_ID', '1532241897265828122'))
CATATAN_LINK = 'resources.json'
CHANNEL_NOTIF_VOICE_ID = int(os.getenv('JULIAN_CHANNEL_NOTIF_VOICE_ID', '1453703811607826596'))
JEDA_SESI_MENIT = 20
CHANNEL_LAPORAN_AFK_ID = int(os.getenv('JULIAN_CHANNEL_LAPORAN_AFK_ID', '1535214549765197834'))
AUTO_AFK = True
AFK_WARN_DETIK = 300
AFK_ROLE_BEBAS = []
DM_SAAT_AFK = True
JEDA_DM_AFK_MENIT = 60
AFK_TOMBOL_AKTIF = True
AFK_TOMBOL_DETIK = 60
AFK_KEBAL_MENIT = 15
AFK_MAKS_KONFIRMASI = 3
PESAN_AFK = ['lo kelamaan diem di {asal}, jadi kelempar ke {afk}. balik aja kalo mau lanjut', 'ketauan afk di {asal} wkwk. udah dipindah, santai bukan ditendang kok', 'mic lo sepi mulu di {asal}, sistem ngira lo ketiduran. sekarang lo di {afk}', 'dari {asal} pindah ke {afk}, soalnya ga ada suara daritadi. tinggal join lagi', 'gua ga ngapa ngapain ya, discord sendiri yang mindahin lo dari {asal} ke {afk}']
CHANNEL_PENGUMUMAN_ID = int(os.getenv('JULIAN_CHANNEL_PENGUMUMAN_ID', '1532238760874348554'))
JAM_KIRIM = 7
CATATAN_UMUM = 'pengumuman.json'
FOLDER_GAMBAR = os.path.join(_BOT_ROOT, 'assets', 'images')
GAMBAR_LIBUR = {'idul fitri': 'idulfitri.jpg', 'idul adha': 'iduladha.jpg', 'tahun baru islam': 'tahunbaruislam.jpg', 'maulid': 'maulid.jpg', 'isra': 'isramiraj.jpg', 'nyepi': 'nyepi.jpg', 'waisak': 'waisak.jpg', 'imlek': 'imlek.jpg', 'natal': 'natal.jpg', 'jumat agung': 'jumatagung.jpg', 'kenaikan': 'kenaikanisa.jpg', 'paskah': 'paskah.jpg'}

def cari_gambar_libur(nama):
    rendah = (nama or '').lower()
    for kata, berkas in GAMBAR_LIBUR.items():
        if kata in rendah:
            return berkas
    return ''
WIB = timezone(timedelta(hours=7))
API_LIBUR = ['https://api-hari-libur.vercel.app/api?year={tahun}', 'https://raw.githubusercontent.com/guangrei/APIHariLibur_V2/main/calendar.min.json']
HARI_PERINGATAN = {'01-01': {'nama': 'Tahun Baru Masehi', 'deskripsi': 'Lembaran baru, target baru, dan resolusi yang biasanya bertahan sampai Februari.', 'kutipan': 'Setiap awal adalah kelanjutan dari sesuatu yang berani kita tinggalkan.', 'gambar': 'tahunbaru.jpg'}, '02-09': {'nama': 'Hari Pers Nasional', 'deskripsi': 'Peringatan buat kerja keras jurnalis yang bikin kita tau apa yang terjadi di luar sana.', 'kutipan': 'Kebebasan pers bukan hadiah, tapi hasil dari orang orang yang menolak diam.', 'gambar': 'haripers.jpg'}, '04-21': {'nama': 'Hari Kartini', 'deskripsi': 'Mengenang R.A. Kartini dan perjuangannya soal hak perempuan buat sekolah dan bersuara.', 'kutipan': 'Habis gelap terbitlah terang.', 'gambar': 'kartini.jpg'}, '05-01': {'nama': 'Hari Buruh', 'deskripsi': 'Hari buat menghargai kerja orang orang yang bikin semuanya tetap jalan.', 'kutipan': 'Tidak ada pekerjaan kecil, yang ada cuma orang yang meremehkannya.', 'gambar': 'hariburuh.jpg'}, '05-02': {'nama': 'Hari Pendidikan Nasional', 'deskripsi': 'Peringatan lahirnya Ki Hajar Dewantara, orang yang bikin sekolah jadi hak semua orang.', 'kutipan': 'Ing ngarsa sung tuladha, ing madya mangun karsa, tut wuri handayani.', 'gambar': 'pendidikan.jpg'}, '05-20': {'nama': 'Hari Kebangkitan Nasional', 'deskripsi': 'Menandai lahirnya Budi Utomo dan awal mula kesadaran berbangsa.', 'kutipan': 'Bangsa yang besar dimulai dari orang orang yang berani sadar duluan.', 'gambar': 'kebangkitan.jpg'}, '06-01': {'nama': 'Hari Lahir Pancasila', 'deskripsi': 'Hari saat lima sila pertama kali diucapkan sebagai dasar negara.', 'kutipan': 'Berbeda beda tetapi tetap satu jua.', 'gambar': 'pancasila.jpg'}, '08-17': {'nama': 'Hari Kemerdekaan Republik Indonesia', 'deskripsi': 'Hari saat Indonesia berdiri sendiri. Selamat ulang tahun, Indonesia.', 'kutipan': 'Bangsa yang besar adalah bangsa yang menghormati jasa pahlawannya.', 'gambar': 'kemerdekaan.jpg'}, '10-01': {'nama': 'Hari Kesaktian Pancasila', 'deskripsi': 'Peringatan bahwa Pancasila tetap berdiri setelah masa paling gelap dalam sejarah kita.', 'kutipan': 'Yang menyatukan kita selalu lebih besar daripada yang memisahkan.', 'gambar': 'kesaktianpancasila.jpg'}, '10-28': {'nama': 'Hari Sumpah Pemuda', 'deskripsi': 'Hari saat anak muda dari berbagai daerah sepakat menyebut diri mereka satu bangsa.', 'kutipan': 'Satu nusa, satu bangsa, satu bahasa.', 'gambar': 'sumpahpemuda.jpg'}, '11-10': {'nama': 'Hari Pahlawan', 'deskripsi': 'Mengenang pertempuran Surabaya dan semua orang yang tidak sempat pulang.', 'kutipan': 'Merdeka atau mati.', 'gambar': 'haripahlawan.jpg'}, '11-25': {'nama': 'Hari Guru Nasional', 'deskripsi': 'Buat semua guru yang namanya masih kita inget sampai sekarang.', 'kutipan': 'Guru yang baik tidak menyuruh kita menghafal, tapi membuat kita ingin tahu.', 'gambar': 'hariguru.jpg'}, '12-22': {'nama': 'Hari Ibu', 'deskripsi': 'Hari buat orang yang paling jarang dikasih ucapan padahal paling sering ada.', 'kutipan': 'Kasih ibu sepanjang jalan, kasih anak sepanjang galah.', 'gambar': 'hariibu.jpg'}}
intents = discord.Intents.default()
intents.members = True
intents.message_content = True
bot = commands.Bot(command_prefix='!', intents=intents)

def baca_warn():
    return penyimpanan.baca_atau_berhenti(CATATAN_WARN, {}, 'catatan warn')

def tulis_warn(data):
    penyimpanan.tulis(CATATAN_WARN, data, rapi=2)

def kasus_baru():
    data = baca_warn()
    meta = data.setdefault('_meta', {})
    meta['kasus'] = meta.get('kasus', 0) + 1
    tulis_warn(data)
    return meta['kasus']

def nomor_kasus(n):
    return f'#{n:04d}'

def warn_aktif(daftar):
    batas = datetime.now(timezone.utc) - timedelta(days=WARN_KADALUARSA_HARI)
    hidup = []
    for w in daftar:
        try:
            kapan = datetime.fromisoformat(w['waktu'])
        except (KeyError, TypeError, ValueError):
            hidup.append(w)
            continue
        if kapan > batas:
            hidup.append(w)
    return hidup

async def kirim_log(guild, judul, isi, warna=discord.Color.blurple(), kasus=None):
    if CHANNEL_LOG_ID == 0:
        return
    channel = guild.get_channel(CHANNEL_LOG_ID)
    if channel is None:
        return
    if kasus:
        judul = f'{nomor_kasus(kasus)}  ·  {judul}'
    embed = discord.Embed(title=judul, description=isi, color=warna, timestamp=datetime.now(timezone.utc))
    if kasus:
        embed.set_footer(text=f'Kasus {nomor_kasus(kasus)}')
    try:
        await channel.send(embed=embed)
    except discord.Forbidden:
        print('[log] bot ga punya izin kirim ke channel log')

@bot.event
async def on_member_join(member):
    if member.bot:
        print(f'[join] {member} itu aplikasi, sambutan dilewat')
        if LOG_KELUAR_MASUK:
            await kirim_log(member.guild, 'Aplikasi ditambahkan', f'{member.mention} ({member}) masuk ke server', discord.Color.blurple())
        return
    kena_raid = await cek_raid(member)
    if ROLE_OTOMATIS:
        role = discord.utils.get(member.guild.roles, name=ROLE_OTOMATIS)
        if role is None:
            print(f"[join] role '{ROLE_OTOMATIS}' ga ketemu")
        else:
            try:
                await member.add_roles(role, reason='Auto role member baru')
            except discord.Forbidden:
                print('[join] gagal kasih role. Cek posisi role bot, harus di atas role target')
    if CHANNEL_SAMBUTAN_ID and (not kena_raid):
        channel = member.guild.get_channel(CHANNEL_SAMBUTAN_ID)
        if channel:
            await kirim_sambutan(channel, member)
    if DM_SAMBUTAN and (not kena_raid):
        await sambut_lewat_dm(member)
    if LOG_KELUAR_MASUK:
        umur = (datetime.now(timezone.utc) - member.created_at).days
        catatan = f'{member.mention} ({member})\nAkun umur {umur} hari'
        if umur < AKUN_BARU_HARI:
            catatan += '  ⚠️ masih baru'
        undangan = await siapa_ngundang(member)
        if undangan:
            pengundang = undangan.inviter.mention if undangan.inviter else 'ga kebaca'
            catatan += f'\nMasuk lewat undangan `{undangan.code}` dari {pengundang} (udah dipakai {undangan.uses}x)'
            if undangan.inviter and (not undangan.inviter.bot):
                catat_undangan(member.guild.id, undangan.inviter.id, undangan.code, member.id)
        elif LACAK_UNDANGAN:
            catatan += '\nUndangannya ga kelacak (mungkin lewat Discovery)'
        await kirim_log(member.guild, 'Member masuk', catatan, discord.Color.green())

async def sambut_lewat_dm(member):
    baris = [f'Halo {member.display_name}, selamat dateng di **{member.guild.name}**.']
    if CHANNEL_ATURAN_ID:
        baris.append(f'Baca dulu aturannya di <#{CHANNEL_ATURAN_ID}> ya.')
    if CHANNEL_RESOURCES_ID:
        baris.append(f'Kalau nyari bahan bacaan atau info, ada di <#{CHANNEL_RESOURCES_ID}>.')
    baris.append('Kalau ada masalah atau mau lapor sesuatu, tinggal buka tiket, nanti diobrolin berdua sama moderator.')
    try:
        await member.send('\n'.join(baris))
    except discord.Forbidden:
        pass
    except discord.HTTPException as e:
        print(f'[join] DM sambutan gagal: {e}')

async def kirim_sambutan(channel, member):
    jumlah = len([m for m in member.guild.members if not m.bot])
    sapaan = f'{member.mention} baru gabung. Yuk disapa.'
    if KARTU_SAMBUTAN and kartu is not None:
        try:
            async with aiohttp.ClientSession() as sesi:
                async with sesi.get(str(member.display_avatar.replace(size=256, format='png'))) as r:
                    foto = await r.read()
            berkas = await asyncio.to_thread(kartu.buat_kartu, member.display_name, foto, jumlah, member.guild.name)
            await channel.send(sapaan, file=discord.File(berkas, filename='welcome.png'))
            return
        except Exception as e:
            print(f'[kartu] gagal bikin kartu, pakai teks biasa: {e}')
    embed = discord.Embed(title='Ada yang baru masuk', description=f'{member.mention} gabung. Sekarang jadi {jumlah} orang.', color=discord.Color.green())
    embed.set_thumbnail(url=member.display_avatar.url)
    try:
        await channel.send(embed=embed)
    except discord.Forbidden:
        print('[join] ga punya izin kirim ke channel sambutan')

@bot.tree.command(name='tessambutan', description='Coba tampilan kartu sambutan pakai akun lo')
@app_commands.checks.has_permissions(manage_guild=True)
async def tessambutan(interaction: discord.Interaction):
    await interaction.response.send_message('Lagi dibikin...', ephemeral=True)
    await kirim_sambutan(interaction.channel, interaction.user)

@bot.event
async def on_member_remove(member):
    if not LOG_KELUAR_MASUK:
        return
    if member.bot:
        await kirim_log(member.guild, 'Aplikasi dikeluarkan', f'{member} dikeluarin dari server', discord.Color.blurple())
        return
    await kirim_log(member.guild, 'Member keluar', f'{member} keluar dari server', discord.Color.orange())

def penguasa():

    async def cek(interaction: discord.Interaction):
        if not ROLE_PENGUASA:
            return True
        punya = {r.name for r in getattr(interaction.user, 'roles', [])}
        if punya & set(ROLE_PENGUASA):
            return True
        raise app_commands.CheckFailure('bukan penguasa')
    return app_commands.check(cek)

@bot.tree.command(name='warn', description='Kasih peringatan ke member')
@app_commands.describe(member='Siapa yang mau ditegur', alasan='Alasannya apa')
@app_commands.checks.has_permissions(moderate_members=True)
async def warn(interaction: discord.Interaction, member: discord.Member, alasan: str):
    await jalankan_warn(interaction, member, alasan)

@bot.tree.command(name='warnings', description='Liat riwayat peringatan member')
@app_commands.describe(member='Siapa yang mau dicek')
@app_commands.checks.has_permissions(moderate_members=True)
async def warnings(interaction: discord.Interaction, member: discord.Member):
    catatan = baca_warn().get(str(member.id), [])
    if not catatan:
        await interaction.response.send_message(f'{member.display_name} bersih, belum pernah kena.', ephemeral=True)
        return
    aktif = warn_aktif(catatan)
    id_aktif = {id(w) for w in aktif}
    baris = []
    for i, w in enumerate(catatan, 1):
        tanda = '🟡' if id(w) in id_aktif else '⚪'
        label = nomor_kasus(w['kasus']) if w.get('kasus') else '—'
        baris.append(f"{tanda} `{i}.` {label} {w['alasan']} (oleh {w['oleh']})")
    embed = discord.Embed(title=f'Peringatan {member.display_name}', description='\n'.join(baris[:15]), color=discord.Color.yellow())
    embed.add_field(name='Aktif', value=f'{len(aktif)}', inline=True)
    embed.add_field(name='Total', value=f'{len(catatan)}', inline=True)
    embed.add_field(name='Gugur setelah', value=f'{WARN_KADALUARSA_HARI} hari', inline=True)
    embed.set_footer(text='🟡 masih dihitung · ⚪ udah gugur · hapus satu pakai /hapuswarn1')
    await interaction.response.send_message(embed=embed, ephemeral=True)

@bot.tree.command(name='hapuswarn', description='Hapus semua peringatan member')
@app_commands.describe(member='Siapa yang mau dibersihin')
@app_commands.checks.has_permissions(manage_guild=True)
@penguasa()
async def hapuswarn(interaction: discord.Interaction, member: discord.Member):
    data = baca_warn()
    if data.pop(str(member.id), None) is None:
        await interaction.response.send_message('Ga ada yang perlu dihapus.', ephemeral=True)
        return
    tulis_warn(data)
    await interaction.response.send_message(f'Catatan {member.display_name} udah dibersihin.')

@bot.tree.command(name='timeout', description='Bisukan member sementara')
@app_commands.describe(member='Siapa yang mau dibisukan', menit='Berapa menit (maks 40320 atau 28 hari)', alasan='Alasannya apa')
@app_commands.checks.has_permissions(moderate_members=True)
@penguasa()
async def timeout(interaction: discord.Interaction, member: discord.Member, menit: int, alasan: str='Ga disebutin'):
    if menit < 1 or menit > 40320:
        await interaction.response.send_message('Menitnya antara 1 sampai 40320.', ephemeral=True)
        return
    try:
        await member.timeout(timedelta(minutes=menit), reason=alasan)
    except discord.Forbidden:
        await interaction.response.send_message('Ga bisa. Posisi role bot harus di atas role dia.', ephemeral=True)
        return
    kasus = kasus_baru()
    await acara.umumkan('timeout', user_id=member.id, guild_id=interaction.guild.id, menit=menit, alasan=alasan)
    await interaction.response.send_message(f'{nomor_kasus(kasus)} — {member.mention} dibisukan {menit} menit. Alasan: {alasan}')
    await kirim_log(interaction.guild, 'Timeout', f'{member.mention} dibisukan {menit} menit oleh {interaction.user.mention}\nAlasan: {alasan}', discord.Color.orange(), kasus=kasus)

@bot.tree.command(name='untimeout', description='Buka bisu member')
@app_commands.describe(member='Siapa yang mau dibuka')
@app_commands.checks.has_permissions(moderate_members=True)
@penguasa()
async def untimeout(interaction: discord.Interaction, member: discord.Member):
    try:
        await member.timeout(None, reason=f'Dibuka {interaction.user}')
    except discord.Forbidden:
        await interaction.response.send_message('Ga punya izin.', ephemeral=True)
        return
    await acara.umumkan('untimeout', user_id=member.id, guild_id=interaction.guild.id)
    await interaction.response.send_message(f'{member.mention} udah bisa ngomong lagi.')

@bot.tree.command(name='kick', description='Keluarkan member dari server')
@app_commands.describe(member='Siapa yang mau dikeluarin', alasan='Alasannya apa')
@app_commands.checks.has_permissions(kick_members=True)
@penguasa()
async def kick(interaction: discord.Interaction, member: discord.Member, alasan: str='Ga disebutin'):
    try:
        await member.kick(reason=f'{alasan} (oleh {interaction.user})')
    except discord.Forbidden:
        await interaction.response.send_message('Ga bisa. Cek izin bot dan posisi role-nya.', ephemeral=True)
        return
    kasus = kasus_baru()
    await interaction.response.send_message(f'{nomor_kasus(kasus)} — {member} dikeluarkan. Alasan: {alasan}')
    await kirim_log(interaction.guild, 'Kick', f'{member} (`{member.id}`) dikeluarkan {interaction.user.mention}\nAlasan: {alasan}', discord.Color.red(), kasus=kasus)

@bot.tree.command(name='ban', description='Blokir member, permanen atau sementara')
@app_commands.describe(member='Siapa yang mau diblokir', alasan='Alasannya apa', hari='Berapa hari. Isi 0 atau kosongin buat permanen')
@app_commands.checks.has_permissions(ban_members=True)
@penguasa()
async def ban(interaction: discord.Interaction, member: discord.Member, alasan: str='Ga disebutin', hari: int=0):
    if hari < 0 or hari > 3650:
        await interaction.response.send_message('Harinya antara 0 (permanen) sampai 3650.', ephemeral=True)
        return
    try:
        await member.ban(reason=f'{alasan} (oleh {interaction.user})', delete_message_days=0)
    except discord.Forbidden:
        await interaction.response.send_message('Ga bisa. Cek izin bot dan posisi role-nya.', ephemeral=True)
        return
    kasus = kasus_baru()
    if hari:
        sampai = datetime.now(timezone.utc) + timedelta(days=hari)
        catat = baca_tempban()
        catat[str(member.id)] = {'guild': interaction.guild.id, 'sampai': sampai.isoformat(), 'alasan': alasan, 'kasus': kasus, 'oleh': str(interaction.user)}
        tulis_tempban(catat)
        await interaction.response.send_message(f'{nomor_kasus(kasus)} — {member} diblokir **{hari} hari**. Kebuka otomatis <t:{int(sampai.timestamp())}:R>.\nAlasan: {alasan}')
        await kirim_log(interaction.guild, 'Tempban', f'{member} (`{member.id}`) diblokir {hari} hari oleh {interaction.user.mention}\nAlasan: {alasan}\nKebuka <t:{int(sampai.timestamp())}:F>', discord.Color.dark_red(), kasus=kasus)
        return
    await interaction.response.send_message(f'{nomor_kasus(kasus)} — {member} diblokir permanen. Alasan: {alasan}')
    await kirim_log(interaction.guild, 'Ban', f'{member} (`{member.id}`) diblokir {interaction.user.mention}\nAlasan: {alasan}', discord.Color.dark_red(), kasus=kasus)
CATATAN_TEMPBAN = 'tempban.json'

def baca_tempban():
    isi, aman = penyimpanan.baca(CATATAN_TEMPBAN, {})
    return isi if aman else {}

def tulis_tempban(data):
    penyimpanan.tulis(CATATAN_TEMPBAN, data)

@tasks.loop(minutes=5)
async def jaga_tempban():
    catat = baca_tempban()
    if not catat:
        return
    sekarang = datetime.now(timezone.utc)
    berubah = False
    for uid, isi in list(catat.items()):
        try:
            sampai = datetime.fromisoformat(isi['sampai'])
        except (KeyError, TypeError, ValueError):
            catat.pop(uid, None)
            berubah = True
            continue
        if sekarang < sampai:
            continue
        guild = bot.get_guild(isi.get('guild', 0))
        if guild is None:
            continue
        catat.pop(uid, None)
        berubah = True
        try:
            orang = await bot.fetch_user(int(uid))
            await guild.unban(orang, reason='Masa tempban habis')
            await kirim_log(guild, 'Tempban habis', f"{orang} (`{uid}`) dibuka otomatis, masa blokirnya udah lewat.\nAlasan dulu: {isi.get('alasan', '-')}", discord.Color.green(), kasus=isi.get('kasus'))
        except discord.NotFound:
            pass
        except Exception as e:
            print(f'[tempban] gagal buka {uid}: {e}')
    if berubah:
        tulis_tempban(catat)

@jaga_tempban.before_loop
async def _sebelum_tempban():
    await bot.wait_until_ready()

@bot.tree.command(name='bersihkan', description='Hapus sejumlah pesan terakhir di channel ini')
@app_commands.describe(jumlah='Berapa pesan (1 sampai 100)')
@app_commands.checks.has_permissions(manage_messages=True)
@penguasa()
async def bersihkan(interaction: discord.Interaction, jumlah: int):
    if jumlah < 1 or jumlah > 100:
        await interaction.response.send_message('Jumlahnya 1 sampai 100.', ephemeral=True)
        return
    await interaction.response.defer(ephemeral=True)
    kehapus = await interaction.channel.purge(limit=jumlah)
    await interaction.followup.send(f'{len(kehapus)} pesan kehapus.', ephemeral=True)
    await kirim_log(interaction.guild, 'Pesan dibersihkan', f'{len(kehapus)} pesan dihapus di {interaction.channel.mention} oleh {interaction.user.mention}', discord.Color.greyple())

@warn.error
@warnings.error
@hapuswarn.error
@timeout.error
@untimeout.error
@kick.error
@ban.error
@bersihkan.error
async def tolak(interaction: discord.Interaction, error):
    if isinstance(error, app_commands.CheckFailure) and (not isinstance(error, app_commands.MissingPermissions)):
        pesan = 'Perintah ini cuma buat ' + ' atau '.join(ROLE_PENGUASA) + '.'
    elif isinstance(error, app_commands.MissingPermissions):
        pesan = 'Lo ga punya izin buat perintah ini.'
    else:
        pesan = f'Ada yang error: {error}'
        print(f'[error] {error}')
    if interaction.response.is_done():
        await interaction.followup.send(pesan, ephemeral=True)
    else:
        await interaction.response.send_message(pesan, ephemeral=True)

@bot.event
async def on_message_delete(message):
    if not LOG_PESAN_DIHAPUS:
        return
    if message.author.bot or not message.guild:
        return
    isi = message.content[:500] if message.content else '(kosong atau cuma lampiran)'
    await kirim_log(message.guild, 'Pesan dihapus', f'Dari {message.author.mention} di {message.channel.mention}\n\n{isi}', discord.Color.red())

@bot.event
async def on_message_edit(sebelum, sesudah):
    if not LOG_PESAN_DIHAPUS:
        return
    if sebelum.author.bot or not sebelum.guild:
        return
    if sebelum.content == sesudah.content:
        return
    await kirim_log(sebelum.guild, 'Pesan diedit', f'Dari {sebelum.author.mention} di {sebelum.channel.mention}\n\nSebelum: {sebelum.content[:300]}\nSesudah: {sesudah.content[:300]}', discord.Color.blue())
BERKAS_PANEL = os.path.join(os.path.dirname(os.path.abspath(__file__)), 'reaction-role.json')
_panel_role = {}

def muat_panel():
    global _panel_role
    isi = penyimpanan.baca_atau_berhenti(BERKAS_PANEL, {}, 'panel reaction role')
    _panel_role = {}
    for kunci, peta in isi.items():
        try:
            _panel_role[int(kunci)] = peta
        except (TypeError, ValueError):
            continue
    print(f'[panel] {len(_panel_role)} panel role dimuat')

def simpan_panel():
    penyimpanan.tulis(BERKAS_PANEL, {str(k): v for k, v in _panel_role.items()})

def cari_role(guild, payload):
    peta = _panel_role.get(payload.message_id) or REACTION_ROLE.get(payload.message_id)
    if not peta:
        return None
    nama = peta.get(str(payload.emoji))
    if not nama:
        return None
    return discord.utils.get(guild.roles, name=nama)

@bot.event
async def on_raw_reaction_add(payload):
    if payload.guild_id is None or payload.member is None or payload.member.bot:
        return
    try:
        await urus_starboard(payload)
    except Exception as e:
        print(f'[starboard] error: {e}')
    guild = bot.get_guild(payload.guild_id)
    role = cari_role(guild, payload)
    if role is None:
        return
    try:
        await payload.member.add_roles(role, reason='Reaction role')
    except discord.Forbidden:
        print(f"[reaction] gagal kasih role '{role.name}'. Naikin posisi role bot")

@bot.event
async def on_raw_reaction_remove(payload):
    if payload.guild_id is None:
        return
    guild = bot.get_guild(payload.guild_id)
    if guild is None:
        return
    member = guild.get_member(payload.user_id)
    if member is None or member.bot:
        return
    role = cari_role(guild, payload)
    if role is None:
        return
    try:
        await member.remove_roles(role, reason='Reaction role dilepas')
    except discord.Forbidden:
        print(f"[reaction] gagal nyabut role '{role.name}'")

@bot.tree.command(name='panelrole', description='Bikin panel role, member tinggal mencet reaksi')
@app_commands.describe(judul='Judul panelnya', isi='Tulisan penjelasannya', pilihan='Format: emoji=NamaRole, pisahin pakai koma. Contoh: 🎮=Gamers, 🔔=Notif Acara')
@app_commands.checks.has_permissions(manage_guild=True)
async def panelrole(interaction: discord.Interaction, judul: str, pilihan: str, isi: str='Pencet reaksi di bawah buat ambil rolenya. Pencet lagi buat ngelepas.'):
    peta, ga_ada, salah = ({}, [], [])
    for bagian in pilihan.split(','):
        bagian = bagian.strip()
        if '=' not in bagian:
            if bagian:
                salah.append(bagian)
            continue
        emoji, nama = bagian.split('=', 1)
        emoji, nama = (emoji.strip(), nama.strip())
        if not emoji or not nama:
            salah.append(bagian)
            continue
        if discord.utils.get(interaction.guild.roles, name=nama) is None:
            ga_ada.append(nama)
            continue
        peta[emoji] = nama
    if salah:
        await interaction.response.send_message('Formatnya salah di: ' + ', '.join((f'`{x}`' for x in salah)) + '\nHarusnya `emoji=NamaRole`, dipisah koma.', ephemeral=True)
        return
    if ga_ada:
        await interaction.response.send_message('Role ini belum ada di server: ' + ', '.join((f'`{x}`' for x in ga_ada)) + '\nBikin dulu rolenya, ditulis persis sama.', ephemeral=True)
        return
    if not peta:
        await interaction.response.send_message('Ga ada pilihan yang kebaca.', ephemeral=True)
        return
    baris = '\n'.join((f'{e}  —  **{n}**' for e, n in peta.items()))
    embed = discord.Embed(title=judul, description=f'{isi}\n\n{baris}', color=discord.Color.blurple())
    await interaction.response.send_message('Panelnya gua kirim.', ephemeral=True)
    pesan = await interaction.channel.send(embed=embed)
    kepasang = []
    for emoji in peta:
        try:
            await pesan.add_reaction(emoji)
            kepasang.append(emoji)
        except Exception:
            pass
    _panel_role[pesan.id] = {e: n for e, n in peta.items() if e in kepasang}
    simpan_panel()
    lapor = f'Panel jadi, {len(kepasang)} pilihan kepasang.'
    if len(kepasang) < len(peta):
        gagal_emoji = [e for e in peta if e not in kepasang]
        lapor += '\nEmoji ini ga bisa dipasang: ' + ' '.join(gagal_emoji) + '. Kemungkinan emoji dari server lain.'
    lapor += '\nPosisi role Beach Guard harus di atas role yang dibagiin.'
    await interaction.followup.send(lapor, ephemeral=True)

@bot.tree.command(name='hapuspanel', description='Matiin panel role, pesannya tetep ada')
@app_commands.describe(pesan_id='ID pesan panelnya')
@app_commands.checks.has_permissions(manage_guild=True)
async def hapuspanel(interaction: discord.Interaction, pesan_id: str):
    try:
        nomor = int(pesan_id.strip())
    except ValueError:
        await interaction.response.send_message('ID-nya angka doang ya.', ephemeral=True)
        return
    if nomor in _panel_role:
        del _panel_role[nomor]
        simpan_panel()
        await interaction.response.send_message('Panel dimatiin. Reaksinya udah ga ngasih role lagi.', ephemeral=True)
    else:
        await interaction.response.send_message('Ga ada panel dengan ID itu.', ephemeral=True)

@bot.tree.command(name='panelverifikasi', description='Kirim panel verifikasi ke channel ini')
@app_commands.describe(judul='Judul panelnya', isi='Tulisan penjelasannya')
@app_commands.checks.has_permissions(manage_guild=True)
async def panelverifikasi(interaction: discord.Interaction, judul: str='Verifikasi', isi: str='Pencet reaksi di bawah buat masuk server.'):
    embed = discord.Embed(title=judul, description=isi, color=discord.Color.blurple())
    await interaction.response.send_message('Panel dikirim.', ephemeral=True)
    pesan = await interaction.channel.send(embed=embed)
    await pesan.add_reaction('✅')
    await interaction.followup.send(f'ID pesannya: `{pesan.id}`\nMasukin ke REACTION_ROLE di kode, formatnya:\n```python\n{pesan.id}: {{"✅": "NamaRole"}},\n```', ephemeral=True)
_cache_libur = {'tahun': None, 'data': {}}

async def ambil_libur(tahun):
    if _cache_libur['tahun'] == tahun and _cache_libur['data']:
        return _cache_libur['data']
    for pola in API_LIBUR:
        url = pola.format(tahun=tahun)
        try:
            async with aiohttp.ClientSession() as sesi:
                async with sesi.get(url, timeout=aiohttp.ClientTimeout(total=20)) as resp:
                    if resp.status != 200:
                        continue
                    isi = await resp.json(content_type=None)
        except Exception as e:
            print(f'[libur] {url} gagal: {e}')
            continue
        hasil = {}
        if isinstance(isi, dict) and 'data' not in isi:
            for tgl, b in isi.items():
                if not isinstance(b, dict) or not b.get('holiday'):
                    continue
                ringkas = b.get('summary') or b.get('description')
                if ringkas:
                    hasil[tgl] = ringkas[0] if isinstance(ringkas, list) else ringkas
        else:
            baris = isi.get('data', isi) if isinstance(isi, dict) else isi
            for b in baris or []:
                if not isinstance(b, dict):
                    continue
                tgl = b.get('date')
                nama = b.get('description') or b.get('name') or b.get('summary')
                if tgl and nama:
                    hasil[tgl] = nama if isinstance(nama, str) else nama[0]
        hasil = {t: n for t, n in hasil.items() if t.startswith(str(tahun))}
        if hasil:
            _cache_libur.update({'tahun': tahun, 'data': hasil})
            print(f'[libur] {len(hasil)} tanggal kebaca buat {tahun}')
            return hasil
    print('[libur] semua sumber gagal, pakai daftar peringatan aja')
    return {}

def baca_umum():
    isi, _aman = penyimpanan.baca(CATATAN_UMUM, {})
    return isi

def tulis_umum(data):
    penyimpanan.tulis(CATATAN_UMUM, data)

async def minta_tulisan_ai(nama):
    if tanya_ai is None:
        return (None, None)
    perintah = f'Hari ini {nama} di Indonesia. Tulis buat pengumuman server Discord.\nBaris 1: satu kalimat menjelaskan hari ini, hangat tapi ga lebay.\nBaris 2: satu kutipan pendek yang relevan.\nCuma dua baris itu, tanpa label, tanpa tanda kutip, tanpa nomor.'
    hasil = await tanya_ai([{'role': 'user', 'parts': [{'text': perintah}]}])
    if not hasil:
        return (None, None)
    baris = [b.strip() for b in hasil.split('\n') if b.strip()]
    deskripsi = baris[0] if baris else None
    kutipan = baris[1] if len(baris) > 1 else None
    return (deskripsi, kutipan)

def rakit_embed(info, tanggal_cantik, tanggal_merah):
    warna = discord.Color.from_rgb(220, 60, 60) if tanggal_merah else discord.Color.from_rgb(70, 130, 200)
    embed = discord.Embed(title=info['nama'], description=info.get('deskripsi') or '', color=warna, timestamp=datetime.now(timezone.utc))
    embed.set_author(name='Hari ini')
    if info.get('kutipan'):
        embed.add_field(name='\u200b', value=f"> *{info['kutipan']}*", inline=False)
    embed.add_field(name='Tanggal', value=tanggal_cantik, inline=True)
    embed.add_field(name='Status', value='Tanggal merah' if tanggal_merah else 'Hari peringatan', inline=True)
    berkas = None
    gambar = (info.get('gambar') or '').strip()
    if gambar.startswith('http'):
        embed.set_image(url=gambar)
    elif gambar:
        jalur = os.path.join(FOLDER_GAMBAR, gambar)
        if os.path.exists(jalur):
            berkas = discord.File(jalur, filename=gambar)
            embed.set_image(url=f'attachment://{gambar}')
        else:
            print(f"[umum] gambar '{jalur}' ga ketemu, embed dikirim tanpa gambar")
    embed.set_footer(text='Pengumuman otomatis')
    return (embed, berkas)

@tasks.loop(minutes=30)
async def cek_hari_besar():
    if CHANNEL_PENGUMUMAN_ID == 0:
        return
    sekarang = datetime.now(WIB)
    if sekarang.hour != JAM_KIRIM:
        return
    hari_ini = sekarang.strftime('%Y-%m-%d')
    catatan = baca_umum()
    if catatan.get('terakhir') == hari_ini:
        return
    libur = await ambil_libur(sekarang.year)
    nama_libur = libur.get(hari_ini)
    manual = HARI_PERINGATAN.get(sekarang.strftime('%m-%d'))
    catatan['terakhir'] = hari_ini
    tulis_umum(catatan)
    if not nama_libur and (not manual):
        return
    if manual:
        info = dict(manual)
        if nama_libur:
            info['nama'] = nama_libur
    else:
        deskripsi, kutipan = await minta_tulisan_ai(nama_libur)
        info = {'nama': nama_libur, 'deskripsi': deskripsi, 'kutipan': kutipan, 'gambar': cari_gambar_libur(nama_libur)}
    channel = bot.get_channel(CHANNEL_PENGUMUMAN_ID)
    if channel is None:
        print('[umum] channel pengumuman ga ketemu')
        return
    embed, berkas = rakit_embed(info, sekarang.strftime('%d %B %Y'), hari_ini in libur)
    try:
        if berkas:
            await channel.send(embed=embed, file=berkas)
        else:
            await channel.send(embed=embed)
        print(f"[umum] kirim: {info['nama']}")
    except discord.Forbidden:
        print('[umum] ga punya izin kirim ke channel pengumuman')

@cek_hari_besar.before_loop
async def sebelum_cek():
    await bot.wait_until_ready()

@bot.tree.command(name='pengumuman', description='Kirim pengumuman ke channel announcement')
@app_commands.describe(judul='Judul pengumuman', isi='Isi pengumumannya')
@app_commands.checks.has_permissions(manage_guild=True)
async def pengumuman(interaction: discord.Interaction, judul: str, isi: str):
    if CHANNEL_PENGUMUMAN_ID == 0:
        await interaction.response.send_message('CHANNEL_PENGUMUMAN_ID belum diisi di kode.', ephemeral=True)
        return
    channel = bot.get_channel(CHANNEL_PENGUMUMAN_ID)
    if channel is None:
        await interaction.response.send_message('Channelnya ga ketemu.', ephemeral=True)
        return
    embed = discord.Embed(title=judul, description=isi.replace('\\n', '\n'), color=discord.Color.gold(), timestamp=datetime.now(timezone.utc))
    embed.set_footer(text=f'Dari {interaction.user.display_name}')
    await channel.send(embed=embed)
    await interaction.response.send_message('Pengumuman terkirim.', ephemeral=True)
    await kirim_log(interaction.guild, 'Pengumuman dikirim', f'{interaction.user.mention} ngirim: {judul}', discord.Color.gold())

@bot.tree.command(name='ceklibur', description='Cek hari besar terdekat')
async def ceklibur(interaction: discord.Interaction):
    await interaction.response.defer()
    sekarang = datetime.now(WIB)
    libur = await ambil_libur(sekarang.year)
    depan = sorted(((t, n) for t, n in libur.items() if t >= sekarang.strftime('%Y-%m-%d')))
    if not depan:
        await interaction.followup.send('Ga ada data libur yang kebaca.')
        return
    baris = []
    for tgl, nama in depan[:5]:
        selisih = (datetime.strptime(tgl, '%Y-%m-%d').date() - sekarang.date()).days
        kapan = 'hari ini' if selisih == 0 else f'{selisih} hari lagi'
        baris.append(f'**{nama}**\n{tgl} ({kapan})')
    embed = discord.Embed(title='Hari besar terdekat', description='\n\n'.join(baris), color=discord.Color.blue())
    await interaction.followup.send(embed=embed)
_sesi = {}
_bubar = {}

def _isi_voice(channel):
    return [m for m in channel.members if not m.bot]

def _rakit_notif(channel, nama, mulai, selesai=False):
    if selesai:
        menit = max(1, int((datetime.now(timezone.utc) - mulai).total_seconds() // 60))
        embed = discord.Embed(title='Voice udah bubar', description=f'Tadi ada **{len(nama)} orang** nongkrong di {channel.mention} selama sekitar {menit} menit.', color=discord.Color.greyple())
        embed.add_field(name='Yang sempat gabung', value=', '.join(nama), inline=False)
        return embed
    if len(nama) == 1:
        judul = f'{nama[0]} lagi sendirian di voice'
        isi = f'Ada di {channel.mention}. Temenin gih.'
    else:
        judul = f'Lagi rame di {channel.mention}'
        isi = f'Sekarang ada **{len(nama)} orang** di dalam.'
    embed = discord.Embed(title=judul, description=isi, color=discord.Color.green())
    embed.add_field(name='Siapa aja', value=', '.join(nama), inline=False)
    embed.set_footer(text='Pesan ini diupdate sendiri, ga bakal spam')
    return embed
_dm_afk = {}
_tugas_afk = {}
_konfirmasi = {}

def _lagi_diem(kondisi):
    return bool(kondisi.self_mute or kondisi.self_deaf)

def _kebal_afk(member):
    punya = {r.name for r in member.roles}
    return any((nama in punya for nama in AFK_ROLE_BEBAS))

async def _lapor_afk(guild, teks, warna=discord.Color.dark_grey()):
    if not CHANNEL_LAPORAN_AFK_ID:
        return
    ch = bot.get_channel(CHANNEL_LAPORAN_AFK_ID)
    if ch is None:
        return
    try:
        await ch.send(embed=discord.Embed(description=teks, color=warna, timestamp=datetime.now(timezone.utc)))
    except discord.Forbidden:
        print('[afk] ga punya izin kirim ke channel laporan')

def _masih_diem(member, channel):
    return bool(member.voice and member.voice.channel == channel and _lagi_diem(member.voice))

class TombolMasihDiSini(discord.ui.View):

    def __init__(self, member_id):
        super().__init__(timeout=AFK_TOMBOL_DETIK)
        self.member_id = member_id
        self.ditekan = asyncio.Event()
        self.pesan = None

    async def interaction_check(self, interaction: discord.Interaction):
        if interaction.user.id != self.member_id:
            await interaction.response.send_message('Tombol ini bukan buat lo.', ephemeral=True)
            return False
        return True

    def _matiin(self):
        for anak in self.children:
            anak.disabled = True

    @discord.ui.button(label='Masih di sini', emoji='🙋', style=discord.ButtonStyle.success)
    async def masih(self, interaction: discord.Interaction, tombol: discord.ui.Button):
        self.ditekan.set()
        self._matiin()
        tombol.label = 'Udah dikonfirmasi'
        try:
            await interaction.response.edit_message(view=self)
        except discord.HTTPException:
            pass
        self.stop()

    async def on_timeout(self):
        self._matiin()
        if self.pesan:
            try:
                await self.pesan.edit(view=self)
            except discord.HTTPException:
                pass

def _sisa_jatah(member):
    if not AFK_MAKS_KONFIRMASI:
        return None
    return max(0, AFK_MAKS_KONFIRMASI - _konfirmasi.get(member.id, 0))

def _embed_peringatan(member, channel):
    embed = discord.Embed(title='Masih di situ ga?', description=f'{member.mention} udah diem **{AFK_WARN_DETIK // 60} menit** di {channel.mention}.\nPencet tombol di bawah dalam **{AFK_TOMBOL_DETIK} detik** kalau ga mau dipindah ke AFK. Mic boleh tetep mati.', color=discord.Color.yellow(), timestamp=datetime.now(timezone.utc))
    embed.set_author(name=member.display_name, icon_url=member.display_avatar.url)
    sisa = _sisa_jatah(member)
    if sisa is None:
        embed.set_footer(text='Didiemin sampai waktunya habis = otomatis dipindah')
    else:
        embed.set_footer(text=f'Sisa jatah konfirmasi: {sisa}. Nyalain mic sekali buat balikin jatahnya penuh.')
    return embed

def _embed_jatah_habis(member, channel):
    embed = discord.Embed(title='Jatah konfirmasi habis', description=f'{member.mention} udah konfirmasi **{AFK_MAKS_KONFIRMASI} kali** tapi mic-nya ga pernah nyala di {channel.mention}.\nKali ini ga ada tombol. **{AFK_TOMBOL_DETIK} detik** lagi dipindah ke AFK kalau mic-nya masih mati.', color=discord.Color.orange(), timestamp=datetime.now(timezone.utc))
    embed.set_author(name=member.display_name, icon_url=member.display_avatar.url)
    embed.set_footer(text='Buka mic buat batalin, terus jatahnya balik penuh')
    return embed

async def _minta_konfirmasi(member, channel):
    ch = bot.get_channel(CHANNEL_LAPORAN_AFK_ID) if CHANNEL_LAPORAN_AFK_ID else None
    if not AFK_TOMBOL_AKTIF or ch is None:
        await asyncio.sleep(AFK_TOMBOL_DETIK)
        return False
    if _sisa_jatah(member) == 0:
        try:
            await ch.send(content=member.mention, embed=_embed_jatah_habis(member, channel))
        except discord.Forbidden:
            print('[afk] ga punya izin kirim ke channel laporan')
        await asyncio.sleep(AFK_TOMBOL_DETIK)
        return False
    tampilan = TombolMasihDiSini(member.id)
    try:
        tampilan.pesan = await ch.send(content=member.mention, embed=_embed_peringatan(member, channel), view=tampilan)
    except discord.Forbidden:
        print('[afk] ga punya izin kirim ke channel laporan')
        await asyncio.sleep(AFK_TOMBOL_DETIK)
        return False
    try:
        await asyncio.wait_for(tampilan.ditekan.wait(), timeout=AFK_TOMBOL_DETIK)
        ditekan = True
    except asyncio.TimeoutError:
        ditekan = False
    except asyncio.CancelledError:
        tampilan.stop()
        await _tutup_peringatan(tampilan, 'Ga jadi, dia udah ngomong lagi.', discord.Color.greyple())
        raise
    tampilan.stop()
    if ditekan:
        _konfirmasi[member.id] = _konfirmasi.get(member.id, 0) + 1
        sisa = _sisa_jatah(member)
        catatan = f'Dikonfirmasi. Aman dari pantauan {AFK_KEBAL_MENIT} menit ke depan.'
        if sisa is not None:
            catatan += f' Sisa jatah: {sisa}.' if sisa else ' Jatahnya habis, konfirmasi berikutnya ga bisa lagi.'
        await _tutup_peringatan(tampilan, catatan, discord.Color.green())
    return ditekan

async def _tutup_peringatan(tampilan, catatan, warna):
    if not tampilan.pesan:
        return
    tampilan._matiin()
    try:
        embed = tampilan.pesan.embeds[0] if tampilan.pesan.embeds else discord.Embed()
        embed.color = warna
        embed.set_footer(text=catatan)
        await tampilan.pesan.edit(embed=embed, view=tampilan)
    except discord.HTTPException:
        pass

async def _hitung_mundur(member, channel):
    try:
        while True:
            await asyncio.sleep(AFK_WARN_DETIK)
            if not _masih_diem(member, channel):
                return
            try:
                afk_ch = member.guild.afk_channel
                afk_mention = afk_ch.mention if afk_ch else 'AFK'
                await member.send(f'lo diem terus di {channel.mention}. buka <#{CHANNEL_LAPORAN_AFK_ID}> terus pencet tombolnya dalam {AFK_TOMBOL_DETIK} detik kalau ga mau dipindah ke {afk_mention}' if AFK_TOMBOL_AKTIF and CHANNEL_LAPORAN_AFK_ID else f'lo diem terus di {channel.mention}. {AFK_TOMBOL_DETIK} detik lagi gua pindahin ke {afk_mention} kalau masih gini')
            except discord.Forbidden:
                pass
            lolos = await _minta_konfirmasi(member, channel)
            if lolos:
                print(f'[afk] {member.display_name} konfirmasi masih online')
                await asyncio.sleep(AFK_KEBAL_MENIT * 60)
                if not _masih_diem(member, channel):
                    return
                continue
            if not _masih_diem(member, channel):
                return
            afk = member.guild.afk_channel
            if afk is None:
                print('[afk] channel AFK belum diset di Server Settings')
                return
            try:
                await member.move_to(afk, reason='Auto AFK: diem kelamaan')
                await acara.umumkan('afk_dipindah', user_id=member.id, guild_id=member.guild.id, asal=channel.name)
                await _lapor_afk(member.guild, f'{member.mention} dipindah dari {channel.mention} ke {afk.mention}', discord.Color.orange())
                try:
                    afk_ch = member.guild.afk_channel
                    afk_mention = afk_ch.mention if afk_ch else 'AFK'
                    await member.send(random.choice(PESAN_AFK).format(asal=channel.mention, afk=afk_mention))
                except discord.Forbidden:
                    pass
                print(f'[afk] {member.display_name} dipindah ke AFK')
            except discord.Forbidden:
                print('[afk] GAGAL mindahin. Julian butuh izin Move Members')
            return
    except asyncio.CancelledError:
        pass
    finally:
        _tugas_afk.pop(member.id, None)

def _atur_pantauan(member, sebelum, sesudah):
    if not AUTO_AFK or member.bot or _kebal_afk(member):
        return
    afk = member.guild.afk_channel
    ch = sesudah.channel
    keluar = ch is None or (afk and ch.id == afk.id)
    if keluar or not _lagi_diem(sesudah):
        _konfirmasi.pop(member.id, None)
        tugas = _tugas_afk.pop(member.id, None)
        if tugas:
            tugas.cancel()
        if not keluar:
            acara.umumkan_nanti('voice_aktif', user_id=member.id, guild_id=member.guild.id)
        return
    if member.id not in _tugas_afk:
        _tugas_afk[member.id] = asyncio.create_task(_hitung_mundur(member, ch))

async def kabari_afk(member, asal):
    if not DM_SAAT_AFK and (not CHANNEL_LAPORAN_AFK_ID):
        return
    terakhir = _dm_afk.get(member.id)
    if terakhir and datetime.now(timezone.utc) - terakhir < timedelta(minutes=JEDA_DM_AFK_MENIT):
        return
    nama_asal = asal.name if asal else 'voice'
    asal_mention = asal.mention if asal else 'voice'
    if CHANNEL_LAPORAN_AFK_ID:
        ch = bot.get_channel(CHANNEL_LAPORAN_AFK_ID)
        if ch:
            embed = discord.Embed(description=f'{member.mention} kelempar ke AFK dari {asal_mention} karena ga ada suara.', color=discord.Color.dark_grey(), timestamp=datetime.now(timezone.utc))
            embed.set_author(name=member.display_name, icon_url=member.display_avatar.url)
            try:
                await ch.send(embed=embed)
            except discord.Forbidden:
                print('[afk] ga punya izin kirim ke channel laporan')
    if DM_SAAT_AFK:
        afk_ch = member.guild.afk_channel
        afk_mention = afk_ch.mention if afk_ch else 'AFK'
        pesan = random.choice(PESAN_AFK).format(asal=asal_mention, afk=afk_mention)
        try:
            await member.send(pesan)
            print(f'[afk] DM terkirim ke {member.display_name}')
        except discord.Forbidden:
            print(f'[afk] DM {member.display_name} ketutup, dilewat')
    _dm_afk[member.id] = datetime.now(timezone.utc)

@bot.event
async def on_voice_state_update(member, sebelum, sesudah):
    if member.bot:
        return
    _atur_pantauan(member, sebelum, sesudah)
    if sebelum.channel == sesudah.channel:
        return
    afk = member.guild.afk_channel
    if afk and sesudah.channel and (sesudah.channel.id == afk.id):
        await kabari_afk(member, sebelum.channel)
    if CHANNEL_NOTIF_VOICE_ID == 0:
        return
    teks = bot.get_channel(CHANNEL_NOTIF_VOICE_ID)
    if teks is None:
        return
    if sesudah.channel and sesudah.channel != afk:
        ch = sesudah.channel
        orang = _isi_voice(ch)
        nama = [m.display_name for m in orang]
        if ch.id in _sesi:
            sesi = _sesi[ch.id]
            for n in nama:
                if n not in sesi['nama']:
                    sesi['nama'].append(n)
            try:
                if sesi.get('pesan'):
                    await sesi['pesan'].edit(embed=_rakit_notif(ch, nama, sesi['mulai']))
            except discord.NotFound:
                _sesi.pop(ch.id, None)
        else:
            bubar = _bubar.get(ch.id)
            if bubar and datetime.now(timezone.utc) - bubar < timedelta(minutes=JEDA_SESI_MENIT):
                print(f'[voice] sesi baru di {ch.name} dilewat, masih dalam jeda')
                return
            _sesi[ch.id] = {'pesan': None, 'mulai': datetime.now(timezone.utc), 'nama': list(nama)}
            try:
                pesan = await teks.send(embed=_rakit_notif(ch, nama, datetime.now(timezone.utc)))
                if ch.id in _sesi:
                    _sesi[ch.id]['pesan'] = pesan
            except discord.Forbidden:
                _sesi.pop(ch.id, None)
                print('[voice] ga punya izin kirim ke channel notif')
                return
            print(f'[voice] sesi mulai di {ch.name}')
    if sebelum.channel and sebelum.channel.id in _sesi:
        ch = sebelum.channel
        sesi = _sesi[ch.id]
        if _isi_voice(ch):
            try:
                if sesi.get('pesan'):
                    await sesi['pesan'].edit(embed=_rakit_notif(ch, [m.display_name for m in _isi_voice(ch)], sesi['mulai']))
            except discord.NotFound:
                _sesi.pop(ch.id, None)
        else:
            await _tutup_sesi_voice(ch, sesi)
            print(f'[voice] sesi bubar di {ch.name}')

async def _tutup_sesi_voice(ch, sesi):
    menit = max(1, int((datetime.now(timezone.utc) - sesi['mulai']).total_seconds() // 60))
    nama = list(sesi.get('nama', []))
    pesan = sesi.get('pesan')
    if pesan is not None:
        try:
            await pesan.delete()
        except (discord.NotFound, discord.Forbidden):
            pass
        except Exception as e:
            print(f'[voice] gagal hapus pesan ajakan: {e}')
    try:
        await kirim_log(ch.guild, 'Voice bubar', f"**{ch.name}** rame sekitar **{menit} menit** sama **{len(nama)} orang**.\nYang sempat gabung: {', '.join(nama) or 'ga kecatat'}", discord.Color.greyple())
    except Exception as e:
        print(f'[voice] gagal nyatet ke log: {e}')
    _sesi.pop(ch.id, None)
    _bubar[ch.id] = datetime.now(timezone.utc)
CATATAN_SESI_VOICE = 'sesi-voice.json'
JEDA_JAGA_VOICE = 60

def _simpan_sesi_voice():
    isi = {}
    for ch_id, sesi in _sesi.items():
        pesan = sesi.get('pesan')
        if pesan is None:
            continue
        isi[str(ch_id)] = {'pesan': pesan.id, 'teks': pesan.channel.id, 'mulai': sesi['mulai'].isoformat(), 'nama': list(sesi.get('nama', []))}
    penyimpanan.tulis(CATATAN_SESI_VOICE, isi, rapi=2)

def _baca_sesi_voice():
    isi, aman = penyimpanan.baca(CATATAN_SESI_VOICE, {})
    if not aman:
        print('[voice] catatan sesi rusak, dianggap ga ada sesi yang jalan')
        return {}
    return isi

async def _pulihin_sesi_voice():
    simpanan = _baca_sesi_voice()
    if not simpanan:
        return
    hidup, ditutup = (0, 0)
    for ch_id, catat in simpanan.items():
        try:
            suara = bot.get_channel(int(ch_id))
            teks = bot.get_channel(int(catat['teks']))
            if suara is None or teks is None:
                continue
            try:
                pesan = await teks.fetch_message(int(catat['pesan']))
            except (discord.NotFound, discord.Forbidden):
                continue
            mulai = datetime.fromisoformat(catat['mulai'])
            nama_lama = list(catat.get('nama', []))
            orang = _isi_voice(suara)
            if orang:
                sekarang = [m.display_name for m in orang]
                for n in sekarang:
                    if n not in nama_lama:
                        nama_lama.append(n)
                _sesi[suara.id] = {'pesan': pesan, 'mulai': mulai, 'nama': nama_lama}
                await pesan.edit(embed=_rakit_notif(suara, sekarang, mulai))
                hidup += 1
            else:
                await _tutup_sesi_voice(suara, {'pesan': pesan, 'mulai': mulai, 'nama': nama_lama})
                ditutup += 1
        except Exception as e:
            print(f'[voice] gagal mungut sesi {ch_id}: {e}')
    print(f'[voice] sesi dipungut dari sebelum restart: {hidup} diterusin, {ditutup} ditutup')
    _simpan_sesi_voice()

@tasks.loop(seconds=JEDA_JAGA_VOICE)
async def jaga_sesi_voice():
    if CHANNEL_NOTIF_VOICE_ID == 0:
        return
    teks = bot.get_channel(CHANNEL_NOTIF_VOICE_ID)
    if teks is None:
        return
    berubah = False
    for guild in bot.guilds:
        afk = guild.afk_channel
        for ch in guild.voice_channels:
            if afk and ch.id == afk.id:
                continue
            orang = _isi_voice(ch)
            nama = [m.display_name for m in orang]
            sesi = _sesi.get(ch.id)
            if orang and (not sesi):
                bubar = _bubar.get(ch.id)
                if bubar and datetime.now(timezone.utc) - bubar < timedelta(minutes=JEDA_SESI_MENIT):
                    continue
                _sesi[ch.id] = {'pesan': None, 'mulai': datetime.now(timezone.utc), 'nama': list(nama)}
                try:
                    pesan = await teks.send(embed=_rakit_notif(ch, nama, datetime.now(timezone.utc)))
                    if ch.id in _sesi:
                        _sesi[ch.id]['pesan'] = pesan
                except discord.Forbidden:
                    _sesi.pop(ch.id, None)
                    print('[voice] ga punya izin kirim ke channel notif')
                    continue
                berubah = True
                print(f'[voice] penjaga: sesi {ch.name} dibikin susulan')
                continue
            if not sesi:
                continue
            if not orang:
                await _tutup_sesi_voice(ch, sesi)
                berubah = True
                print(f'[voice] penjaga: sesi {ch.name} ditutup susulan')
                continue
            for n in nama:
                if n not in sesi['nama']:
                    sesi['nama'].append(n)
                    berubah = True
            if sesi.get('terakhir') != nama:
                sesi['terakhir'] = list(nama)
                try:
                    if sesi.get('pesan'):
                        await sesi['pesan'].edit(embed=_rakit_notif(ch, nama, sesi['mulai']))
                    berubah = True
                except discord.NotFound:
                    _sesi.pop(ch.id, None)
                    berubah = True
                except Exception as e:
                    print(f'[voice] penjaga gagal ngupdate {ch.name}: {e}')
    if berubah:
        _simpan_sesi_voice()

@jaga_sesi_voice.before_loop
async def _sebelum_jaga_sesi_voice():
    await bot.wait_until_ready()
KATEGORI = ['Beasiswa', 'Lowongan', 'Kuliah', 'Tutorial', 'Lainnya']

def baca_link():
    return penyimpanan.baca_atau_berhenti(CATATAN_LINK, [], 'daftar link')

def tulis_link(data):
    penyimpanan.tulis(CATATAN_LINK, data, rapi=2)

@bot.tree.command(name='simpan', description='Simpan link biar ga tenggelam di chat')
@app_commands.describe(judul='Judulnya apa', link='Alamat webnya', kategori='Masuk kategori mana', catatan='Keterangan tambahan (opsional)')
@app_commands.choices(kategori=[app_commands.Choice(name=k, value=k) for k in KATEGORI])
async def simpan(interaction: discord.Interaction, judul: str, link: str, kategori: app_commands.Choice[str], catatan: str=''):
    if not link.startswith('http'):
        await interaction.response.send_message('Linknya harus diawali http atau https.', ephemeral=True)
        return
    data = baca_link()
    if any((d['link'] == link for d in data)):
        await interaction.response.send_message('Link itu udah pernah disimpen.', ephemeral=True)
        return
    catat = {'id': max((d['id'] for d in data), default=0) + 1, 'judul': judul, 'link': link, 'kategori': kategori.value, 'catatan': catatan, 'oleh': interaction.user.display_name, 'waktu': datetime.now(timezone.utc).strftime('%d %b %Y')}
    data.append(catat)
    tulis_link(data)
    embed = discord.Embed(title=judul, url=link, description=catatan or '', color=discord.Color.teal())
    embed.add_field(name='Kategori', value=kategori.value, inline=True)
    embed.add_field(name='Nomor', value=f"#{catat['id']}", inline=True)
    embed.set_footer(text=f"Disimpen {catat['oleh']} • cari lagi pakai /cari")
    tujuan = bot.get_channel(CHANNEL_RESOURCES_ID) if CHANNEL_RESOURCES_ID else None
    if tujuan:
        await tujuan.send(embed=embed)
        await interaction.response.send_message(f'Kesimpen dan diposting ke {tujuan.mention}.', ephemeral=True)
    else:
        await interaction.response.send_message(embed=embed)

@bot.tree.command(name='cari', description='Cari link yang pernah disimpen')
@app_commands.describe(kata='Kata kunci, boleh judul atau kategori')
async def cari(interaction: discord.Interaction, kata: str):
    kunci = kata.lower()
    hasil = [d for d in baca_link() if kunci in d['judul'].lower() or kunci in d['kategori'].lower() or kunci in (d.get('catatan') or '').lower()]
    if not hasil:
        await interaction.response.send_message(f"Ga nemu apa apa buat '{kata}'.", ephemeral=True)
        return
    baris = [f"**#{d['id']} {d['judul']}**\n{d['link']}\n{d['kategori']} • {d['oleh']} • {d['waktu']}" for d in hasil[-8:]]
    embed = discord.Embed(title=f'Hasil cari: {kata}', description='\n\n'.join(baris), color=discord.Color.teal())
    embed.set_footer(text=f'{len(hasil)} ketemu, ditampilin {len(baris)} terbaru')
    await interaction.response.send_message(embed=embed)

@bot.tree.command(name='daftarlink', description='Liat semua link per kategori')
@app_commands.describe(kategori='Mau liat kategori apa')
@app_commands.choices(kategori=[app_commands.Choice(name=k, value=k) for k in KATEGORI])
async def daftarlink(interaction: discord.Interaction, kategori: app_commands.Choice[str]):
    hasil = [d for d in baca_link() if d['kategori'] == kategori.value]
    if not hasil:
        await interaction.response.send_message(f'Belum ada yang disimpen di kategori {kategori.value}.', ephemeral=True)
        return
    baris = [f"**#{d['id']}** [{d['judul']}]({d['link']}) • {d['oleh']}" for d in hasil[-15:]]
    embed = discord.Embed(title=f'Kategori {kategori.value}', description='\n'.join(baris), color=discord.Color.teal())
    embed.set_footer(text=f'Total {len(hasil)} link')
    await interaction.response.send_message(embed=embed)

@bot.tree.command(name='hapuslink', description='Hapus link dari daftar')
@app_commands.describe(nomor='Nomor link yang mau dihapus')
@app_commands.checks.has_permissions(manage_messages=True)
async def hapuslink(interaction: discord.Interaction, nomor: int):
    data = baca_link()
    sisa = [d for d in data if d['id'] != nomor]
    if len(sisa) == len(data):
        await interaction.response.send_message(f'Nomor #{nomor} ga ketemu.', ephemeral=True)
        return
    tulis_link(sisa)
    await interaction.response.send_message(f'Link #{nomor} udah dihapus.', ephemeral=True)
_strike = {}
_riwayat_chat = {}
TEGURAN = {'kata': ['eh jaga mulut dikit, pesan lo gua hapus', 'kata kata lo kena filter. santai aja ngomongnya', 'pesan lo dihapus, ada kata yang ga boleh di server ini'], 'kata_ringan': ['santai dikit ngomongnya, pesan lo gua hapus', 'yang ini kena filter. tenang, ga gua catat kok'], 'undangan': ['jangan promosi server lain di sini ya', 'link undangan server lain gua hapus, maaf'], 'spam': ['pelan pelan ngetiknya, kecepetan itu', 'santai bro, chatnya kebanyakan dalam sedetik'], 'kapital': ['ga usah caps lock semua, kaya lagi teriak', 'kecilin huruf lo dikit, berisik bacanya'], 'mention': ['kebanyakan nge-tag orang sekaligus, jangan gitu', 'jangan mention rame rame, ganggu yang lain']}

def _bersih(teks):
    tukar = {'0': 'o', '1': 'i', '3': 'e', '4': 'a', '5': 's', '7': 't', '@': 'a', '$': 's'}
    hasil = teks.lower()
    for a, b in tukar.items():
        hasil = hasil.replace(a, b)
    return hasil

def _pola(daftar):
    return [re.compile('(?<![a-z])' + re.escape(k) + '[a-z]{0,2}(?![a-z])') for k in daftar]
_POLA_BERAT = _pola(KATA_BERAT)
_POLA_RINGAN = _pola(KATA_RINGAN)

def _kebal(member, channel):
    if channel.id in AUTOMOD_BEBAS:
        return True
    izin = channel.permissions_for(member)
    return izin.manage_messages or izin.administrator

def _cek_pelanggaran(pesan):
    isi = pesan.content
    if FILTER_KATA:
        polos = _bersih(isi)
        if any((p.search(polos) for p in _POLA_BERAT)):
            return 'kata'
        if any((p.search(polos) for p in _POLA_RINGAN)):
            return 'kata_ringan'
    if BLOKIR_UNDANGAN:
        polos = isi.lower().replace(' ', '')
        if 'discord.gg/' in polos or 'discord.com/invite' in polos:
            return 'undangan'
    if MAX_MENTION and len(pesan.mentions) > MAX_MENTION:
        return 'mention'
    if BATAS_KAPITAL and len(isi) >= KAPITAL_MINIMAL:
        huruf = [c for c in isi if c.isalpha()]
        if huruf:
            persen = sum((1 for c in huruf if c.isupper())) / len(huruf) * 100
            if persen >= KAPITAL_PERSEN:
                return 'kapital'
    if ANTI_SPAM:
        sekarang = datetime.now(timezone.utc)
        jejak = [w for w in _riwayat_chat.get(pesan.author.id, []) if sekarang - w < timedelta(seconds=SPAM_DETIK)]
        jejak.append(sekarang)
        _riwayat_chat[pesan.author.id] = jejak
        if len(jejak) >= SPAM_JUMLAH:
            _riwayat_chat[pesan.author.id] = []
            return 'spam'
    return None

@bot.event
async def on_message(pesan):
    if not AUTOMOD_AKTIF or pesan.author.bot or (not pesan.guild):
        return
    if _kebal(pesan.author, pesan.channel):
        return
    jenis = _cek_pelanggaran(pesan)
    if jenis is None:
        return
    try:
        await pesan.delete()
    except (discord.Forbidden, discord.NotFound):
        pass
    teguran = random.choice(TEGURAN[jenis])
    try:
        await pesan.author.send(teguran)
    except discord.Forbidden:
        try:
            await pesan.channel.send(f'{pesan.author.mention} {teguran}', delete_after=8)
        except discord.Forbidden:
            pass
    if jenis == 'spam' and AUTO_SLOWMODE:
        await coba_slowmode(pesan)
    if jenis == 'kata_ringan':
        await kirim_log(pesan.guild, 'Automod: kata ringan', f'Pesan {pesan.author.mention} dihapus di {pesan.channel.mention}\nGa dihitung sebagai pelanggaran', discord.Color.light_grey())
        return
    sekarang = datetime.now(timezone.utc)
    catat = [w for w in _strike.get(pesan.author.id, []) if sekarang - w < timedelta(minutes=10)]
    catat.append(sekarang)
    _strike[pesan.author.id] = catat
    await kirim_log(pesan.guild, f'Automod: {jenis}', f'Pesan {pesan.author.mention} dihapus di {pesan.channel.mention}\nPelanggaran ke-{len(catat)} dalam 10 menit terakhir', discord.Color.orange())
    if len(catat) >= STRIKE_SEBELUM_TIMEOUT:
        _strike[pesan.author.id] = []
        try:
            await pesan.author.timeout(timedelta(minutes=TIMEOUT_OTOMATIS_MENIT), reason=f'Automod: {jenis} berulang')
            await pesan.author.send(f'lo kena timeout {TIMEOUT_OTOMATIS_MENIT} menit, udah {STRIKE_SEBELUM_TIMEOUT} kali kena filter. tenangin diri dulu')
            await kirim_log(pesan.guild, 'Automod: timeout otomatis', f'{pesan.author.mention} kena timeout {TIMEOUT_OTOMATIS_MENIT} menit', discord.Color.red())
        except discord.Forbidden:
            print('[automod] ga bisa timeout, cek posisi role bot')
_panas = {}
_slowmode_bot = {}

async def coba_slowmode(pesan):
    channel = pesan.channel
    sekarang = datetime.now(timezone.utc)
    jejak = [(w, o) for w, o in _panas.get(channel.id, []) if sekarang - w < timedelta(seconds=SLOWMODE_JENDELA)]
    jejak.append((sekarang, pesan.author.id))
    _panas[channel.id] = jejak
    if len(jejak) < SLOWMODE_PEMICU:
        return
    if len({o for _w, o in jejak}) < SLOWMODE_ORANG_BEDA:
        return
    if channel.slowmode_delay >= SLOWMODE_DETIK:
        return
    try:
        await channel.edit(slowmode_delay=SLOWMODE_DETIK, reason='Automod: channel lagi rame banget')
    except discord.Forbidden:
        print('[automod] mau pasang slowmode tapi ga punya izin Manage Channels')
        return
    except discord.HTTPException as e:
        print(f'[automod] slowmode gagal: {e}')
        return
    _panas[channel.id] = []
    _slowmode_bot[channel.id] = sekarang
    await channel.send(f'Rame banget di sini. Gua pasang jeda **{SLOWMODE_DETIK} detik** biar kebaca. Kebuka sendiri {SLOWMODE_LAMA_MENIT} menit lagi.', delete_after=60)
    await kirim_log(pesan.guild, 'Automod: slowmode otomatis', f'{channel.mention} diset {SLOWMODE_DETIK} detik karena {len(jejak)} kejadian spam dari {len({o for _w, o in jejak})} orang berbeda', discord.Color.orange())

@tasks.loop(minutes=1)
async def lepas_slowmode():
    sekarang = datetime.now(timezone.utc)
    for ch_id, kapan in list(_slowmode_bot.items()):
        if sekarang - kapan < timedelta(minutes=SLOWMODE_LAMA_MENIT):
            continue
        _slowmode_bot.pop(ch_id, None)
        channel = bot.get_channel(ch_id)
        if channel is None:
            continue
        if channel.slowmode_delay != SLOWMODE_DETIK:
            continue
        try:
            await channel.edit(slowmode_delay=0, reason='Automod: channelnya udah adem')
            await kirim_log(channel.guild, 'Automod: slowmode dilepas', f'{channel.mention} balik normal', discord.Color.green())
        except discord.HTTPException:
            pass

@lepas_slowmode.before_loop
async def _sebelum_lepas_slowmode():
    await bot.wait_until_ready()

@bot.tree.command(name='automod', description='Liat setelan automod yang lagi aktif')
@app_commands.checks.has_permissions(manage_guild=True)
async def automod(interaction: discord.Interaction):
    baris = [f"Status: **{('nyala' if AUTOMOD_AKTIF else 'mati')}**", f"Filter kata: {('nyala' if FILTER_KATA else 'mati')} ({len(KATA_BERAT)} berat + {len(KATA_RINGAN)} ringan)", f"Blokir undangan: {('nyala' if BLOKIR_UNDANGAN else 'mati')}", f'Anti spam: {SPAM_JUMLAH} pesan / {SPAM_DETIK} detik' if ANTI_SPAM else 'Anti spam: mati', f'Batas kapital: {KAPITAL_PERSEN}%' if BATAS_KAPITAL else 'Batas kapital: mati', f'Maks mention: {MAX_MENTION} orang', f'Slowmode otomatis: {SLOWMODE_DETIK} detik kalau {SLOWMODE_PEMICU} spam dari {SLOWMODE_ORANG_BEDA}+ orang dalam {SLOWMODE_JENDELA} detik' if AUTO_SLOWMODE else 'Slowmode otomatis: mati', f'Timeout otomatis: setelah {STRIKE_SEBELUM_TIMEOUT} pelanggaran, selama {TIMEOUT_OTOMATIS_MENIT} menit', f'Channel bebas: {len(AUTOMOD_BEBAS)}']
    embed = discord.Embed(title='Setelan Automod', description='\n'.join(baris), color=discord.Color.orange())
    embed.set_footer(text='Admin dan moderator kebal automod')
    await interaction.response.send_message(embed=embed, ephemeral=True)
BERKAS_KUNCI = os.path.join(os.path.dirname(os.path.abspath(__file__)), 'channel-terkunci.json')

@bot.tree.command(name='unban', description='Buka blokir orang yang pernah di-ban')
@app_commands.describe(user_id='ID orangnya (angka)', alasan='Alasannya apa')
@app_commands.checks.has_permissions(ban_members=True)
@penguasa()
async def unban(interaction: discord.Interaction, user_id: str, alasan: str='Ga disebutin'):
    try:
        nomor = int(user_id.strip())
    except ValueError:
        await interaction.response.send_message('ID-nya angka doang ya. Liat daftarnya pakai `/daftarban`.', ephemeral=True)
        return
    try:
        orang = await bot.fetch_user(nomor)
    except discord.NotFound:
        await interaction.response.send_message('Ga ada akun dengan ID itu.', ephemeral=True)
        return
    except discord.HTTPException as e:
        await interaction.response.send_message(f'Gagal ngecek: {e}', ephemeral=True)
        return
    try:
        await interaction.guild.unban(orang, reason=f'{alasan} (oleh {interaction.user})')
    except discord.NotFound:
        await interaction.response.send_message(f'{orang} emang lagi ga di-ban.', ephemeral=True)
        return
    except discord.Forbidden:
        await interaction.response.send_message('Ga bisa. Cek izin Ban Members punya bot.', ephemeral=True)
        return
    await interaction.response.send_message(f'{orang} udah dibuka blokirnya.')
    await kirim_log(interaction.guild, 'Unban', f'{orang} ({orang.id}) dibuka {interaction.user.mention}\nAlasan: {alasan}', discord.Color.green())

@bot.tree.command(name='daftarban', description='Liat siapa aja yang lagi di-ban')
@app_commands.checks.has_permissions(ban_members=True)
async def daftarban(interaction: discord.Interaction):
    await interaction.response.defer(ephemeral=True)
    catatan = []
    try:
        async for entri in interaction.guild.bans(limit=50):
            catatan.append(entri)
    except discord.Forbidden:
        await interaction.followup.send('Bot ga punya izin liat daftar ban.', ephemeral=True)
        return
    if not catatan:
        await interaction.followup.send('Ga ada yang di-ban. Bersih.', ephemeral=True)
        return
    baris = [f"**{e.user}**\n`{e.user.id}` • {(e.reason or 'tanpa alasan')[:80]}" for e in catatan[:15]]
    embed = discord.Embed(title=f'{len(catatan)} orang lagi di-ban', description='\n\n'.join(baris), color=discord.Color.dark_red())
    embed.set_footer(text='Buka pakai /unban user_id:<angka di atas>')
    await interaction.followup.send(embed=embed, ephemeral=True)

@bot.tree.command(name='hapuswarn1', description='Hapus satu peringatan tertentu')
@app_commands.describe(member='Siapa yang mau dikoreksi', nomor='Nomor peringatannya, liat di /warnings')
@app_commands.checks.has_permissions(manage_guild=True)
@penguasa()
async def hapuswarn1(interaction: discord.Interaction, member: discord.Member, nomor: int):
    data = baca_warn()
    catatan = data.get(str(member.id), [])
    if not catatan:
        await interaction.response.send_message(f'{member.display_name} ga punya catatan.', ephemeral=True)
        return
    if not 1 <= nomor <= len(catatan):
        await interaction.response.send_message(f'Nomornya antara 1 sampai {len(catatan)}. Cek pakai `/warnings`.', ephemeral=True)
        return
    kebuang = catatan.pop(nomor - 1)
    if catatan:
        data[str(member.id)] = catatan
    else:
        del data[str(member.id)]
    tulis_warn(data)
    await interaction.response.send_message(f"Peringatan ke-{nomor} punya {member.display_name} dihapus.\nYang dihapus: {kebuang['alasan']}\nSisa: {len(catatan)} catatan.")
    await kirim_log(interaction.guild, 'Peringatan dihapus', f"{interaction.user.mention} hapus peringatan ke-{nomor} punya {member.mention}\nIsinya: {kebuang['alasan']}", discord.Color.yellow())

@bot.tree.command(name='slowmode', description='Atur jeda antar pesan di channel ini')
@app_commands.describe(detik='0 sampai 21600 (6 jam). Isi 0 buat matiin')
@app_commands.checks.has_permissions(manage_channels=True)
async def slowmode(interaction: discord.Interaction, detik: int):
    if not 0 <= detik <= 21600:
        await interaction.response.send_message('Antara 0 sampai 21600 detik aja.', ephemeral=True)
        return
    try:
        await interaction.channel.edit(slowmode_delay=detik, reason=f'Diatur {interaction.user}')
    except discord.Forbidden:
        await interaction.response.send_message('Bot ga punya izin Manage Channels di sini.', ephemeral=True)
        return
    if detik:
        await interaction.response.send_message(f'Slowmode nyala: satu pesan tiap {detik} detik.')
    else:
        await interaction.response.send_message('Slowmode dimatiin.')
    await kirim_log(interaction.guild, 'Slowmode', f'{interaction.channel.mention} diset {detik} detik oleh {interaction.user.mention}', discord.Color.greyple())

@bot.tree.command(name='bersihkanuser', description='Hapus pesan dari satu orang aja')
@app_commands.describe(member='Pesan siapa yang mau dihapus', jumlah='Dicari dalam berapa pesan terakhir (1-200)')
@app_commands.checks.has_permissions(manage_messages=True)
@penguasa()
async def bersihkanuser(interaction: discord.Interaction, member: discord.Member, jumlah: int=100):
    if not 1 <= jumlah <= 200:
        await interaction.response.send_message('Jumlahnya 1 sampai 200.', ephemeral=True)
        return
    await interaction.response.defer(ephemeral=True)
    try:
        kehapus = await interaction.channel.purge(limit=jumlah, check=lambda m: m.author.id == member.id)
    except discord.Forbidden:
        await interaction.followup.send('Bot ga punya izin hapus pesan di sini.', ephemeral=True)
        return
    await interaction.followup.send(f'{len(kehapus)} pesan punya {member.display_name} kehapus (dari {jumlah} pesan terakhir).', ephemeral=True)
    await kirim_log(interaction.guild, 'Pesan user dibersihkan', f'{len(kehapus)} pesan {member.mention} dihapus di {interaction.channel.mention} oleh {interaction.user.mention}', discord.Color.greyple())

async def _atur_kunci(channel, guild, kunci):
    izin = channel.overwrites_for(guild.default_role)
    if kunci:
        if izin.send_messages is False:
            return False
        izin.send_messages = False
    else:
        if izin.send_messages is not False:
            return False
        izin.send_messages = None
    try:
        await channel.set_permissions(guild.default_role, overwrite=izin, reason='Lockdown' if kunci else 'Buka lockdown')
        return True
    except discord.Forbidden:
        return False

@bot.tree.command(name='kunci', description='Kunci channel ini, member ga bisa nulis')
@app_commands.describe(semua='Kunci SEMUA channel teks, bukan cuma yang ini')
@app_commands.checks.has_permissions(manage_channels=True)
@penguasa()
async def kunci(interaction: discord.Interaction, semua: bool=False):
    await interaction.response.defer()
    guild = interaction.guild
    if not semua:
        berubah = await _atur_kunci(interaction.channel, guild, True)
        await interaction.followup.send('Channel ini dikunci. Buka lagi pakai `/buka`.' if berubah else 'Channel ini emang udah kekunci.')
        if berubah:
            await kirim_log(guild, 'Channel dikunci', f'{interaction.channel.mention} dikunci {interaction.user.mention}', discord.Color.red())
        return
    kena = await kunci_server(guild, f'manual oleh {interaction.user}')
    await interaction.followup.send(f'{len(kena)} channel dikunci. Buka semua pakai `/buka semua:True`.')

@bot.tree.command(name='buka', description='Buka kunci channel')
@app_commands.describe(semua='Buka SEMUA channel yang tadi dikunci')
@app_commands.checks.has_permissions(manage_channels=True)
@penguasa()
async def buka(interaction: discord.Interaction, semua: bool=False):
    await interaction.response.defer()
    guild = interaction.guild
    if not semua:
        berubah = await _atur_kunci(interaction.channel, guild, False)
        await interaction.followup.send('Channel ini dibuka.' if berubah else 'Channel ini emang ga kekunci.')
        if berubah:
            await kirim_log(guild, 'Channel dibuka', f'{interaction.channel.mention} dibuka {interaction.user.mention}', discord.Color.green())
        return
    jumlah = await buka_server(guild, f'manual oleh {interaction.user}')
    await interaction.followup.send(f'{jumlah} channel dibuka lagi.')
_jejak_join = {}

async def kunci_server(guild, alasan):
    kena = []
    for channel in guild.text_channels:
        try:
            if await _atur_kunci(channel, guild, True):
                kena.append(channel.id)
        except Exception as e:
            print(f'[raid] gagal ngunci {channel.name}: {e}')
    if kena:
        penyimpanan.tulis(BERKAS_KUNCI, {'guild': guild.id, 'channel': kena, 'kapan': datetime.now(timezone.utc).isoformat(), 'alasan': alasan})
    print(f'[raid] {len(kena)} channel dikunci ({alasan})')
    return kena

async def buka_server(guild, alasan):
    isi, aman = penyimpanan.baca(BERKAS_KUNCI, {})
    daftar = isi.get('channel', []) if aman else []
    jumlah = 0
    for ch_id in daftar:
        channel = guild.get_channel(ch_id)
        if channel is None:
            continue
        try:
            if await _atur_kunci(channel, guild, False):
                jumlah += 1
        except Exception as e:
            print(f'[raid] gagal ngebuka {channel.name}: {e}')
    penyimpanan.tulis(BERKAS_KUNCI, {})
    print(f'[raid] {jumlah} channel dibuka ({alasan})')
    return jumlah

async def cek_raid(member):
    if not ANTI_RAID:
        return False
    # [BUG FIX] Buffer join dibuat per-guild supaya join di server lain
    # nggak kepakai buat ngerakit "rombongan" yang bikin false-positive raid.
    sekarang = datetime.now(timezone.utc)
    jejak = _jejak_join.setdefault(member.guild.id, [])
    jejak.append((sekarang, member))
    batas = sekarang - timedelta(seconds=RAID_DETIK)
    while jejak and jejak[0][0] < batas:
        jejak.pop(0)
    if len(jejak) < RAID_JUMLAH:
        return False
    rombongan = [m for _w, m in jejak]
    baru = [m for m in rombongan if sekarang - m.created_at < timedelta(days=AKUN_BARU_HARI)]
    _jejak_join[member.guild.id] = []
    daftar = '\n'.join((f'{m} (`{m.id}`) — akun umur {(sekarang - m.created_at).days} hari' for m in rombongan[:10]))
    isi = f'**{len(rombongan)} akun masuk dalam {RAID_DETIK} detik.**\n{len(baru)} di antaranya akun baru (< {AKUN_BARU_HARI} hari).\n\n{daftar}'
    if RAID_AUTO_KUNCI:
        kena = await kunci_server(member.guild, 'raid kedeteksi')
        isi += f'\n\n**{len(kena)} channel gua kunci otomatis.**\nKebuka sendiri {RAID_KUNCI_MENIT} menit lagi kalau ga ada yang ngapa ngapain. Buka lebih cepat pakai `/buka semua:True`.'
        if not buka_otomatis.is_running():
            buka_otomatis.start(member.guild)
    else:
        isi += '\n\nKunci otomatis lagi mati. Kunci manual pakai `/kunci semua:True`.'
    await kirim_log(member.guild, '⚠️ KEMUNGKINAN RAID', isi, discord.Color.red())
    if ROLE_SIAGA and CHANNEL_LOG_ID:
        channel = member.guild.get_channel(CHANNEL_LOG_ID)
        role = discord.utils.get(member.guild.roles, name=ROLE_SIAGA)
        if channel and role:
            try:
                await channel.send(f'{role.mention} ada yang aneh, cek di atas.')
            except discord.Forbidden:
                pass
    try:
        acara.umumkan_nanti('raid_on', guild_id=member.guild.id, jumlah=len(rombongan))
    except Exception as e:
        print(f'[raid] gagal umumin raid_on: {e}')
    return True

@tasks.loop(minutes=RAID_KUNCI_MENIT, count=2)
async def buka_otomatis(guild):
    if buka_otomatis.current_loop == 0:
        return
    jumlah = await buka_server(guild, 'buka otomatis setelah lockdown raid')
    if jumlah:
        await kirim_log(guild, 'Lockdown dibuka otomatis', f'{jumlah} channel dibuka lagi setelah {RAID_KUNCI_MENIT} menit.', discord.Color.green())

@bot.event
async def on_member_ban(guild, user):
    pelaku, alasan = await _cari_pelaku(guild, discord.AuditLogAction.ban, user)
    await kirim_log(guild, 'Ban tercatat', f'{user} (`{user.id}`) di-ban' + (f' oleh {pelaku.mention}' if pelaku else '') + (f'\nAlasan: {alasan}' if alasan else ''), discord.Color.dark_red())

@bot.event
async def on_member_unban(guild, user):
    pelaku, alasan = await _cari_pelaku(guild, discord.AuditLogAction.unban, user)
    await kirim_log(guild, 'Unban tercatat', f'{user} (`{user.id}`) dibuka blokirnya' + (f' oleh {pelaku.mention}' if pelaku else ''), discord.Color.green())

async def _cari_pelaku(guild, aksi, sasaran):
    try:
        async for entri in guild.audit_logs(limit=6, action=aksi):
            if entri.target and entri.target.id == sasaran.id:
                return (entri.user, entri.reason)
    except discord.Forbidden:
        print('[log] bot ga punya izin View Audit Log')
    except Exception as e:
        print(f'[log] audit log error: {e}')
    return (None, None)
_jejak_hapus = {}

async def catat_perusakan(guild, pelaku, apa, nama):
    if not ANTI_NUKE or pelaku is None:
        return
    if pelaku.id == bot.user.id or pelaku.id == guild.owner_id:
        return
    sekarang = datetime.now(timezone.utc)
    jejak = [w for w in _jejak_hapus.get(pelaku.id, []) if sekarang - w < timedelta(seconds=NUKE_DETIK)]
    jejak.append(sekarang)
    _jejak_hapus[pelaku.id] = jejak
    await kirim_log(guild, f'{apa} dihapus', f'**{nama}** dihapus {pelaku.mention}\nPenghapusan ke-{len(jejak)} dalam {NUKE_DETIK} detik terakhir', discord.Color.orange())
    if len(jejak) < NUKE_JUMLAH:
        return
    _jejak_hapus[pelaku.id] = []
    kasus = kasus_baru()
    dicabut = []
    if NUKE_CABUT_ROLE and isinstance(pelaku, discord.Member):
        buang = [r for r in pelaku.roles if r.name != '@everyone' and (not r.managed)]
        try:
            await pelaku.remove_roles(*buang, reason=f'Anti nuke {nomor_kasus(kasus)}')
            dicabut = [r.name for r in buang]
        except discord.Forbidden:
            print('[nuke] mau cabut role tapi posisi role bot kalah tinggi')
        except Exception as e:
            print(f'[nuke] gagal cabut role: {e}')
    isi = f'**{pelaku.mention} ngehapus {len(jejak)} {apa.lower()} dalam {NUKE_DETIK} detik.**\n\nIni pola perusakan, bukan beres beres biasa. Kemungkinan akunnya kebobolan.'
    if dicabut:
        isi += f"\n\n**Semua role-nya gua cabut** sebagai rem darurat:\n{', '.join(dicabut)}\nKalau ternyata salah tebak, tinggal dibalikin manual."
    else:
        isi += '\n\nRole-nya BELUM dicabut. Cek sekarang juga.'
    await kirim_log(guild, '🚨 KEMUNGKINAN NUKE', isi, discord.Color.red(), kasus=kasus)
    if ROLE_SIAGA and CHANNEL_LOG_ID:
        channel = guild.get_channel(CHANNEL_LOG_ID)
        role = discord.utils.get(guild.roles, name=ROLE_SIAGA)
        if channel and role:
            try:
                await channel.send(f'{role.mention} DARURAT, baca yang di atas.')
            except discord.Forbidden:
                pass

@bot.event
async def on_guild_channel_delete(channel):
    pelaku, _alasan = await _cari_pelaku_umum(channel.guild, discord.AuditLogAction.channel_delete, channel.id)
    await catat_perusakan(channel.guild, pelaku, 'Channel', channel.name)

@bot.event
async def on_guild_role_delete(role):
    pelaku, _alasan = await _cari_pelaku_umum(role.guild, discord.AuditLogAction.role_delete, role.id)
    await catat_perusakan(role.guild, pelaku, 'Role', role.name)

@bot.event
async def on_member_update(sebelum, sesudah):
    if not PANTAU_ROLE_ADMIN:
        return
    baru = [r for r in sesudah.roles if r not in sebelum.roles]
    if not baru:
        return
    bahaya = []
    for r in baru:
        izin = r.permissions
        if izin.administrator:
            bahaya.append((r, 'Administrator'))
        elif izin.manage_guild:
            bahaya.append((r, 'Manage Server'))
        elif izin.manage_roles:
            bahaya.append((r, 'Manage Roles'))
        elif izin.manage_channels:
            bahaya.append((r, 'Manage Channels'))
    if not bahaya:
        return
    pelaku, _alasan = await _cari_pelaku_umum(sesudah.guild, discord.AuditLogAction.member_role_update, sesudah.id)
    rincian = '\n'.join((f'**{r.name}** — izin {ket}' for r, ket in bahaya))
    await kirim_log(sesudah.guild, '⚠️ Role berizin tinggi dikasih', f'{sesudah.mention} baru dapet:\n{rincian}\n\n' + (f'Yang ngasih: {pelaku.mention}' if pelaku else 'Ga kebaca siapa yang ngasih') + '\n\nKalau ini bukan lo yang ngelakuin, cabut sekarang juga.', discord.Color.red())

async def _cari_pelaku_umum(guild, aksi, target_id):
    try:
        async for entri in guild.audit_logs(limit=6, action=aksi):
            if entri.target and getattr(entri.target, 'id', None) == target_id:
                return (entri.user, entri.reason)
    except discord.Forbidden:
        print('[nuke] bot ga punya izin View Audit Log, anti nuke jadi buta')
    except Exception as e:
        print(f'[nuke] audit log error: {e}')
    return (None, None)
_undangan = {}

async def segarkan_undangan(guild):
    if not LACAK_UNDANGAN:
        return
    try:
        _undangan[guild.id] = {i.code: i.uses for i in await guild.invites()}
    except discord.Forbidden:
        print('[undangan] bot ga punya izin Manage Server, pelacakan mati')
    except Exception as e:
        print(f'[undangan] gagal disegerin: {e}')

async def siapa_ngundang(member):
    if not LACAK_UNDANGAN:
        return None
    lama = _undangan.get(member.guild.id, {})
    try:
        sekarang = await member.guild.invites()
    except (discord.Forbidden, discord.HTTPException):
        return None
    ketemu = None
    for undangan in sekarang:
        if undangan.uses > lama.get(undangan.code, 0):
            ketemu = undangan
            break
    _undangan[member.guild.id] = {i.code: i.uses for i in sekarang}
    return ketemu
CATATAN_UNDANGAN = 'undangan.json'

def baca_undangan():
    isi, aman = penyimpanan.baca(CATATAN_UNDANGAN, {})
    return isi if aman else {}

def tulis_undangan(data):
    penyimpanan.tulis(CATATAN_UNDANGAN, data)

def catat_undangan(guild_id, inviter_id, kode, member_id):
    data = baca_undangan()
    guild_data = data.setdefault(str(guild_id), {})
    inviter_data = guild_data.setdefault(str(inviter_id), {})
    riwayat = inviter_data.setdefault('riwayat', [])
    if str(member_id) not in riwayat:
        riwayat.append(str(member_id))
    inviter_data['total'] = len(riwayat)
    tulis_undangan(data)

@bot.tree.command(name='undanganku', description='Liat berapa orang yang berhasil lo ajak masuk')
async def cmd_undanganku(interaction: discord.Interaction):
    data = baca_undangan().get(str(interaction.guild.id), {})
    punya = data.get(str(interaction.user.id), {})
    total = punya.get('total', 0)
    embed = discord.Embed(title='📨  Catatan Undangan', description=f'Lo udah berhasil ngajak **{total} orang** gabung ke server ini.', color=discord.Color.green())
    if total > 0:
        embed.set_footer(text='Makasih udah bantu ramein server!')
    await interaction.response.send_message(embed=embed)

@bot.tree.command(name='topundang', description='Liat siapa yang paling banyak ngajak orang')
async def cmd_topundang(interaction: discord.Interaction):
    data = baca_undangan().get(str(interaction.guild.id), {})
    if not data:
        await interaction.response.send_message('Belum ada catatan undangan di server ini.', ephemeral=True)
        return
    urut = sorted(data.items(), key=lambda x: x[1].get('total', 0), reverse=True)
    atas = urut[:10]
    baris = []
    for i, (uid, info) in enumerate(atas, 1):
        total = info.get('total', 0)
        if total <= 0:
            continue
        anggota = interaction.guild.get_member(int(uid))
        nama = anggota.display_name if anggota else 'Member keluar'
        baris.append(f'**{i}.** {nama} — {total} orang')
    if not baris:
        await interaction.response.send_message('Belum ada yang berhasil ngajak masuk.', ephemeral=True)
        return
    embed = discord.Embed(title='📨  Papan Peringkat Undangan', description='\\n'.join(baris), color=discord.Color.gold())
    await interaction.response.send_message(embed=embed)

def baca_starboard():
    isi, aman = penyimpanan.baca(CATATAN_STARBOARD, {})
    return isi if aman else {}

def tulis_starboard(data):
    penyimpanan.tulis(CATATAN_STARBOARD, data)

async def urus_starboard(payload):
    if not STARBOARD_AKTIF or not CHANNEL_STARBOARD_ID:
        return
    if str(payload.emoji) != STARBOARD_EMOJI or payload.guild_id is None:
        return
    if payload.channel_id == CHANNEL_STARBOARD_ID:
        return
    guild = bot.get_guild(payload.guild_id)
    asal = guild.get_channel(payload.channel_id) if guild else None
    if asal is None:
        return
    try:
        pesan = await asal.fetch_message(payload.message_id)
    except (discord.NotFound, discord.Forbidden):
        return
    if pesan.author.bot:
        return
    reaksi = discord.utils.get(pesan.reactions, emoji=STARBOARD_EMOJI)
    jumlah = reaksi.count if reaksi else 0
    if jumlah < STARBOARD_AMBANG:
        return
    papan = bot.get_channel(CHANNEL_STARBOARD_ID)
    if papan is None:
        return
    catat = baca_starboard()
    kunci = str(payload.message_id)
    if kunci in catat:
        try:
            lama = await papan.fetch_message(catat[kunci])
            await lama.edit(content=f'{STARBOARD_EMOJI} **{jumlah}**  ·  {asal.mention}')
        except (discord.NotFound, discord.Forbidden):
            catat.pop(kunci, None)
            tulis_starboard(catat)
        return
    embed = discord.Embed(description=pesan.content or '', color=discord.Color.gold(), timestamp=pesan.created_at)
    embed.set_author(name=pesan.author.display_name, icon_url=pesan.author.display_avatar.url)
    embed.add_field(name='\u200b', value=f'[lompat ke pesannya]({pesan.jump_url})', inline=False)
    gambar = next((a for a in pesan.attachments if (a.content_type or '').startswith('image/')), None)
    if gambar:
        embed.set_image(url=gambar.url)
    try:
        kiriman = await papan.send(f'{STARBOARD_EMOJI} **{jumlah}**  ·  {asal.mention}', embed=embed)
    except discord.Forbidden:
        print('[starboard] ga punya izin kirim ke channel starboard')
        return
    catat[kunci] = kiriman.id
    tulis_starboard(catat)

@bot.tree.command(name='say', description='Bikin bot ngomong di channel ini')
@app_commands.describe(teks='Yang mau diomongin', channel='Kirim ke channel lain (opsional)')
@app_commands.checks.has_permissions(manage_guild=True)
@penguasa()
async def say(interaction: discord.Interaction, teks: str, channel: discord.TextChannel=None):
    tujuan = channel or interaction.channel
    try:
        await tujuan.send(teks.replace('\\n', '\n'))
    except discord.Forbidden:
        await interaction.response.send_message(f'Ga punya izin kirim ke {tujuan.mention}.', ephemeral=True)
        return
    await interaction.response.send_message(f'Terkirim ke {tujuan.mention}.', ephemeral=True)
    await kirim_log(interaction.guild, 'Bot disuruh ngomong', f'{interaction.user.mention} di {tujuan.mention}\n\n{teks[:500]}', discord.Color.greyple())

@bot.tree.command(name='embed', description='Kirim pesan kotak rapi pakai muka bot')
@app_commands.describe(judul='Judulnya', isi='Isinya, pakai \\n buat ganti baris', warna='Kode warna hex, misal 4AA8D8', channel='Kirim ke channel lain (opsional)', gambar='Link gambar (opsional)')
@app_commands.checks.has_permissions(manage_guild=True)
@penguasa()
async def embed_perintah(interaction: discord.Interaction, judul: str, isi: str, warna: str='4AA8D8', channel: discord.TextChannel=None, gambar: str=''):
    try:
        nilai = int(warna.lstrip('#'), 16)
    except ValueError:
        await interaction.response.send_message('Warnanya ditulis hex, misal `4AA8D8`.', ephemeral=True)
        return
    tujuan = channel or interaction.channel
    kotak = discord.Embed(title=judul, description=isi.replace('\\n', '\n'), color=discord.Color(nilai))
    if gambar.startswith('http'):
        kotak.set_image(url=gambar)
    try:
        await tujuan.send(embed=kotak)
    except discord.Forbidden:
        await interaction.response.send_message(f'Ga punya izin kirim ke {tujuan.mention}.', ephemeral=True)
        return
    except discord.HTTPException as e:
        await interaction.response.send_message(f'Ditolak Discord: {e}', ephemeral=True)
        return
    await interaction.response.send_message(f'Terkirim ke {tujuan.mention}.', ephemeral=True)
    await kirim_log(interaction.guild, 'Embed dikirim', f'{interaction.user.mention} di {tujuan.mention}\nJudul: {judul}', discord.Color.greyple())

class AlasanWarn(discord.ui.Modal, title='Kasih peringatan'):
    alasan = discord.ui.TextInput(label='Alasannya apa', placeholder='Ditulis apa adanya, ini kecatat permanen', max_length=300, required=True, style=discord.TextStyle.paragraph)

    def __init__(self, sasaran):
        super().__init__()
        self.sasaran = sasaran

    async def on_submit(self, interaction: discord.Interaction):
        await jalankan_warn(interaction, self.sasaran, str(self.alasan))

async def jalankan_warn(interaction, member, alasan):
    if member.bot:
        await interaction.response.send_message('Bot ga bisa ditegur.', ephemeral=True)
        return
    data = baca_warn()
    kunci = str(member.id)
    meta = data.setdefault('_meta', {})
    meta['kasus'] = meta.get('kasus', 0) + 1
    kasus = meta['kasus']
    data.setdefault(kunci, []).append({'kasus': kasus, 'alasan': alasan, 'oleh': str(interaction.user), 'waktu': datetime.now(timezone.utc).isoformat()})
    tulis_warn(data)
    semua = data[kunci]
    aktif = warn_aktif(semua)
    jumlah = len(aktif)
    menit = 0
    for ambang in sorted(WARN_ESKALASI):
        if jumlah >= ambang:
            menit = WARN_ESKALASI[ambang]
    kata = f'{nomor_kasus(kasus)} — {member.mention} kena peringatan.\nAlasan: {alasan}\nPeringatan aktif: **{jumlah}**' + (f' (dari {len(semua)} total, sisanya udah gugur)' if len(semua) > jumlah else '')
    await acara.umumkan('warn', user_id=member.id, guild_id=interaction.guild.id, jumlah=jumlah, alasan=alasan, kasus=kasus)
    kena_timeout = False
    if menit:
        try:
            await member.timeout(timedelta(minutes=menit), reason=f'Eskalasi {jumlah} peringatan ({nomor_kasus(kasus)})')
            kena_timeout = True
            await acara.umumkan('timeout', user_id=member.id, guild_id=interaction.guild.id, menit=menit, alasan=f'eskalasi {jumlah} peringatan')
            lama = f'{menit // 60} jam' if menit >= 60 else f'{menit} menit'
            kata += f'\n\n⚠️ **Otomatis kena timeout {lama}** karena udah {jumlah} peringatan dalam {WARN_KADALUARSA_HARI} hari.'
        except discord.Forbidden:
            kata += '\n\n⚠️ Harusnya kena timeout otomatis, tapi posisi role bot ada di bawah role dia.'
    await interaction.response.send_message(kata)
    try:
        pesan_dm = f'Lo kena peringatan di {interaction.guild.name} ({nomor_kasus(kasus)}). Alasan: {alasan}\nIni peringatan aktif ke-{jumlah} lo. Peringatan gugur sendiri setelah {WARN_KADALUARSA_HARI} hari.'
        if kena_timeout:
            pesan_dm += '\nLo lagi dibisukan sementara karena udah kebanyakan.'
        await member.send(pesan_dm)
    except discord.Forbidden:
        pass
    await kirim_log(interaction.guild, 'Peringatan', f'{member.mention} ditegur {interaction.user.mention}\nAlasan: {alasan}\nAktif: {jumlah} · Total: {len(semua)}' + (f'\nOtomatis timeout {menit} menit' if kena_timeout else ''), discord.Color.yellow(), kasus=kasus)

@bot.tree.context_menu(name='Tegur orang ini')
@app_commands.checks.has_permissions(moderate_members=True)
async def menu_warn(interaction: discord.Interaction, member: discord.Member):
    await interaction.response.send_modal(AlasanWarn(member))

@bot.tree.context_menu(name='Tegur penulisnya')
@app_commands.checks.has_permissions(moderate_members=True)
async def menu_warn_pesan(interaction: discord.Interaction, message: discord.Message):
    if not isinstance(message.author, discord.Member):
        await interaction.response.send_message('Penulisnya udah ga ada di server.', ephemeral=True)
        return
    await interaction.response.send_modal(AlasanWarn(message.author))

@bot.tree.context_menu(name='Hapus pesan ini')
@app_commands.checks.has_permissions(manage_messages=True)
async def menu_hapus(interaction: discord.Interaction, message: discord.Message):
    isi = message.content[:400] or '(kosong atau cuma lampiran)'
    penulis = message.author
    try:
        await message.delete()
    except discord.Forbidden:
        await interaction.response.send_message('Ga punya izin hapus pesan di sini.', ephemeral=True)
        return
    except discord.NotFound:
        await interaction.response.send_message('Pesannya udah ga ada.', ephemeral=True)
        return
    kasus = kasus_baru()
    await interaction.response.send_message(f'{nomor_kasus(kasus)} — pesan {penulis.display_name} dihapus.', ephemeral=True)
    await kirim_log(interaction.guild, 'Pesan dihapus manual', f'Punya {penulis.mention} di {message.channel.mention}\nDihapus {interaction.user.mention}\n\n{isi}', discord.Color.red(), kasus=kasus)

@bot.tree.context_menu(name='Bersihin pesan dia')
@app_commands.checks.has_permissions(manage_messages=True)
async def menu_bersihin(interaction: discord.Interaction, message: discord.Message):
    penulis = message.author
    await interaction.response.defer(ephemeral=True)
    try:
        kehapus = await interaction.channel.purge(limit=50, check=lambda m: m.author.id == penulis.id)
    except discord.Forbidden:
        await interaction.followup.send('Ga punya izin hapus pesan di sini.', ephemeral=True)
        return
    kasus = kasus_baru()
    await interaction.followup.send(f'{nomor_kasus(kasus)} — {len(kehapus)} pesan punya {penulis.display_name} kehapus.', ephemeral=True)
    await kirim_log(interaction.guild, 'Pesan dibersihkan (klik kanan)', f'{len(kehapus)} pesan {penulis.mention} dihapus di {interaction.channel.mention} oleh {interaction.user.mention}', discord.Color.greyple(), kasus=kasus)

@bot.tree.context_menu(name='Info singkat')
async def menu_info(interaction: discord.Interaction, member: discord.Member):
    sekarang = datetime.now(timezone.utc)
    umur = (sekarang - member.created_at).days
    catatan = baca_warn().get(str(member.id), [])
    aktif = warn_aktif(catatan)
    baris = [f'**{member}** · `{member.id}`', f'Akun umur **{umur} hari**' + ('  ⚠️ masih baru' if umur < AKUN_BARU_HARI else '')]
    if member.joined_at:
        baris.append(f'Gabung <t:{int(member.joined_at.timestamp())}:R>')
    baris.append(f'Peringatan aktif: **{len(aktif)}** (total {len(catatan)})')
    if member.is_timed_out():
        baris.append(f'🔇 Lagi dibisukan sampai <t:{int(member.timed_out_until.timestamp())}:R>')
    embed = discord.Embed(description='\n'.join(baris), color=member.color if member.color.value else discord.Color.blurple())
    embed.set_thumbnail(url=member.display_avatar.url)
    await interaction.response.send_message(embed=embed, ephemeral=True)

@menu_warn.error
@menu_warn_pesan.error
@menu_hapus.error
@menu_bersihin.error
@menu_info.error
async def _tolak_menu(interaction: discord.Interaction, error):
    await tolak(interaction, error)

@bot.tree.command(name='userinfo', description='Info lengkap soal member')
@app_commands.describe(member='Kosongin kalau mau liat punya sendiri')
async def userinfo(interaction: discord.Interaction, member: discord.Member=None):
    member = member or interaction.user
    sekarang = datetime.now(timezone.utc)
    umur_akun = (sekarang - member.created_at).days
    lama_gabung = (sekarang - member.joined_at).days if member.joined_at else None
    embed = discord.Embed(title=member.display_name, description=f'{member.mention} • `{member.id}`', color=member.color if member.color.value else discord.Color.blurple(), timestamp=sekarang)
    embed.set_thumbnail(url=member.display_avatar.url)
    embed.add_field(name='Akun dibikin', value=f'<t:{int(member.created_at.timestamp())}:D>\n{umur_akun} hari lalu', inline=True)
    if member.joined_at:
        embed.add_field(name='Gabung server', value=f'<t:{int(member.joined_at.timestamp())}:D>\n{lama_gabung} hari lalu', inline=True)
    if umur_akun < AKUN_BARU_HARI:
        embed.add_field(name='⚠️ Perhatian', value=f'Akun baru, umurnya di bawah {AKUN_BARU_HARI} hari', inline=False)
    role = [r.mention for r in reversed(member.roles) if r.name != '@everyone']
    embed.add_field(name=f'Role ({len(role)})', value=' '.join(role[:15]) if role else 'ga punya role', inline=False)
    jumlah_warn = len(baca_warn().get(str(member.id), []))
    status = []
    if jumlah_warn:
        status.append(f'{jumlah_warn} peringatan')
    if member.is_timed_out():
        status.append(f'lagi dibisukan sampai <t:{int(member.timed_out_until.timestamp())}:R>')
    if member.bot:
        status.append('ini aplikasi, bukan orang')
    embed.add_field(name='Catatan', value='\n'.join(status) if status else 'bersih, ga ada apa apa', inline=False)
    await interaction.response.send_message(embed=embed)

@bot.tree.command(name='serverinfo', description='Ringkasan server ini')
async def serverinfo(interaction: discord.Interaction):
    guild = interaction.guild
    manusia = sum((1 for m in guild.members if not m.bot))
    aplikasi = guild.member_count - manusia
    embed = discord.Embed(title=guild.name, description=guild.description or '', color=discord.Color.blurple(), timestamp=datetime.now(timezone.utc))
    if guild.icon:
        embed.set_thumbnail(url=guild.icon.url)
    embed.add_field(name='Member', value=f'{manusia} orang\n{aplikasi} aplikasi', inline=True)
    embed.add_field(name='Channel', value=f'{len(guild.text_channels)} teks\n{len(guild.voice_channels)} voice', inline=True)
    embed.add_field(name='Role', value=f'{len(guild.roles) - 1}', inline=True)
    embed.add_field(name='Dibikin', value=f'<t:{int(guild.created_at.timestamp())}:D>', inline=True)
    embed.add_field(name='Boost', value=f'Level {guild.premium_tier}\n{guild.premium_subscription_count} boost', inline=True)
    embed.add_field(name='Pemilik', value=guild.owner.mention if guild.owner else 'ga kebaca', inline=True)
    if guild.afk_channel:
        embed.add_field(name='Channel AFK', value=f'{guild.afk_channel.mention} (setelah {guild.afk_timeout // 60} menit)', inline=False)
    embed.set_footer(text=f'ID: {guild.id}')
    await interaction.response.send_message(embed=embed)
HURUF_POLL = ['🇦', '🇧', '🇨', '🇩', '🇪']

@bot.tree.command(name='poll', description='Bikin polling')
@app_commands.describe(pertanyaan='Yang mau ditanyain', pilihan='Pisahin pakai | . Contoh: Bakso | Mie ayam | Sate', jam='Berapa jam pollingnya dibuka (1 sampai 168)')
async def poll(interaction: discord.Interaction, pertanyaan: str, pilihan: str, jam: int=24):
    isi = [p.strip() for p in pilihan.split('|') if p.strip()]
    if len(isi) < 2:
        await interaction.response.send_message('Minimal dua pilihan, dipisah pakai `|`.\nContoh: `Bakso | Mie ayam | Sate`', ephemeral=True)
        return
    if len(isi) > 5:
        await interaction.response.send_message('Maksimal lima pilihan.', ephemeral=True)
        return
    if not 1 <= jam <= 168:
        await interaction.response.send_message('Lamanya antara 1 sampai 168 jam (seminggu).', ephemeral=True)
        return
    try:
        bawaan = discord.Poll(question=pertanyaan[:300], duration=timedelta(hours=jam))
        for i, teks in enumerate(isi):
            bawaan.add_answer(text=teks[:55], emoji=HURUF_POLL[i])
        await interaction.response.send_message(poll=bawaan)
        return
    except Exception as e:
        print(f'[poll] polling bawaan ga kepakai ({e}), pindah ke reaksi')
    baris = '\n'.join((f'{HURUF_POLL[i]}  {teks}' for i, teks in enumerate(isi)))
    tutup = datetime.now(timezone.utc) + timedelta(hours=jam)
    embed = discord.Embed(title=pertanyaan, description=baris, color=discord.Color.blurple())
    embed.set_footer(text=f'Dibikin {interaction.user.display_name}')
    embed.add_field(name='Ditutup', value=f'<t:{int(tutup.timestamp())}:R>')
    await interaction.response.send_message(embed=embed)
    pesan = await interaction.original_response()
    for i in range(len(isi)):
        try:
            await pesan.add_reaction(HURUF_POLL[i])
        except Exception:
            pass

def baca_tiket():
    isi, aman = penyimpanan.baca(CATATAN_TIKET, {})
    return isi if aman else {}

def tulis_tiket(data):
    penyimpanan.tulis(CATATAN_TIKET, data)

class TombolTiket(discord.ui.View):

    def __init__(self):
        super().__init__(timeout=None)

    @discord.ui.button(label='Buka Tiket', emoji='🎫', style=discord.ButtonStyle.primary, custom_id='julian:buka_tiket')
    async def buka_tiket(self, interaction: discord.Interaction, tombol: discord.ui.Button):
        data = baca_tiket()
        punya = [t for t in data.values() if t.get('orang') == interaction.user.id and t.get('status') == 'buka']
        if len(punya) >= TIKET_MAKS_PER_ORANG:
            await interaction.response.send_message(f'Lo udah punya {len(punya)} tiket yang belum ditutup. Selesaiin dulu yang itu.', ephemeral=True)
            return
        await interaction.response.defer(ephemeral=True)
        nama = f'tiket-{interaction.user.display_name}'[:95]
        try:
            utas = await interaction.channel.create_thread(name=nama, type=discord.ChannelType.private_thread, invitable=False, reason=f'Tiket dari {interaction.user}')
            await utas.add_user(interaction.user)
        except discord.Forbidden:
            await interaction.followup.send('Gagal bikin tiket. Bot butuh izin Create Private Threads di channel ini.', ephemeral=True)
            return
        except discord.HTTPException as e:
            await interaction.followup.send(f'Gagal bikin tiket: {e}', ephemeral=True)
            return
        data[str(utas.id)] = {'orang': interaction.user.id, 'dibuka': datetime.now(timezone.utc).isoformat(), 'status': 'buka'}
        tulis_tiket(data)
        sapaan = f'{interaction.user.mention} ceritain aja di sini, ada apa. Yang bisa baca cuma lo sama moderator.\nKalau udah kelar, ketik `/tiket tutup`.'
        penjaga = discord.utils.get(interaction.guild.roles, name=ROLE_PENJAGA_TIKET)
        if penjaga:
            sapaan += f'\n\n{penjaga.mention} ada tiket baru.'
        await utas.send(sapaan)
        await interaction.followup.send(f'Tiket lo kebuka di {utas.mention}.', ephemeral=True)
        await kirim_log(interaction.guild, 'Tiket dibuka', f'{interaction.user.mention} buka tiket di {utas.mention}', discord.Color.blurple())

@bot.tree.command(name='paneltiket', description='Pasang panel tiket di channel ini')
@app_commands.describe(judul='Judul panelnya', isi='Tulisan penjelasannya')
@app_commands.checks.has_permissions(manage_guild=True)
async def paneltiket(interaction: discord.Interaction, judul: str='Butuh bantuan?', isi: str='Pencet tombol di bawah buat ngobrol berdua sama moderator. Ga ada yang lain yang bisa baca.'):
    embed = discord.Embed(title=judul, description=isi, color=discord.Color.blurple())
    await interaction.response.send_message('Panel tiketnya gua pasang.', ephemeral=True)
    await interaction.channel.send(embed=embed, view=TombolTiket())

@bot.tree.command(name='tiket', description='Urus tiket yang lagi kebuka')
@app_commands.describe(aksi='Mau ngapain')
@app_commands.choices(aksi=[app_commands.Choice(name='tutup', value='tutup')])
async def tiket(interaction: discord.Interaction, aksi: app_commands.Choice[str]):
    utas = interaction.channel
    data = baca_tiket()
    catat = data.get(str(utas.id))
    if catat is None:
        await interaction.response.send_message('Perintah ini cuma jalan di dalam tiket.', ephemeral=True)
        return
    izin = utas.permissions_for(interaction.user)
    if interaction.user.id != catat['orang'] and (not izin.manage_threads):
        await interaction.response.send_message('Cuma yang buka tiket atau moderator yang bisa nutup.', ephemeral=True)
        return
    catat['status'] = 'tutup'
    catat['ditutup'] = datetime.now(timezone.utc).isoformat()
    catat['ditutup_oleh'] = interaction.user.id
    tulis_tiket(data)
    await interaction.response.send_message(f'Tiket ditutup sama {interaction.user.mention}. Transkripnya gua kirim ke moderator.')
    berkas, jumlah = await rakit_transkrip(utas, catat)
    try:
        await utas.edit(archived=True, locked=True, reason=f'Ditutup {interaction.user}')
    except discord.Forbidden:
        pass
    await kirim_transkrip(interaction.guild, utas, interaction.user, berkas, jumlah)

async def rakit_transkrip(utas, catat):
    baris = [f'Transkrip tiket: {utas.name}', f'ID thread : {utas.id}', f"Dibuka    : {catat.get('dibuka', '-')}", f"Ditutup   : {catat.get('ditutup', '-')}", '=' * 60, '']
    jumlah = 0
    try:
        async for pesan in utas.history(limit=500, oldest_first=True):
            waktu = pesan.created_at.strftime('%d/%m/%Y %H:%M')
            isi = pesan.content or ''
            for lampiran in pesan.attachments:
                isi += f'\n    [lampiran] {lampiran.filename} — {lampiran.url}'
            for tempel in pesan.embeds:
                isi += f"\n    [embed] {tempel.title or ''} {tempel.description or ''}"
            baris.append(f'[{waktu}] {pesan.author.display_name}: {isi}')
            jumlah += 1
    except discord.Forbidden:
        baris.append('(ga bisa baca isi thread, bot kurang izin)')
    baris.append('')
    baris.append(f'Total {jumlah} pesan.')
    data = '\n'.join(baris).encode('utf-8')
    return (io.BytesIO(data), jumlah)

async def kirim_transkrip(guild, utas, penutup, berkas, jumlah):
    if CHANNEL_LOG_ID == 0:
        return
    channel = guild.get_channel(CHANNEL_LOG_ID)
    if channel is None:
        return
    embed = discord.Embed(title='Tiket ditutup', description=f'{utas.mention} ditutup {penutup.mention}\nTranskrip {jumlah} pesan terlampir.', color=discord.Color.greyple(), timestamp=datetime.now(timezone.utc))
    try:
        await channel.send(embed=embed, file=discord.File(berkas, filename=f'tiket-{utas.id}.txt'))
    except discord.Forbidden:
        print('[tiket] ga punya izin kirim transkrip ke log')
    except Exception as e:
        print(f'[tiket] transkrip gagal dikirim: {e}')

@bot.tree.command(name='editlink', description='Betulin link yang udah disimpen')
@app_commands.describe(nomor='Nomor linknya, liat di /cari', judul='Judul baru (kosongin kalau ga diubah)', link='Alamat baru (kosongin kalau ga diubah)', catatan='Keterangan baru (kosongin kalau ga diubah)')
@app_commands.checks.has_permissions(manage_messages=True)
async def editlink(interaction: discord.Interaction, nomor: int, judul: str='', link: str='', catatan: str=''):
    if link and (not link.startswith('http')):
        await interaction.response.send_message('Linknya harus diawali http atau https.', ephemeral=True)
        return
    data = baca_link()
    catat = next((d for d in data if d['id'] == nomor), None)
    if catat is None:
        await interaction.response.send_message(f'Nomor #{nomor} ga ketemu.', ephemeral=True)
        return
    if not (judul or link or catatan):
        await interaction.response.send_message('Isi minimal satu yang mau diubah.', ephemeral=True)
        return
    ubahan = []
    if judul:
        ubahan.append(f"judul: {catat['judul']} -> {judul}")
        catat['judul'] = judul
    if link:
        ubahan.append('alamatnya diganti')
        catat['link'] = link
    if catatan:
        ubahan.append('keterangannya diganti')
        catat['catatan'] = catatan
    catat['diedit_oleh'] = interaction.user.display_name
    catat['diedit'] = datetime.now(timezone.utc).strftime('%d %b %Y')
    tulis_link(data)
    embed = discord.Embed(title=catat['judul'], url=catat['link'], description=catat.get('catatan') or '', color=discord.Color.teal())
    embed.add_field(name='Yang diubah', value='\n'.join(ubahan), inline=False)
    embed.set_footer(text=f'#{nomor} • diedit {interaction.user.display_name}')
    await interaction.response.send_message(embed=embed, ephemeral=True)

@unban.error
@daftarban.error
@hapuswarn1.error
@slowmode.error
@bersihkanuser.error
@kunci.error
@buka.error
@paneltiket.error
@editlink.error
@say.error
@embed_perintah.error
async def _tolak_baru(interaction: discord.Interaction, error):
    await tolak(interaction, error)

# [BUG FIX] Background task untuk membersihkan memory leak pada dict automod
@tasks.loop(minutes=30)
async def bersih_memori_automod():
    sekarang = datetime.now(timezone.utc)
    for uid in list(_riwayat_chat.keys()):
        _riwayat_chat[uid] = [w for w in _riwayat_chat[uid] if sekarang - w < timedelta(seconds=SPAM_DETIK)]
        if not _riwayat_chat[uid]:
            _riwayat_chat.pop(uid, None)
    for uid in list(_strike.keys()):
        _strike[uid] = [w for w in _strike[uid] if sekarang - w < timedelta(minutes=10)]
        if not _strike[uid]:
            _strike.pop(uid, None)

@bersih_memori_automod.before_loop
async def sebelum_bersih_memori():
    await bot.wait_until_ready()

@tasks.loop(seconds=10)
async def jaga_status_julian():
    if bot.is_closed():
        return
    tulisan = random.choice(STATUS_LIST_JULIAN)
    try:
        await bot.change_presence(activity=discord.CustomActivity(name=tulisan))
    except Exception as e:
        if 'closing transport' not in str(e):
            print(f'[status] gagal dipasang: {e}')

@jaga_status_julian.before_loop
async def sebelum_status_julian():
    await bot.wait_until_ready()

@bot.event
async def on_ready():
    print(f'Bot admin {bot.user} online')
    muat_panel()
    bot.add_view(TombolTiket())
    if not jaga_status_julian.is_running():
        jaga_status_julian.start()
    try:
        await bot.tree.sync()
        print('[setup] slash command kedaftar')
    except Exception as e:
        print(f'[setup] gagal sync: {e}')
    if not cek_hari_besar.is_running():
        cek_hari_besar.start()
    await _pulihin_sesi_voice()
    if not jaga_sesi_voice.is_running():
        jaga_sesi_voice.start()
    if AUTO_SLOWMODE and (not lepas_slowmode.is_running()):
        lepas_slowmode.start()
    if not jaga_tempban.is_running():
        jaga_tempban.start()
    if not bersih_memori_automod.is_running():
        bersih_memori_automod.start()
    for g in bot.guilds:
        await segarkan_undangan(g)
    print(f"[setup] anti nuke={('nyala' if ANTI_NUKE else 'mati')} | {NUKE_JUMLAH} hapus / {NUKE_DETIK} detik | cabut role={('ya' if NUKE_CABUT_ROLE else 'engga')}")
    print(f"[setup] lacak undangan={('nyala' if LACAK_UNDANGAN else 'mati')} | {sum((len(v) for v in _undangan.values()))} undangan kecatat")
    print(f"[setup] starboard={('nyala' if STARBOARD_AKTIF else 'mati')} | {STARBOARD_EMOJI} x{STARBOARD_AMBANG} -> {CHANNEL_STARBOARD_ID or 'BELUM DIISI'}")
    print(f'[setup] warn: gugur setelah {WARN_KADALUARSA_HARI} hari | eskalasi {WARN_ESKALASI}')
    print(f'[setup] jembatan antar bot: {acara.siapa_dengerin()}')
    print(f"[setup] sambutan={('aktif' if CHANNEL_SAMBUTAN_ID else 'mati')} | log={('aktif' if CHANNEL_LOG_ID else 'mati')} | auto role={ROLE_OTOMATIS or 'mati'} | reaction role={len(REACTION_ROLE) + len(_panel_role)} pesan")
    print(f"[setup] log isi pesan={('nyala' if LOG_PESAN_DIHAPUS else 'MATI')} | log keluar masuk={('nyala' if LOG_KELUAR_MASUK else 'MATI')}")
    print(f"[setup] pengumuman={('aktif' if CHANNEL_PENGUMUMAN_ID else 'mati')} | ngecek hari besar tiap jam {JAM_KIRIM} WIB")
    print(f"[setup] notif voice={('aktif' if CHANNEL_NOTIF_VOICE_ID else 'mati')} | jeda antar sesi {JEDA_SESI_MENIT} menit")
    print(f'[setup] penjaga sesi voice=nyala, ngecek tiap {JEDA_JAGA_VOICE} detik | catatan: {CATATAN_SESI_VOICE}')
    print(f"[setup] resources={('aktif' if CHANNEL_RESOURCES_ID else 'mati')} | {len(baca_link())} link tersimpan")
    afk = None
    for g in bot.guilds:
        afk = g.afk_channel
        break
    print(f"[setup] DM saat AFK={('nyala' if DM_SAAT_AFK else 'mati')} | channel AFK server={(afk.name if afk else 'BELUM DISETEL')} | laporan={('aktif' if CHANNEL_LAPORAN_AFK_ID else 'mati')}")
    print(f"[setup] auto AFK={('nyala' if AUTO_AFK else 'mati')} | peringatan {AFK_WARN_DETIK}d, pindah {AFK_TOMBOL_DETIK}d setelahnya | kebal: {', '.join(AFK_ROLE_BEBAS) or 'ga ada'}")
    print(f"[setup] tombol konfirmasi AFK={('nyala' if AFK_TOMBOL_AKTIF else 'mati')} | jendela {AFK_TOMBOL_DETIK} detik | jeda tenang {AFK_KEBAL_MENIT} menit | maks {AFK_MAKS_KONFIRMASI or 'tanpa batas'} kali | channel={CHANNEL_LAPORAN_AFK_ID or 'BELUM DIISI'}")
    if KARTU_SAMBUTAN and kartu:
        print(f'[setup] kartu sambutan=nyala | dimuat dari {kartu.__file__}')
    else:
        print('[setup] kartu sambutan=MATI, cek pesan [kartu] di atas')
    print(f"[setup] perintah berbahaya cuma buat role: {', '.join(ROLE_PENGUASA) or 'SIAPA AJA (ga dibatasi)'}")
    print(f"[setup] automod={('nyala' if AUTOMOD_AKTIF else 'mati')} | {len(KATA_BERAT)} kata berat + {len(KATA_RINGAN)} ringan | timeout otomatis setelah {STRIKE_SEBELUM_TIMEOUT} pelanggaran")
if __name__ == '__main__':
    if not TOKEN:
        raise SystemExit('JULIAN_TOKEN belum keisi di file .env')
    bot.run(TOKEN)
