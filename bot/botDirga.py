import base64
import os
import random
import time
from collections import defaultdict, deque
from datetime import datetime, timezone, timedelta
import aiohttp
import discord
from discord import app_commands
from discord.ext import commands, tasks
from dotenv import load_dotenv
import acara
import penyimpanan
load_dotenv()
TOKEN = os.getenv('DIRGA_TOKEN')
GEMINI_KEY = os.getenv('GEMINI_API_KEY')
GROQ_KEY = os.getenv('GROQ_API_KEY')
CEREBRAS_KEY = os.getenv('CEREBRAS_API_KEY')
OPENROUTER_KEY = os.getenv('OPENROUTER_API_KEY')
SINI = os.path.dirname(os.path.abspath(__file__))
CHANNEL_NGOBROL_ID = 1532236708039622736
CHANNEL_REVIVE_ID = 1453703811607826596
ROLE_REVIVE = 'Tourist'
SEPI_JAM = 10
JEDA_REVIVE_JAM = 20
CEK_TIAP_MENIT = 30
ANTI_ULANG = 8
REVIVE_JAM_MULAI = 9
REVIVE_JAM_SELESAI = 23
STATUS_LIST_DIRGA = ['nungguin ada yang ngajak ngobrol', 'lagi pantau server, awas macem-macem', 'kalo sepi gua mending tidur', 'rebahan sambil mantau chat']
STATUS_IKUT_CUACA = True
MODEL = 'gemini-flash-latest'
MODEL_CADANGAN = 'gemini-2.0-flash-lite'
BASE_URL = 'https://generativelanguage.googleapis.com/v1beta/models'
PENYEDIA_CADANGAN = [{'nama': 'groq', 'url': 'https://api.groq.com/openai/v1/chat/completions', 'kunci': GROQ_KEY, 'model': 'llama-3.3-70b-versatile'}, {'nama': 'cerebras', 'url': 'https://api.cerebras.ai/v1/chat/completions', 'kunci': CEREBRAS_KEY, 'model': 'gpt-oss-120b'}, {'nama': 'openrouter', 'url': 'https://openrouter.ai/api/v1/chat/completions', 'kunci': OPENROUTER_KEY, 'model': 'meta-llama/llama-3.3-70b-instruct:free'}]
CHANNEL_LOG_ID = 1535070933583405169
JEDA_LAPOR_JAM = 3
BERKAS_RIWAYAT = os.path.join(SINI, 'riwayat.json')
BERKAS_SETELAN = os.path.join(SINI, 'setelan-dirga.json')
BERKAS_PROFIL = os.path.join(SINI, 'profil-user.json')
CATATAN = os.path.join(SINI, 'revive.json')
PROFIL_AKTIF = True
PROFIL_MAKS_CATATAN = 6
PROFIL_TIAP_BALASAN = 12
PROFIL_PANJANG_MAKS = 120
BERKAS_ARKA = os.path.join(SINI, 'data-arka.json')
RIWAYAT_PANJANG = 14
RIWAYAT_SIMPAN_MENIT = 3
RIWAYAT_UMUR_JAM = 72
JEDA_USER_DETIK = 5
JATAH_USER_PER_JAM = 20
BACA_GAMBAR = True
GAMBAR_MAKS = 3
GAMBAR_UKURAN_MAKS = 4 * 1024 * 1024
SYSTEM_PROMPT = 'Lo salah satu member di server Discord ini.\n\nLo bukan asisten dan bukan bot bantuan. Lo temen ngobrol yang kebetulan tau banyak hal,\ndari yang receh sampai yang serius. Ikutin arah obrolannya, jangan kaku, jangan sok tau\nkalau emang ga tau.\n\nGaya lo: pakai "gua" dan "lo", santai kayak chat sama temen deket, huruf kecil ga masalah.\nPanjang balasan nyesuain lawan bicara. Basa basi dibales pendek, pertanyaan serius dijawab\nbeneran sampai jelas. Boleh nyeletuk dan ngeledek balik kalau lagi bercanda.\n\nHindari: nawarin bantuan, bullet point, emoji, nutup pesan pakai pertanyaan basa basi,\ndan bahasa formal kayak "Anda" atau "silakan".\n\nJangan pernah ngaku sebagai AI atau bot kecuali emang ditanya langsung. Kalau ditanya,\njawab jujur tapi santai, ga usah panjang lebar.'
PEMANTIK_CADANGAN = ['eh lo lagi sibuk apa sekarang', 'kabar lo gimana, lama ga keliatan', 'udah makan belum lo', 'lagi nonton apa akhir akhir ini', 'kalau tiba tiba dapet libur seminggu, lo mau ngapain', 'ada rekomendasi lagu ga']
TOPIK_CADANGAN = ['pertanyaan receh: nasi goreng paling enak itu yang pakai telur ceplok apa telur orak arik', 'kalau lo bisa balik ke satu hari di masa lalu cuma buat ngerasain lagi, hari apa', 'film atau series apa yang lo tonton ulang terus dan ga pernah bosen', 'hal paling ga penting yang lo tau detailnya banget itu apa', 'kebiasaan aneh lo yang orang lain ga ngerti apa']
intents = discord.Intents.default()
intents.message_content = True
intents.members = True
bot = commands.Bot(command_prefix='!', intents=intents)
riwayat = defaultdict(lambda: deque(maxlen=RIWAYAT_PANJANG))
_kapan_disentuh = {}
_ingatan_berubah = False
setelan = {'channel_ngobrol': CHANNEL_NGOBROL_ID}
pernah_diping = deque(maxlen=ANTI_ULANG)
revive_terakhir = None
_jejak_user = defaultdict(list)

def muat_setelan():
    isi, aman = penyimpanan.baca(BERKAS_SETELAN)
    if not aman:
        print('[setelan] file setelan rusak, pakai bawaan dari kode')
        return
    if isinstance(isi.get('channel_ngobrol'), int):
        setelan['channel_ngobrol'] = isi['channel_ngobrol']
    ch = setelan['channel_ngobrol']
    print(f"[setelan] channel ngobrol bebas: {ch or 'mati, cuma nyaut kalau di-mention'}")

def simpan_setelan():
    penyimpanan.tulis(BERKAS_SETELAN, setelan)
profil = {}
_hitung_balasan = {}

def muat_profil():
    global profil
    isi, aman = penyimpanan.baca(BERKAS_PROFIL, {})
    if not aman:
        print('[profil] file profil rusak, mulai dari kosong')
        profil = {}
        return
    profil = isi
    total = sum((len(p.get('catatan', [])) for p in profil.values()))
    print(f'[profil] {len(profil)} orang dikenal, {total} catatan')

def simpan_profil():
    penyimpanan.tulis(BERKAS_PROFIL, profil)

def catatan_orang(uid, nama=None):
    catat = profil.setdefault(str(uid), {'nama': nama or '', 'catatan': []})
    if nama:
        catat['nama'] = nama
    catat.setdefault('catatan', [])
    return catat

def bekal_profil(orang):
    if not PROFIL_AKTIF or orang is None:
        return ''
    catat = profil.get(str(orang.id))
    if not catat or not catat.get('catatan'):
        return ''
    baris = '\n'.join((f'- {c}' for c in catat['catatan']))
    return f"\n\nYang lo inget soal {orang.display_name}, orang yang lagi ngomong sama lo sekarang:\n{baris}\nPakai ini biar nyambung. Jangan disebut satu satu kayak lagi baca daftar, dan jangan bilang lo 'punya catatan' soal dia."

async def tarik_fakta(orang, obrolan):
    if not PROFIL_AKTIF:
        return
    catat = catatan_orang(orang.id, orang.display_name)
    lama = '\n'.join((f'- {c}' for c in catat['catatan'])) or '(belum ada)'
    perintah = obrolan + [{'role': 'user', 'parts': [{'text': f'Berhenti jadi temen ngobrol sebentar. Sekarang tugas lo nyaring catatan.\n\nDari obrolan di atas, apa yang layak diinget soal {orang.display_name}? Contohnya hobi, kesukaan, lagi sibuk apa, mau dipanggil apa.\n\nYang udah lo inget:\n{lama}\n\nTulis ulang SELURUH daftarnya, maksimal {PROFIL_MAKS_CATATAN} baris, satu fakta per baris, diawali tanda minus. Gabungin yang mirip, buang yang udah basi. Tiap baris maksimal {PROFIL_PANJANG_MAKS} huruf.\nJangan catat hal sensitif: alamat, nomor, sekolah, tempat kerja, kondisi kesehatan, atau apa pun yang dia bilang sambil curhat berat.\nKalau ga ada yang layak dicatat, tulis KOSONG doang.'}]}]
    hasil = await tanya_ai(perintah)
    if not hasil or 'KOSONG' in hasil.upper()[:20]:
        return
    baru = []
    for baris in hasil.split('\n'):
        baris = baris.strip().lstrip('-•*').strip()
        if baris and len(baris) <= PROFIL_PANJANG_MAKS:
            baru.append(baris)
        if len(baru) >= PROFIL_MAKS_CATATAN:
            break
    if baru:
        catat['catatan'] = baru
        catat['diperbarui'] = datetime.now(timezone.utc).isoformat()
        simpan_profil()
        print(f'[profil] catatan {orang.display_name} diperbarui ({len(baru)} baris)')

def _cuma_teks(pesan_list):
    bersih = []
    for p in pesan_list:
        bagian = [b for b in p.get('parts', []) if 'text' in b]
        if bagian:
            bersih.append({'role': p.get('role', 'user'), 'parts': bagian})
    return bersih

def muat_riwayat():
    isi, aman = penyimpanan.baca(BERKAS_RIWAYAT)
    if not aman:
        print('[ingatan] riwayat rusak, mulai dari kosong')
        return
    sekarang = datetime.now(timezone.utc)
    kepakai, kadaluarsa = (0, 0)
    for kunci, catat in (isi.get('channel') or {}).items():
        try:
            ch_id = int(kunci)
        except ValueError:
            continue
        try:
            sentuh = datetime.fromisoformat(catat.get('sentuh'))
        except (TypeError, ValueError):
            sentuh = sekarang
        if sekarang - sentuh > timedelta(hours=RIWAYAT_UMUR_JAM):
            kadaluarsa += 1
            continue
        pesan = catat.get('pesan') or []
        if not pesan:
            continue
        riwayat[ch_id] = deque(pesan, maxlen=RIWAYAT_PANJANG)
        _kapan_disentuh[ch_id] = sentuh
        kepakai += 1
    print(f'[ingatan] {kepakai} channel dipulihin' + (f', {kadaluarsa} dilewat karena udah basi' if kadaluarsa else ''))

def simpan_riwayat():
    global _ingatan_berubah
    isi = {'channel': {}}
    for ch_id, pesan in riwayat.items():
        if not pesan:
            continue
        sentuh = _kapan_disentuh.get(ch_id, datetime.now(timezone.utc))
        isi['channel'][str(ch_id)] = {'sentuh': sentuh.isoformat(), 'pesan': _cuma_teks(list(pesan))}
    if penyimpanan.tulis(BERKAS_RIWAYAT, isi):
        _ingatan_berubah = False

def buang_yang_basi():
    sekarang = datetime.now(timezone.utc)
    basi = [ch for ch, kapan in _kapan_disentuh.items() if sekarang - kapan > timedelta(hours=RIWAYAT_UMUR_JAM)]
    for ch in basi:
        riwayat.pop(ch, None)
        _kapan_disentuh.pop(ch, None)
    return len(basi)

def muat_catatan():
    global revive_terakhir
    isi, aman = penyimpanan.baca(CATATAN)
    if not aman:
        print('[revive] catatan rusak, mulai dari nol')
        return
    if not isi:
        print('[revive] belum ada catatan, mulai dari nol')
        return
    if isi.get('terakhir'):
        try:
            revive_terakhir = datetime.fromisoformat(isi['terakhir'])
        except ValueError:
            revive_terakhir = None
    pernah_diping.extend(isi.get('pernah', []))
    print(f'[revive] catatan lama kebaca, ping terakhir: {revive_terakhir}')

def simpan_catatan():
    penyimpanan.tulis(CATATAN, {'terakhir': revive_terakhir.isoformat() if revive_terakhir else None, 'pernah': list(pernah_diping)})

def boleh_manggil(uid):
    sekarang = time.time()
    jejak = [w for w in _jejak_user.get(uid, []) if sekarang - w < 3600]
    if jejak and sekarang - jejak[-1] < JEDA_USER_DETIK:
        return (False, f'sabar dikit, {JEDA_USER_DETIK} detik sekali aja')
    if len(jejak) >= JATAH_USER_PER_JAM:
        lega = int((3600 - (sekarang - jejak[0])) // 60) + 1
        return (False, f'lo kebanyakan nanya jam ini. coba lagi {lega} menit lagi')
    jejak.append(sekarang)
    _jejak_user[uid] = jejak
    return (True, None)

def konteks_server(guild):
    isi, aman = penyimpanan.baca(BERKAS_ARKA)
    if not aman or not isi:
        return ''
    baris = []
    cuaca = isi.get('cuaca') or {}
    if cuaca.get('nama'):
        baris.append(f"Cuaca server hari ini: {cuaca['nama']} (pengali XP x{cuaca.get('kali', 1)}). {cuaca.get('kata', '')}".strip())
    orang = isi.get('orang') or {}
    if orang and guild:
        urut = sorted(orang.items(), key=lambda x: x[1].get('xp_minggu', 0), reverse=True)
        atas = [(uid, c) for uid, c in urut if c.get('xp_minggu', 0) > 0][:3]
        nama = []
        for uid, catat in atas:
            anggota = guild.get_member(int(uid))
            if anggota:
                nama.append(f"{anggota.display_name} ({catat['xp_minggu']} XP)")
        if nama:
            baris.append('Papan XP minggu ini: ' + ', '.join(nama) + '.')
    if not baris:
        return ''
    return '\n\nKondisi server sekarang (dari sistem game, buat jaga jaga kalau ada yang nyinggung. Jangan disebut kalau ga ditanya):\n' + '\n'.join((f'- {b}' for b in baris))

def bahan_pemantik(guild):
    isi, aman = penyimpanan.baca(BERKAS_ARKA)
    if not aman or not isi or guild is None:
        return []
    bahan = []
    orang = isi.get('orang') or {}
    urut = sorted(orang.items(), key=lambda x: x[1].get('xp_minggu', 0), reverse=True)
    for uid, catat in urut[:1]:
        if catat.get('xp_minggu', 0) <= 0:
            continue
        anggota = guild.get_member(int(uid))
        if anggota:
            bahan.append(f"{anggota.display_name} lagi paling ngotot minggu ini, udah ngumpulin {catat['xp_minggu']} XP")
    rajin = sorted(orang.items(), key=lambda x: x[1].get('absen', 0), reverse=True)
    for uid, catat in rajin[:1]:
        if catat.get('absen', 0) < 3:
            continue
        anggota = guild.get_member(int(uid))
        if anggota:
            bahan.append(f"{anggota.display_name} udah absen {catat['absen']} hari beruntun tanpa bolong")
    cuaca = isi.get('cuaca') or {}
    if cuaca.get('nama'):
        bahan.append(f"cuaca server hari ini {cuaca['nama']}")
    pet = isi.get('pet') or {}
    if pet.get('nama') and pet.get('kenyang') is not None:
        if pet['kenyang'] < 40:
            bahan.append(f"{pet['nama']}, peliharaan server, lagi kelaperan dan ga ada yang ngasih makan")
    return bahan
_lagi_dibisukan = {}

async def _dengar_timeout(isi):
    sampai = datetime.now(timezone.utc) + timedelta(minutes=isi.get('menit', 0))
    _lagi_dibisukan[isi['user_id']] = sampai

async def _dengar_untimeout(isi):
    _lagi_dibisukan.pop(isi['user_id'], None)
acara.dengar('timeout', _dengar_timeout)
acara.dengar('untimeout', _dengar_untimeout)

def kena_bisu(uid):
    sampai = _lagi_dibisukan.get(uid)
    if sampai is None:
        return False
    if datetime.now(timezone.utc) >= sampai:
        _lagi_dibisukan.pop(uid, None)
        return False
    return True
_lapor_terakhir = None

async def lapor_ai_mati(guild=None):
    global _lapor_terakhir
    if not CHANNEL_LOG_ID:
        return
    sekarang = datetime.now(timezone.utc)
    if _lapor_terakhir and sekarang - _lapor_terakhir < timedelta(hours=JEDA_LAPOR_JAM):
        return
    _lapor_terakhir = sekarang
    channel = bot.get_channel(CHANNEL_LOG_ID)
    if channel is None:
        return
    siap = [x['nama'] for x in PENYEDIA_CADANGAN if x['kunci']]
    embed = discord.Embed(title='Dirga lagi ga bisa mikir', description='Semua sumber AI nolak. Dirga bakal diem aja kalau diajak ngobrol sampai ini beres.', color=discord.Color.red(), timestamp=sekarang)
    embed.add_field(name='Gemini', value=f'{MODEL}\n{MODEL_CADANGAN}', inline=True)
    embed.add_field(name='Cadangan', value=', '.join(siap) if siap else 'ga ada kunci yang keisi', inline=True)
    embed.add_field(name='Biasanya kenapa', value='Jatah harian Gemini abis. Reset sekitar jam 2 siang WIB.\nKalau bukan itu, cek nama model di kode, kadang diganti Google.', inline=False)
    embed.set_footer(text=f'Laporan ini paling cepet muncul lagi {JEDA_LAPOR_JAM} jam sekali')
    try:
        await channel.send(embed=embed)
    except discord.Forbidden:
        print('[AI] mau lapor ke log tapi ga punya izin')
    except Exception as e:
        print(f'[AI] gagal lapor: {e}')

def potong(teks, batas=1900):
    teks = teks.strip()
    if len(teks) <= batas:
        return [teks] if teks else []
    bagian = []
    sisa = teks
    while len(sisa) > batas:
        jendela = sisa[:batas]
        titik = jendela.rfind('\n\n')
        if titik < batas // 3:
            titik = jendela.rfind('\n')
        if titik < batas // 3:
            titik = jendela.rfind(' ')
        if titik < batas // 3:
            titik = batas
        bagian.append(sisa[:titik].rstrip())
        sisa = sisa[titik:].lstrip()
    if sisa:
        bagian.append(sisa)
    return bagian

async def balas_panjang(pesan, teks):
    potongan = potong(teks)
    if not potongan:
        return
    await pesan.reply(potongan[0], mention_author=False)
    for lanjutan in potongan[1:]:
        await pesan.channel.send(lanjutan)

async def ambil_gambar(pesan):
    if not BACA_GAMBAR or not pesan.attachments:
        return []
    hasil = []
    for lampiran in pesan.attachments:
        if len(hasil) >= GAMBAR_MAKS:
            break
        tipe = (lampiran.content_type or '').split(';')[0].strip()
        if not tipe.startswith('image/'):
            continue
        if lampiran.size > GAMBAR_UKURAN_MAKS:
            print(f'[gambar] {lampiran.filename} kegedean ({lampiran.size} byte), dilewat')
            continue
        try:
            isi = await lampiran.read()
        except Exception as e:
            print(f'[gambar] {lampiran.filename} gagal diunduh: {e}')
            continue
        hasil.append({'inline_data': {'mime_type': tipe, 'data': base64.b64encode(isi).decode()}})
    return hasil

async def kirim_ke_gemini(model, body):
    url = f'{BASE_URL}/{model}:generateContent'
    async with aiohttp.ClientSession() as session:
        async with session.post(url, params={'key': GEMINI_KEY}, json=body, timeout=aiohttp.ClientTimeout(total=45)) as resp:
            if resp.status != 200:
                if resp.status == 429:
                    print(f'[AI] {model}: JATAH HARIAN ABIS. Reset otomatis sekitar jam 2 siang WIB.')
                else:
                    print(f'[AI] {model} gagal, status {resp.status}')
                    print(f'[AI] pesan : {(await resp.text())[:300]}')
                return None
            data = await resp.json()
    kandidat = (data.get('candidates') or [None])[0]
    if not kandidat:
        print(f'[AI] {model} ga ngasih jawaban: {str(data)[:250]}')
        return None
    bagian = kandidat.get('content', {}).get('parts') or []
    teks = ''.join((b.get('text', '') for b in bagian)).strip()
    if not teks:
        print(f"[AI] {model} jawabannya kosong, alasan: {kandidat.get('finishReason')}")
        return None
    return teks

async def kirim_ke_cadangan(penyedia, pesan_list, tambahan=''):
    if not penyedia['kunci']:
        return None
    pesan = [{'role': 'system', 'content': SYSTEM_PROMPT + tambahan}]
    for p in pesan_list:
        isi = ''.join((b.get('text', '') for b in p.get('parts', [])))
        if not isi.strip():
            continue
        pesan.append({'role': 'assistant' if p.get('role') == 'model' else 'user', 'content': isi})
    async with aiohttp.ClientSession() as session:
        async with session.post(penyedia['url'], headers={'Authorization': f"Bearer {penyedia['kunci']}"}, json={'model': penyedia['model'], 'messages': pesan, 'temperature': 1.0, 'max_tokens': 800}, timeout=aiohttp.ClientTimeout(total=45)) as resp:
            if resp.status != 200:
                if resp.status == 429:
                    print(f"[AI] {penyedia['nama']}: jatahnya abis juga")
                else:
                    print(f"[AI] {penyedia['nama']} gagal, status {resp.status}")
                    print(f'[AI] pesan : {(await resp.text())[:250]}')
                return None
            data = await resp.json()
    teks = (data.get('choices') or [{}])[0].get('message', {}).get('content', '').strip()
    if teks:
        print(f"[AI] dijawab sama {penyedia['nama']}")
    return teks or None

async def tanya_ai(pesan_list, tambahan='', system_override=None):
    sys_prompt = system_override if system_override is not None else SYSTEM_PROMPT + tambahan
    body = {'system_instruction': {'parts': [{'text': sys_prompt}]}, 'contents': pesan_list, 'generationConfig': {'temperature': 1.0, 'maxOutputTokens': 2000}}
    for model in (MODEL, MODEL_CADANGAN):
        try:
            hasil = await kirim_ke_gemini(model, body)
            if hasil:
                return hasil
            print('[AI] pindah ke cadangan...')
        except Exception as e:
            print(f'[AI] {model} error: {e}')
    cadangan_tambahan = system_override if system_override is not None else tambahan
    for penyedia in PENYEDIA_CADANGAN:
        try:
            hasil = await kirim_ke_cadangan(penyedia, pesan_list, cadangan_tambahan)
            if hasil:
                return hasil
        except Exception as e:
            print(f"[AI] {penyedia['nama']} error: {e}")
    print('[AI] semua sumber gagal')
    try:
        bot.loop.create_task(lapor_ai_mati())
    except Exception:
        pass
    return None

@bot.event
async def on_ready():
    print(f'Bot {bot.user} online')
    muat_setelan()
    muat_riwayat()
    muat_profil()
    muat_catatan()
    await pasang_status()
    try:
        await bot.tree.sync()
    except Exception as e:
        print(f'[sync] error: {e}')
    print(f'[revive] role={ROLE_REVIVE} | ngintip tiap {CEK_TIAP_MENIT} menit | nembak kalau sepi >{SEPI_JAM} jam | jeda antar ping >{JEDA_REVIVE_JAM} jam | cuma jam {REVIVE_JAM_MULAI}-{REVIVE_JAM_SELESAI} WIB')
    print(f'[ingatan] {RIWAYAT_PANJANG} pesan per channel | ditulis tiap {RIWAYAT_SIMPAN_MENIT} menit | channel nganggur dibuang setelah {RIWAYAT_UMUR_JAM} jam')
    print(f'[rem] {JEDA_USER_DETIK} detik antar panggilan, maks {JATAH_USER_PER_JAM} per orang per jam')
    print(f"[gambar] baca gambar={('nyala' if BACA_GAMBAR else 'mati')} (cuma lewat Gemini, cadangan ga bisa liat gambar)")
    await cek_model()
    if not cek_sepi.is_running():
        cek_sepi.start()
    if not jaga_ingatan.is_running():
        jaga_ingatan.start()
    if not jaga_status.is_running():
        jaga_status.start()

async def pasang_status():
    pilihan = list(STATUS_LIST_DIRGA)
    if STATUS_IKUT_CUACA:
        isi, aman = penyimpanan.baca(BERKAS_ARKA)
        cuaca = (isi or {}).get('cuaca') or {}
        if aman and cuaca.get('nama'):
            pilihan.append(f"cuaca hari ini: {cuaca['nama'].lower()}")
    tulisan = random.choice(pilihan)
    try:
        await bot.change_presence(activity=discord.CustomActivity(name=tulisan))
    except Exception as e:
        print(f'[status] gagal dipasang: {e}')

async def cek_model():
    url = 'https://generativelanguage.googleapis.com/v1beta/models'
    try:
        async with aiohttp.ClientSession() as session:
            async with session.get(url, params={'key': GEMINI_KEY}) as resp:
                if resp.status != 200:
                    print(f'[model] ga bisa ngecek, status {resp.status}')
                    print(f'[model] pesan: {(await resp.text())[:300]}')
                    return
                data = await resp.json()
        bisa = [m['name'].replace('models/', '') for m in data.get('models', []) if 'generateContent' in m.get('supportedGenerationMethods', [])]
        print(f'[model] gemini: {MODEL} lalu {MODEL_CADANGAN}')
        siap = [x['nama'] for x in PENYEDIA_CADANGAN if x['kunci']]
        belum = [x['nama'] for x in PENYEDIA_CADANGAN if not x['kunci']]
        print(f"[model] cadangan siap  : {(', '.join(siap) if siap else '(belum ada)')}")
        if belum:
            print(f"[model] kunci belum ada: {', '.join(belum)}")
        print(f"[model] yang tersedia : {(', '.join(bisa[:15]) if bisa else '(kosong)')}")
        if MODEL not in bisa:
            print(f"[model] >>> '{MODEL}' GA ADA di daftar. Ganti baris MODEL pakai salah satu di atas.")
    except Exception as e:
        print(f'[model] error: {e}')

async def dirga_level_up(isi):
    uid = isi.get('user_id')
    nama = isi.get('nama')
    baru = isi.get('tingkat')
    if not uid or not nama:
        return
    ch_id = setelan.get('channel_ngobrol')
    if not ch_id:
        return
    channel = bot.get_channel(ch_id)
    if not channel:
        return
    pesan_list = [{'role': 'user', 'parts': [{'text': f"SYSTEM: {nama} baru aja naik level jadi '{baru}'. Kasih ucapan selamat yang santai, gaya lo banget, cukup 1 atau 2 kalimat aja."}]}]
    jawaban = await tanya_ai(pesan_list)
    if jawaban:
        await channel.send(f'<@{uid}> {jawaban}')
acara.dengar('level_up', dirga_level_up)

@bot.event
async def on_member_join(member):
    embed = discord.Embed(title='👋 Selamat Datang di Summer Tide!', description=f'Halo **{member.name}**, selamat gabung di server **Summer Tide**!\n\nGua **Dirga**, bot asisten pribadi di sini. Biar lo ga bingung, gua bakal kenalin temen-temen bot gua yang punya tugas masing-masing di server ini:', color=2829617)
    embed.add_field(name='🧑\u200d💻 Dirga (Gua Sendiri)', value='Gua bot pinter yang bisa diajak ngobrol, nerjemahin bahasa, bantuin tugas, sampai ngebangunin server pas lagi sepi.', inline=False)
    embed.add_field(name='🛡️ Julian', value='Satpam server. Dia mantau log, ngapus pesan nyasar, dan mastiin server aman dari kerusuhan.', inline=False)
    embed.add_field(name='🎮 Arka', value='Tukang ngurusin *experience* (XP) & sistem level. Rajin /absen tiap hari di Arka biar level lu naik!', inline=False)
    embed.add_field(name='🎧 Kevin', value='Spesialis musik Lo-Fi 24/7. Ketik `/lofi` buat nge-chill bareng lagu-lagu santai racikan Kevin.', inline=False)
    embed.add_field(name='🎵 TideTunes', value='Bot musik andalan server buat muterin lagu *request*-an lu di voice channel.', inline=False)
    embed.add_field(name='🍿 WaveFlix', value='Temen nobar! Lu bisa nyari film & bikin jadwal nonton bareng pakai `/nobar_cari` (khusus di WaveFlix Room).', inline=False)
    embed.set_footer(text='💡 Psst! Lu punya waktu 1 JAM buat nanya-nanya apa aja soal server ini ke gua lewat DM ini. Yuk, tanya aja!')
    try:
        await member.send(embed=embed)
    except Exception as e:
        print(f'[on_member_join] Gagal kirim DM ke {member.name}: {e}')

@bot.event
async def on_message(message):
    global _ingatan_berubah
    if message.author.bot:
        return
    tulisan = message.content
    if message.attachments and (not tulisan.strip()):
        tulisan = '(ngirim gambar)'
    riwayat[message.channel.id].append({'role': 'user', 'parts': [{'text': f'{message.author.display_name}: {tulisan}'}]})
    _kapan_disentuh[message.channel.id] = datetime.now(timezone.utc)
    _ingatan_berubah = True
    di_dm = message.guild is None
    orientasi_aktif = False
    if di_dm:
        guild = message.author.mutual_guilds[0] if message.author.mutual_guilds else None
        if guild:
            member = guild.get_member(message.author.id)
            if member and member.joined_at:
                umur = datetime.now(timezone.utc) - member.joined_at
                if umur > timedelta(hours=1):
                    await message.channel.send('Masa orientasi lu udah habis bro! Kalau masih ada yang bingung atau butuh bantuan lain, bisa langsung nanya ke Admin di server ya.')
                    return
                else:
                    orientasi_aktif = True
    kena_mention = bot.user.mentioned_in(message) and (not message.mention_everyone)
    bales_ke_bot = message.reference is not None and message.reference.resolved is not None and (getattr(message.reference.resolved, 'author', None) == bot.user)
    di_channel_ngobrol = message.channel.id == setelan['channel_ngobrol']
    if not (di_dm or kena_mention or bales_ke_bot or di_channel_ngobrol):
        await bot.process_commands(message)
        return
    boleh, alasan = boleh_manggil(message.author.id)
    if not boleh:
        try:
            await message.add_reaction('🥱')
        except Exception:
            pass
        if kena_mention and alasan and ('kebanyakan' in alasan):
            await message.reply(alasan, mention_author=False, delete_after=15)
        await bot.process_commands(message)
        return
    gambar = await ambil_gambar(message)
    if gambar:
        antrian = list(riwayat[message.channel.id])
        antrian[-1] = {'role': 'user', 'parts': antrian[-1]['parts'] + gambar}
    else:
        antrian = list(riwayat[message.channel.id])
    bekal = konteks_server(message.guild) + bekal_profil(message.author)
    if orientasi_aktif:
        bekal += '\n\nSISTEM [SANGAT PENTING]: User ini baru saja bergabung dengan server Summer Tide (kurang dari 1 jam yang lalu). Lu sedang melayani dia lewat DM sebagai pemandu server. Jawab pertanyaan dia tentang server, channel, role, atau bot lain dengan ramah dan gaya lu yang biasa. FOKUS HANYA pada membantu dia mengenal server ini. JIKA dia bertanya atau mengobrol tentang hal lain di luar konteks server Discord ini, TOLAK dengan sopan dan alihkan pembicaraan kembali ke fitur-fitur server.'
    async with message.channel.typing():
        jawaban = await tanya_ai(antrian, bekal)
    if jawaban:
        riwayat[message.channel.id].append({'role': 'model', 'parts': [{'text': jawaban}]})
        _ingatan_berubah = True
        try:
            await balas_panjang(message, jawaban)
        except discord.HTTPException as e:
            print(f'[chat] gagal ngirim balasan: {e}')
        if PROFIL_AKTIF:
            uid = message.author.id
            _hitung_balasan[uid] = _hitung_balasan.get(uid, 0) + 1
            if _hitung_balasan[uid] >= PROFIL_TIAP_BALASAN:
                _hitung_balasan[uid] = 0
                bot.loop.create_task(tarik_fakta(message.author, list(riwayat[message.channel.id])))
    await bot.process_commands(message)

@tasks.loop(minutes=RIWAYAT_SIMPAN_MENIT)
async def jaga_ingatan():
    dibuang = buang_yang_basi()
    if _ingatan_berubah or dibuang:
        simpan_riwayat()

@jaga_ingatan.before_loop
async def _sebelum_ingatan():
    await bot.wait_until_ready()

@tasks.loop(seconds=10)
async def jaga_status():
    await pasang_status()

@jaga_status.before_loop
async def _sebelum_status():
    await bot.wait_until_ready()

@tasks.loop(minutes=CEK_TIAP_MENIT)
async def cek_sepi():
    global revive_terakhir
    if CHANNEL_REVIVE_ID == 0:
        return
    channel = bot.get_channel(CHANNEL_REVIVE_ID)
    if channel is None:
        return
    sekarang = datetime.now(timezone.utc)
    jam_wib = (sekarang + timedelta(hours=7)).hour
    if not REVIVE_JAM_MULAI <= jam_wib < REVIVE_JAM_SELESAI:
        return
    if revive_terakhir and sekarang - revive_terakhir < timedelta(hours=JEDA_REVIVE_JAM):
        return
    pesan_terakhir = None
    async for msg in channel.history(limit=100):
        if not msg.author.bot:
            pesan_terakhir = msg
            break
    if pesan_terakhir is not None:
        if sekarang - pesan_terakhir.created_at < timedelta(hours=SEPI_JAM):
            return
    role = discord.utils.get(channel.guild.roles, name=ROLE_REVIVE)
    if role is None:
        print(f"[revive] role '{ROLE_REVIVE}' belum dibuat")
        return
    kandidat = [m for m in role.members if not m.bot and m.id not in pernah_diping and (not m.is_timed_out()) and (not kena_bisu(m.id))]
    if not kandidat:
        pernah_diping.clear()
        kandidat = [m for m in role.members if not m.bot and (not m.is_timed_out()) and (not kena_bisu(m.id))]
    if not kandidat:
        return
    target = random.choice(kandidat)
    bahan = bahan_pemantik(channel.guild)
    bumbu = ''
    if bahan:
        bumbu = '\n\nBoleh dipakai kalau nyambung, pilih SATU aja, jangan disebut semua:\n' + '\n'.join((f'- {b}' for b in bahan))
    prompt = [{'role': 'user', 'parts': [{'text': f'Sapa {target.display_name} pakai satu pertanyaan ringan yang bikin dia pengen bales. Topik bebas, apa aja yang santai. Jangan formal, jangan nyebut kalau server lagi sepi. Satu kalimat aja.{bumbu}'}]}]
    pemantik = await tanya_ai(prompt) or random.choice(PEMANTIK_CADANGAN)
    await channel.send(f'{target.mention} {potong(pemantik)[0]}')
    pernah_diping.append(target.id)
    revive_terakhir = sekarang
    simpan_catatan()
    print(f'[revive] ping ke {target.display_name}')

@cek_sepi.before_loop
async def sebelum_loop():
    await bot.wait_until_ready()

@bot.tree.command(name='halo', description='Cek bot masih idup apa engga')
async def halo(interaction: discord.Interaction):
    jumlah = len(riwayat.get(interaction.channel_id, []))
    await interaction.response.send_message(f'idup kok, santai. gua inget {jumlah} pesan terakhir di sini.')

@bot.tree.command(name='lupa', description='Bikin Dirga lupa obrolan di channel ini')
@app_commands.checks.has_permissions(manage_messages=True)
async def lupa(interaction: discord.Interaction):
    global _ingatan_berubah
    jumlah = len(riwayat.get(interaction.channel_id, []))
    riwayat.pop(interaction.channel_id, None)
    _kapan_disentuh.pop(interaction.channel_id, None)
    _ingatan_berubah = True
    simpan_riwayat()
    if jumlah:
        await interaction.response.send_message(f'oke, {jumlah} pesan terakhir gua lupain. mulai dari kosong lagi.')
    else:
        await interaction.response.send_message('emang belum ada yang gua inget di sini.', ephemeral=True)

@bot.tree.command(name='ringkas', description='Ringkas obrolan terakhir di channel ini')
async def ringkas(interaction: discord.Interaction):
    pesan = [p for p in riwayat.get(interaction.channel_id, []) if p.get('role') == 'user']
    if len(pesan) < 4:
        await interaction.response.send_message('obrolannya masih dikit, ga ada yang perlu diringkas.', ephemeral=True)
        return
    boleh, alasan = boleh_manggil(interaction.user.id)
    if not boleh:
        await interaction.response.send_message(alasan, ephemeral=True)
        return
    await interaction.response.defer()
    perintah = list(riwayat[interaction.channel_id]) + [{'role': 'user', 'parts': [{'text': 'Ringkas obrolan di atas jadi 3 sampai 5 poin pendek. Sebut siapa ngomong apa. Cuma poinnya aja, ga usah kasih pengantar atau penutup. Kalau obrolannya cuma basa basi, bilang aja ga ada yang penting.'}]}]
    hasil = await tanya_ai(perintah)
    if not hasil:
        await interaction.followup.send('lagi ga bisa mikir, coba lagi nanti.')
        return
    potongan = potong(hasil)
    await interaction.followup.send(potongan[0])
    for lanjutan in potongan[1:]:
        await interaction.channel.send(lanjutan)

@bot.tree.command(name='topik', description='Minta Dirga lempar topik obrolan baru')
async def topik(interaction: discord.Interaction):
    boleh, alasan = boleh_manggil(interaction.user.id)
    if not boleh:
        await interaction.response.send_message(alasan, ephemeral=True)
        return
    await interaction.response.defer()
    bekal = list(riwayat.get(interaction.channel_id, []))[-6:]
    perintah = bekal + [{'role': 'user', 'parts': [{'text': 'Lempar satu topik obrolan baru yang bikin orang pengen nimbrung. Nyambung sama suasana obrolan di atas kalau ada, kalau ga ada ya bebas. Satu atau dua kalimat, jangan formal, jangan nyebut kalau server lagi sepi.'}]}]
    hasil = await tanya_ai(perintah) or random.choice(TOPIK_CADANGAN)
    await interaction.followup.send(potong(hasil)[0])

@bot.tree.command(name='ngobroldi', description='Atur channel tempat Dirga nyaut tanpa di-mention')
@app_commands.describe(channel='Kosongin kalau mau dimatiin')
@app_commands.checks.has_permissions(manage_guild=True)
async def ngobroldi(interaction: discord.Interaction, channel: discord.TextChannel=None):
    setelan['channel_ngobrol'] = channel.id if channel else 0
    simpan_setelan()
    if channel:
        await interaction.response.send_message(f'oke, mulai sekarang gua nyaut sendiri di {channel.mention} tanpa perlu di-mention.')
    else:
        await interaction.response.send_message('channel ngobrol bebas gua matiin. sekarang gua cuma nyaut kalau di-mention atau dibales.')

@bot.tree.command(name='jatah', description='Sisa kuota nanya lo jam ini')
async def jatah(interaction: discord.Interaction):
    sekarang = time.time()
    jejak = [w for w in _jejak_user.get(interaction.user.id, []) if sekarang - w < 3600]
    kepakai = len(jejak)
    sisa = max(0, JATAH_USER_PER_JAM - kepakai)
    panjang = 12
    isi = round(sisa / JATAH_USER_PER_JAM * panjang)
    batang = '▰' * isi + '▱' * (panjang - isi)
    embed = discord.Embed(title='Jatah nanya lo', description=f'`{batang}`  **{sisa}** dari {JATAH_USER_PER_JAM}', color=discord.Color.green() if sisa > 5 else discord.Color.orange())
    if jejak:
        pulih = int(jejak[0] + 3600 - sekarang)
        embed.add_field(name='Jatah mulai balik', value=f'<t:{int(sekarang + pulih)}:R>', inline=True)
    embed.add_field(name='Jeda antar tanya', value=f'{JEDA_USER_DETIK} detik', inline=True)
    embed.set_footer(text='Batas ini biar satu orang ga ngabisin jatah harian buat seluruh server')
    await interaction.response.send_message(embed=embed, ephemeral=True)

@bot.tree.command(name='terjemah', description='Terjemahin teks')
@app_commands.describe(teks='Yang mau diterjemahin', ke='Mau ke bahasa apa')
@app_commands.choices(ke=[app_commands.Choice(name='Indonesia', value='Indonesia'), app_commands.Choice(name='Inggris', value='Inggris'), app_commands.Choice(name='Korea', value='Korea'), app_commands.Choice(name='Jepang', value='Jepang'), app_commands.Choice(name='Arab', value='Arab')])
async def terjemah(interaction: discord.Interaction, teks: str, ke: app_commands.Choice[str]):
    if len(teks) > 1000:
        await interaction.response.send_message('kepanjangan, maksimal 1000 huruf.', ephemeral=True)
        return
    boleh, alasan = boleh_manggil(interaction.user.id)
    if not boleh:
        await interaction.response.send_message(alasan, ephemeral=True)
        return
    await interaction.response.defer()
    perlu_baca = ke.value in ('Korea', 'Jepang', 'Arab')
    tambahan = '\nBaris kedua: cara bacanya pakai huruf latin.\nBaris ketiga: satu catatan singkat soal susunan kalimatnya atau kata yang menarik.' if perlu_baca else ''
    perintah = [{'role': 'user', 'parts': [{'text': f'Tugas: Terjemahkan teks berikut ke bahasa {ke.value}:\n\n{teks}\n\nBaris pertama: hasil terjemahannya doang.{tambahan}\nJangan kasih pengantar, jangan kasih tanda kutip.'}]}]
    system_override = 'Kamu adalah asisten penerjemah bahasa yang sangat profesional dan ketat. Tugasmu hanya menerjemahkan teks. ATURAN MUTLAK: Jika teks yang diberikan mengandung kata-kata kasar, makian, vulgar, atau pornografi (contoh: memek, kontol, anjing, bangsat, dll), kamu WAJIB menolak perintah ini dengan HANYA membalas teks berikut tanpa tambahan apa pun:\n⚠️ Maaf, kata tersebut terlalu tidak pantas untuk diterjemahkan di channel publik.'
    hasil = await tanya_ai(perintah, system_override=system_override)
    if not hasil:
        await interaction.followup.send('lagi ga bisa mikir, coba lagi nanti.')
        return
    if 'terlalu tidak pantas' in hasil:
        teks_asli = '[Disensor]'
    else:
        teks_asli = teks[:1000]
    embed = discord.Embed(title=f'Terjemahan ke {ke.value}', description=potong(hasil, 3900)[0], color=discord.Color.blurple())
    embed.add_field(name='Aslinya', value=teks_asli, inline=False)
    await interaction.followup.send(embed=embed)

@bot.tree.command(name='ingat', description='Kasih tau Dirga sesuatu soal lo')
@app_commands.describe(catatan='Hal yang mau dia inget soal lo')
async def ingat(interaction: discord.Interaction, catatan: str):
    if not PROFIL_AKTIF:
        await interaction.response.send_message('fitur ingatan lagi mati.', ephemeral=True)
        return
    if len(catatan) > PROFIL_PANJANG_MAKS:
        await interaction.response.send_message(f'kepanjangan, maksimal {PROFIL_PANJANG_MAKS} huruf.', ephemeral=True)
        return
    catat = catatan_orang(interaction.user.id, interaction.user.display_name)
    if len(catat['catatan']) >= PROFIL_MAKS_CATATAN:
        catat['catatan'].pop(0)
    catat['catatan'].append(catatan)
    catat['diperbarui'] = datetime.now(timezone.utc).isoformat()
    simpan_profil()
    await interaction.response.send_message(f"oke, gua inget: *{catatan}*\nsekarang gua nyatet {len(catat['catatan'])} hal soal lo. liat semua pakai `/ingatan`.", ephemeral=True)

@bot.tree.command(name='ingatan', description='Liat apa aja yang Dirga inget soal lo')
async def ingatan(interaction: discord.Interaction):
    catat = profil.get(str(interaction.user.id))
    if not catat or not catat.get('catatan'):
        await interaction.response.send_message('gua belum inget apa apa soal lo. ngobrol dulu sana, atau kasih tau langsung pakai `/ingat`.', ephemeral=True)
        return
    baris = '\n'.join((f'`{i}.` {c}' for i, c in enumerate(catat['catatan'], 1)))
    embed = discord.Embed(title='Yang gua inget soal lo', description=baris, color=discord.Color.blurple())
    embed.set_footer(text='Hapus semua pakai /lupakan · ini cuma lo yang bisa liat')
    await interaction.response.send_message(embed=embed, ephemeral=True)

@bot.tree.command(name='lupakan', description='Hapus semua yang Dirga inget soal lo')
async def lupakan(interaction: discord.Interaction):
    catat = profil.pop(str(interaction.user.id), None)
    if not catat or not catat.get('catatan'):
        await interaction.response.send_message('emang ga ada yang gua inget.', ephemeral=True)
        return
    simpan_profil()
    await interaction.response.send_message(f"oke, {len(catat['catatan'])} catatan soal lo gua hapus. kita mulai dari awal lagi.", ephemeral=True)

@bot.tree.command(name='revive', description='Ikut atau keluar dari panggilan kalau server lagi sepi')
async def cmd_revive(interaction: discord.Interaction):
    if interaction.guild is None:
        await interaction.response.send_message('Cuma bisa dipakai di dalam server.', ephemeral=True)
        return
    role = discord.utils.get(interaction.guild.roles, name=ROLE_REVIVE)
    if role is None:
        await interaction.response.send_message(f'Role **{ROLE_REVIVE}** belum dibikin sama admin.', ephemeral=True)
        return
    try:
        if role in interaction.user.roles:
            await interaction.user.remove_roles(role)
            await interaction.response.send_message(f'Sip, lo ga bakal kena ping dari Dirga lagi.', ephemeral=True)
        else:
            await interaction.user.add_roles(role)
            await interaction.response.send_message(f'Mantap, lo masuk daftar sasaran ping. Kalo lagi sepi ntar gua cari.', ephemeral=True)
    except discord.Forbidden:
        await interaction.response.send_message('gua ga punya izin buat ngatur role itu. minta admin cek posisi role gua ya.', ephemeral=True)

@bot.tree.command(name='ajakmabar', description='Suruh Dirga nge-tag dan ngajak orang mabar')
@app_commands.describe(siapa1='Siapa yang mau lu ajak? (Wajib pilih orang/role)', siapa2='Ada lagi yang mau diajak? (Opsional)', siapa3='Satu lagi deh? (Opsional)', game='Game apa nih? (Opsional)', pesan_tambahan='Ada pesan khusus dari lu? (Opsional)')
async def ajakmabar(interaction: discord.Interaction, siapa1: discord.Member, siapa2: discord.Member=None, siapa3: discord.Member=None, game: str='', pesan_tambahan: str=''):
    await interaction.response.defer()
    targets = [t.mention for t in (siapa1, siapa2, siapa3) if t is not None]
    mentions_str = ' '.join(targets)
    prompt_game = f' game {game}' if game else ' game bebas'
    prompt_tambahan = f' Pesan tambahan dari yang ngajak: "{pesan_tambahan}".' if pesan_tambahan else ''
    prompt = [{'role': 'user', 'parts': [{'text': f'Tugas lu adalah jadi perantara buat ngajakin orang mabar{prompt_game}. Target yang diajak: {mentions_str}.{prompt_tambahan} Buat satu kalimat ajakan santai ala anak tongkrongan yang ngegas dikit biar pada mau join. Lu posisinya mewakili si pengajak ({interaction.user.display_name}). Jangan kaku.'}]}]
    ajakan = await tanya_ai(prompt) or 'woi login ga lo pada, mabar buruan.'
    await interaction.channel.send(f'{mentions_str} {potong(ajakan)[0]}')
    await interaction.followup.send(f'Beres, udah gua panggilin {mentions_str}.', ephemeral=True)

@lupa.error
@ngobroldi.error
@ajakmabar.error
async def _tolak(interaction: discord.Interaction, error):
    if isinstance(error, app_commands.MissingPermissions):
        pesan = 'perintah ini cuma buat yang punya izin.'
    else:
        pesan = f'ada yang error: {error}'
        print(f'[error] {error}')
    if interaction.response.is_done():
        await interaction.followup.send(pesan, ephemeral=True)
    else:
        await interaction.response.send_message(pesan, ephemeral=True)
if __name__ == '__main__':
    if not TOKEN:
        raise SystemExit('DIRGA_TOKEN belum keisi di file .env')
    if not GEMINI_KEY:
        raise SystemExit('GEMINI_API_KEY belum keisi di file .env')
    bot.run(TOKEN)