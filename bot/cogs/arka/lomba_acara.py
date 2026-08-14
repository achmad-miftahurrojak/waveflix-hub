import asyncio
from datetime import datetime, timedelta
import discord
from discord import app_commands
from discord.ext import commands, tasks
from . import core
from .config import *

def lomba_aktif(channel_id):
    return core.data.setdefault('lomba', {}).get(str(channel_id))

def _boleh_urus_acara(inter, acara):
    if inter.user.id == acara.get('pembuat'):
        return True
    if inter.user.guild_permissions.manage_events or inter.user.guild_permissions.manage_guild:
        return True
    if ROLE_SOAL:
        punya = {r.name for r in getattr(inter.user, 'roles', [])}
        if punya & set(ROLE_SOAL):
            return True
    return False

async def tutup_lomba(bot, channel, otomatis=False):
    lomba = lomba_aktif(channel.id)
    if not lomba:
        return False
    pemilih_global = {}
    hasil_mentah = []
    for i, kiriman in enumerate(lomba['kiriman']):
        try:
            pesan = await channel.fetch_message(kiriman['pesan'])
        except Exception:
            continue
        pemilih = []
        for reaksi in pesan.reactions:
            if str(reaksi.emoji) != EMOJI_VOTE:
                continue
            async for orang in reaksi.users():
                if orang.bot or orang.id == kiriman['orang']:
                    continue
                pemilih.append(orang.id)
        hasil_mentah.append((kiriman['orang'], pesan.jump_url, pemilih, i))
    skor = {i: 0 for i in range(len(hasil_mentah))}
    for _orang_id, _tautan, pemilih, idx in hasil_mentah:
        for uid in pemilih:
            if uid in pemilih_global:
                continue
            pemilih_global[uid] = idx
            skor[idx] += 1
    hasil = [(orang_id, skor[idx], tautan) for orang_id, tautan, _pemilih, idx in hasil_mentah]
    hasil.sort(key=lambda x: x[1], reverse=True)
    del core.data['lomba'][str(channel.id)]
    core.simpan()
    if not hasil:
        await channel.send(f"🏁 Lomba **{lomba['judul']}** ditutup, tapi ga ada yang ngirim. Sayang banget.")
        return True
    baris = []
    for i, (uid, suara_masuk, tautan) in enumerate(hasil[:3]):
        anggota = channel.guild.get_member(int(uid))
        nama = anggota.display_name if anggota else 'entah siapa'
        tanda = ['🥇', '🥈', '🥉'][i]
        hadiah = LOMBA_HADIAH[i] if i < len(LOMBA_HADIAH) else 0
        baris.append(f'{tanda} **{nama}** — {suara_masuk} suara · +{hadiah} XP\n\u3000\u3000[liat karyanya]({tautan})')
        if anggota and hadiah:
            core.catat_main(anggota.id, 'lomba', menang=i == 0)
            await core.tambah_xp(bot, anggota, hadiah)
    isi = discord.Embed(color=WARNA, title=f"🏁  Hasil {lomba['judul']}", description='\n'.join(baris))
    isi.set_footer(text=f'{len(hasil)} karya masuk · 1 orang = 1 suara' + (' · ditutup otomatis' if otomatis else ''))
    await channel.send(embed=isi)
    return True

def acara_aktif():
    return core.data.setdefault('acara', [])

async def peserta_acara(channel, id_pesan):
    try:
        pesan = await channel.fetch_message(id_pesan)
    except Exception:
        return []
    for reaksi in pesan.reactions:
        if str(reaksi.emoji) == EMOJI_IKUT:
            return [u async for u in reaksi.users() if not u.bot]
    return []

async def dm_peserta(orang, isi):
    kelewat = []
    for o in orang:
        try:
            await o.send(embed=isi)
        except discord.Forbidden:
            kelewat.append(o.display_name)
        except Exception as e:
            print(f'[arka] DM acara ke {o} gagal: {e}')
    return kelewat

def _cari_acara(nama):
    nama_l = nama.strip().lower()
    return [a for a in acara_aktif() if a['nama'].lower() == nama_l and (not a.get('dimulai'))]

class LombaAcara(commands.Cog):

    def __init__(self, bot):
        self.bot = bot
        self.jaga_lomba.start()
        self.jaga_acara.start()

    def cog_unload(self):
        self.jaga_lomba.cancel()
        self.jaga_acara.cancel()

    @commands.Cog.listener()
    async def on_message(self, pesan):
        if pesan.author.bot or not pesan.guild:
            return
        lomba = lomba_aktif(pesan.channel.id)
        if lomba and pesan.attachments:
            lomba['kiriman'].append({'pesan': pesan.id, 'orang': pesan.author.id})
            core.simpan()
            try:
                await pesan.add_reaction(EMOJI_VOTE)
            except Exception:
                pass

    @commands.Cog.listener()
    async def on_raw_reaction_add(self, payload):
        if payload.user_id == self.bot.user.id:
            return
        if str(payload.emoji) != EMOJI_VOTE:
            return
        lomba = lomba_aktif(payload.channel_id)
        if not lomba:
            return
        channel = self.bot.get_channel(payload.channel_id)
        if channel is None:
            return
        kiriman = next((k for k in lomba['kiriman'] if k['pesan'] == payload.message_id), None)
        if not kiriman:
            return
        if payload.user_id == kiriman['orang']:
            try:
                pesan = await channel.fetch_message(payload.message_id)
                user = payload.member or self.bot.get_user(payload.user_id)
                if user:
                    await pesan.remove_reaction(payload.emoji, user)
            except Exception:
                pass
            return
        for lain in lomba['kiriman']:
            if lain['pesan'] == payload.message_id:
                continue
            try:
                pesan_lain = await channel.fetch_message(lain['pesan'])
            except Exception:
                continue
            for reaksi in pesan_lain.reactions:
                if str(reaksi.emoji) != EMOJI_VOTE:
                    continue
                async for orang in reaksi.users():
                    if orang.id == payload.user_id:
                        try:
                            pesan_baru = await channel.fetch_message(payload.message_id)
                            user = payload.member or self.bot.get_user(payload.user_id)
                            if user:
                                await pesan_baru.remove_reaction(payload.emoji, user)
                        except Exception:
                            pass
                        return

    @tasks.loop(minutes=2)
    async def jaga_lomba(self):
        try:
            sekarang = datetime.now(WIB)
            for ch_id, lomba in list(core.data.get('lomba', {}).items()):
                if datetime.fromisoformat(lomba['tutup']) <= sekarang:
                    channel = self.bot.get_channel(int(ch_id))
                    if channel:
                        await tutup_lomba(self.bot, channel, otomatis=True)
        except Exception as e:
            print(f'[arka] jaga lomba error: {e}')

    @jaga_lomba.before_loop
    async def before_jaga_lomba(self):
        await self.bot.wait_until_ready()

    @tasks.loop(minutes=1)
    async def jaga_acara(self):
        try:
            sekarang = datetime.now(WIB)
            sisa = []
            for acara in acara_aktif():
                mulai = datetime.fromisoformat(acara['waktu'])
                channel = self.bot.get_channel(acara['channel'])
                if not channel:
                    continue
                menit_lagi = (mulai - sekarang).total_seconds() / 60
                if menit_lagi <= 0 and (not acara.get('dimulai')):
                    orang = await peserta_acara(channel, acara['pesan'])
                    sebut = ' '.join((o.mention for o in orang)) or 'ga ada yang daftar'
                    await channel.send(f"🎉 **{acara['nama']}** mulai sekarang!\n{sebut}")
                    if ACARA_DM and orang:
                        kabar = discord.Embed(color=WARNA, title=f"🎉  {acara['nama']} mulai sekarang", description=f'Di **{channel.guild.name}**, gabung sekarang.')
                        kabar.add_field(name='Tempatnya', value=f'[Buka channelnya]({channel.jump_url})')
                        await dm_peserta(orang, kabar)
                    acara['dimulai'] = True
                    continue
                if 0 < menit_lagi <= ACARA_INGETIN and (not acara.get('diingetin')):
                    orang = await peserta_acara(channel, acara['pesan'])
                    sebut = ' '.join((o.mention for o in orang))
                    await channel.send(f"⏰ **{acara['nama']}** {int(menit_lagi)} menit lagi. " + (sebut if sebut else 'Belum ada yang daftar nih.'))
                    if ACARA_DM and orang:
                        kabar = discord.Embed(color=WARNA, title=f"⏰  {acara['nama']} sebentar lagi", description=f'Mulai **{int(menit_lagi)} menit lagi** di **{channel.guild.name}**.')
                        kabar.add_field(name='Tempatnya', value=f'[Buka channelnya]({channel.jump_url})')
                        kabar.set_footer(text='Lo dapet ini karena mencet ✅ di pengumuman')
                        kelewat = await dm_peserta(orang, kabar)
                        if kelewat:
                            print(f"[arka] DM acara ketutup buat: {', '.join(kelewat)}")
                    acara['diingetin'] = True
                sisa.append(acara)
            if len(sisa) != len(acara_aktif()):
                core.data['acara'] = sisa
                core.simpan()
        except Exception as e:
            print(f'[arka] jaga acara error: {e}')

    @jaga_acara.before_loop
    async def before_jaga_acara(self):
        await self.bot.wait_until_ready()

    @app_commands.command(name='lomba', description='Buka lomba kiriman foto atau gambar')
    @app_commands.describe(judul='Nama lombanya', jam='Lomba dibuka berapa jam. Contoh: 24')
    async def cmd_lomba(self, inter: discord.Interaction, judul: str, jam: int=24):
        if not await core.di_arena(inter):
            return
        if lomba_aktif(inter.channel_id):
            await inter.response.send_message('Masih ada lomba jalan di sini. Tutup dulu pakai `/lombatutup`.', ephemeral=True)
            return
        if not 1 <= jam <= 168:
            await inter.response.send_message('Antara 1 sampai 168 jam ya.', ephemeral=True)
            return
        tutup = datetime.now(WIB) + timedelta(hours=jam)
        isi = discord.Embed(color=WARNA, title=f'🏖️  {judul}', description='Kirim karya lo ke channel ini, satu pesan berisi gambar.\nNanti gua tempelin 🌊, tinggal member yang milih.\n**Satu orang cuma boleh vote satu karya.**')
        isi.add_field(name='Ditutup', value=f"{tutup.strftime('%d %b, %H:%M')} WIB ({jam} jam lagi)")
        isi.set_footer(text=f'Ga boleh vote karya sendiri. Juara 1 dapet {LOMBA_HADIAH[0]} XP')
        await inter.response.send_message(embed=isi)
        core.data.setdefault('lomba', {})[str(inter.channel_id)] = {'judul': judul, 'tutup': tutup.isoformat(), 'kiriman': []}
        core.simpan()

    @app_commands.command(name='lombatutup', description='Tutup lomba lebih cepat')
    async def cmd_lombatutup(self, inter: discord.Interaction):
        if not await core.di_arena(inter):
            return
        if not lomba_aktif(inter.channel_id):
            await inter.response.send_message('Ga ada lomba yang jalan di sini.', ephemeral=True)
            return
        await inter.response.send_message('Oke, gua itung suaranya...')
        await tutup_lomba(self.bot, inter.channel)

    @app_commands.command(name='acara', description='Bikin acara, orang bisa daftar ikut')
    @app_commands.describe(nama='Nama acaranya', jam='Jam mulai, format 19:30', catatan='Keterangan tambahan, boleh dikosongin')
    async def cmd_acara(self, inter: discord.Interaction, nama: str, jam: str, catatan: str=None):
        if not await core.di_arena(inter):
            return
        try:
            j, m = [int(x) for x in jam.strip().split(':')]
            assert 0 <= j <= 23 and 0 <= m <= 59
        except Exception:
            await inter.response.send_message('Jamnya ditulis kayak `19:30` ya.', ephemeral=True)
            return
        sekarang = datetime.now(WIB)
        mulai = sekarang.replace(hour=j, minute=m, second=0, microsecond=0)
        if mulai <= sekarang:
            mulai += timedelta(days=1)
        selisih = mulai - sekarang
        jam_lagi = int(selisih.total_seconds() // 3600)
        menit_lagi = int(selisih.total_seconds() % 3600 // 60)
        isi = discord.Embed(color=WARNA, title=f'📣  {nama}', description=catatan or 'Mencet ✅ kalau mau ikut.')
        isi.add_field(name='Mulai', value=f"{mulai.strftime('%d %b, %H:%M')} WIB\n({jam_lagi} jam {menit_lagi} menit lagi)")
        isi.set_footer(text=f'Dibikin {inter.user.display_name} · peserta di-ping {ACARA_INGETIN} menit sebelum mulai' + (' dan dapet DM' if ACARA_DM else ''))
        await inter.response.send_message(embed=isi)
        pesan = await inter.original_response()
        try:
            await pesan.add_reaction(EMOJI_IKUT)
        except Exception:
            pass
        acara_aktif().append({'nama': nama, 'waktu': mulai.isoformat(), 'channel': inter.channel_id, 'pesan': pesan.id, 'pembuat': inter.user.id, 'diingetin': False, 'dimulai': False})
        core.simpan()

    @app_commands.command(name='acaralist', description='Acara apa aja yang bakal jalan')
    async def cmd_acaralist(self, inter: discord.Interaction):
        if not await core.di_arena(inter):
            return
        daftar = sorted([a for a in acara_aktif() if not a.get('dimulai')], key=lambda a: a['waktu'])
        if not daftar:
            await inter.response.send_message('Belum ada acara yang dijadwalin.')
            return
        baris = []
        for acara in daftar:
            mulai = datetime.fromisoformat(acara['waktu'])
            baris.append(f"**{acara['nama']}** — {mulai.strftime('%d %b, %H:%M')} WIB di <#{acara['channel']}>")
        isi = discord.Embed(color=WARNA, title='📅  Acara mendatang', description='\n'.join(baris))
        isi.set_footer(text='Ubah jam: /acaraedit · Batalin: /acarahapus')
        await inter.response.send_message(embed=isi)

    @app_commands.command(name='acarahapus', description='Batalin acara yang belum mulai')
    @app_commands.describe(nama='Nama acara yang mau dibatalin')
    async def cmd_acarahapus(self, inter: discord.Interaction, nama: str):
        if not await core.di_arena(inter):
            return
        cocok = _cari_acara(nama)
        if not cocok:
            await inter.response.send_message(f'Ga nemu acara **{nama}** yang belum mulai. Cek `/acaralist`.', ephemeral=True)
            return
        acara = cocok[0]
        if not _boleh_urus_acara(inter, acara):
            await inter.response.send_message('Cuma yang bikin acara (atau admin) yang boleh batalin.', ephemeral=True)
            return
        core.data['acara'] = [a for a in acara_aktif() if a is not acara]
        core.simpan()
        channel = self.bot.get_channel(acara['channel'])
        if channel:
            try:
                pesan = await channel.fetch_message(acara['pesan'])
                await pesan.reply(f"Acara **{acara['nama']}** dibatalin sama {inter.user.display_name}.")
            except Exception:
                pass
        await inter.response.send_message(f"Oke, **{acara['nama']}** gua batalin.")

    @app_commands.command(name='acaraedit', description='Ubah jam mulai acara')
    @app_commands.describe(nama='Nama acaranya', jam='Jam baru, format 19:30')
    async def cmd_acaraedit(self, inter: discord.Interaction, nama: str, jam: str):
        if not await core.di_arena(inter):
            return
        try:
            j, m = [int(x) for x in jam.strip().split(':')]
            assert 0 <= j <= 23 and 0 <= m <= 59
        except Exception:
            await inter.response.send_message('Jamnya ditulis kayak `19:30` ya.', ephemeral=True)
            return
        cocok = _cari_acara(nama)
        if not cocok:
            await inter.response.send_message(f'Ga nemu acara **{nama}** yang belum mulai. Cek `/acaralist`.', ephemeral=True)
            return
        acara = cocok[0]
        if not _boleh_urus_acara(inter, acara):
            await inter.response.send_message('Cuma yang bikin acara (atau admin) yang boleh ngubah.', ephemeral=True)
            return
        sekarang = datetime.now(WIB)
        mulai = sekarang.replace(hour=j, minute=m, second=0, microsecond=0)
        if mulai <= sekarang:
            mulai += timedelta(days=1)
        acara['waktu'] = mulai.isoformat()
        acara['diingetin'] = False
        core.simpan()
        channel = self.bot.get_channel(acara['channel'])
        if channel:
            try:
                pesan = await channel.fetch_message(acara['pesan'])
                isi = pesan.embeds[0] if pesan.embeds else discord.Embed(color=WARNA, title=f"📣  {acara['nama']}")
                selisih = mulai - sekarang
                jam_lagi = int(selisih.total_seconds() // 3600)
                menit_lagi = int(selisih.total_seconds() % 3600 // 60)
                fields = [(f.name, f.value, f.inline) for f in isi.fields if f.name != 'Mulai']
                isi.clear_fields()
                isi.add_field(name='Mulai', value=f"{mulai.strftime('%d %b, %H:%M')} WIB\n({jam_lagi} jam {menit_lagi} menit lagi)")
                for n, v, inl in fields:
                    isi.add_field(name=n, value=v, inline=inl)
                await pesan.edit(embed=isi)
            except Exception as e:
                print(f'[arka] edit embed acara gagal: {e}')
        await inter.response.send_message(f"Jam **{acara['nama']}** dipindah ke **{mulai.strftime('%d %b, %H:%M')} WIB**.")

async def setup(bot):
    await bot.add_cog(LombaAcara(bot))