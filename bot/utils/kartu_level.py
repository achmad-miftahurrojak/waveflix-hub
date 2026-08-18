import os
import io
from PIL import Image, ImageDraw, ImageFont
LEBAR, TINGGI = (1200, 630)
_UTILS_DIR = os.path.dirname(os.path.abspath(__file__))
_BOT_ROOT = os.path.dirname(_UTILS_DIR)
FOLDER = os.path.join(_BOT_ROOT, 'gambar')
NAMA_LATAR = ['level-bg.png', 'welcome-bg.png', 'welcome-bg.jpg']
WHITE_GLASS = (255, 255, 255, 225)
DARK_BLUE = (15, 30, 70)
OCEAN_BLUE = (0, 150, 220)
SUNNY_YELLOW = (255, 200, 50)
TEXT_DIM = (70, 90, 120)

def _latar():
    for n in NAMA_LATAR:
        j = os.path.join(FOLDER, n)
        if os.path.exists(j):
            try:
                g = Image.open(j).convert('RGBA')
                g = g.resize((LEBAR, TINGGI), Image.Resampling.LANCZOS)
                return g
            except Exception:
                pass
    g = Image.new('RGBA', (LEBAR, TINGGI), (135, 206, 235, 255))
    return g

def get_font(name, size):
    for f in (f'C:/Windows/Fonts/{name}.ttf', f'C:/Windows/Fonts/{name}.ttc', f'C:/Windows/Fonts/{name}bd.ttf'):
        if os.path.exists(f):
            try:
                return ImageFont.truetype(f, size)
            except:
                pass
    return ImageFont.load_default()

def draw_text(draw, pos, text, font, fill, anchor='lt', shadow=False):
    if shadow:
        draw.text((pos[0] + 2, pos[1] + 2), text, font=font, fill=(0, 0, 0, 80), anchor=anchor)
    draw.text(pos, text, font=font, fill=fill, anchor=anchor)

def buat_kartu_level(nama, avatar_bytes, tingkat, xp, xp_bawah, xp_atas, peringkat=None, lencana=None, catat=None):
    if catat is None:
        catat = {}
    dasar = _latar()
    lapis = Image.new('RGBA', (LEBAR, TINGGI), (0, 0, 0, 0))
    gambar = ImageDraw.Draw(lapis)
    f_title = get_font('bahnschrift', 75)
    f_rank = get_font('bahnschrift', 55)
    f_sub = get_font('bahnschrift', 30)
    AVATAR_SIZE = 360
    AVATAR_X, AVATAR_Y = (60, 130)
    gambar.ellipse([AVATAR_X - 15, AVATAR_Y - 15, AVATAR_X + AVATAR_SIZE + 15, AVATAR_Y + AVATAR_SIZE + 15], fill=(255, 200, 50, 150))
    gambar.ellipse([AVATAR_X - 5, AVATAR_Y - 5, AVATAR_X + AVATAR_SIZE + 5, AVATAR_Y + AVATAR_SIZE + 5], fill=WHITE_GLASS[:3])
    try:
        avatar = Image.open(io.BytesIO(avatar_bytes)).convert('RGBA')
        avatar = avatar.resize((AVATAR_SIZE, AVATAR_SIZE), Image.Resampling.LANCZOS)
        mask = Image.new('L', (AVATAR_SIZE, AVATAR_SIZE), 0)
        ImageDraw.Draw(mask).ellipse((0, 0, AVATAR_SIZE, AVATAR_SIZE), fill=255)
        lapis.paste(avatar, (AVATAR_X, AVATAR_Y), mask)
    except:
        gambar.ellipse([AVATAR_X, AVATAR_Y, AVATAR_X + AVATAR_SIZE, AVATAR_Y + AVATAR_SIZE], fill=OCEAN_BLUE)
    draw_text(gambar, (AVATAR_X + AVATAR_SIZE // 2, AVATAR_Y + AVATAR_SIZE + 30), nama.upper()[:15], f_title, WHITE_GLASS[:3], anchor='mt', shadow=True)
    PANEL_X = 480
    PANEL_W = 670
    gambar.rounded_rectangle([PANEL_X, 60, PANEL_X + PANEL_W, 580], radius=25, fill=WHITE_GLASS)
    draw_text(gambar, (PANEL_X + 40, 70), tingkat.upper(), f_rank, DARK_BLUE)
    if xp_atas:
        xp_text = f'{xp:,} / {xp_atas:,} XP'
        ratio = min(1.0, (xp - xp_bawah) / max(1, xp_atas - xp_bawah))
    else:
        xp_text = f'{xp:,} XP (MAX)'
        ratio = 1.0
    draw_text(gambar, (PANEL_X + PANEL_W - 40, 135), xp_text, f_sub, TEXT_DIM, anchor='rm')
    BAR_Y = 155
    BAR_H = 20
    gambar.rounded_rectangle([PANEL_X + 40, BAR_Y, PANEL_X + PANEL_W - 40, BAR_Y + BAR_H], radius=10, fill=(220, 220, 230, 255))
    gambar.rounded_rectangle([PANEL_X + 40, BAR_Y, PANEL_X + 40 + int((PANEL_W - 80) * ratio), BAR_Y + BAR_H], radius=10, fill=OCEAN_BLUE)
    STAT_Y = 210
    draw_text(gambar, (PANEL_X + 40, STAT_Y), 'SUMMER STATS', get_font('bahnschrift', 22), TEXT_DIM)
    f_stat = get_font('bahnschrift', 28)
    stats_list = [('PESAN', f"{catat.get('pesan', 0):,}"), ('WAKTU NONGKRONG (MENIT)', f"{catat.get('menit', 0):,}"), ('ABSEN BERUNTUN', f"{catat.get('absen', 0):,} HARI")]
    for i, (label, val) in enumerate(stats_list):
        sy = STAT_Y + 40 + i * 50
        draw_text(gambar, (PANEL_X + 40, sy), label, f_stat, DARK_BLUE)
        draw_text(gambar, (PANEL_X + PANEL_W - 40, sy), val, f_stat, OCEAN_BLUE, anchor='rt')
        gambar.line([(PANEL_X + 40, sy + 38), (PANEL_X + PANEL_W - 40, sy + 38)], fill=(200, 210, 220, 255), width=2)
    COMBAT_Y = 410
    draw_text(gambar, (PANEL_X + 40, COMBAT_Y), 'ARENA RECORD', get_font('bahnschrift', 22), TEXT_DIM)
    lencana_count = len(catat.get('lencana', [])) if 'lencana' in catat else 0
    combat_list = [('MENANG', f"{catat.get('menang', 0):,}"), ('KALAH', f"{catat.get('kalah', 0):,}"), ('LENCANA', f'{lencana_count}')]
    col_w = (PANEL_W - 80) // 3
    for i, (label, val) in enumerate(combat_list):
        cx = PANEL_X + 40 + i * col_w + col_w // 2
        draw_text(gambar, (cx, COMBAT_Y + 40), label, f_sub, TEXT_DIM, anchor='mt')
        draw_text(gambar, (cx, COMBAT_Y + 80), val, get_font('bahnschrift', 45), DARK_BLUE, anchor='mt')
    if peringkat:
        tag_w = 120
        tag_h = 50
        gambar.rounded_rectangle([LEBAR - tag_w, 30, LEBAR + 10, 30 + tag_h], radius=15, fill=SUNNY_YELLOW)
        draw_text(gambar, (LEBAR - tag_w // 2 + 5, 30 + tag_h // 2 - 2), f'#{peringkat}', get_font('impact', 35), DARK_BLUE, anchor='mm')
    dasar.alpha_composite(lapis)
    keluar = io.BytesIO()
    dasar.convert('RGB').save(keluar, format='PNG', optimize=True)
    keluar.seek(0)
    return keluar