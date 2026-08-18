import io
import os
from PIL import Image, ImageDraw, ImageFilter, ImageFont, ImageSequence
LEBAR, TINGGI = (1000, 500)
_UTILS_DIR = os.path.dirname(os.path.abspath(__file__))
_BOT_ROOT = os.path.dirname(_UTILS_DIR)
FOLDER = os.path.join(_BOT_ROOT, 'gambar')
NAMA_LATAR = ['welcome-bg.gif', 'welcome-bg.png', 'welcome-bg.jpg', 'welcome-bg.webp']
GELAP = 0.3
ZOOM = 1.0
GESER_X = 0.0
GESER_Y = 0.0
FONT_SENDIRI = os.path.join(FOLDER, 'font.ttf')
LEBAR_GIF = 600
MAKS_FRAME = 40
FPS = 10
KATA_ATAS = 'WELCOME'
KATA_BAWAH = 'YOU ARE OUR {nomor}{akhiran} MEMBER ♥'
AVATAR = 200
AVATAR_ATAS = 42
CINCIN = 7
PUSAT = LEBAR // 2
Y_ATAS = 306
Y_NAMA = 379
Y_BAWAH = 440
UKURAN_ATAS = 74
UKURAN_NAMA = 42
UKURAN_BAWAH = 27
WARNA_ATAS = (28, 32, 56)
WARNA_BAWAH = (68, 42, 92)
WARNA_AKSEN = (238, 246, 255)
GARIS_TEPI = (0, 0, 0)

def _cari_latar():
    for nama in NAMA_LATAR:
        jalur = os.path.join(FOLDER, nama)
        if os.path.exists(jalur):
            return jalur
    return None

def _cek_gerak(jalur):
    if not jalur:
        return False
    try:
        with Image.open(jalur) as gbr:
            return getattr(gbr, 'n_frames', 1) > 1
    except Exception:
        return False
JALUR_LATAR = _cari_latar()
BERGERAK = _cek_gerak(JALUR_LATAR)
NAMA_KELUARAN = 'welcome.gif' if BERGERAK else 'welcome.png'
print(f"[kartu] latar: {JALUR_LATAR or 'gradasi bawaan'} | bergerak: {('ya' if BERGERAK else 'engga')} | keluaran: {NAMA_KELUARAN}")

def _akhiran(n):
    if 10 <= n % 100 <= 20:
        return 'TH'
    return {1: 'ST', 2: 'ND', 3: 'RD'}.get(n % 10, 'TH')

def _font(ukuran, tebal=True):
    kandidat = [FONT_SENDIRI, '/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf' if tebal else '/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf', 'C:/Windows/Fonts/arialbd.ttf' if tebal else 'C:/Windows/Fonts/arial.ttf', '/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf']
    for jalur in kandidat:
        if jalur and os.path.exists(jalur):
            try:
                return ImageFont.truetype(jalur, ukuran)
            except Exception:
                continue
    return ImageFont.load_default()

def _font_muat(teks, gambar, ukuran, maks, tebal=True):
    while ukuran > 16:
        font = _font(ukuran, tebal)
        if gambar.textlength(teks, font=font) <= maks:
            return font
        ukuran -= 2
    return _font(16, tebal)

def _akhiran(n):
    if 10 <= n % 100 <= 20:
        return 'TH'
    return {1: 'ST', 2: 'ND', 3: 'RD'}.get(n % 10, 'TH')

def _tulis(gambar, y, teks, font, warna, tepi=3):
    gambar.text((PUSAT, y), teks, font=font, fill=warna, anchor='mm', stroke_width=tepi, stroke_fill=GARIS_TEPI)

def _potong_pas(gbr):
    lebar_asli, tinggi_asli = gbr.size
    skala = max(LEBAR / lebar_asli, TINGGI / tinggi_asli) * max(1.0, ZOOM)
    baru = (max(LEBAR, int(lebar_asli * skala)), max(TINGGI, int(tinggi_asli * skala)))
    gbr = gbr.resize(baru, Image.LANCZOS)
    sisa_x = baru[0] - LEBAR
    sisa_y = baru[1] - TINGGI
    x = int(sisa_x / 2 + GESER_X * sisa_x / 2)
    y = int(sisa_y / 2 + GESER_Y * sisa_y / 2)
    x = max(0, min(sisa_x, x))
    y = max(0, min(sisa_y, y))
    return gbr.crop((x, y, x + LEBAR, y + TINGGI))

def _gelapin(gbr):
    hitam = Image.new('RGB', (LEBAR, TINGGI), (0, 0, 0))
    return Image.blend(gbr, hitam, GELAP)

def _gradasi():
    gbr = Image.new('RGB', (LEBAR, TINGGI))
    gambar = ImageDraw.Draw(gbr)
    for y in range(TINGGI):
        rasio = y / TINGGI
        warna = tuple((int(WARNA_ATAS[i] + (WARNA_BAWAH[i] - WARNA_ATAS[i]) * rasio) for i in range(3)))
        gambar.line([(0, y), (LEBAR, y)], fill=warna)
    return gbr

def _bulatin(avatar, ukuran):
    avatar = avatar.convert('RGBA').resize((ukuran, ukuran))
    topeng = Image.new('L', (ukuran * 4, ukuran * 4), 0)
    ImageDraw.Draw(topeng).ellipse((0, 0, ukuran * 4, ukuran * 4), fill=255)
    topeng = topeng.resize((ukuran, ukuran), Image.LANCZOS)
    hasil = Image.new('RGBA', (ukuran, ukuran), (0, 0, 0, 0))
    hasil.paste(avatar, (0, 0), topeng)
    return hasil

def _lapisan(nama, avatar_bytes, nomor, nama_server):
    lapis = Image.new('RGBA', (LEBAR, TINGGI), (0, 0, 0, 0))
    gambar = ImageDraw.Draw(lapis)
    kiri = PUSAT - AVATAR // 2
    atas = AVATAR_ATAS
    bayangan = Image.new('RGBA', (LEBAR, TINGGI), (0, 0, 0, 0))
    ImageDraw.Draw(bayangan).ellipse((kiri - 10, atas - 6, kiri + AVATAR + 10, atas + AVATAR + 14), fill=(0, 0, 0, 130))
    bayangan = bayangan.filter(ImageFilter.GaussianBlur(12))
    lapis.alpha_composite(bayangan)
    gambar.ellipse((kiri - CINCIN, atas - CINCIN, kiri + AVATAR + CINCIN, atas + AVATAR + CINCIN), fill=(255, 255, 255, 255))
    try:
        avatar = Image.open(io.BytesIO(avatar_bytes))
        bulat = _bulatin(avatar, AVATAR)
        lapis.paste(bulat, (kiri, atas), bulat)
    except Exception as e:
        print(f'[kartu] avatar gagal: {e}')
        gambar.ellipse((kiri, atas, kiri + AVATAR, atas + AVATAR), fill=(90, 90, 110, 255))
    ruang = min(PUSAT, LEBAR - PUSAT) * 2 - 60
    f_atas = _font_muat(KATA_ATAS, gambar, UKURAN_ATAS, ruang)
    _tulis(gambar, Y_ATAS, KATA_ATAS, f_atas, (255, 255, 255), tepi=4)
    f_nama = _font_muat(nama, gambar, UKURAN_NAMA, ruang)
    _tulis(gambar, Y_NAMA, nama, f_nama, (255, 255, 255), tepi=3)
    bawah = KATA_BAWAH.format(nomor=nomor, server=nama_server, akhiran=_akhiran(nomor))
    f_bawah = _font_muat(bawah, gambar, UKURAN_BAWAH, ruang)
    _tulis(gambar, Y_BAWAH, bawah, f_bawah, WARNA_AKSEN, tepi=3)
    return lapis

def _kartu_diam(lapis):
    if JALUR_LATAR:
        try:
            with Image.open(JALUR_LATAR) as gbr:
                gbr.seek(0)
                dasar = _gelapin(_potong_pas(gbr.convert('RGB')))
        except Exception as e:
            print(f'[kartu] latar gagal dibaca: {e}')
            dasar = _gradasi()
    else:
        dasar = _gradasi()
    dasar = dasar.convert('RGBA')
    dasar.alpha_composite(lapis)
    keluar = io.BytesIO()
    dasar.convert('RGB').save(keluar, format='PNG')
    keluar.seek(0)
    return keluar

def _kartu_gerak(lapis):
    tinggi_gif = int(TINGGI * LEBAR_GIF / LEBAR)
    with Image.open(JALUR_LATAR) as sumber:
        jeda_asli = sumber.info.get('duration') or 80
        total = getattr(sumber, 'n_frames', 1)
        langkah = max(1, round(1000 / FPS / jeda_asli))
        if total / langkah > MAKS_FRAME:
            langkah = max(1, total // MAKS_FRAME)
        jeda = int(jeda_asli * langkah)
        frames = []
        for i, frame in enumerate(ImageSequence.Iterator(sumber)):
            if i % langkah:
                continue
            if len(frames) >= MAKS_FRAME:
                break
            dasar = _gelapin(_potong_pas(frame.convert('RGB'))).convert('RGBA')
            dasar.alpha_composite(lapis)
            frames.append(dasar.convert('RGB').resize((LEBAR_GIF, tinggi_gif), Image.LANCZOS).convert('P', palette=Image.ADAPTIVE, colors=128))
    keluar = io.BytesIO()
    frames[0].save(keluar, format='GIF', save_all=True, append_images=frames[1:], duration=jeda, loop=0, disposal=2, optimize=True)
    keluar.seek(0)
    print(f'[kartu] kartu bergerak: {len(frames)} frame, {len(keluar.getvalue()) / 1024 / 1024:.1f} MB')
    return keluar

def buat_kartu(nama, avatar_bytes, nomor, nama_server):
    lapis = _lapisan(nama, avatar_bytes, nomor, nama_server)
    if BERGERAK:
        try:
            return _kartu_gerak(lapis)
        except Exception as e:
            print(f'[kartu] gagal bikin versi bergerak, pakai gambar diam: {e}')
    return _kartu_diam(lapis)