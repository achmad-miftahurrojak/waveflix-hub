import aiohttp
import urllib.parse
import os
import re
from unidecode import unidecode
import cloudscraper
import asyncio

def _cek_ketersediaan_sinkron(url):
    try:
        teks = cloudscraper.create_scraper().get(url, timeout=5).text
        if '<title>IDLIX / Nonton Film' in teks:
            return False
        return True
    except Exception:
        return False

async def cari_film(query, batas=5):
    # [BUG FIX] Batasi jumlah hasil yang diverifikasi. Sebelumnya SEMUA hasil TMDB
    # (bisa 20+) diperiksa satu-satu ke idlix + tinyurl padahal hasil akhir cuma dipakai [:5].
    # Sekarang cukup verifikasi batas + 3 (margin buat yang ternyata ga tersedia).
    TMDB_API_KEY = os.getenv('TMDB_API_KEY')
    if not TMDB_API_KEY:
        raise ValueError('TMDB_API_KEY belum di-set di .env!')
    url = 'https://api.themoviedb.org/3/search/multi'
    headers = {'accept': 'application/json'}
    params = {'query': query, 'language': 'en-US'}
    if len(TMDB_API_KEY) > 50:
        headers['Authorization'] = f'Bearer {TMDB_API_KEY}'
    else:
        params['api_key'] = TMDB_API_KEY
    hasil = []
    async with aiohttp.ClientSession() as session:
        async with session.get(url, headers=headers, params=params) as response:
            if response.status != 200:
                print(f'Error TMDB: {response.status}')
                return []
            data = await response.json()
            hasil_mentah = []
            for item in data.get('results', []):
                tipe = item.get('media_type')
                if tipe not in ['movie', 'tv']:
                    continue
                judul = item.get('title') if tipe == 'movie' else item.get('name')
                tahun_mentah = item.get('release_date') if tipe == 'movie' else item.get('first_air_date')
                if not judul or not tahun_mentah:
                    continue
                tahun = tahun_mentah.split('-')[0]
                judul_roman = unidecode(judul)
                slug_judul = re.sub('[^a-zA-Z0-9\\s-]', '', judul_roman).strip().lower()
                slug_judul = re.sub('[\\s]+', '-', slug_judul)
                slug_lengkap = f'{slug_judul}-{tahun}'
                idlix_tipe = 'movie' if tipe == 'movie' else 'series'
                poster = ''
                if item.get('poster_path'):
                    poster = f"https://image.tmdb.org/t/p/w500{item.get('poster_path')}"
                if tipe == 'tv':
                    tv_url = f"https://api.themoviedb.org/3/tv/{item.get('id')}"
                    tv_params = {'language': 'en-US'}
                    if 'api_key' in params:
                        tv_params['api_key'] = params['api_key']
                    try:
                        async with session.get(tv_url, headers=headers, params=tv_params) as tv_resp:
                            if tv_resp.status == 200:
                                tv_data = await tv_resp.json()
                                valid_seasons = [s for s in tv_data.get('seasons', []) if s.get('season_number', 0) > 0 and s.get('poster_path')]
                                if valid_seasons:
                                    valid_seasons.sort(key=lambda x: x['season_number'], reverse=True)
                                    poster = f"https://image.tmdb.org/t/p/w500{valid_seasons[0]['poster_path']}"
                    except Exception as e:
                        print(f'Gagal ngambil poster season terbaru: {e}')
                hasil_mentah.append({'judul': f'{judul} ({tahun})', 'url': f'https://z2.idlixku.com/{idlix_tipe}/{slug_lengkap}', 'deskripsi': item.get('overview', 'Tidak ada deskripsi.'), 'poster': poster, 'tipe': idlix_tipe})

            async def verifikasi(item):
                tersedia = await asyncio.to_thread(_cek_ketersediaan_sinkron, item['url'])
                if tersedia:
                    import urllib.parse, urllib.request, urllib.error, random, string
                    
                    # Ekstrak slug
                    slug = item['url'].strip('/').split('/')[-1]
                    safe_slug = "".join([c for c in slug if c.isalnum() or c == '-'])
                    alias = "waveflix-" + safe_slug
                    
                    # Batasi alias maksimal 30 karakter agar tidak ditolak TinyURL (HTTP 400)
                    alias = alias[:30]
                    # Hilangkan strip di akhir jika ada akibat pemotongan
                    alias = alias.rstrip('-')
                    
                    # Menggunakan tinyurl
                    api_url_alias = f"https://tinyurl.com/api-create.php?url={urllib.parse.quote(item['url'])}&alias={alias}"
                    
                    try:
                        short_url = await asyncio.to_thread(lambda: urllib.request.urlopen(api_url_alias, timeout=5).read().decode('utf-8'))
                        item['url'] = short_url
                    except urllib.error.HTTPError as e:
                        # Jika alias sudah dipakai (422) atau alias tidak valid/kepanjangan (400)
                        if e.code in (422, 400):
                            try:
                                api_url_baru = f"https://tinyurl.com/api-create.php?url={urllib.parse.quote(item['url'])}"
                                short_url = await asyncio.to_thread(lambda: urllib.request.urlopen(api_url_baru, timeout=5).read().decode('utf-8'))
                                item['url'] = short_url
                            except Exception as e2:
                                print(f'Gagal memendekkan URL dengan tinyurl (tanpa alias): {e2}')
                        else:
                            print(f'Gagal memendekkan URL dengan tinyurl: {e}')
                    except Exception as e:
                        print(f'Gagal memendekkan URL dengan tinyurl: {e}')
                    return item
                return None
            # [BUG FIX] Hanya verifikasi segelintir calon, bukan semua hasil TMDB.
            # asyncio.gather tetap jalan paralel, tapi jumlahnya dibatasi supaya
            # idlix + tinyurl nggak kena banjir request dan hasil lebih cepet.
            target_verifikasi = min(len(hasil_mentah), batas + 3)
            tasks = [verifikasi(item) for item in hasil_mentah[:target_verifikasi]]
            verified_results = await asyncio.gather(*tasks)
            hasil = [v for v in verified_results if v is not None][:batas]
    return hasil

async def cari_trending():
    from bs4 import BeautifulSoup
    scraper = cloudscraper.create_scraper()
    try:
        html = await asyncio.to_thread(lambda: scraper.get('https://z2.idlixku.com/', timeout=10).text)
    except Exception as e:
        print(f'Gagal scrape idlix: {e}')
        return []
    soup = BeautifulSoup(html, 'html.parser')
    judul_trending = []
    h2 = soup.find('h2', string='Trending Now')
    if h2:
        ul = h2.find_next_sibling('ul')
        if ul:
            for a in ul.find_all('a')[:10]:
                judul_trending.append(a.text)
    if not judul_trending:
        return []
    hasil_akhir = []
    for judul in judul_trending:
        try:
            # [BUG FIX] Cuma butuh 1 hasil per judul trending, jadi verifikasi dibatasi
            # ke segelintir calon alih-alih semua hasil TMDB (hebatnya turun drastis).
            hasil_pencarian = await cari_film(judul, batas=1)
            if hasil_pencarian:
                hasil_akhir.append(hasil_pencarian[0])
        except Exception as e:
            print(f'Gagal cari metadata untuk {judul}: {e}')
    return hasil_akhir