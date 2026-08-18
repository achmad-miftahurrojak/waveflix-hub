import discord
from discord.ext import commands, tasks
from discord import app_commands
import os
import sys
from dotenv import load_dotenv
import asyncio
import random
from datetime import datetime, timedelta

# Tambahkan utils/ ke sys.path untuk import pencarian_nobar
_BOT_ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
sys.path.insert(0, os.path.join(_BOT_ROOT, 'utils'))

from pencarian_nobar import cari_film, cari_trending

load_dotenv()
TOKEN = os.getenv('WAVEFLIX_TOKEN')

# [REFACTOR] Memindahkan Hardcoded ID ke environment variable dengan fallback
WAVEFLIX_CATEGORY_ID = int(os.getenv('WAVEFLIX_CATEGORY_ID', '1537065953995788338'))
WAVEFLIX_CHANNEL_ID = int(os.getenv('WAVEFLIX_CHANNEL_ID', '1537224641062506557'))

class WaveFlixBot(commands.Bot):
    def __init__(self):
        super().__init__(command_prefix='!', intents=discord.Intents.default(), help_command=None)

    async def setup_hook(self):
        await self.tree.sync()
        print('[WaveFlix] Slash commands di-sync.')

bot = WaveFlixBot()
STATUS_LIST_WAVEFLIX = ['siapin popcorn kita nobar', 'nonton film gratis tiap hari', 'cari film bioskop terbaru', 'jangan lupa ajak temen nobar']

@tasks.loop(seconds=10)
async def jaga_status_waveflix():
    if bot.is_closed():
        return
    tulisan = random.choice(STATUS_LIST_WAVEFLIX)
    try:
        await bot.change_presence(activity=discord.CustomActivity(name=tulisan))
    except Exception as e:
        if 'closing transport' not in str(e):
            print(f'[WaveFlix] gagal dipasang: {e}')

@jaga_status_waveflix.before_loop
async def sebelum_status_waveflix():
    await bot.wait_until_ready()

@bot.event
async def on_ready():
    print(f'[WaveFlix] Login sebagai {bot.user} (ID: {bot.user.id})')
    if not jaga_status_waveflix.is_running():
        jaga_status_waveflix.start()
    if not rekomendasi_harian.is_running():
        rekomendasi_harian.start()

class HasilNobarView(discord.ui.View):
    def __init__(self, hasil):
        super().__init__(timeout=180)
        # [BUG FIX] Mencegah crash jika jumlah hasil lebih dari 15.
        # Discord membatasi maksimal 5 baris (0-4) dan 5 tombol per baris.
        hasil = hasil[:15]
        for i, h in enumerate(hasil):
            btn = discord.ui.Button(label=f"{h['judul'][:20]}...", style=discord.ButtonStyle.link, url=h['url'], row=i // 3)
            self.add_item(btn)

@tasks.loop(hours=24)
async def rekomendasi_harian():
    if bot.is_closed():
        return
    channel = bot.get_channel(WAVEFLIX_CHANNEL_ID)
    if not channel:
        return
        
    def check_trending_msg(msg):
        if msg.author == bot.user and msg.embeds:
            author_name = getattr(msg.embeds[0].author, 'name', None)
            return author_name and '🌟 FILM TRENDING MINGGU INI 🌟' in str(author_name)
        return False
        
    try:
        # [REFACTOR] Menggunakan .purge untuk menghapus pesan secara massal dan efisien
        await channel.purge(limit=50, check=check_trending_msg)
    except Exception as e:
        print(f'[WaveFlix] Gagal hapus pesan lama: {e}')
        
    hasil = await cari_trending()
    if not hasil:
        return
        
    embeds = []
    # [BUG FIX] Batasi maksimal embed list agar tidak melanggar batas Discord (maksimal 10 embeds)
    for i, h in enumerate(hasil[:10]):
        embed = discord.Embed(title=h['judul'], description=f"[{h['tipe'].upper()}] {h['deskripsi'][:250]}...", color=discord.Color.gold())
        if i == 0:
            embed.set_author(name='🌟 FILM TRENDING MINGGU INI 🌟', icon_url=bot.user.display_avatar.url if bot.user.display_avatar else None)
        if h['poster']:
            embed.set_image(url=h['poster'])
        embeds.append(embed)
        
    view = HasilNobarView(hasil)
    await channel.send(embeds=embeds, view=view)

@rekomendasi_harian.before_loop
async def sebelum_rekomendasi():
    await bot.wait_until_ready()

@bot.tree.command(name='nobar_cari', description='Cari film di WaveFlix buat nobar!')
@app_commands.describe(judul='Judul film atau serial yang mau ditonton')
async def nobar_cari(interaction: discord.Interaction, judul: str):
    if not interaction.channel or getattr(interaction.channel, 'category_id', None) != WAVEFLIX_CATEGORY_ID:
        await interaction.response.send_message('❌ Command ini cuma bisa dipakai di dalam kategori **WaveFlix Room**!', ephemeral=True)
        return
        
    if not interaction.user.voice or not interaction.user.voice.channel:
        await interaction.response.send_message('❌ Lu harus join Voice Channel dulu sebelum bisa nyari film buat nobar!', ephemeral=True)
        return
        
    await interaction.response.defer(ephemeral=True)
    try:
        hasil = await cari_film(judul)
    except Exception as e:
        await interaction.followup.send(f'⚠️ Error: {e}', ephemeral=True)
        return
        
    if not hasil:
        await interaction.followup.send(f'🎬 Waduh, film **{judul}** nggak ketemu nih.', ephemeral=True)
        return
        
    embeds = []
    # [BUG FIX] Batasi pengiriman maksimal 10 embeds per pesan
    for i, h in enumerate(hasil[:10]):
        embed = discord.Embed(title=h['judul'], description=f"[{h['tipe'].upper()}] {h['deskripsi'][:250]}...", color=discord.Color.blue())
        if i == 0:
            embed.set_author(name=f"🍿 Hasil Pencarian WaveFlix ({len(hasil)} hasil buat '{judul}')")
        if h['poster']:
            embed.set_image(url=h['poster'])
        embeds.append(embed)
        
    view = HasilNobarView(hasil)
    await interaction.followup.send(embeds=embeds, view=view)

@bot.tree.command(name='nobar_jadwal', description='Bikin jadwal nobar dan kumpulin massa!')
@app_commands.describe(judul='Judul film yang mau dinobar', waktu="Jam berapa? (contoh: 19:30 atau 'nanti malam')", link='Link film dari WaveFlix (opsional)')
async def nobar_jadwal(interaction: discord.Interaction, judul: str, waktu: str, link: str=None):
    if not interaction.channel or getattr(interaction.channel, 'category_id', None) != WAVEFLIX_CATEGORY_ID:
        await interaction.response.send_message('❌ Command ini cuma bisa dipakai di dalam kategori **WaveFlix Room**!', ephemeral=True)
        return
        
    deskripsi = f'Nanti kita bakal nobar **{judul}** jam **{waktu}**!\nSiapin cemilan dan kopi ☕'
    deskripsi += f'\n\n📍 **Tempat:** {interaction.channel.mention}'
    if link:
        deskripsi += f'\n🔗 [Link Film]({link})'
        
    embed = discord.Embed(title='🎬 JADWAL NOBAR BARU!', description=deskripsi, color=discord.Color.red())
    embed.set_author(name=interaction.user.display_name, icon_url=interaction.user.display_avatar.url if interaction.user.display_avatar else None)
    embed.set_footer(text='Host: Silakan buka link dan mulai Share Screen di Voice Channel ya!')
    
    channel_reservasi = interaction.guild.get_channel(WAVEFLIX_CHANNEL_ID)
    if channel_reservasi:
        await channel_reservasi.send(content='📢 @here Pengumuman Nobar!', embed=embed)
        await interaction.response.send_message(f'✅ Jadwal nobar berhasil disebar ke {channel_reservasi.mention}!', ephemeral=True)
    else:
        await interaction.response.send_message(content='📢 Pengumuman Nobar!', embed=embed)

if __name__ == '__main__':
    if TOKEN:
        bot.run(TOKEN)
    else:
        print('Error: WAVEFLIX_TOKEN belum ada di .env')