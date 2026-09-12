import torrentStream from 'torrent-stream';
import fetch from 'node-fetch';

process.env.NODE_TLS_REJECT_UNAUTHORIZED = "0"; 


async function fetchTMDB(endpoint) {
    const TMDB_KEY = process.env.TMDB_API_KEY || 'cff0f315183dd0830f0ef2ef924ae25c';
    const url = `https://api.themoviedb.org/3${endpoint}?api_key=${TMDB_KEY}`;
    const res = await fetch(url);
    if (!res.ok) throw new Error(`TMDB error: ${res.status}`);
    return await res.json();
}


export const activeEngines = new Map(); 

export async function getTorrentInfo(tmdbId, hostUrl) {
    if (activeEngines.has(tmdbId)) {
        return {
            provider: 'Torrent Stream',
            sources: [{ file: `${hostUrl}/stream-video/${tmdbId}`, type: 'mp4' }],
            tracks: []
        };
    }

    try {
        const externalIds = await fetchTMDB(`/movie/${tmdbId}/external_ids`);
        const imdbId = externalIds.imdb_id;
        if (!imdbId) throw new Error('No IMDB ID found for TMDB ID ' + tmdbId);

        console.log(`[torrent] Searching YTS for ${imdbId}...`);
        
        const ytsRes = await fetch(`https://yts.mx/api/v2/list_movies.json?query_term=${imdbId}`, {
            headers: {
                'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/137.0.0.0 Safari/537.36'
            }
        });
        const ytsText = await ytsRes.text();
        let ytsData;
        try {
            ytsData = JSON.parse(ytsText);
        } catch (e) {
            console.log(`[torrent] Error parsing YTS response: ${ytsText.substring(0, 100)}`);
            return null;
        }
        
        if (!ytsData.data || !ytsData.data.movies || ytsData.data.movies.length === 0) {
            console.log(`[torrent] Movie not found on YTS, falling back to Puppeteer...`);
            return null;
        }
        
        const movie = ytsData.data.movies[0];
        let selectedTorrent = movie.torrents.find(t => t.quality === '1080p') || movie.torrents[0];
        
        const magnetURI = `magnet:?xt=urn:btih:${selectedTorrent.hash}&dn=${encodeURIComponent(movie.title)}&tr=udp://open.demonii.com:1337/announce&tr=udp://tracker.openbittorrent.com:80`;
        console.log(`[torrent] Found magnet: ${magnetURI}`);

        return new Promise((resolve, reject) => {
            const engine = torrentStream(magnetURI, { connections: 100 });
            
            engine.on('ready', () => {
                const file = engine.files.reduce((a, b) => a.length > b.length ? a : b);
                console.log(`[torrent] Selected file: ${file.name}`);
                
                activeEngines.set(tmdbId, { engine, file });
                
                resolve({
                    provider: 'Torrent Stream',
                    sources: [{ file: `${hostUrl}/stream-video/${tmdbId}`, type: 'mp4' }],
                    tracks: []
                });
            });
            
            
            setTimeout(() => {
                if (!activeEngines.has(tmdbId)) reject(new Error('Torrent timeout'));
            }, 10000);
        });
    } catch (e) {
        console.error('[torrent] Error:', e.message);
        return null;
    }
}
