import os
import discord
from discord.ext import commands
from discord import app_commands
import asyncio
from dotenv import load_dotenv
import re
from google import genai
import wavelink
import cloudscraper
from bs4 import BeautifulSoup
import json
from collections import deque

load_dotenv()
TOKEN = os.getenv("TIDETUNES_TOKEN")
GEMINI_KEY = os.getenv("GEMINI_API_KEY")
LAVALINK_URI = os.getenv("LAVALINK_URI", "http://127.0.0.1:2333")
LAVALINK_PASSWORD = os.getenv("LAVALINK_PASSWORD", "youshallnotpass")

genai_client = None
if GEMINI_KEY:
    genai_client = genai.Client(api_key=GEMINI_KEY)

class TideTunesBot(commands.Bot):
    def __init__(self):
        intents = discord.Intents.default()
        intents.message_content = True
        intents.voice_states = True
        super().__init__(command_prefix="!", intents=intents)
        
    async def setup_hook(self):
        nodes = [wavelink.Node(uri=LAVALINK_URI, password=LAVALINK_PASSWORD)]
        await wavelink.Pool.connect(nodes=nodes, client=self, cache_capacity=100)
        await self.tree.sync()
        print("[TideTunes] Bot is ready, Lavalink connected, and slash commands synced.")

bot = TideTunesBot()

# [REFACTOR] Memori untuk state per-guild
play_history = {}
autoplay_status = {}
autoplay_locks = {}

def _track_history(guild_id, title):
    """Menyimpan history lagu menggunakan deque untuk operasi O(1)"""
    if guild_id not in play_history:
        play_history[guild_id] = deque(maxlen=10)
    play_history[guild_id].append(title)

def _get_autoplay_lock(guild_id):
    if guild_id not in autoplay_locks:
        autoplay_locks[guild_id] = asyncio.Lock()
    return autoplay_locks[guild_id]

def _clear_guild_state(guild_id):
    """[BUG FIX] Mencegah Memory Leak saat bot dihentikan atau disconnect"""
    play_history.pop(guild_id, None)
    autoplay_status.pop(guild_id, None)
    autoplay_locks.pop(guild_id, None)

@bot.event
async def on_wavelink_node_ready(payload: wavelink.NodeReadyEventPayload):
    print(f"[TideTunes] Lavalink Node connected: {payload.node.identifier}")

@bot.event
async def on_wavelink_track_start(payload: wavelink.TrackStartEventPayload):
    player = payload.player
    if not player: return
    _track_history(player.guild.id, payload.track.title)
    
    # Trigger background AI DJ fetch if queue is low
    guild_id = player.guild.id
    if len(player.queue) < 2 and autoplay_status.get(guild_id, False):
        asyncio.run_coroutine_threadsafe(process_autoplay(guild_id, player), bot.loop)

@bot.event
async def on_wavelink_track_end(payload: wavelink.TrackEndEventPayload):
    player = payload.player
    if not player: return
    
    guild_id = player.guild.id
    if player.queue.is_empty:
        # [BUG FIX] Tambahkan flag state checking agar tidak asal disconnect
        await asyncio.sleep(10)
        if not player.playing and player.queue.is_empty:
            await player.disconnect()
            _clear_guild_state(guild_id)
            print(f"[TideTunes] Disconnected from {guild_id} due to inactivity.")

async def process_autoplay(guild_id, player):
    if not autoplay_status.get(guild_id, False):
        return
    if not GEMINI_KEY:
        autoplay_status[guild_id] = False
        print("[TideTunes] AI Auto-DJ dimatikan: GEMINI_API_KEY belum disetel.")
        return

    lock = _get_autoplay_lock(guild_id)
    if lock.locked():
        return

    async with lock:
        if len(player.queue) >= 3:
            return
    
        history = list(play_history.get(guild_id, []))
        if not history:
            return
    
        prompt = (
            f"Gua habis dengerin lagu-lagu ini: {', '.join(history)}. "
            "Tolong kasih gua 10 rekomendasi lagu baru yang vibes, genre, atau beat-nya mirip dan nyambung banget sama lagu-lagu itu. "
            "Jangan kasih lagu yang udah ada di list itu. "
            "Balas HANYA dengan 10 baris, setiap baris formatnya: 'Judul Lagu - Nama Artis'. "
            "Jangan ada nomor urut, jangan ada teks pembuka/penutup."
        )
        try:
            # [CRITICAL FIX] gemini-3.5-flash belum eksis, diubah ke gemini-1.5-flash
            response = await asyncio.to_thread(
                genai_client.models.generate_content,
                model='gemini-1.5-flash',
                contents=prompt
            )
            rekomendasi_list = [line.strip() for line in response.text.strip().split('\n') if line.strip()]
            valid_recs = [r for r in rekomendasi_list if '-' in r and not r.startswith('*') and not r.lower().startswith('berikut')]
            
            if valid_recs:
                for rec in valid_recs:
                    tracks = await wavelink.Playable.search(rec, source=wavelink.TrackSource.SoundCloud)
                    if tracks:
                        player.queue.put(tracks[0])
                
                if not player.playing and not player.queue.is_empty:
                    await player.play(player.queue.get())
                    
        except Exception as e:
            print(f"[TideTunes] Error pre-fetch AI Auto-DJ: {e}")

# ─── Scrapers for Spotify ─────────────────────────────────────────────────────

async def ambil_playlist_spotify(url):
    match = re.search(r'playlist/([a-zA-Z0-9]+)', url)
    if not match: return []
    playlist_id = match.group(1)
    
    embed_url = f"https://open.spotify.com/embed/playlist/{playlist_id}"
    try:
        scraper = cloudscraper.create_scraper()
        html = await asyncio.to_thread(lambda: scraper.get(embed_url, timeout=10).text)
        soup = BeautifulSoup(html, 'html.parser')
        script = soup.find('script', id='__NEXT_DATA__')
        
        if not script:
            raise Exception("Gagal menemukan data playlist dari Spotify Embed")
            
        data = json.loads(script.string)
        track_list = data.get('props', {}).get('pageProps', {}).get('state', {}).get('data', {}).get('entity', {}).get('trackList', [])
        
        tracks = []
        for track in track_list:
            title = track.get('title', '')
            subtitle = track.get('subtitle', '')
            if title:
                tracks.append(f"{title} {subtitle}".strip())
                
        return tracks
    except Exception as e:
        raise Exception(f"Gagal ngambil playlist Spotify: {e}")

async def ekstrak_metadata_single_spotify(query):
    if "spotify.com/track/" in query:
        match = re.search(r'track/([a-zA-Z0-9]+)', query)
        if match:
            track_id = match.group(1)
            embed_url = f"https://open.spotify.com/embed/track/{track_id}"
            try:
                scraper = cloudscraper.create_scraper()
                html = await asyncio.to_thread(lambda: scraper.get(embed_url, timeout=10).text)
                soup = BeautifulSoup(html, 'html.parser')
                script = soup.find('script', id='__NEXT_DATA__')
                if script:
                    data = json.loads(script.string)
                    entity = data.get('props', {}).get('pageProps', {}).get('state', {}).get('data', {}).get('entity', {})
                    title = entity.get('name', entity.get('title', ''))
                    artists = " ".join([a.get('name', '') for a in entity.get('artists', [])])
                    if title:
                        return f"{title} {artists}".strip()
            except Exception as e:
                print(f"[TideTunes] Gagal ekstrak Spotify single metadata: {e}")
    return query

# ─── Slash Commands ───────────────────────────────────────────────────────────

@bot.tree.command(name="play", description="Putar lagu atau baca link playlist (Spotify/YouTube/SoundCloud)")
@app_commands.describe(query="Judul lagu yang mau diputar")
async def play(interaction: discord.Interaction, query: str):
    if not interaction.user.voice:
        await interaction.response.send_message("❌ Lu harus join Voice Channel dulu!", ephemeral=True)
        return
        
    await interaction.response.defer(ephemeral=True)
    channel = interaction.user.voice.channel
    player: wavelink.Player = interaction.guild.voice_client
    
    if not player:
        try:
            player = await channel.connect(cls=wavelink.Player)
            player.autoplay = wavelink.AutoPlayMode.partial
        except Exception as e:
            return await interaction.followup.send(f"❌ Gagal masuk ke Voice Channel: {e}")
    elif player.channel.id != channel.id:
        await player.move_to(channel)

    # Scrape Spotify
    if "spotify.com/playlist/" in query:
        await interaction.followup.send("🔍 Mengambil data dari Spotify Playlist...")
        playlist_tracks = await ambil_playlist_spotify(query)
        if not playlist_tracks:
            return await interaction.followup.send("❌ Gagal mendapatkan lagu dari playlist atau playlist kosong.")
        
        first_track = playlist_tracks.pop(0)
        tracks = await wavelink.Playable.search(first_track, source=wavelink.TrackSource.SoundCloud)
        if tracks:
            track = tracks[0]
            player.queue.put(track)
            if not player.playing:
                await player.play(player.queue.get())
                await interaction.followup.send(f"🎵 Memutar lagu pertama: **{track.title}**\n📝 **{len(playlist_tracks)} lagu lainnya** sedang dicari di background!")
            else:
                await interaction.followup.send(f"📝 Berhasil memasukkan **{len(playlist_tracks) + 1} lagu** ke dalam antrean (via background)!")
        
        async def fetch_background():
            for t in playlist_tracks:
                try:
                    res = await wavelink.Playable.search(t, source=wavelink.TrackSource.SoundCloud)
                    if res:
                        player.queue.put(res[0])
                except Exception as e:
                    print(f"[TideTunes] Gagal memuat {t}: {e}")
                # [REFACTOR] Hindari API rate limit dari SoundCloud / Lavalink dengan memberi nafas 0.5 detik
                await asyncio.sleep(0.5)
                
        bot.loop.create_task(fetch_background())
        return

    if "spotify.com/track/" in query:
        query = await ekstrak_metadata_single_spotify(query)

    try:
        # Pindah ke SoundCloud sebagai default karena YouTube sedang memblokir total IP server dan API
        tracks = await wavelink.Playable.search(query, source=wavelink.TrackSource.SoundCloud)
        if not tracks:
            return await interaction.followup.send("❌ Gagal menemukan lagu tersebut di SoundCloud.")

        if isinstance(tracks, wavelink.Playlist):
            added = player.queue.put(tracks)
            if not player.playing:
                first_track = player.queue.get()
                await player.play(first_track)
                await interaction.followup.send(f"🎵 Memutar lagu pertama: **{first_track.title}**\n📝 **{added - 1} lagu lainnya** berhasil dimasukkan ke antrean!")
            else:
                await interaction.followup.send(f"📝 Berhasil memasukkan playlist berisi **{added} lagu** ke dalam antrean!")
        else:
            track = tracks[0]
            player.queue.put(track)
            if not player.playing:
                await player.play(player.queue.get())
                await interaction.followup.send(f"🎵 Sedang memutar: **{track.title}**")
            else:
                await interaction.followup.send(f"📝 Dimasukkan ke antrean: **{track.title}**")
                
    except Exception as e:
        await interaction.followup.send(f"❌ Error saat memutar lagu: {e}")

@bot.tree.command(name="skip", description="Lewati lagu yang sedang diputar")
async def skip(interaction: discord.Interaction):
    player: wavelink.Player = interaction.guild.voice_client
    if player and player.playing:
        await player.skip(force=True)
        await interaction.response.send_message("⏭️ Lagu dilewati!", ephemeral=True)
    else:
        await interaction.response.send_message("❌ Nggak ada lagu yang lagi diputar.", ephemeral=True)

@bot.tree.command(name="stop", description="Hentikan musik dan bot keluar dari VC")
async def stop(interaction: discord.Interaction):
    player: wavelink.Player = interaction.guild.voice_client
    if player:
        player.queue.clear()
        _clear_guild_state(interaction.guild_id)
        await player.disconnect()
        await interaction.response.send_message("🛑 Musik dihentikan dan TideTunes keluar dari VC.", ephemeral=True)
    else:
        await interaction.response.send_message("❌ TideTunes nggak lagi di Voice Channel.", ephemeral=True)

@bot.tree.command(name="queue", description="Lihat antrean lagu")
async def queue(interaction: discord.Interaction):
    player: wavelink.Player = interaction.guild.voice_client
    if player and not player.queue.is_empty:
        q = list(player.queue)
        tampil = q[:15]
        antrean = "\n".join([f"{i+1}. {track.title} - {track.author}" for i, track in enumerate(tampil)])
        if len(q) > 15:
            antrean += f"\n\n... dan **{len(q) - 15} lagu lainnya**"
        antrean += f"\n\n📊 Total: **{len(q)} lagu** dalam antrean"
        await interaction.response.send_message(f"🎶 **Antrean Lagu:**\n{antrean}", ephemeral=True)
    else:
        await interaction.response.send_message("📭 Antrean kosong.", ephemeral=True)

@bot.tree.command(name="autoplay", description="Nyalakan/matikan AI Auto-DJ Gemini")
@app_commands.choices(status=[
    app_commands.Choice(name="On", value="on"),
    app_commands.Choice(name="Off", value="off")
])
async def autoplay(interaction: discord.Interaction, status: app_commands.Choice[str]):
    guild_id = interaction.guild_id
    
    if status.value == "on":
        if not GEMINI_KEY:
            return await interaction.response.send_message("❌ AI Auto-DJ butuh `GEMINI_API_KEY` di file `.env`.", ephemeral=True)
        autoplay_status[guild_id] = True
        await interaction.response.send_message("🤖 **AI Auto-DJ diaktifkan!** TideTunes bakal nyariin lagu otomatis kalau antrean habis.", ephemeral=True)
        player: wavelink.Player = interaction.guild.voice_client
        if player:
            asyncio.run_coroutine_threadsafe(process_autoplay(guild_id, player), bot.loop)
    else:
        autoplay_status[guild_id] = False
        await interaction.response.send_message("🛑 **AI Auto-DJ dimatikan!** TideTunes akan keluar kalau antrean lagu habis.", ephemeral=True)

@bot.tree.command(name="pause", description="Jeda lagu yang sedang diputar")
async def pause(interaction: discord.Interaction):
    player: wavelink.Player = interaction.guild.voice_client
    if player and player.playing and not player.paused:
        await player.pause(True)
        await interaction.response.send_message("⏸️ Lagu dijeda.", ephemeral=True)
    else:
        await interaction.response.send_message("❌ Nggak ada lagu yang lagi diputar.", ephemeral=True)

@bot.tree.command(name="resume", description="Lanjutkan lagu yang dijeda")
async def resume(interaction: discord.Interaction):
    player: wavelink.Player = interaction.guild.voice_client
    if player and player.paused:
        await player.pause(False)
        await interaction.response.send_message("▶️ Lagu dilanjutkan.", ephemeral=True)
    else:
        await interaction.response.send_message("❌ Nggak ada lagu yang lagi dijeda.", ephemeral=True)

@bot.tree.command(name="nowplaying", description="Lihat lagu yang sedang diputar")
async def nowplaying(interaction: discord.Interaction):
    player: wavelink.Player = interaction.guild.voice_client
    if not player or not player.current:
        return await interaction.response.send_message("❌ Nggak ada lagu yang lagi diputar.", ephemeral=True)

    track = player.current
    status_icon = "⏸️ Dijeda" if player.paused else "▶️ Sedang Diputar"

    if track.length:
        mins, secs = divmod(int(track.length / 1000), 60)
        dur_str = f"{mins}:{secs:02d}"
    else:
        dur_str = "Live / Tidak diketahui"

    embed = discord.Embed(
        title="🎵 Now Playing",
        description=f"**{track.title}** - {track.author}",
        color=discord.Color.from_rgb(30, 215, 96),
        url=track.uri or ""
    )
    embed.add_field(name="Status", value=status_icon, inline=True)
    embed.add_field(name="Durasi", value=dur_str, inline=True)
    embed.add_field(name="Antrean", value=f"{len(player.queue)} lagu", inline=True)

    if getattr(track, "artwork", None):
        embed.set_thumbnail(url=track.artwork)

    auto_dj = "🟢 Aktif" if autoplay_status.get(interaction.guild_id, False) else "🔴 Mati"
    embed.set_footer(text=f"AI Auto-DJ: {auto_dj}")

    await interaction.response.send_message(embed=embed, ephemeral=True)

@bot.tree.command(name="volume", description="Atur volume musik (0-100)")
@app_commands.describe(level="Volume level (0-100)")
async def volume(interaction: discord.Interaction, level: int):
    if level < 0 or level > 100:
        return await interaction.response.send_message("❌ Volume harus antara 0 dan 100.", ephemeral=True)

    player: wavelink.Player = interaction.guild.voice_client
    if player:
        await player.set_volume(level)
        emoji = "🔊" if level >= 70 else "🔉" if level >= 30 else "🔈" if level > 0 else "🔇"
        await interaction.response.send_message(f"{emoji} Volume diset ke **{level}%**", ephemeral=True)
    else:
        await interaction.response.send_message("❌ Nggak ada lagu yang lagi diputar.", ephemeral=True)

@bot.tree.command(name="shuffle", description="Acak urutan antrean lagu")
async def shuffle(interaction: discord.Interaction):
    player: wavelink.Player = interaction.guild.voice_client
    if player and len(player.queue) > 1:
        player.queue.shuffle()
        await interaction.response.send_message(f"🔀 Antrean diacak! ({len(player.queue)} lagu)", ephemeral=True)
    else:
        await interaction.response.send_message("❌ Antrean kosong atau cuma ada 1 lagu.", ephemeral=True)

@bot.tree.command(name="remove", description="Hapus lagu dari antrean berdasarkan nomor")
@app_commands.describe(nomor="Nomor lagu di antrean (lihat /queue)")
async def remove(interaction: discord.Interaction, nomor: int):
    player: wavelink.Player = interaction.guild.voice_client
    if not player or player.queue.is_empty:
        return await interaction.response.send_message("❌ Antrean kosong.", ephemeral=True)

    if nomor < 1 or nomor > len(player.queue):
        return await interaction.response.send_message(f"❌ Nomor nggak valid. Pilih antara 1 - {len(player.queue)}.", ephemeral=True)

    removed = player.queue.delete(nomor - 1)
    await interaction.response.send_message(f"🗑️ Dihapus dari antrean: **{removed.title}**", ephemeral=True)

# ─── Entry Point ──────────────────────────────────────────────────────────────

def jalankan():
    if TOKEN:
        bot.run(TOKEN)
    else:
        print("[TideTunes] Token tidak ditemukan di .env (TIDETUNES_TOKEN). Bot tidak dijalankan.")

if __name__ == "__main__":
    jalankan()
