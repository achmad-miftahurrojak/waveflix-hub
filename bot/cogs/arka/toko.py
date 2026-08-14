import asyncio
import random
from datetime import datetime, timedelta
import discord
from discord import app_commands
from discord.ext import commands, tasks
from . import core
from .config import *

def cari_barang(kode):
    for b in BARANG:
        if b[0] == kode:
            return b
    return None

def stok_toko():
    kunci = datetime.now(WIB).strftime('%Y-W%W')
    if core.data.get('toko', {}).get('minggu') != kunci:
        pilihan = []
        for kelompok, (isi, berapa) in TOKO_KELOMPOK.items():
            pilihan += random.sample(isi, min(berapa, len(isi)))
        for kode in TOKO_SUPER:
            if random.random() < TOKO_SUPER_PELUANG:
                pilihan.append(kode)
        urut = {b[0]: b[3] for b in BARANG}
        pilihan.sort(key=lambda k: urut.get(k, 0))
        core.data['toko'] = {'minggu': kunci, 'stok': pilihan, 'diumumkan': False}
        core.simpan()
    return core.data['toko']['stok']

def senin_depan():
    sekarang = datetime.now(WIB)
    lagi = 7 - sekarang.weekday() or 7
    depan = (sekarang + timedelta(days=lagi)).replace(hour=0, minute=0, second=0, microsecond=0)
    sisa = depan - sekarang
    return (int(sisa.total_seconds() // 86400), int(sisa.total_seconds() % 86400 // 3600))

def punya_barang(uid):
    return core.catatan(uid).setdefault('barang', {})

def embed_toko(catat=None):
    stok = stok_toko()
    baris = []
    for kode in stok:
        b = cari_barang(kode)
        if not b:
            continue
        _k, emoji, nama, harga, ket = b
        cukup = ''
        ekor = ''
        if catat is not None:
            punya = catat.get('barang', {}).get(kode, 0)
            cukup = '' if catat['xp'] >= harga else '  ⚠️ XP lo belum cukup'
            ekor = f'  ·  punya {punya}' if punya else ''
            if kode in BATAS_BELI:
                udah = catat.get('beli_total', {}).get(kode, 0)
                ekor += f'  ·  **{udah}/{BATAS_BELI[kode]}** jatah kepakai' if udah else f'  ·  batas {BATAS_BELI[kode]}x seumur hidup'
        elif kode in BATAS_BELI:
            ekor = f'  ·  batas {BATAS_BELI[kode]}x seumur hidup'
        langka = '  🌟 LANGKA' if kode in TOKO_SUPER else ''
        baris.append(f'{emoji}  **{nama}** — {harga:,} XP{cukup}{langka}\n\u3000\u3000{ket}\n\u3000\u3000`/beli {kode}`{ekor}')
    libur = [b for b in BARANG if b[0] not in stok]
    hari, jam = senin_depan()
    isi = discord.Embed(color=WARNA, title='🎣  Toko Pantai — stok minggu ini', description='\n\n'.join(baris))
    if libur:
        isi.add_field(name='Lagi ga dijual', value=' · '.join((f'{b[1]} {b[2]}' for b in libur)) + '\nMungkin muncul lagi minggu depan.', inline=False)
    kaki = f'Stok diundi ulang {hari} hari {jam} jam lagi'
    if catat is not None:
        kaki = f"XP lo sekarang: {catat['xp']:,} · " + kaki.lower()
    else:
        kaki += ' · ketik /toko buat liat XP dan jatah beli lo'
    isi.set_footer(text=kaki)
    return isi

def wishlist_orang(uid):
    return core.catatan(uid).setdefault('wishlist', [])

async def kabarin_wishlist(guild, stok):
    if not WISHLIST_AKTIF:
        return 0
    dikabarin = 0
    for uid, catat in core.data['orang'].items():
        incaran = catat.get('wishlist') or []
        kena = [k for k in incaran if k in stok]
        if not kena:
            continue
        anggota = guild.get_member(int(uid))
        if anggota is None:
            continue
        baris = []
        for kode in kena:
            info = cari_barang(kode)
            if not info:
                continue
            _k, emoji, nama, harga, _ket = info
            cukup = '✅ XP lo cukup' if catat.get('xp', 0) >= harga else f"⚠️ kurang {harga - catat.get('xp', 0):,} XP"
            baris.append(f'{emoji} **{nama}** — {harga:,} XP  ·  {cukup}')
        try:
            await anggota.send('⭐ **Barang incaran lo masuk stok minggu ini.**\n\n' + '\n'.join(baris) + '\n\nStok ganti lagi Senin depan. Beli pakai `/beli`.')
            dikabarin += 1
        except discord.Forbidden:
            pass
        except Exception as e:
            print(f'[wishlist] gagal DM {uid}: {e}')
    if dikabarin:
        print(f'[wishlist] {dikabarin} orang dikabarin soal stok baru')
    return dikabarin

class Toko(commands.Cog):

    def __init__(self, bot):
        self.bot = bot
        self.jaga_julukan.start()
        self.umumin_toko.start()

    def cog_unload(self):
        self.jaga_julukan.cancel()
        self.umumin_toko.cancel()

    @tasks.loop(hours=1)
    async def jaga_julukan(self):
        try:
            sekarang = datetime.now(WIB)
            for uid, catat in list(core.data['orang'].items()):
                efek = catat.get('efek', {})
                sampai = efek.get('julukan_sampai')
                if not sampai:
                    continue
                try:
                    if datetime.fromisoformat(sampai) > sekarang:
                        continue
                except (TypeError, ValueError):
                    pass
                role_id = efek.pop('julukan_role', None)
                efek.pop('julukan_sampai', None)
                core.simpan()
                if not role_id:
                    continue
                for guild in self.bot.guilds:
                    role = guild.get_role(role_id)
                    if role:
                        try:
                            await role.delete(reason='Masa julukan habis')
                        except discord.HTTPException:
                            pass
                        break
        except Exception as e:
            print(f'[arka] jaga julukan error: {e}')

    @jaga_julukan.before_loop
    async def before_jaga_julukan(self):
        await self.bot.wait_until_ready()

    @tasks.loop(minutes=10)
    async def umumin_toko(self):
        try:
            stok_toko()
            toko = core.data.get('toko', {})
            jam = datetime.now(WIB).hour
            if CHANNEL_TOKO and (not toko.get('diumumkan')) and (jam >= JAM_TOKO):
                channel = self.bot.get_channel(CHANNEL_TOKO)
                if channel:
                    await channel.send('🛒 **Stok toko minggu ini udah ganti.**', embed=embed_toko())
                    toko['diumumkan'] = True
                    core.simpan()
                    await kabarin_wishlist(channel.guild, toko.get('stok', []))
        except Exception as e:
            print(f'[arka] pengumuman toko error: {e}')

    @umumin_toko.before_loop
    async def before_umumin_toko(self):
        await self.bot.wait_until_ready()

    @app_commands.command(name='toko', description='Tukar XP jadi barang')
    async def cmd_toko(self, inter: discord.Interaction):
        if not await core.di_arena(inter):
            return
        await inter.response.send_message(embed=embed_toko(core.catatan(inter.user.id)))

    @app_commands.command(name='beli', description='Beli barang di toko')
    @app_commands.describe(barang='Kode barangnya, liat di /toko')
    @app_commands.choices(barang=[app_commands.Choice(name='Kerang Ajaib — 300', value='kerang'), app_commands.Choice(name='Kail Perak — 450', value='kail'), app_commands.Choice(name='Umpan Emas — 500', value='umpan'), app_commands.Choice(name='Air Pasang — 600', value='cepat'), app_commands.Choice(name='Sahabat Lumba lumba — 700', value='lumba'), app_commands.Choice(name='Perisai Beruntun — 800', value='perisai'), app_commands.Choice(name='Tiket Kapal — 1000', value='tiket'), app_commands.Choice(name='Paus Raksasa — 1500', value='paus'), app_commands.Choice(name='Julukan Sendiri — 1200', value='julukan'), app_commands.Choice(name='Hak Nama Nemo — 1800', value='namapet')])
    async def cmd_beli(self, inter: discord.Interaction, barang: app_commands.Choice[str]):
        if not await core.di_arena(inter):
            return
        isi_barang = cari_barang(barang.value)
        if not isi_barang:
            await inter.response.send_message('Barangnya ga ada.', ephemeral=True)
            return
        kode, emoji, nama, harga, _ket = isi_barang
        if kode not in stok_toko():
            hari, jam = senin_depan()
            await inter.response.send_message(f'{emoji} **{nama}** lagi ga dijual minggu ini. Stok diundi ulang {hari} hari {jam} jam lagi, siapa tau muncul.\nLiat yang lagi ada di `/toko`.', ephemeral=True)
            return
        catat = core.catatan(inter.user.id)
        batas = BATAS_BELI.get(kode)
        if batas:
            udah = catat.setdefault('beli_total', {}).get(kode, 0)
            if udah >= batas:
                await inter.response.send_message(f'{emoji} **{nama}** cuma boleh dibeli **{batas} kali seumur hidup**, dan lo udah pakai semua jatahnya. Buff-nya kegedean buat diobral.', ephemeral=True)
                return
        if catat['xp'] < harga:
            kurang = harga - catat['xp']
            await inter.response.send_message(f'XP lo kurang **{kurang:,}** lagi buat {nama}.', ephemeral=True)
            return
        catat['xp'] -= harga
        catat.setdefault('beli_total', {})[kode] = catat.setdefault('beli_total', {}).get(kode, 0) + 1
        tambahan = 'Barangnya kepakai sendiri nanti, ga usah diapa apain.'
        if kode == 'paus':
            catat.setdefault('efek', {})['paus_sampai'] = (datetime.now(WIB) + timedelta(days=1)).isoformat()
            sisa_jatah = batas - catat['beli_total'][kode] if batas else None
            tambahan = '🐋 Langsung nyala. Semua XP lo naik 50 persen selama 24 jam.' + (f'\nSisa jatah beli: **{sisa_jatah}** kali lagi.' if sisa_jatah is not None else '')
        else:
            punya_barang(inter.user.id)[kode] = punya_barang(inter.user.id).get(kode, 0) + 1
        catat.setdefault('riwayat_beli', []).append({'kode': kode, 'harga': harga, 'kapan': datetime.now(WIB).strftime('%d %b %Y %H:%M')})
        if len(catat['riwayat_beli']) > 100:
            del catat['riwayat_beli'][:-100]
        core.simpan()
        await inter.response.send_message(f"{emoji} **{nama}** kebeli. Sisa XP lo **{catat['xp']:,}**.\n{tambahan}")

    @app_commands.command(name='tas', description='Liat barang yang lo punya')
    async def cmd_tas(self, inter: discord.Interaction):
        if not await core.di_arena(inter):
            return
        catat = core.catatan(inter.user.id)
        punya = punya_barang(inter.user.id)
        efek = catat.setdefault('efek', {})
        baris = []
        for kode, emoji, nama, _harga, ket in BARANG:
            jumlah = punya.get(kode, 0)
            if jumlah:
                baris.append(f'{emoji}  **{nama}** × {jumlah}\n\u3000\u3000{ket}')
        aktif = []
        if efek.get('umpan', 0):
            aktif.append(f"🪝 Umpan Emas nyala, sisa **{efek['umpan']}** jelajah")
        sampai = efek.get('cepat_sampai')
        if sampai and datetime.fromisoformat(sampai) > datetime.now(WIB):
            sisa = int((datetime.fromisoformat(sampai) - datetime.now(WIB)).total_seconds() // 60)
            aktif.append(f'⏩ Air Pasang nyala, sisa **{sisa} menit**')
        isi = discord.Embed(color=WARNA, title=f'🎒  Tas {inter.user.display_name}', description='\n\n'.join(baris) if baris else 'Tasnya kosong. Beli di `/toko`.')
        if aktif:
            isi.add_field(name='Lagi jalan', value='\n'.join(aktif), inline=False)
        isi.set_footer(text=f"XP lo: {catat['xp']:,}")
        await inter.response.send_message(embed=isi)

    @app_commands.command(name='riwayat', description='Liat riwayat belanjo lo')
    async def cmd_riwayat(self, inter: discord.Interaction):
        if not await core.di_arena(inter):
            return
        catat = core.catatan(inter.user.id)
        riwayat = catat.get('riwayat_beli', [])
        if not riwayat:
            await inter.response.send_message('Lo belum pernah belanja apa apa.', ephemeral=True)
            return
        baris = []
        for item in reversed(riwayat[-20:]):
            info = cari_barang(item['kode'])
            nama = info[2] if info else item['kode']
            emoji = info[1] if info else '📦'
            baris.append(f"{item['kapan']} - {emoji} **{nama}** ({item['harga']} XP)")
        isi = discord.Embed(color=WARNA, title=f'📜 Riwayat Belanja {inter.user.display_name}', description='\n'.join(baris))
        await inter.response.send_message(embed=isi, ephemeral=True)

    @app_commands.command(name='kasihitem', description='Kasih barang dari tas lo ke orang lain')
    @app_commands.describe(orang='Mau dikasih ke siapa', barang='Barang yang mana', jumlah='Berapa biji')
    @app_commands.choices(barang=[app_commands.Choice(name=f'{b[1]} {b[2]}', value=b[0]) for b in BARANG if b[0] not in KASIH_ITEM_DILARANG])
    async def cmd_kasihitem(self, inter: discord.Interaction, orang: discord.Member, barang: app_commands.Choice[str], jumlah: int=1):
        if not await core.di_arena(inter):
            return
        if not KASIH_ITEM_AKTIF:
            await inter.response.send_message('Fitur ini lagi dimatiin.', ephemeral=True)
            return
        kode = barang.value
        if orang.bot:
            await inter.response.send_message('Bot ga punya tas.', ephemeral=True)
            return
        if orang.id == inter.user.id:
            await inter.response.send_message('Mindahin barang dari tas lo ke tas lo sendiri. Buat apa.', ephemeral=True)
            return
        if jumlah < 1:
            await inter.response.send_message('Minimal satu biji.', ephemeral=True)
            return
        if kode in KASIH_ITEM_DILARANG:
            await inter.response.send_message('Barang ini nempel ke orangnya, ga bisa dioper.', ephemeral=True)
            return
        catat = core.catatan(inter.user.id)
        tas = punya_barang(inter.user.id)
        punya = tas.get(kode, 0)
        info = cari_barang(kode)
        if punya < jumlah:
            await inter.response.send_message(f'Lo cuma punya **{punya}** {info[1]} {info[2]}. Cek `/tas`.', ephemeral=True)
            return
        hari = core.hari_ini()
        jatah = catat.setdefault('kasih_item_harian', {'hari': hari, 'jumlah': 0})
        if jatah.get('hari') != hari:
            jatah['hari'] = hari
            jatah['jumlah'] = 0
        sisa = KASIH_ITEM_HARIAN - jatah['jumlah']
        if jumlah > sisa:
            await inter.response.send_message(f'Sisa jatah lo hari ini **{sisa} barang**. Batasnya {KASIH_ITEM_HARIAN} per hari.', ephemeral=True)
            return
        tas[kode] = punya - jumlah
        if not tas[kode]:
            del tas[kode]
        jatah['jumlah'] += jumlah
        tas_tujuan = punya_barang(orang.id)
        tas_tujuan[kode] = tas_tujuan.get(kode, 0) + jumlah
        penerima = catat.setdefault('kasih_ke', [])
        if orang.id not in penerima:
            penerima.append(orang.id)
        core.simpan()
        _kode, emoji, nama, harga, ket = info
        isi = discord.Embed(color=WARNA, title='🎁  Barang dioper', description=f'{inter.user.mention} ngasih **{jumlah}× {emoji} {nama}** ke {orang.mention}.')
        isi.add_field(name='Gunanya', value=ket, inline=False)
        isi.set_footer(text=f"Sisa jatah {inter.user.display_name} hari ini: {KASIH_ITEM_HARIAN - jatah['jumlah']}")
        await inter.response.send_message(embed=isi)
        try:
            await orang.send(f'{inter.user.display_name} ngasih lo **{jumlah}× {emoji} {nama}** di {inter.guild.name}.\n{ket}\nCek `/tas`.')
        except discord.Forbidden:
            pass

    @app_commands.command(name='wishlist', description='Barang incaran lo, dikabarin kalau masuk stok')
    @app_commands.describe(aksi='Mau ngapain', barang='Barang yang mana')
    @app_commands.choices(aksi=[app_commands.Choice(name='liat', value='liat'), app_commands.Choice(name='tambah', value='tambah'), app_commands.Choice(name='hapus', value='hapus')])
    @app_commands.choices(barang=[app_commands.Choice(name=f'{b[1]} {b[2]}', value=b[0]) for b in BARANG])
    async def cmd_wishlist(self, inter: discord.Interaction, aksi: app_commands.Choice[str]=None, barang: app_commands.Choice[str]=None):
        if not await core.di_arena(inter):
            return
        if not WISHLIST_AKTIF:
            await inter.response.send_message('Fitur ini lagi dimatiin.', ephemeral=True)
            return
        pilih = aksi.value if aksi else 'liat'
        daftar = wishlist_orang(inter.user.id)
        stok = stok_toko()
        if pilih in ('tambah', 'hapus') and barang is None:
            await inter.response.send_message(f'Isi juga barangnya mau yang mana.', ephemeral=True)
            return
        if pilih == 'tambah':
            kode = barang.value
            if kode in daftar:
                await inter.response.send_message(f'**{cari_barang(kode)[2]}** udah ada di incaran lo.', ephemeral=True)
                return
            if len(daftar) >= WISHLIST_MAKS:
                await inter.response.send_message(f'Incaran lo udah {WISHLIST_MAKS} barang. Hapus dulu salah satu.', ephemeral=True)
                return
            daftar.append(kode)
            core.simpan()
            info = cari_barang(kode)
            pesan = f'{info[1]} **{info[2]}** masuk incaran lo. Gua DM kalau dia nongol di stok mingguan.'
            if kode in stok:
                pesan += f'\n\nEh, dia **lagi dijual minggu ini**. `/beli {kode}`.'
            await inter.response.send_message(pesan, ephemeral=True)
            return
        if pilih == 'hapus':
            kode = barang.value
            if kode not in daftar:
                await inter.response.send_message('Itu ga ada di incaran lo.', ephemeral=True)
                return
            daftar.remove(kode)
            core.simpan()
            await inter.response.send_message(f'**{cari_barang(kode)[2]}** dicoret dari incaran.', ephemeral=True)
            return
        if not daftar:
            await inter.response.send_message('Incaran lo masih kosong.\nIsi pakai `/wishlist aksi:tambah barang:...`, nanti gua DM tiap barangnya masuk stok mingguan.', ephemeral=True)
            return
        catat = core.catatan(inter.user.id)
        baris = []
        for kode in daftar:
            info = cari_barang(kode)
            if not info:
                continue
            _k, emoji, nama, harga, _ket = info
            if kode in stok:
                tanda = '🟢 lagi dijual'
            else:
                hari, jam = senin_depan()
                tanda = f'⚪ diundi ulang {hari}h {jam}j lagi'
            kurang = '' if catat['xp'] >= harga else f"  ·  kurang {harga - catat['xp']:,} XP"
            baris.append(f'{emoji} **{nama}** — {harga:,} XP  ·  {tanda}{kurang}')
        isi = discord.Embed(color=WARNA, title='⭐  Barang incaran lo', description='\n'.join(baris))
        isi.set_footer(text=f'{len(daftar)}/{WISHLIST_MAKS} · gua DM kalau ada yang masuk stok')
        await inter.response.send_message(embed=isi, ephemeral=True)

    @app_commands.command(name='pakai', description='Pakai barang sosial yang lo punya')
    @app_commands.describe(barang='Barang mana', nilai='Isian yang dibutuhin barangnya')
    @app_commands.choices(barang=[app_commands.Choice(name='Julukan Sendiri', value='julukan'), app_commands.Choice(name='Hak Nama Nemo', value='namapet')])
    async def cmd_pakai(self, inter: discord.Interaction, barang: app_commands.Choice[str], nilai: str):
        if not await core.di_arena(inter):
            return
        kode = barang.value
        tas = punya_barang(inter.user.id)
        if tas.get(kode, 0) < 1:
            info = cari_barang(kode)
            await inter.response.send_message(f"Lo ga punya {(info[1] if info else '')} **{(info[2] if info else kode)}**. Cek `/toko`.", ephemeral=True)
            return
        if kode == 'namapet':
            if not PET_AKTIF:
                await inter.response.send_message('Peliharaan lagi dimatiin.', ephemeral=True)
                return
            nama_baru = nilai.strip()[:20]
            if len(nama_baru) < 2 or not nama_baru.replace(' ', '').isalnum():
                await inter.response.send_message('Namanya 2 sampai 20 huruf, huruf sama angka doang.', ephemeral=True)
                return
            pet = core.kondisi_pet()
            lama = pet['nama']
            pet['nama'] = nama_baru
            tas[kode] -= 1
            if not tas[kode]:
                del tas[kode]
            core.simpan()
            await inter.response.send_message(f'🐣 **{lama}** sekarang namanya **{nama_baru}**.\n{inter.user.mention} yang ngasih nama. Sekarang seluruh server manggil dia gitu.')
            return
        pecah = nilai.rsplit(' ', 1)
        nama_role = pecah[0].strip()[:24]
        warna_hex = pecah[1] if len(pecah) > 1 else '4AA8D8'
        try:
            nilai_warna = int(warna_hex.lstrip('#'), 16)
        except ValueError:
            await inter.response.send_message('Formatnya: `nama julukan WARNAHEX`\nContoh: `/pakai julukan nilai:Raja Pantai FF6B35`', ephemeral=True)
            return
        if len(nama_role) < 2:
            await inter.response.send_message('Nama julukannya kependekan.', ephemeral=True)
            return
        await inter.response.defer()
        catat = core.catatan(inter.user.id)
        lama_id = catat.get('efek', {}).get('julukan_role')
        if lama_id:
            role_lama = inter.guild.get_role(lama_id)
            if role_lama:
                try:
                    await role_lama.delete(reason='Julukan diganti')
                except discord.HTTPException:
                    pass
        try:
            role = await inter.guild.create_role(name=nama_role, colour=discord.Colour(nilai_warna), reason=f'Julukan dibeli {inter.user}')
            await inter.user.add_roles(role, reason='Julukan')
        except discord.Forbidden:
            await inter.followup.send('Arka butuh izin **Manage Roles** buat bikin julukan. Barang lo belum kepakai kok.', ephemeral=True)
            return
        except discord.HTTPException as e:
            await inter.followup.send(f'Ditolak Discord: {e}', ephemeral=True)
            return
        tas[kode] -= 1
        if not tas[kode]:
            del tas[kode]
        sampai = datetime.now(WIB) + timedelta(days=JULUKAN_HARI)
        catat.setdefault('efek', {})['julukan_role'] = role.id
        catat['efek']['julukan_sampai'] = sampai.isoformat()
        core.simpan()
        await inter.followup.send(f'🏷️ {inter.user.mention} sekarang **{nama_role}**.\nBertahan {JULUKAN_HARI} hari, sampai <t:{int(sampai.timestamp())}:D>.')

async def setup(bot):
    await bot.add_cog(Toko(bot))