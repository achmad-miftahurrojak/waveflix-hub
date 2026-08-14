import asyncio
import os
import random
import re
import time
from datetime import datetime, timedelta
import aiohttp
import discord
from discord.ext import commands
import acara as papan
import penyimpanan
from .config import *
from .voice import suara
try:
    import kartu_level
except Exception as e:
    print(f'[arka] kartu level ga kepakai, /level pakai embed biasa: {e}')
    kartu_level = None
data = {'orang': {}, 'cuaca': {}}
jeda_terakhir = {}
sesi_tebak = {}
loop_udah_jalan = False

def muat():
    """Load player data dari file. Raise SystemExit jika corrupt."""
    global data
    data = penyimpanan.baca_atau_berhenti(BERKAS, {'orang': {}, 'cuaca': {}}, 'data pemain Arka')
    data.setdefault('orang', {})
    data.setdefault('cuaca', {})
    if not isinstance(data['orang'], dict):
        raise SystemExit(f"[arka] data['orang'] harus dict, dapat {type(data['orang']).__name__}")
    print(f"[arka] data dimuat, {len(data['orang'])} orang tercatat")

def simpan():
    """Simpan data pemain. Tolak jika data kosong tapi file asli punya data."""
    if not data.get('orang'):
        lama, aman = penyimpanan.baca(BERKAS, {})
        if aman and lama.get('orang'):
            print(f"[arka] SIMPAN DITOLAK. Data di memori kosong tapi data-arka.json punya {len(lama['orang'])} orang.")
            print('[arka] Ini tanda muat() belum kepanggil. Filenya gua biarin utuh.')
            return False
    try:
        penyimpanan.tulis(BERKAS, data)
        return True
    except Exception as e:
        print(f'[arka] ERROR simpan data: {e}')
        return False

def _validasi_uid(uid):
    """Validasi UID format. Raise ValueError jika invalid."""
    try:
        uid_int = int(uid)
        if uid_int <= 0:
            raise ValueError(f'UID harus positif, dapat {uid_int}')
        return uid_int
    except (ValueError, TypeError) as e:
        raise ValueError(f'Invalid UID: {uid}') from e

def catatan(uid):
    """Get atau create player record. Validasi UID sebelum akses."""
    uid_int = _validasi_uid(uid)
    uid_str = str(uid_int)
    if uid_str not in data['orang']:
        data['orang'][uid_str] = {'xp': 0, 'xp_minggu': 0, 'pesan': 0, 'menit': 0, 'absen': 0, 'absen_terakhir': '1970-01-01', 'absen_panjang': 0, 'menang': 0, 'kalah': 0, 'juara': 0}
    return data['orang'][uid_str]

def catat_main(uid, game, menang=False):
    c = catatan(uid)
    per_game = c.setdefault('game', {}).setdefault(game, {})
    per_game['main'] = per_game.get('main', 0) + 1
    per_game['minggu'] = per_game.get('minggu', 0) + 1
    if menang:
        per_game['menang'] = per_game.get('menang', 0) + 1

def hari_ini():
    return datetime.now(WIB).strftime('%Y-%m-%d')

def tingkat(xp):
    for batas, nama, role in reversed(TINGKAT):
        if xp >= batas:
            return (batas, nama, role)
    return TINGKAT[0]

def berikutnya(xp):
    for batas, nama, _role in TINGKAT:
        if xp < batas:
            return (batas, nama)
    return None

def batang(xp):
    sekarang = tingkat(xp)[0]
    depan = berikutnya(xp)
    if not depan:
        return '▰' * 10
    seluruh = depan[0] - sekarang
    jalan = xp - sekarang
    persen = jalan / seluruh
    isi = min(10, int(persen * 10))
    return '▰' * isi + '▱' * (10 - isi)

def lencana_punya(catat):
    punya = []
    for kode, emoji, nama, _, kunci, target in LENCANA:
        if catat.get(kunci, 0) >= target:
            punya.append((kode, emoji, nama))
    return punya

def lencana_khusus_punya(catat):
    punya = []
    if koleksi_lengkap(catat):
        punya.append(('kolektor', '🐚', 'Kolektor Pantai'))
    if len(catat.get('kasih_ke', [])) >= 5:
        punya.append(('dermawan', '🤝', 'Dermawan'))
    return punya

def koleksi_lengkap(catat):
    punya = catat.get('koleksi', {})
    return len(punya) == len(TANGKAPAN)

def deret_tangkapan():
    return ' '.join((t[1] for t in TANGKAPAN))

def cuaca_hari_ini():
    hari = hari_ini()
    if data['cuaca'].get('tanggal') != hari:
        c = random.choices(CUACA, weights=[c[1] for c in CUACA], k=1)[0]
        data['cuaca'] = {'tanggal': hari, 'nama': c[0], 'kali': c[1], 'emoji': c[2], 'kata': c[3], 'diumumkan': False}
        simpan()
    return data['cuaca']

def suasana_pet():
    pet = data.setdefault('pet', {'kenyang': 0.0, 'terakhir': time.time()})
    kenyang = pet['kenyang']
    for batas, nama, emoji, kali in PET_SUASANA:
        if kenyang >= batas:
            return (nama, emoji, kali)
    return PET_SUASANA[-1][1:]

def kondisi_pet():
    pet = data.setdefault('pet', {'kenyang': 0.0, 'terakhir': time.time()})
    sekarang = time.time()
    lalu = sekarang - pet['terakhir']
    jam = lalu / 3600.0
    pet['kenyang'] = max(0.0, pet['kenyang'] - jam * PET_LAPAR_PER_JAM)
    pet['terakhir'] = sekarang
    return pet

async def _dengar_timeout(isi):
    catat = catatan(isi['user_id'])
    sampai = datetime.now(WIB) + timedelta(minutes=isi.get('menit', 0))
    catat.setdefault('efek', {})['beku_sampai'] = sampai.isoformat()
    simpan()
    print(f"[arka] XP {isi['user_id']} dibekuin {isi.get('menit')} menit (dari Julian)")

async def _dengar_untimeout(isi):
    catat = data['orang'].get(str(isi['user_id']))
    if catat and catat.get('efek', {}).pop('beku_sampai', None):
        simpan()
        print(f"[arka] XP {isi['user_id']} dicairin lagi")
_dipindah_afk = {}

async def _dengar_afk(isi):
    _dipindah_afk[isi['user_id']] = True

async def _dengar_balik_voice(isi):
    _dipindah_afk.pop(isi['user_id'], None)
papan.dengar('timeout', _dengar_timeout)
papan.dengar('untimeout', _dengar_untimeout)
papan.dengar('afk_dipindah', _dengar_afk)
papan.dengar('voice_aktif', _dengar_balik_voice)

def lagi_beku(catat):
    sampai = catat.get('efek', {}).get('beku_sampai')
    if not sampai:
        return False
    try:
        return datetime.fromisoformat(sampai) > datetime.now(WIB)
    except (TypeError, ValueError):
        return False

async def kabari_hampir(anggota, catat, sebelum):
    if not KABAR_HAMPIR:
        return
    depan = berikutnya(catat['xp'])
    if depan is None:
        return
    kurang = depan[0] - catat['xp']
    if kurang > HAMPIR_AMBANG:
        return
    if depan[0] - sebelum <= HAMPIR_AMBANG:
        return
    if catat.get('hampir_dikabarin') == depan[1]:
        return
    catat['hampir_dikabarin'] = depan[1]
    simpan()
    try:
        await anggota.send(f'🌊 Dikit lagi. Tinggal **{kurang:,} XP** buat nyampe **{depan[1]}**.\nSekali ngobrol bentar atau `/hunt` sekali dua kali juga nyampe kok.')
    except discord.Forbidden:
        pass
    except Exception as e:
        print(f'[arka] kabar hampir naik gagal: {e}')

async def cek_comeback(bot_instance, anggota, catat):
    hari = hari_ini()
    lama = catat.get('terakhir_aktif')
    catat['terakhir_aktif'] = hari
    if not COMEBACK_AKTIF or not lama or lama == hari:
        return
    try:
        selisih = (datetime.strptime(hari, '%Y-%m-%d') - datetime.strptime(lama, '%Y-%m-%d')).days
    except ValueError:
        return
    if selisih < COMEBACK_HARI:
        return
    sampai = datetime.now(WIB) + timedelta(hours=COMEBACK_JAM)
    catat.setdefault('efek', {})['comeback_sampai'] = sampai.isoformat()
    catat['xp'] += COMEBACK_XP
    catat['xp_minggu'] += COMEBACK_XP
    simpan()
    channel = bot_instance.get_channel(CHANNEL_ARENA) if CHANNEL_ARENA else None
    if channel:
        try:
            await channel.send(f'🌊 {anggota.mention} balik lagi setelah **{selisih} hari** ilang. Nih **{COMEBACK_XP} XP** buat ngejar ketinggalan, plus semua XP lo **x{COMEBACK_PENGALI}** selama {COMEBACK_JAM} jam ke depan.\nJangan ilang lagi.')
        except Exception:
            pass
    try:
        await anggota.send(f'Eh, lo balik. Udah {selisih} hari ga keliatan.\nGua kasih **{COMEBACK_XP} XP** sama pengali **x{COMEBACK_PENGALI}** selama {COMEBACK_JAM} jam, biar ga kerasa ketinggalan banget. Cek `/profil` buat liat posisi lo sekarang.')
    except discord.Forbidden:
        pass

def pengali_comeback(catat):
    sampai = catat.get('efek', {}).get('comeback_sampai')
    if not sampai:
        return 1.0
    try:
        if datetime.fromisoformat(sampai) > datetime.now(WIB):
            return COMEBACK_PENGALI
    except (TypeError, ValueError):
        pass
    return 1.0

async def tambah_xp(bot_instance, anggota, jumlah):
    if anggota.bot:
        return
    catat = catatan(anggota.id)
    if lagi_beku(catat):
        return
    await cek_comeback(bot_instance, anggota, catat)
    lama = tingkat(catat['xp'])[1]
    cuaca = cuaca_hari_ini()
    dapat = max(1, int(jumlah * cuaca['kali']))
    if PET_AKTIF:
        dapat = max(1, int(dapat * suasana_pet()[2]))
    dapat = max(1, int(dapat * pengali_comeback(catat)))
    efek_paus = catat.setdefault('efek', {})
    sampai = efek_paus.get('paus_sampai')
    if sampai and datetime.fromisoformat(sampai) > datetime.now(WIB):
        dapat = int(dapat * (1 + PAUS_TAMBAHAN))
    sebelum = catat['xp']
    catat['xp'] += dapat
    catat['xp_minggu'] += dapat
    simpan()
    await kabari_hampir(anggota, catat, sebelum)
    baru = tingkat(catat['xp'])
    if baru[1] != lama:
        channel = bot_instance.get_channel(CHANNEL_ARENA) if CHANNEL_ARENA else None
        if channel:
            try:
                await channel.send(suara('naik', n=anggota.mention, x=baru[1]))
            except Exception:
                pass
        try:
            papan.umumkan_nanti('level_up', user_id=anggota.id, guild_id=anggota.guild.id if hasattr(anggota, 'guild') and anggota.guild else 0, tingkat=baru[1], tingkat_lama=lama, xp=catat['xp'], nama=anggota.display_name)
        except Exception as e:
            print(f'[arka] gagal umumin level_up: {e}')
        if baru[2]:
            semua = {t[2] for t in TINGKAT if t[2]}
            try:
                role_baru = discord.utils.get(anggota.guild.roles, name=baru[2])
                if role_baru and role_baru not in anggota.roles:
                    await anggota.add_roles(role_baru, reason='Naik tingkat')
                lama_dipakai = [r for r in anggota.roles if r.name in semua and r.name != baru[2]]
                if lama_dipakai:
                    await anggota.remove_roles(*lama_dipakai, reason='Tingkat lama')
            except discord.Forbidden:
                print('[arka] ga bisa atur role tingkatan.')
            except Exception as e:
                print(f'[arka] role tingkatan bermasalah: {e}')

def channel_perintah(nama):
    isi = CHANNEL_PERINTAH.get(nama, 0)
    if isi == 'semua':
        return []
    isi = isi or CHANNEL_ARENA
    if not isi:
        return []
    return [isi] if isinstance(isi, int) else list(isi)

async def di_arena(inter):
    nama = inter.command.name if inter.command else ''
    tujuan = channel_perintah(nama)
    if not tujuan or inter.channel_id in tujuan:
        return True
    kemana = ' atau '.join((f'<#{c}>' for c in tujuan))
    await inter.response.send_message(f'`/{nama}` mainnya di {kemana} ya, biar sini ga penuh.', ephemeral=True)
    return False

def masih_anggota(guild, uid):
    return guild is not None and guild.get_member(int(uid)) is not None

def peringkat_orang(guild, uid):
    hidup = [(u, c.get('xp', 0)) for u, c in data['orang'].items() if c.get('xp', 0) > 0 and masih_anggota(guild, u)]
    hidup.sort(key=lambda x: x[1], reverse=True)
    for i, (u, _x) in enumerate(hidup, 1):
        if u == str(uid):
            return i
    return None

async def rakit_kartu_level(inter, orang, catat):
    xp = catat['xp']
    sekarang = tingkat(xp)
    depan = berikutnya(xp)
    async with aiohttp.ClientSession() as sesi:
        async with sesi.get(str(orang.display_avatar.replace(size=256, format='png'))) as r:
            foto = await r.read()
    return await asyncio.to_thread(kartu_level.buat_kartu_level, orang.display_name, foto, sekarang[1], xp, sekarang[0], depan[0] if depan else None, peringkat_orang(inter.guild, orang.id), lencana_punya(catat), catat)