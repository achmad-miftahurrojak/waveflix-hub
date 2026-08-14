import asyncio
from datetime import datetime, timedelta
import discord
from discord import app_commands
from discord.ext import commands, tasks
import random
import os
from . import core
from .config import *

def lencana_punya(catat):
    hasil = []
    for kode, emoji, nama, kalimat, kunci, target in LENCANA:
        if catat.get(kunci, 0) >= target:
            hasil.append((kode, emoji, nama, kalimat))
    if core.koleksi_lengkap(catat):
        hasil.append(('kolektor', '🐚', 'Kolektor Pantai', 'Nemu semua jenis tangkapan'))
    if len(catat.get('kasih_ke', [])) >= 5:
        hasil.append(('dermawan', '🤝', 'Dermawan', 'Kasih XP ke 5 orang berbeda'))
    return hasil

def lencana_berikutnya(catat):
    dekat = None
    for kode, emoji, nama, kalimat, kunci, target in LENCANA:
        punya = catat.get(kunci, 0)
        if punya >= target:
            continue
        rasio = punya / target
        if dekat is None or rasio > dekat[0]:
            dekat = (rasio, emoji, nama, punya, target, kalimat)
    return dekat

def _kata_sah(teks):
    bersih = teks.strip().lower()
    if len(bersih) < RANTAI_HURUF_MIN:
        return None
    if not bersih.isalpha():
        return None
    return bersih

def season_sekarang():
    musim = core.data.setdefault('season', {})
    musim.setdefault('nomor', 1)
    musim.setdefault('mulai', core.hari_ini())
    musim.setdefault('hall', [])
    return musim

def season_berakhir(musim):
    try:
        mulai = datetime.strptime(musim['mulai'], '%Y-%m-%d').replace(tzinfo=WIB)
    except (ValueError, KeyError):
        return None
    return mulai + timedelta(days=SEASON_BULAN * 30)

async def tutup_season(bot, channel):
    musim = season_sekarang()
    guild = channel.guild
    hidup = [(uid, c) for uid, c in core.data['orang'].items() if c.get('xp', 0) > 0 and core.masih_anggota(guild, uid)]
    hidup.sort(key=lambda x: x[1]['xp'], reverse=True)
    atas = hidup[:SEASON_JUARA_DISIMPAN]
    papan = []
    for i, (uid, catat) in enumerate(atas, 1):
        anggota = guild.get_member(int(uid))
        papan.append({'peringkat': i, 'id': int(uid), 'nama': anggota.display_name if anggota else 'entah siapa', 'xp': catat['xp'], 'tingkat': core.tingkat(catat['xp'])[1]})
    musim.setdefault('hall', []).append({'musim': musim['nomor'], 'mulai': musim['mulai'], 'selesai': core.hari_ini(), 'juara': papan})
    for catat in core.data['orang'].values():
        catat['xp_abadi'] = catat.get('xp_abadi', 0) + catat.get('xp', 0)
        catat['xp'] = int(catat.get('xp', 0) * SEASON_SISA_PERSEN)
        catat['xp_minggu'] = 0
    musim['nomor'] += 1
    musim['mulai'] = core.hari_ini()
    core.simpan()
    semua_role = {t[2] for t in TINGKAT if t[2]}
    dirapiin = 0
    for uid, catat in core.data['orang'].items():
        anggota = guild.get_member(int(uid))
        if anggota is None:
            continue
        benar = core.tingkat(catat['xp'])[2]
        salah = [r for r in anggota.roles if r.name in semua_role and r.name != benar]
        if not salah:
            continue
        try:
            await anggota.remove_roles(*salah, reason='Musim baru')
            dirapiin += 1
        except (discord.Forbidden, discord.HTTPException):
            pass
    baris = '\n'.join((f"{['🥇', '🥈', '🥉', '4️⃣', '5️⃣'][j['peringkat'] - 1]} **{j['nama']}** — {j['xp']:,} XP · {j['tingkat']}" for j in papan)) or '_ga ada yang nyemplung musim ini_'
    isi = discord.Embed(color=WARNA, title=f"🌅  Musim {musim['nomor'] - 1} selesai", description=f'Papan ditutup. Ini yang paling dalam musim kemarin:\n\n{baris}')
    isi.add_field(name=f"Musim {musim['nomor']} dimulai", value=f'XP semua orang dipangkas jadi **{int(SEASON_SISA_PERSEN * 100)}%**. Total XP sepanjang masa lo tetep kesimpen dan bisa diliat di `/profil`.\nPapan sekarang rata lagi. Yang baru gabung punya kesempatan.', inline=False)
    if dirapiin:
        isi.set_footer(text=f'Role tingkatan {dirapiin} orang dirapiin')
    await channel.send(embed=isi)

async def buka_kapsul(bot, daftar):
    tujuan = CHANNEL_REKAP or CHANNEL_ARENA
    channel = bot.get_channel(tujuan) if tujuan else None
    if channel is None:
        return
    await channel.send(f'⛵ **Kapal Waktu sandar.** Ada **{len(daftar)} pesan** yang dititipin buat hari ini. Yang nulis mungkin udah lupa.')
    for k in daftar:
        anggota = channel.guild.get_member(k['orang'])
        nama = anggota.display_name if anggota else 'entah siapa'
        jarak = '?'
        try:
            d1 = datetime.strptime(k['ditulis'], '%Y-%m-%d')
            d2 = datetime.strptime(k['buka'], '%Y-%m-%d')
            jarak = (d2 - d1).days
        except (ValueError, KeyError):
            pass
        isi = discord.Embed(color=WARNA, title=f'Dari {nama}, {jarak} hari yang lalu', description=k['isi'])
        isi.set_footer(text=f"Ditulis {k['ditulis']}")
        if anggota:
            isi.set_thumbnail(url=anggota.display_avatar.url)
        await channel.send(embed=isi)
        await asyncio.sleep(2)

def _boleh_atur(inter):
    if not ROLE_SOAL:
        return True
    punya = {r.name for r in getattr(inter.user, 'roles', [])}
    return bool(punya & set(ROLE_SOAL))

class Sosial(commands.Cog):

    def __init__(self, bot):
        self.bot = bot
        self.jaga_season.start()
        self.jaga_kapsul.start()

    def cog_unload(self):
        self.jaga_season.cancel()
        self.jaga_kapsul.cancel()

    async def cek_rantai(self, pesan):
        if not CHANNEL_RANTAI or pesan.channel.id != CHANNEL_RANTAI:
            return False
        kata = _kata_sah(pesan.content)
        if kata is None:
            return False
        rantai = core.data.setdefault('rantai', {'terakhir': '', 'dipakai': [], 'panjang': 0, 'rekor': 0, 'orang_terakhir': 0})
        if rantai['panjang'] and pesan.author.id == rantai.get('orang_terakhir'):
            await pesan.add_reaction('🙅')
            return True
        if rantai['terakhir']:
            harus = rantai['terakhir'][-1]
            if kata[0] != harus:
                await pesan.add_reaction('❌')
                await pesan.channel.send(f"Putus di **{rantai['panjang']}** kata. Harusnya diawali huruf **{harus.upper()}**, bukan **{kata[0].upper()}**.\nMulai lagi dari kata bebas." + (f"\nRekor masih **{rantai['rekor']}** kata." if rantai['rekor'] else ''))
                rantai.update({'terakhir': '', 'dipakai': [], 'panjang': 0, 'orang_terakhir': 0})
                core.simpan()
                return True
        if kata in rantai['dipakai']:
            await pesan.add_reaction('♻️')
            await pesan.channel.send(f"**{kata}** udah kepakai di rantai ini. Putus di **{rantai['panjang']}** kata, mulai lagi.")
            rantai.update({'terakhir': '', 'dipakai': [], 'panjang': 0, 'orang_terakhir': 0})
            core.simpan()
            return True
        rantai['terakhir'] = kata
        rantai['dipakai'].append(kata)
        rantai['panjang'] += 1
        rantai['orang_terakhir'] = pesan.author.id
        hadiah = RANTAI_XP
        rekor_baru = rantai['panjang'] > rantai.get('rekor', 0)
        if rekor_baru:
            rantai['rekor'] = rantai['panjang']
            hadiah += RANTAI_XP_REKOR
        core.catat_main(pesan.author.id, 'rantai', menang=True)
        await core.tambah_xp(self.bot, pesan.author, hadiah)
        await pesan.add_reaction('✅')
        if rekor_baru and rantai['panjang'] > 5:
            await pesan.channel.send(f"🔥 **REKOR BARU: {rantai['panjang']} kata!** {pesan.author.display_name} dapet bonus **{RANTAI_XP_REKOR} XP**. Lanjut, jangan putus.")
        elif rantai['panjang'] % 10 == 0:
            await pesan.channel.send(f"**{rantai['panjang']} kata** tanpa putus. Rekornya {rantai['rekor']}. Hati hati.")
        core.simpan()
        return True

    @app_commands.command(name='rantai', description='Liat status rantai kata sekarang')
    async def cmd_rantai(self, inter: discord.Interaction):
        if not await core.di_arena(inter):
            return
        rantai = core.data.get('rantai') or {}
        panjang = rantai.get('panjang', 0)
        rekor = rantai.get('rekor', 0)
        terakhir = rantai.get('terakhir') or '(belum mulai)'
        uid = rantai.get('orang_terakhir')
        orang = None
        if uid and inter.guild:
            orang = inter.guild.get_member(int(uid))
        isi = discord.Embed(color=WARNA, title='🔠  Rantai Kata')
        if panjang:
            lanjut = terakhir[-1].upper() if terakhir and terakhir != '(belum mulai)' else '?'
            isi.description = f'Sekarang **{panjang}** kata tanpa putus.\nKata terakhir: **{terakhir}** → lanjut dari **{lanjut}**…'
        else:
            isi.description = 'Rantai lagi kosong. Ketik satu kata bebas di ' + (f'<#{CHANNEL_RANTAI}>' if CHANNEL_RANTAI else 'channel rantai') + ' buat mulai.'
        isi.add_field(name='Rekor', value=f'**{rekor}** kata', inline=True)
        if orang:
            isi.add_field(name='Terakhir nulis', value=orang.display_name, inline=True)
        isi.set_footer(text=f'+{RANTAI_XP} XP per kata · bonus rekor +{RANTAI_XP_REKOR} XP')
        await inter.response.send_message(embed=isi, ephemeral=True)

    @tasks.loop(hours=1)
    async def jaga_season(self):
        try:
            if SEASON_AKTIF:
                musim = season_sekarang()
                habis = season_berakhir(musim)
                if habis and datetime.now(WIB) >= habis:
                    tujuan = CHANNEL_REKAP or CHANNEL_ARENA
                    channel = self.bot.get_channel(tujuan) if tujuan else None
                    if channel:
                        await tutup_season(self.bot, channel)
        except Exception as e:
            print(f'[arka] season error: {e}')

    @jaga_season.before_loop
    async def before_jaga_season(self):
        await self.bot.wait_until_ready()

    @tasks.loop(minutes=30)
    async def jaga_kapsul(self):
        try:
            if KAPSUL_AKTIF:
                sekarang = datetime.now(WIB)
                hari = sekarang.strftime('%Y-%m-%d')
                penanda = core.data.get('kapsul_terakhir')
                if sekarang.hour >= KAPSUL_JAM_BUKA and penanda != hari:
                    jatuh = [k for k in core.data.get('kapsul', []) if not k.get('dibuka') and k['buka'] <= hari]
                    core.data['kapsul_terakhir'] = hari
                    for k in jatuh:
                        k['dibuka'] = True
                    core.simpan()
                    if jatuh:
                        await buka_kapsul(self.bot, jatuh)
        except Exception as e:
            print(f'[arka] kapsul error: {e}')

    @jaga_kapsul.before_loop
    async def before_jaga_kapsul(self):
        await self.bot.wait_until_ready()

    @app_commands.command(name='profil', description='Semua data lo dalam satu tampilan')
    @app_commands.describe(orang='Kosongin kalau mau liat punya sendiri')
    async def cmd_profil(self, inter: discord.Interaction, orang: discord.Member=None):
        if not await core.di_arena(inter):
            return
        orang = orang or inter.user
        catat = core.catatan(orang.id)
        xp = catat['xp']
        depan = core.berikutnya(xp)
        isi = discord.Embed(color=WARNA, title=f'🌊  {orang.display_name} — {core.tingkat(xp)[1]}', description=f"**{xp:,} XP** sepanjang masa · **{catat['xp_minggu']:,} XP** minggu ini")
        isi.set_thumbnail(url=orang.display_avatar.url)
        if depan:
            isi.add_field(name=f'Menuju {depan[1]}', value=f'`{core.batang(xp)}`  kurang {depan[0] - xp:,} XP', inline=False)
        isi.add_field(name='Aktivitas', value=f"{catat['pesan']:,} pesan\n{catat['menit']:,} menit voice", inline=True)
        isi.add_field(name='Absen', value=f"{catat['absen']} hari beruntun\nrekor {catat['absen_panjang']} hari", inline=True)
        isi.add_field(name='Tanding', value=f"{catat['menang']} menang\n{catat['kalah']} kalah", inline=True)
        misi_cog = self.bot.get_cog('Misi')
        if misi_cog:
            import sys
            misi_hari_ini = sys.modules[misi_cog.__module__].misi_hari_ini
            misi_orang = sys.modules[misi_cog.__module__].misi_orang
            daftar_misi, catat_misi = misi_orang(orang.id)
            baris_misi = []
            for m in daftar_misi:
                maju = catat_misi['maju'].get(m['kode'], 0)
                kelar = maju >= m['target']
                diambil = m['kode'] in catat_misi.get('diambil', [])
                tanda = '💰' if diambil else '✅' if kelar else '⬜'
                baris_misi.append(f"{tanda} {m['kalimat'].format(n=m['target'])} ({min(maju, m['target'])}/{m['target']})")
            if baris_misi:
                isi.add_field(name='Misi hari ini', value='\n'.join(baris_misi), inline=False)
        tas = catat.get('barang', {})
        if tas:
            rincian = []
            toko_cog = self.bot.get_cog('Toko')
            if toko_cog:
                import sys
                cari_barang = sys.modules[toko_cog.__module__].cari_barang
                for kode, banyak in tas.items():
                    butir = cari_barang(kode)
                    if butir:
                        rincian.append(f'{butir[1]} {butir[2]} ×{banyak}')
            if rincian:
                isi.add_field(name='Tas', value='\n'.join(rincian), inline=True)
        punya_koleksi = catat.get('koleksi', {})
        if punya_koleksi:
            isi.add_field(name='Koleksi', value=f'{len(punya_koleksi)}/{len(TANGKAPAN)} jenis' + ('  ✨ lengkap' if core.koleksi_lengkap(catat) else ''), inline=True)
        if catat.get('juara'):
            isi.add_field(name='Gelar juara', value=f"🏆 {catat['juara']}x", inline=True)
        lencana = lencana_punya(catat)
        if lencana:
            isi.add_field(name=f'Lencana ({len(lencana)})', value='  '.join((f'{e} {n}' for _k, e, n, _ket in lencana)), inline=False)
        dekat = lencana_berikutnya(catat)
        if dekat:
            _rasio, emoji, nama, punya, target, kalimat = dekat
            isi.add_field(name='Lencana terdekat', value=f'{emoji} **{nama}** — {kalimat}  ({punya}/{target})', inline=False)
        isi.set_footer(text='Semua data lo dalam satu tempat. Rincian per bagian ada di /level /misi /tas /koleksi')
        await inter.response.send_message(embed=isi)

    @app_commands.command(name='absenbulan', description='Rekap absen lo bulan ini')
    async def cmd_absenbulan(self, inter: discord.Interaction):
        if not await core.di_arena(inter):
            return
        catat = core.catatan(inter.user.id)
        sekarang = datetime.now(WIB)
        bulan_ini = sekarang.strftime('%Y-%m')
        bulan_lalu = (sekarang.replace(day=1) - timedelta(days=1)).strftime('%Y-%m')
        riwayat_absen = catat.get('absen_riwayat', [])
        ini = [t for t in riwayat_absen if t.startswith(bulan_ini)]
        lalu = [t for t in riwayat_absen if t.startswith(bulan_lalu)]
        if not riwayat_absen:
            await inter.response.send_message('Rekap bulanan baru mulai ngitung dari absen lo berikutnya. Runtutan lo yang sekarang tetep aman kok.', ephemeral=True)
            return
        lewat = sekarang.day
        persen = len(ini) / lewat * 100 if lewat else 0
        isi = discord.Embed(color=WARNA, title=f"📅  Absen {sekarang.strftime('%B %Y')}", description=f'**{len(ini)} dari {lewat} hari** ({persen:.0f}%)')
        isi.set_thumbnail(url=inter.user.display_avatar.url)
        tanda = []
        for h in range(1, lewat + 1):
            tgl = sekarang.replace(day=h).strftime('%Y-%m-%d')
            tanda.append('🟦' if tgl in ini else '⬜')
        isi.add_field(name='Sebulan ini', value=''.join(tanda[:31]), inline=False)
        if lalu:
            selisih = len(ini) - len(lalu)
            arah = 'lebih rajin' if selisih > 0 else 'lebih males' if selisih < 0 else 'sama aja'
            isi.add_field(name='Bulan lalu', value=f'{len(lalu)} hari — bulan ini {arah} ({selisih:+d} hari)', inline=True)
        isi.add_field(name='Runtutan sekarang', value=f"{catat['absen']} hari\nrekor {catat['absen_panjang']}", inline=True)
        await inter.response.send_message(embed=isi)

    @app_commands.command(name='kapsul', description='Titip pesan buat dibuka di masa depan')
    @app_commands.describe(pesan='Yang mau lo titipin', hari='Dibuka berapa hari lagi')
    async def cmd_kapsul(self, inter: discord.Interaction, pesan: str, hari: int=30):
        if not await core.di_arena(inter):
            return
        if not KAPSUL_AKTIF:
            await inter.response.send_message('Fitur ini lagi dimatiin.', ephemeral=True)
            return
        if not KAPSUL_HARI_MIN <= hari <= KAPSUL_HARI_MAKS:
            await inter.response.send_message(f'Antara {KAPSUL_HARI_MIN} sampai {KAPSUL_HARI_MAKS} hari.', ephemeral=True)
            return
        if len(pesan) > 900:
            await inter.response.send_message('Kepanjangan, maksimal 900 huruf.', ephemeral=True)
            return
        semua = core.data.setdefault('kapsul', [])
        punya = [k for k in semua if k['orang'] == inter.user.id and (not k.get('dibuka'))]
        if len(punya) >= KAPSUL_MAKS_PER_ORANG:
            await inter.response.send_message(f'Lo udah punya **{len(punya)} kapsul** yang belum kebuka. Tunggu itu dulu.', ephemeral=True)
            return
        buka_tanggal = (datetime.now(WIB) + timedelta(days=hari)).strftime('%Y-%m-%d')
        semua.append({'orang': inter.user.id, 'isi': pesan, 'ditulis': core.hari_ini(), 'buka': buka_tanggal, 'dibuka': False})
        core.simpan()
        await inter.response.send_message(f'⛵ Kapsul lo gua kunci. Dibuka **{buka_tanggal}** ({hari} hari lagi), dipost rame rame di <#{CHANNEL_REKAP or CHANNEL_ARENA}>.\nIsinya ga ada yang bisa baca sampai hari itu, termasuk lo sendiri.', ephemeral=True)

    @app_commands.command(name='kasih', description='Kasih sebagian XP lo ke orang lain')
    @app_commands.describe(orang='Mau dikasih ke siapa', jumlah='Berapa XP')
    async def cmd_kasih(self, inter: discord.Interaction, orang: discord.Member, jumlah: int):
        if not await core.di_arena(inter):
            return
        if not KASIH_AKTIF:
            await inter.response.send_message('Fitur ini lagi dimatiin.', ephemeral=True)
            return
        if orang.bot:
            await inter.response.send_message('Bot ga butuh XP.', ephemeral=True)
            return
        if orang.id == inter.user.id:
            await inter.response.send_message('Ga bisa kasih ke diri sendiri. Ya iyalah.', ephemeral=True)
            return
        if jumlah < KASIH_MIN:
            await inter.response.send_message(f'Minimal **{KASIH_MIN} XP** sekali kasih.', ephemeral=True)
            return
        catat = core.catatan(inter.user.id)
        hari = core.hari_ini()
        jatah = catat.setdefault('kasih_harian', {'hari': hari, 'jumlah': 0})
        if jatah.get('hari') != hari:
            jatah['hari'] = hari
            jatah['jumlah'] = 0
        sisa_jatah = KASIH_MAKS_HARIAN - jatah['jumlah']
        if jumlah > sisa_jatah:
            await inter.response.send_message(f'Sisa jatah lo hari ini **{sisa_jatah:,} XP**. Batasnya {KASIH_MAKS_HARIAN:,} per hari, biar ga dipakai muter muterin XP antar dua akun.', ephemeral=True)
            return
        if catat['xp'] < jumlah:
            await inter.response.send_message(f"XP lo cuma **{catat['xp']:,}**.", ephemeral=True)
            return
        nyampe = max(1, int(jumlah * (1 - KASIH_POTONGAN)))
        nguap = jumlah - nyampe
        catat['xp'] -= jumlah
        jatah['jumlah'] += jumlah
        penerima = catat.setdefault('kasih_ke', [])
        if orang.id not in penerima:
            penerima.append(orang.id)
        tujuan = core.catatan(orang.id)
        tujuan['xp'] += nyampe
        tujuan['xp_minggu'] += nyampe
        core.simpan()
        await inter.response.send_message(f"🤝 {inter.user.mention} ngasih **{nyampe:,} XP** ke {orang.mention}.\nKepotong {nguap:,} XP di jalan. Sisa jatah hari ini: **{KASIH_MAKS_HARIAN - jatah['jumlah']:,} XP**.")
        try:
            await orang.send(f'{inter.user.display_name} baru ngasih lo **{nyampe:,} XP** di {inter.guild.name}. Lumayan.')
        except discord.Forbidden:
            pass

    @app_commands.command(name='pet', description=f'Liat kabar {PET_NAMA}, peliharaan server')
    async def cmd_pet(self, inter: discord.Interaction):
        if not await core.di_arena(inter):
            return
        if not PET_AKTIF:
            await inter.response.send_message('Fitur peliharaan lagi dimatiin.', ephemeral=True)
            return
        pet = core.kondisi_pet()
        kenyang = pet['kenyang']
        nama_suasana, emoji, kali = ('mati', '', 1.0)
        for batas, n, e, k in PET_SUASANA:
            if kenyang >= batas:
                nama_suasana, emoji, kali = (n, e, k)
                break
        core.simpan()
        panjang = 14
        isi_batang = max(0, min(panjang, round(kenyang / 100 * panjang)))
        batang = '▰' * isi_batang + '▱' * (panjang - isi_batang)
        isi = discord.Embed(color=WARNA, title=f"{emoji}  {pet['nama']} lagi {nama_suasana}", description=f'`{batang}`  {int(kenyang)}/100')
        isi.add_field(name='Pengali XP buat SEMUA orang', value=f'**x{kali}**', inline=True)
        isi.add_field(name='Udah dikasih makan', value=f"{pet.get('total_makan', 0)} kali", inline=True)
        isi.add_field(name='Cara ngerawat', value=f'Kasih makan pakai `/kasihmakan` — **{PET_HARGA_MAKAN} XP** jadi **{PET_ISI_MAKAN} kenyang**.\nDia juga kenyang sendiri kalau server rame ngobrol.\nKenyangnya turun **{PET_LAPAR_PER_JAM} per jam**, jadi jangan ditinggal lama lama.', inline=False)
        isi.set_footer(text='Dia punya server, bukan punya satu orang. Kalau dia seneng, XP semua orang ikut naik.')
        await inter.response.send_message(embed=isi)

    @app_commands.command(name='kasihmakan', description=f'Kasih makan {PET_NAMA} pakai XP lo')
    async def cmd_kasihmakan(self, inter: discord.Interaction):
        if not await core.di_arena(inter):
            return
        if not PET_AKTIF:
            await inter.response.send_message('Fitur peliharaan lagi dimatiin.', ephemeral=True)
            return
        catat = core.catatan(inter.user.id)
        hari = core.hari_ini()
        jatah = catat.setdefault('pet_makan', {'hari': hari, 'kali': 0})
        if jatah.get('hari') != hari:
            jatah['hari'] = hari
            jatah['kali'] = 0
        if jatah['kali'] >= PET_MAKAN_MAKS_HARIAN:
            await inter.response.send_message(f'Lo udah kasih makan **{PET_MAKAN_MAKS_HARIAN} kali** hari ini. Biar yang lain kebagian ikut ngerawat juga.', ephemeral=True)
            return
        if catat['xp'] < PET_HARGA_MAKAN:
            await inter.response.send_message(f"XP lo kurang. Butuh **{PET_HARGA_MAKAN}**, punya lo **{catat['xp']:,}**.", ephemeral=True)
            return
        pet = core.kondisi_pet()
        if pet['kenyang'] >= 100:
            await inter.response.send_message(f"{pet['nama']} udah kenyang banget, ga mau makan lagi. Simpen XP lo.", ephemeral=True)
            return
        catat['xp'] -= PET_HARGA_MAKAN
        jatah['kali'] += 1
        pet['kenyang'] = min(100.0, pet['kenyang'] + PET_ISI_MAKAN)
        pet['total_makan'] = pet.get('total_makan', 0) + 1
        kenyang = pet['kenyang']
        nama_suasana, emoji, kali = ('mati', '', 1.0)
        for batas, n, e, k in PET_SUASANA:
            if kenyang >= batas:
                nama_suasana, emoji, kali = (n, e, k)
                break
        core.simpan()
        panjang = 14
        isi_batang = max(0, min(panjang, round(kenyang / 100 * panjang)))
        batang = '▰' * isi_batang + '▱' * (panjang - isi_batang)
        kata = f"{emoji} Lo kasih makan **{pet['nama']}**. Kenyangnya sekarang **{int(kenyang)}/100**, dia lagi **{nama_suasana}**.\n`{batang}`\nPengali XP buat semua orang: **x{kali}**"
        sisa = PET_MAKAN_MAKS_HARIAN - jatah['kali']
        if sisa:
            kata += f'\nSisa jatah lo hari ini: {sisa} kali.'
        await inter.response.send_message(kata)

    @app_commands.command(name='hall', description='Hall of Fame juara musim musim lalu')
    async def cmd_hall(self, inter: discord.Interaction):
        if not await core.di_arena(inter):
            return
        musim = season_sekarang()
        hall = musim.get('hall', [])
        isi = discord.Embed(color=WARNA, title='🏛️  Hall of Fame')
        habis = season_berakhir(musim)
        isi.description = f"**Musim {musim['nomor']}** lagi jalan sejak {musim['mulai']}." + (f'\nTutup <t:{int(habis.timestamp())}:R>.' if habis else '')
        if not hall:
            isi.add_field(name='Belum ada musim yang selesai', value='Musim pertama masih jalan. Yang masuk lima besar pas musim ini tutup bakal diabadiin di sini selamanya.', inline=False)
        else:
            for catat in reversed(hall[-5:]):
                baris = '\n'.join((f"`{j['peringkat']}.` **{j['nama']}** — {j['xp']:,} XP" for j in catat['juara'])) or '_kosong_'
                isi.add_field(name=f"Musim {catat['musim']} · {catat['mulai']} → {catat['selesai']}", value=baris, inline=False)
        await inter.response.send_message(embed=isi)

    @app_commands.command(name='aturxp', description='Betulin XP member (khusus admin)')
    @app_commands.describe(orang='Siapa yang mau dibetulin', aksi='Mau diapain', jumlah='Berapa XP (ga kepakai kalau reset)')
    @app_commands.choices(aksi=[app_commands.Choice(name='tambah', value='tambah'), app_commands.Choice(name='kurangi', value='kurangi'), app_commands.Choice(name='set ke angka tertentu', value='set'), app_commands.Choice(name='reset semua data dia', value='reset')])
    async def cmd_aturxp(self, inter: discord.Interaction, orang: discord.Member, aksi: app_commands.Choice[str], jumlah: int=0):
        if not _boleh_atur(inter):
            await inter.response.send_message('Perintah ini cuma buat ' + ' atau '.join(ROLE_SOAL) + '.', ephemeral=True)
            return
        catat = core.catatan(orang.id)
        lama = catat['xp']
        if aksi.value == 'reset':
            core.data['orang'].pop(str(orang.id), None)
            core.simpan()
            await inter.response.send_message(f'⚠️ Semua data **{orang.display_name}** dihapus. XP lamanya {lama:,}. Ini ga bisa dibatalin.')
            return
        if jumlah < 0:
            await inter.response.send_message('Jumlahnya jangan minus.', ephemeral=True)
            return
        if aksi.value == 'tambah':
            catat['xp'] += jumlah
        elif aksi.value == 'kurangi':
            catat['xp'] = max(0, catat['xp'] - jumlah)
        else:
            catat['xp'] = jumlah
        core.simpan()
        await inter.response.send_message(f"XP **{orang.display_name}**: {lama:,} → **{catat['xp']:,}** ({aksi.value} {jumlah:,}).\nTingkatan sekarang: **{core.tingkat(catat['xp'])[1]}**")

    @app_commands.command(name='bantuan', description='Arka bisa apa aja?')
    async def cmd_bantuan(self, inter: discord.Interaction):
        if not await core.di_arena(inter):
            return
        isi = discord.Embed(color=WARNA, title='🌊  Arka bisa apa aja', description='XP dapet otomatis dari ngobrol di channel mana pun dan dari nongkrong bareng di voice.')
        isi.add_field(name='Data diri', value='`/profil` `/level` `/papan` `/rekap` `/statistik` `/cuaca` `/absen` `/absenbulan` `/rantai`', inline=False)
        isi.add_field(name='Main main', value='`/tebak` `/suit` `/duel` `/balon` `/trivia` `/hunt` `/koleksi` `/lagu` `/akinator` `/turnamen`', inline=False)
        isi.add_field(name='Misi & toko', value='`/misi` `/quest` `/gantimisi` `/toko` `/beli` `/tas` `/pakai` `/riwayat` `/wishlist`', inline=False)
        isi.add_field(name='Bagi bagi', value='`/kasih` — bagi XP ke orang lain.\n`/kasihitem` — oper barang dari tas lo. Anak baru butuh kail, lo punya lebih, ya kasih aja.\n`/wishlist` — incar barang, gua DM kalau dia masuk stok Senin.', inline=False)
        isi.add_field(name='Musim', value=f'Papan direset tiap {SEASON_BULAN} bulan, lima teratas diabadiin di `/hall`. XP lo ga hangus, cuma dipangkas biar yang baru gabung punya kesempatan.', inline=False)
        isi.add_field(name=f'Bareng bareng', value=f'`/pet` `/kasihmakan` — {PET_NAMA} peliharaan server. Kalau dia seneng, XP **semua orang** naik. Kalau kelaparan, semua ikut turun.\n`/kasih` — bagi XP lo ke orang lain.\n`/kapsul` — titip pesan buat dibuka di masa depan.', inline=False)
        isi.add_field(name='Acara & lomba', value='`/acara` `/acaralist` `/acaraedit` `/acarahapus` `/lomba` `/lombatutup`', inline=False)
        isi.add_field(name='Bantuan channel', value='`/panduan` — cara main di channel tempat lo ngetik.\n`/backup` — cadangan data (Island Owner).', inline=False)
        isi.add_field(name='Trivia', value='Jawabannya diketik langsung di chat, ga usah pakai perintah. Jawab sebelum petunjuk keluar dapet XP lebih.', inline=False)
        isi.add_field(name='Cuaca', value=f'Tiap hari beda, dan ngubah banyak XP yang masuk. Diumumin jam {JAM_CUACA} pagi.', inline=False)
        isi.add_field(name='Rekap mingguan', value='Tiap Senin pagi gua umumin tiga teratas minggu itu, terus hitungannya gua reset. XP sepanjang masa aman, yang direset cuma papan mingguan.', inline=False)
        await inter.response.send_message(embed=isi, ephemeral=True)

async def setup(bot):
    await bot.add_cog(Sosial(bot))