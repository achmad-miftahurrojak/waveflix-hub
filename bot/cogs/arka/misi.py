import asyncio
import random
from datetime import datetime
import discord
from discord import app_commands
from discord.ext import commands, tasks
from . import core
from .config import *

def misi_hari_ini():
    kunci = core.hari_ini()
    if core.data.get('misi', {}).get('tanggal') != kunci:
        if core.data.get('misi', {}).get('daftar'):
            core.data['misi_kemarin_daftar'] = core.data['misi']['daftar']
        pilihan = random.sample(MISI_PILIHAN, MISI_JUMLAH)
        core.data['misi'] = {'tanggal': kunci, 'daftar': [{'kode': k, 'kalimat': t.format(n=n), 'target': n, 'xp': x} for k, t, n, x in pilihan], 'diumumkan': False}
        core.simpan()
    return core.data['misi']

def misi_orang(uid):
    hari = misi_hari_ini()
    catat = core.catatan(uid)
    misi = catat.setdefault('misi', {})
    if misi.get('tanggal') != hari['tanggal']:
        if misi.get('tanggal'):
            belum = 0
            for i, m in enumerate(core.data.get('misi_kemarin_daftar', [])):
                if m['kode'] not in misi.get('diambil', []):
                    belum += m['xp'] // 2
            if belum:
                catat['misi_kemarin'] = belum
        misi.clear()
        misi.update({'tanggal': hari['tanggal'], 'maju': {}, 'diambil': [], 'dikabarin': [], 'ganti': {}})
    for kunci, awal in [('maju', {}), ('diambil', []), ('dikabarin', []), ('ganti', {})]:
        misi.setdefault(kunci, awal)
    daftar = []
    for i, m in enumerate(hari['daftar']):
        daftar.append(misi['ganti'].get(str(i), m))
    return (daftar, misi)

async def kabarin_misi(bot, uid, m):
    try:
        orang = bot.get_user(uid) or await bot.fetch_user(uid)
        isi = discord.Embed(color=WARNA, title='✅  Satu misi kelar', description=f"**{m['kalimat']}**\n+{m['xp']} XP nunggu diambil.")
        isi.set_footer(text='Ketik /misi di channel bot buat ngambilnya')
        await orang.send(embed=isi)
    except Exception:
        pass

def catat_minggu(uid):
    catat = core.catatan(uid)
    minggu = catat.setdefault('misi_minggu', {})
    kunci = datetime.now(WIB).strftime('%Y-W%W')
    if minggu.get('minggu') != kunci:
        minggu.clear()
        minggu.update({'minggu': kunci, 'hari': [], 'diambil': False})
    hari = core.hari_ini()
    if hari not in minggu.setdefault('hari', []):
        minggu['hari'].append(hari)
    core.simpan()

def minggu_ini():
    return datetime.now(WIB).strftime('%Y-W%W')

def quest_sekarang():
    if not QUEST_AKTIF:
        return None
    quest = core.data.setdefault('quest', {})
    if quest.get('minggu') != minggu_ini():
        kode, kalimat, target = random.choice(QUEST_PILIHAN)
        quest.clear()
        quest.update({'minggu': minggu_ini(), 'kode': kode, 'kalimat': kalimat.format(n=target), 'target': target, 'maju': 0, 'penyumbang': {}, 'kelar': False, 'diumumkan': False})
        core.simpan()
    return quest

def maju_quest(uid, kode, jumlah=1):
    quest = quest_sekarang()
    if not quest or quest['kelar'] or quest['kode'] != kode:
        return False
    quest['maju'] += jumlah
    kunci = str(uid)
    quest['penyumbang'][kunci] = quest['penyumbang'].get(kunci, 0) + jumlah
    if quest['maju'] >= quest['target']:
        quest['kelar'] = True
        core.simpan()
        return True
    core.simpan()
    return False

def embed_misi():
    hari = misi_hari_ini()
    baris = [f"**{i}.** {m['kalimat']}  ·  +{m['xp']} XP" for i, m in enumerate(hari['daftar'], 1)]
    total = sum((m['xp'] for m in hari['daftar'])) + MISI_BONUS
    isi = discord.Embed(color=WARNA, title='🏝️  Misi Hari Ini', description='\n'.join(baris))
    isi.add_field(name='Kalau ketiganya kelar', value=f'Bonus **{MISI_BONUS} XP**, total sehari bisa **{total:,} XP**', inline=False)
    isi.add_field(name='Cara ngambilnya', value='Ketik `/misi` di sini. Kemajuan lo cuma keliatan sama lo sendiri, dan hadiahnya masuk pas lo ngetik itu.', inline=False)
    isi.set_footer(text='Ganti tiap jam 00.00 WIB · pakai /gantimisi kalau ada yang ga sreg dan lo punya Kerang Ajaib')
    return isi

def embed_quest():
    quest = quest_sekarang()
    if not quest:
        return None
    rasio = min(1.0, quest['maju'] / max(1, quest['target']))
    panjang = 16
    isi_batang = round(rasio * panjang)
    batang_quest = '▰' * isi_batang + '▱' * (panjang - isi_batang)
    isi = discord.Embed(color=WARNA, title='🤝  Quest server minggu ini', description=f"## {quest['kalimat']}\n`{batang_quest}`  **{quest['maju']}/{quest['target']}**")
    if quest['kelar']:
        isi.add_field(name='Status', value='✅ Udah tembus. Hadiah kebagi.', inline=False)
    else:
        isi.add_field(name='Hadiah', value=f'Kalau tembus, **semua yang nyumbang** dapet **{QUEST_BONUS} XP**. Nyumbang satu pun tetep kebagian.', inline=False)
    penyumbang = quest.get('penyumbang', {})
    if penyumbang:
        isi.add_field(name='Yang udah ikut', value=f'{len(penyumbang)} orang', inline=True)
    return isi

class Misi(commands.Cog):

    def __init__(self, bot):
        self.bot = bot
        self.umumin_misi.start()
        self.jaga_quest.start()

    def cog_unload(self):
        self.umumin_misi.cancel()
        self.jaga_quest.cancel()

    def maju_misi(self, uid, kode, jumlah=1):
        maju_quest(uid, kode, jumlah)
        daftar, misi = misi_orang(uid)
        ikut = [m for m in daftar if m['kode'] == kode]
        if not ikut:
            return
        sebelum = misi['maju'].get(kode, 0)
        misi['maju'][kode] = sebelum + jumlah
        sesudah = misi['maju'][kode]
        for m in ikut:
            baru_kelar = sebelum < m['target'] <= sesudah
            if baru_kelar and MISI_DM and (kode not in misi['dikabarin']):
                misi['dikabarin'].append(kode)
                try:
                    self.bot.loop.create_task(kabarin_misi(self.bot, uid, m))
                except Exception:
                    pass
        core.simpan()

    async def bayar_quest(self, guild, channel):
        quest = core.data.get('quest') or {}
        penyumbang = quest.get('penyumbang', {})
        if not penyumbang:
            return
        dibayar = []
        for uid, banyak in sorted(penyumbang.items(), key=lambda x: x[1], reverse=True):
            anggota = guild.get_member(int(uid))
            if anggota is None:
                continue
            await core.tambah_xp(self.bot, anggota, QUEST_BONUS)
            dibayar.append((anggota.display_name, banyak))
        baris = '\n'.join((f'**{nama}** — nyumbang {banyak}' for nama, banyak in dibayar[:15]))
        isi = discord.Embed(color=WARNA, title='🎉  Quest server TEMBUS', description=f"**{quest['kalimat']}**\nTarget {quest['target']} kelewat bareng bareng.\n\n{baris}")
        isi.set_footer(text=f'{len(dibayar)} orang dapet {QUEST_BONUS} XP masing masing. Quest baru diundi Senin.')
        await channel.send(embed=isi)

    @tasks.loop(minutes=10)
    async def umumin_misi(self):
        try:
            hari = misi_hari_ini()
            jam = datetime.now(WIB).hour
            if CHANNEL_MISI and (not hari.get('diumumkan')) and (jam >= JAM_MISI):
                channel = self.bot.get_channel(CHANNEL_MISI)
                if channel:
                    hari['diumumkan'] = True
                    core.simpan()
                    try:
                        await channel.send(embed=embed_misi())
                    except Exception:
                        hari['diumumkan'] = False
                        core.simpan()
                        raise
        except Exception as e:
            print(f'[arka] pengumuman misi error: {e}')

    @umumin_misi.before_loop
    async def before_umumin_misi(self):
        await self.bot.wait_until_ready()

    @tasks.loop(minutes=10)
    async def jaga_quest(self):
        try:
            if QUEST_AKTIF:
                quest = quest_sekarang()
                tujuan = CHANNEL_QUEST or CHANNEL_ARENA
                channel = self.bot.get_channel(tujuan) if tujuan else None
                if channel and quest and (not quest.get('diumumkan')):
                    quest['diumumkan'] = True
                    core.simpan()
                    isi = embed_quest()
                    if isi:
                        await channel.send('Quest baru minggu ini. Ini ga bisa dikelarin sendirian.', embed=isi)
                if channel and quest and quest.get('kelar') and (not quest.get('dibayar')):
                    quest['dibayar'] = True
                    core.simpan()
                    await self.bayar_quest(channel.guild, channel)
        except Exception as e:
            print(f'[arka] quest error: {e}')

    @jaga_quest.before_loop
    async def before_jaga_quest(self):
        await self.bot.wait_until_ready()

    @app_commands.command(name='misi', description='Misi hari ini, sekalian ambil hadiahnya')
    async def cmd_misi(self, inter: discord.Interaction):
        if not await core.di_arena(inter):
            return
        catat = core.catatan(inter.user.id)
        bayar_lumba = 0
        lumba_kepakai = False
        sisa_kemarin = catat.get('misi_kemarin')
        if sisa_kemarin and catat.setdefault('barang', {}).get('lumba', 0) > 0:
            tas_lumba = catat['barang']
            tas_lumba['lumba'] -= 1
            if not tas_lumba['lumba']:
                del tas_lumba['lumba']
            bayar_lumba = sisa_kemarin
            lumba_kepakai = True
        catat.pop('misi_kemarin', None)
        daftar, misi = misi_orang(inter.user.id)
        maju, diambil = (misi['maju'], misi['diambil'])
        baris, baru_kelar, total_xp = ([], [], 0)
        for m in daftar:
            punya = maju.get(m['kode'], 0)
            kelar = punya >= m['target']
            if kelar and m['kode'] not in diambil:
                diambil.append(m['kode'])
                baru_kelar.append(m)
                total_xp += m['xp']
            tanda = '✅' if kelar else '⬜'
            baris.append(f"{tanda}  {m['kalimat']}  ·  `{min(punya, m['target'])}/{m['target']}`  ·  +{m['xp']} XP")
        semua_kelar = all((maju.get(m['kode'], 0) >= m['target'] for m in daftar))
        bonus = 0
        if semua_kelar and 'BONUS' not in diambil:
            diambil.append('BONUS')
            bonus = MISI_BONUS
            total_xp += bonus
            catat_minggu(inter.user.id)
        total_xp += bayar_lumba
        if total_xp:
            await core.tambah_xp(self.bot, inter.user, total_xp)
        minggu = catat.setdefault('misi_minggu', {})
        kunci_minggu = datetime.now(WIB).strftime('%Y-W%W')
        if minggu.get('minggu') != kunci_minggu:
            minggu.clear()
            minggu.update({'minggu': kunci_minggu, 'hari': [], 'diambil': False})
        kelar_minggu = len(minggu.get('hari', []))
        hadiah_minggu = 0
        if kelar_minggu >= MISI_MINGGU_HARI and (not minggu.get('diambil')):
            minggu['diambil'] = True
            hadiah_minggu = MISI_MINGGU_XP
            await core.tambah_xp(self.bot, inter.user, hadiah_minggu)
        core.simpan()
        isi = discord.Embed(color=WARNA, title='🏝️  Misi Hari Ini', description='\n'.join(baris))
        catatan_bawah = []
        if baru_kelar:
            catatan_bawah.append('Baru kelar: ' + ', '.join((m['kalimat'] for m in baru_kelar)))
        if bonus:
            catatan_bawah.append(f'🎁 **Ketiganya kelar!** Bonus {bonus} XP.')
        if lumba_kepakai:
            catatan_bawah.append(f'🐬 **Sahabat Lumba lumba kepakai.** Misi kemarin yang ga kelar dibayar setengah: +{bayar_lumba} XP.')
        if catatan_bawah:
            isi.add_field(name=f'Masuk +{total_xp:,} XP', value='\n'.join(catatan_bawah), inline=False)
        elif semua_kelar:
            isi.add_field(name='Udah kelar semua', value='Hadiahnya udah lo ambil. Besok ganti misi baru.', inline=False)
        bar = '▰' * min(kelar_minggu, MISI_MINGGU_HARI) + '▱' * max(0, MISI_MINGGU_HARI - kelar_minggu)
        if hadiah_minggu:
            nilai = f'`{bar}` **KELAR!**\n🏆 Misi mingguan tuntas, +{hadiah_minggu:,} XP.'
        elif minggu.get('diambil'):
            nilai = f'`{bar}` udah diambil minggu ini. Reset Senin.'
        else:
            nilai = f'`{bar}` {kelar_minggu} dari {MISI_MINGGU_HARI} hari\nKelarin misi harian {MISI_MINGGU_HARI} hari dalam seminggu buat dapet {MISI_MINGGU_XP:,} XP.'
        isi.add_field(name='🗓️ Misi Mingguan', value=nilai, inline=False)
        punya_kerang = catat.get('barang', {}).get('kerang', 0)
        isi.set_footer(text='Cuma lo yang liat pesan ini. ' + (f'Punya {punya_kerang} Kerang Ajaib, pakai /gantimisi kalau ada misi yang ga sreg.' if punya_kerang else 'Misi ganti tiap jam 00.00 WIB.'))
        await inter.response.send_message(embed=isi, ephemeral=True)

    @app_commands.command(name='gantimisi', description='Tukar satu misi pakai Kerang Ajaib')
    @app_commands.describe(nomor='Misi ke berapa yang mau diganti, 1 sampai 3')
    async def cmd_gantimisi(self, inter: discord.Interaction, nomor: int):
        if not await core.di_arena(inter):
            return
        daftar, misi = misi_orang(inter.user.id)
        if not 1 <= nomor <= len(daftar):
            await inter.response.send_message(f'Nomornya 1 sampai {len(daftar)} aja.', ephemeral=True)
            return
        catat = core.catatan(inter.user.id)
        tas = catat.setdefault('barang', {})
        if tas.get('kerang', 0) < 1:
            await inter.response.send_message('Lo ga punya Kerang Ajaib. Beli dulu di `/toko`.', ephemeral=True)
            return
        lama_misi = daftar[nomor - 1]
        if misi['maju'].get(lama_misi['kode'], 0) >= lama_misi['target']:
            await inter.response.send_message('Misi itu udah kelar, ngapain diganti.', ephemeral=True)
            return
        kepakai = {m['kode'] for m in daftar}
        sisa = [x for x in MISI_PILIHAN if x[0] not in kepakai]
        if not sisa:
            await inter.response.send_message('Ga ada misi lain yang bisa dituker.', ephemeral=True)
            return
        kode, kalimat, target, xp = random.choice(sisa)
        misi['ganti'][str(nomor - 1)] = {'kode': kode, 'kalimat': kalimat.format(n=target), 'target': target, 'xp': xp}
        tas['kerang'] -= 1
        if not tas['kerang']:
            del tas['kerang']
        core.simpan()
        await inter.response.send_message(f"🐚 Kerang Ajaib kepakai.\n~~{lama_misi['kalimat']}~~\n→ **{kalimat.format(n=target)}**  ·  +{xp} XP", ephemeral=True)

    @app_commands.command(name='quest', description='Quest bareng satu server minggu ini')
    async def cmd_quest(self, inter: discord.Interaction):
        if not await core.di_arena(inter):
            return
        isi = embed_quest()
        if isi is None:
            await inter.response.send_message('Quest komunal lagi dimatiin.', ephemeral=True)
            return
        quest = core.data.get('quest') or {}
        punya = quest.get('penyumbang', {}).get(str(inter.user.id), 0)
        isi.set_footer(text=f'Sumbangan lo: {punya}' if punya else 'Lo belum nyumbang apa apa minggu ini')
        await inter.response.send_message(embed=isi)

async def setup(bot):
    await bot.add_cog(Misi(bot))