import asyncio
import os
import random
import discord
from discord.ext import tasks
from discord import app_commands
from dotenv import load_dotenv
load_dotenv()
TOKEN = os.getenv('KEVIN_TOKEN')
CHANNEL_STUDIO_ID = int(os.getenv('KEVIN_CHANNEL_STUDIO_ID', '1536889743013314570'))
KEVIN_RADIO_URL = os.getenv('KEVIN_RADIO_URL', 'http://ice1.somafm.com/groovesalad-128-mp3')
KEVIN_YT_URL = os.getenv('KEVIN_YT_URL', 'ytsearch1:lofi hip hop radio - beats to relax/study to live')
VOLUME = float(os.getenv('KEVIN_VOLUME', '0.4'))
KEVIN_GUILD_ID = os.getenv('KEVIN_GUILD_ID') # Pengaman agar tidak nyasar ke server lain
WATCHDOG_DETIK = 120
YT_RENEW_JAM = 5
intents = discord.Intents.default()
intents.voice_states = True
bot = discord.Client(intents=intents)
pohon = app_commands.CommandTree(bot)
_voice_client: discord.VoiceClient | None = None
_sedang_reconnect: bool = False
_stream_aktif: bool = True
FFMPEG_OPTS = {'before_options': '-reconnect 1 -reconnect_streamed 1 -reconnect_delay_max 5 -thread_queue_size 4096 -nostdin -loglevel warning', 'options': '-vn'}
_BOT_ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
FFMPEG_PATH = os.path.join(_BOT_ROOT, 'bin', 'ffmpeg.exe')
if not os.path.exists(FFMPEG_PATH):
    FFMPEG_PATH = 'ffmpeg'  # fallback ke ffmpeg di PATH sistem

def _ambil_url_sync(sumber: str) -> str:
    if 'youtube.com' in sumber or 'youtu.be' in sumber or sumber.startswith('ytsearch'):
        try:
            import yt_dlp
            ydl_opts = {'format': 'bestaudio/best', 'quiet': True, 'no_warnings': True}
            with yt_dlp.YoutubeDL(ydl_opts) as ydl:
                info = ydl.extract_info(sumber, download=False)
                if 'entries' in info:
                    info = info['entries'][0]
                return info['url']
        except Exception as e:
            print(f'[kevin] yt-dlp error (YouTube): {e}')
            print('[kevin] Mencoba fallback ke SoundCloud...')
            try:
                ydl_opts_sc = {'format': 'bestaudio/best', 'quiet': True, 'no_warnings': True, 'default_search': 'scsearch'}
                with yt_dlp.YoutubeDL(ydl_opts_sc) as ydl:
                    info_sc = ydl.extract_info('scsearch1:lofi hip hop chillhop mix', download=False)
                    if 'entries' in info_sc:
                        info_sc = info_sc['entries'][0]
                    return info_sc['url']
            except Exception as e2:
                print(f'[kevin] yt-dlp error (SoundCloud): {e2}')
                raise
    return sumber

async def _ambil_url(sumber: str) -> str:
    return await asyncio.to_thread(_ambil_url_sync, sumber)

async def _mulai_stream(vc: discord.VoiceClient) -> bool:
    try:
        sumber_url = KEVIN_YT_URL if KEVIN_YT_URL else KEVIN_RADIO_URL
        url = await _ambil_url(sumber_url)
        if vc.is_playing():
            vc.stop()
            await asyncio.sleep(1.0)

        def setelah_selesai(error):
            if error:
                print(f'[kevin] stream berhenti: {error}')
        audio = discord.FFmpegPCMAudio(url, executable=FFMPEG_PATH, **FFMPEG_OPTS)
        sumber = discord.PCMVolumeTransformer(audio, volume=VOLUME)
        vc.play(sumber, after=setelah_selesai)
        print('[kevin] streaming berjalan')
        return True
    except Exception as e:
        print(f'[kevin] gagal mulai stream: {e}')
        return False

async def _join_dan_putar(guild: discord.Guild) -> bool:
    global _voice_client
    channel = guild.get_channel(CHANNEL_STUDIO_ID)
    if not isinstance(channel, discord.VoiceChannel):
        print(f'[kevin] channel ID {CHANNEL_STUDIO_ID} ga ketemu atau bukan voice')
        return False
    try:
        if _voice_client and _voice_client.is_connected():
            if _voice_client.channel.id != channel.id:
                await _voice_client.move_to(channel)
        else:
            _voice_client = await channel.connect(self_deaf=True, reconnect=True)
        return await _mulai_stream(_voice_client)
    except discord.ClientException as e:
        print(f'[kevin] ClientException saat join: {e}')
        try:
            if guild.voice_client:
                # [BUG FIX] Pastikan FFMPEG mati sebelum disconnect untuk mencegah zombie processes
                if guild.voice_client.is_playing():
                    guild.voice_client.stop()
                await guild.voice_client.disconnect(force=True)
            _voice_client = None
            await asyncio.sleep(2)
            _voice_client = await channel.connect(self_deaf=True, reconnect=True)
            return await _mulai_stream(_voice_client)
        except Exception as e2:
            print(f'[kevin] retry join juga gagal: {e2}')
            return False
    except Exception as e:
        print(f'[kevin] gagal join: {e}')
        return False

async def watchdog():
    global _sedang_reconnect
    await bot.wait_until_ready()
    await asyncio.sleep(10)
    print('[kevin] watchdog jalan')
    while not bot.is_closed():
        await asyncio.sleep(WATCHDOG_DETIK)
        if not _stream_aktif:
            continue
        try:
            guild = bot.get_guild(int(KEVIN_GUILD_ID)) if KEVIN_GUILD_ID else next(iter(bot.guilds), None)
            if not guild:
                continue
            perlu_reconnect = _voice_client is None or not _voice_client.is_connected() or (not _voice_client.is_playing() and (not _voice_client.is_paused()))
            if perlu_reconnect and (not _sedang_reconnect):
                _sedang_reconnect = True
                print('[kevin] watchdog: putus, reconnect...')
                oke = await _join_dan_putar(guild)
                _sedang_reconnect = False
                if not oke:
                    print('[kevin] reconnect gagal, coba lagi di iterasi berikutnya')
        except Exception as e:
            print(f'[kevin] watchdog error: {e}')
            _sedang_reconnect = False

async def renew_yt_loop():
    if not KEVIN_YT_URL:
        return
    await bot.wait_until_ready()
    await asyncio.sleep(YT_RENEW_JAM * 3600)
    while not bot.is_closed():
        if _stream_aktif and _voice_client and _voice_client.is_connected():
            print('[kevin] renew YouTube URL...')
            try:
                guild = bot.get_guild(int(KEVIN_GUILD_ID)) if KEVIN_GUILD_ID else next(iter(bot.guilds), None)
                if guild:
                    await _join_dan_putar(guild)
            except Exception as e:
                print(f'[kevin] renew error: {e}')
        await asyncio.sleep(YT_RENEW_JAM * 3600)
STATUS_LIST_KEVIN = ['nemenin lo nugas atau rebahan', 'play lofi hip hop 24 jam', 'santai dulu ga sih', 'temen setia lo begadang']

@tasks.loop(seconds=10)
async def jaga_status_kevin():
    if bot.is_closed():
        return
    tulisan = random.choice(STATUS_LIST_KEVIN)
    try:
        await bot.change_presence(activity=discord.CustomActivity(name=tulisan))
    except Exception as e:
        if 'closing transport' not in str(e):
            print(f'[kevin] gagal dipasang: {e}')

@jaga_status_kevin.before_loop
async def sebelum_status_kevin():
    await bot.wait_until_ready()

@bot.event
async def on_ready():
    print(f'[kevin] {bot.user} online')
    if not jaga_status_kevin.is_running():
        jaga_status_kevin.start()
    try:
        await pohon.sync()
    except Exception as e:
        print(f'[kevin] sync command error: {e}')
    guild = bot.get_guild(int(KEVIN_GUILD_ID)) if KEVIN_GUILD_ID else next(iter(bot.guilds), None)
    if guild:
        oke = await _join_dan_putar(guild)
        if oke:
            print('[kevin] stream berjalan otomatis saat startup')
        else:
            print('[kevin] startup stream gagal, watchdog akan coba lagi')
    else:
        print('[kevin] belum ada guild, watchdog akan handle')
    bot.loop.create_task(watchdog())
    bot.loop.create_task(renew_yt_loop())

@bot.event
async def on_voice_state_update(member, before, after):
    if member.id == bot.user.id and before.channel and (not after.channel):
        print('[kevin] bot terputus dari voice channel. Watchdog akan reconnect otomatis (Fitur 24/7 aktif).')

@pohon.command(name='play', description='Mulai atau restart lofi radio (admin)')
@app_commands.guild_only()
@app_commands.checks.has_permissions(manage_guild=True)
async def play(inter: discord.Interaction):
    global _stream_aktif
    _stream_aktif = True
    await inter.response.defer(ephemeral=True)
    oke = await _join_dan_putar(inter.guild)
    if oke:
        await inter.followup.send('🎶 **Lofi radio mulai diputar.**', ephemeral=True)
    else:
        await inter.followup.send('Gagal memutar radio. Cek log terminal untuk detailnya.', ephemeral=True)

@pohon.command(name='stop', description='Hentikan lofi radio dan keluarkan bot (admin)')
@app_commands.guild_only()
@app_commands.checks.has_permissions(manage_guild=True)
async def stop(inter: discord.Interaction):
    global _stream_aktif, _voice_client
    _stream_aktif = False
    if _voice_client and _voice_client.is_connected():
        # [BUG FIX] Stop FFMPEG sebelum disconnect
        if _voice_client.is_playing():
            _voice_client.stop()
        await _voice_client.disconnect(force=True)
        _voice_client = None
    await inter.response.send_message('Lofi radio dihentikan. Pakai `/play` buat nyalain lagi.', ephemeral=True)

@pohon.command(name='lofi', description='Cek status stream lofi radio saat ini')
@app_commands.guild_only()
async def lofi(inter: discord.Interaction):
    if not _stream_aktif:
        status = 'Dimatikan manual (Ketik `/play` untuk menyalakan)'
    elif not _voice_client or not _voice_client.is_connected():
        status = 'Mencoba terhubung ke voice channel...'
    elif _voice_client.is_paused():
        status = f'Dijeda di <#{_voice_client.channel.id}>'
    elif _voice_client.is_playing():
        status = f'Sedang memutar di <#{_voice_client.channel.id}> 🎧'
    else:
        status = 'Terhubung tapi tidak memutar (menunggu audio)'
    embed = discord.Embed(title='Lofi Radio Status', description=status, color=2829617)
    embed.set_footer(text='Lofi Radio is a bot that plays Lo-Fi Hip Hop music 24/7.')
    await inter.response.send_message(embed=embed, ephemeral=True)

@pohon.command(name='settings', description='Pengaturan lofi radio (admin)')
@app_commands.describe(volume='Atur volume radio (0 sampai 100)')
@app_commands.guild_only()
@app_commands.checks.has_permissions(manage_guild=True)
async def settings(inter: discord.Interaction, volume: int=None):
    global VOLUME
    if volume is None:
        await inter.response.send_message(f'Volume saat ini: **{int(VOLUME * 100)}%**\nGunakan `/settings volume:<angka>` untuk mengubah.', ephemeral=True)
        return
    if not 0 <= volume <= 100:
        await inter.response.send_message('Angka volume harus antara 0 sampai 100.', ephemeral=True)
        return
    VOLUME = volume / 100
    if _voice_client and _voice_client.is_connected() and isinstance(_voice_client.source, discord.PCMVolumeTransformer):
        _voice_client.source.volume = VOLUME
        await inter.response.send_message(f'Volume diubah menjadi: **{volume}%**', ephemeral=True)
    else:
        await inter.response.send_message(f'Volume diubah menjadi: **{volume}%** (Efektif saat radio diputar ulang)', ephemeral=True)

@pohon.command(name='help', description='Panduan lengkap penggunaan lofi radio')
async def help_cmd(inter: discord.Interaction):
    embed = discord.Embed(title='Bantuan Lofi Radio', description='Bot ini memutar musik Lo-Fi Hip Hop 24/7 secara otomatis.', color=2829617)
    embed.add_field(name='`/play`', value='Mulai atau restart radio (Admin)', inline=False)
    embed.add_field(name='`/stop`', value='Matikan radio (Admin)', inline=False)
    embed.add_field(name='`/lofi`', value='Cek status radio saat ini', inline=False)
    embed.add_field(name='`/settings`', value='Atur volume radio (Admin)', inline=False)
    embed.add_field(name='Info', value='Bot akan otomatis reconnect jika koneksi terputus.', inline=False)
    await inter.response.send_message(embed=embed, ephemeral=True)

# [BUG FIX] Penanganan error saat user bukan admin mengetik command
@pohon.error
async def on_app_command_error(inter: discord.Interaction, error: app_commands.AppCommandError):
    if isinstance(error, app_commands.MissingPermissions):
        await inter.response.send_message("Lo ga punya izin (harus Admin) buat pakai perintah ini.", ephemeral=True)
    else:
        print(f"[kevin] Command error: {error}")

if __name__ == '__main__':
    if not TOKEN:
        print('[kevin] KEVIN_TOKEN belum diisi di .env')
    else:
        bot.run(TOKEN)