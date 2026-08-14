// ============================================================================
// KONFIGURASI API
// ============================================================================
const TMDB_API_KEY = "cff0f315183dd0830f0ef2ef924ae25c";
const TMDB_BASE_URL = "https://api.themoviedb.org/3";
const TMDB_IMAGE_URL = "https://image.tmdb.org/t/p/original";
const TMDB_POSTER_URL = "https://image.tmdb.org/t/p/w500";
// [BUG FIX] Dinamisasi URL Backend agar jalan di VPS (Localhost Trap)
const isLocal = window.location.hostname === "localhost" || window.location.hostname === "127.0.0.1" || window.location.hostname === "";
const BACKEND_URL = isLocal ? "http://localhost:8080" : "http://IP_VPS_ANDA:8080"; 

// ============================================================================
// UI LOGIC
// ============================================================================

window.addEventListener('scroll', () => {
    const navbar = document.getElementById('navbar');
    if (window.scrollY > 50) {
        navbar.classList.add('scrolled');
    } else {
        navbar.classList.remove('scrolled');
    }
});

// Matikan lompatan layar saat navbar kosong diklik
document.querySelectorAll('.nav-links a').forEach(link => {
    link.addEventListener('click', (e) => {
        if(link.getAttribute('href') === '#') e.preventDefault();
    });
});

const playerModal = document.getElementById('player-modal');
const iframe = document.getElementById('video-frame');
const detailsModal = document.getElementById('details-modal');

let currentHeroMovie = null;
let currentDetailsMovie = null;

function closePlayer() {
    playerModal.classList.remove('active');
    iframe.src = ""; 
    // Jangan ubah overflow jika details modal masih aktif
    if (!detailsModal.classList.contains('active')) {
        document.body.classList.remove('modal-open');
        document.body.style.overflow = 'auto';
    }
}

function closeDetails() {
    detailsModal.classList.remove('active');
    document.body.classList.remove('modal-open');
    document.body.style.overflow = 'auto';
}

// Tambahkan UX Kontrol Tombol ESC (Windows Habit)
document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') {
        if (playerModal.classList.contains('active')) {
            closePlayer();
        } else if (detailsModal.classList.contains('active')) {
            closeDetails();
        }
    }
});

playerModal.addEventListener('click', (e) => {
    if (e.target === playerModal) closePlayer();
});
detailsModal.addEventListener('click', (e) => {
    if (e.target === detailsModal) closeDetails();
});

// ============================================================================
// MY LIST LOGIC (LOCAL STORAGE)
// ============================================================================

function getMyList() {
    const list = localStorage.getItem('summertide_mylist');
    return list ? JSON.parse(list) : [];
}

function saveMyList(list) {
    localStorage.setItem('summertide_mylist', JSON.stringify(list));
}

function toggleMyList() {
    if (!currentDetailsMovie) return;
    let list = getMyList();
    const index = list.findIndex(m => m.id === currentDetailsMovie.id);
    const btn = document.getElementById('details-mylist-btn');
    
    if (index > -1) {
        list.splice(index, 1);
        btn.innerHTML = '<i class="fas fa-plus"></i> Daftar Saya';
    } else {
        list.push(currentDetailsMovie);
        btn.innerHTML = '<i class="fas fa-check"></i> Tersimpan';
    }
    saveMyList(list);
    renderMyList();
}

document.getElementById('details-mylist-btn').addEventListener('click', toggleMyList);

// ============================================================================
// API LOGIC (TMDB & GOLANG)
// ============================================================================

async function fetchTMDB(endpoint) {
    if (TMDB_API_KEY === "ISI_DENGAN_API_KEY_TMDB_ANDA") return { results: [] };
    const char = endpoint.includes('?') ? '&' : '?';
    try {
        const response = await fetch(`${TMDB_BASE_URL}${endpoint}${char}api_key=${TMDB_API_KEY}&language=id-ID`);
        return await response.json();
    } catch (error) {
        console.error("TMDB Error:", error);
        return { results: [] };
    }
}

// Modifikasi PlayMovie untuk mensupport tipe (movie/tv), season, dan episode
async function playMovie(title, year, type = 'movie', season = '', episode = '') {
    const notifMsg = type === 'tv' ? `(S${season}E${episode})` : `(${year})`;
    const loadingOverlay = document.getElementById('loading-overlay');
    loadingOverlay.classList.add('active');
    
    try {
        const url = `${BACKEND_URL}/api/play?title=${encodeURIComponent(title)}&year=${year}&type=${type}&season=${season}&episode=${episode}`;
        const response = await fetch(url);
        const data = await response.json();
        
        loadingOverlay.classList.remove('active');
        
        if (data.iframeUrl) {
            iframe.src = data.iframeUrl;
            playerModal.classList.add('active');
            document.body.classList.add('modal-open');
        } else {
            alert("Maaf, video tidak ditemukan di server penyedia.");
        }
    } catch (error) {
        loadingOverlay.classList.remove('active');
        alert("Gagal menghubungi server backend Golang.");
    }
}

function playHeroMovie() {
    if (currentHeroMovie) {
        const year = (currentHeroMovie.release_date || currentHeroMovie.first_air_date || "").substring(0, 4);
        playMovie(currentHeroMovie.title || currentHeroMovie.name, year, currentHeroMovie.media_type || 'movie');
    }
}

document.getElementById('details-play-btn').addEventListener('click', () => {
    if (currentDetailsMovie) {
        const title = currentDetailsMovie.title || currentDetailsMovie.name;
        const isTv = currentDetailsMovie.media_type === 'tv' || currentDetailsMovie.first_air_date;
        const type = isTv ? 'tv' : 'movie';
        let year = "";
        
        if (isTv) {
            year = currentDetailsMovie.first_air_date ? currentDetailsMovie.first_air_date.substring(0, 4) : "";
            // Default mainkan S1 E1 jika klik putar utama
            playMovie(title, year, type, 1, 1);
        } else {
            year = currentDetailsMovie.release_date ? currentDetailsMovie.release_date.substring(0, 4) : "";
            playMovie(title, year, type);
        }
    }
});

// ============================================================================
// DETAILS MODAL LOGIC
// ============================================================================

async function openDetails(movie) {
    currentDetailsMovie = movie;
    const isTv = movie.media_type === 'tv' || movie.first_air_date;
    const title = movie.title || movie.name || "Tanpa Judul";
    const year = (isTv ? movie.first_air_date : movie.release_date) || "";
    
    document.getElementById('details-title').textContent = title;
    document.getElementById('details-year').textContent = year.substring(0,4);
    document.getElementById('details-desc').textContent = movie.overview || "Tidak ada deskripsi.";
    document.getElementById('details-match').textContent = movie.vote_average ? Math.round(movie.vote_average * 10) + "% Match" : "N/A";
    
    const bgUrl = movie.backdrop_path ? `${TMDB_IMAGE_URL}${movie.backdrop_path}` : 'https://images.unsplash.com/photo-1489599849927-2ee91cede3ba?q=80&w=2070';
    document.getElementById('details-hero').style.backgroundImage = `url('${bgUrl}')`;
    
    // Cek My List Status
    const list = getMyList();
    const btn = document.getElementById('details-mylist-btn');
    if (list.find(m => m.id === movie.id)) {
        btn.innerHTML = '<i class="fas fa-check"></i> Tersimpan';
    } else {
        btn.innerHTML = '<i class="fas fa-plus"></i> Daftar Saya';
    }
    
    const seriesSection = document.getElementById('series-section');
    if (isTv) {
        seriesSection.style.display = 'block';
        await fetchTVSeasons(movie.id);
    } else {
        seriesSection.style.display = 'none';
    }
    
    detailsModal.classList.add('active');
    document.body.classList.add('modal-open');
}

async function fetchTVSeasons(tvId) {
    const data = await fetchTMDB(`/tv/${tvId}`);
    const selector = document.getElementById('season-selector');
    selector.innerHTML = "";
    
    if (data.seasons && data.seasons.length > 0) {
        // Abaikan season 0 (Specials)
        const validSeasons = data.seasons.filter(s => s.season_number > 0);
        validSeasons.forEach(s => {
            const opt = document.createElement('option');
            opt.value = s.season_number;
            opt.textContent = `Season ${s.season_number}`;
            selector.appendChild(opt);
        });
        if(validSeasons.length > 0) {
            loadEpisodes(validSeasons[0].season_number);
        }
    }
}

window.loadEpisodes = async function(seasonNumber) {
    if (!currentDetailsMovie) return;
    const tvId = currentDetailsMovie.id;
    const data = await fetchTMDB(`/tv/${tvId}/season/${seasonNumber}`);
    const listContainer = document.getElementById('episode-list');
    listContainer.innerHTML = "";
    
    if (data.episodes) {
        const fragment = document.createDocumentFragment();
        data.episodes.forEach(ep => {
            const item = document.createElement('div');
            item.className = 'episode-item';
            
            const img = ep.still_path ? `${TMDB_POSTER_URL}${ep.still_path}` : 'https://images.unsplash.com/photo-1489599849927-2ee91cede3ba?q=80&w=2070';
            const epNum = ep.episode_number;
            
            item.onclick = () => {
                const title = currentDetailsMovie.name;
                const year = currentDetailsMovie.first_air_date ? currentDetailsMovie.first_air_date.substring(0, 4) : "";
                playMovie(title, year, 'tv', seasonNumber, epNum);
            };
            
            item.tabIndex = 0;
            item.onkeydown = (e) => { if(e.key === 'Enter') item.onclick(); };
            
            item.innerHTML = `
                <img src="${img}" alt="${ep.name}" loading="lazy">
                <div class="episode-info">
                    <h4>${epNum}. ${escapeHTML(ep.name)}</h4>
                    <span>${ep.runtime ? ep.runtime + 'm' : ''}</span>
                </div>
            `;
            fragment.appendChild(item);
        });
        listContainer.appendChild(fragment);
    }
};

// ============================================================================
// RENDER LOGIC
// ============================================================================

function setHeroMovie(movie) {
    currentHeroMovie = movie;
    const bgUrl = movie.backdrop_path ? `${TMDB_IMAGE_URL}${movie.backdrop_path}` : 'https://images.unsplash.com/photo-1489599849927-2ee91cede3ba?q=80&w=2070';
    document.getElementById('hero-section').style.backgroundImage = `url('${bgUrl}')`;
    document.getElementById('hero-title').textContent = movie.title || movie.name;
    
    let desc = movie.overview || "Tidak ada deskripsi tersedia.";
    if (desc.length > 200) desc = desc.substring(0, 200) + "...";
    document.getElementById('hero-desc').textContent = desc;
    
    const year = (movie.release_date || movie.first_air_date || "").substring(0, 4);
    document.querySelector('.hero-meta .year').textContent = year;
    document.querySelector('.hero-meta .match').textContent = movie.vote_average ? Math.round(movie.vote_average * 10) + "% Match" : "N/A";
}

function escapeHTML(str) {
    const p = document.createElement('p');
    p.appendChild(document.createTextNode(str));
    return p.innerHTML;
}

function renderMovieRow(containerId, movies) {
    const container = document.getElementById(containerId);
    if(!container) return;
    container.innerHTML = ""; 
    const fragment = document.createDocumentFragment();
    
    movies.forEach(movie => {
        if (!movie.poster_path) return; 
        const card = document.createElement('div');
        card.className = 'movie-card';
        
        const isTv = movie.media_type === 'tv' || movie.first_air_date;
        const year = (isTv ? movie.first_air_date : movie.release_date) || "";
        const title = movie.title || movie.name || "Tanpa Judul";
        const rating = movie.vote_average ? movie.vote_average.toFixed(1) : "N/A";
        const safeTitle = escapeHTML(title);
        
        // Buka Details Modal alih-alih langsung memutar
        card.onclick = () => openDetails(movie);
        
        // [A11Y FIX] Tambahkan aksesibilitas keyboard
        card.tabIndex = 0;
        card.onkeydown = (e) => { if(e.key === 'Enter') openDetails(movie); };
        
        card.innerHTML = `
            <div class="poster-container">
                <img src="${TMDB_POSTER_URL}${movie.poster_path}" alt="${safeTitle}" loading="lazy">
                <span class="quality-badge">${isTv ? 'SERIES' : 'HD'}</span>
                <span class="rating-badge"><i class="fas fa-star"></i> ${rating}</span>
            </div>
            <div class="movie-info">
                <h4>${safeTitle}</h4>
                <span class="year-text">${year.substring(0,4)}</span>
            </div>
        `;
        fragment.appendChild(card);
    });
    container.appendChild(fragment); 
}

function renderMyList() {
    let row = document.getElementById('mylist-row');
    if (!row) {
        const main = document.querySelector('.content');
        const section = document.createElement('section');
        section.className = 'movie-row';
        section.innerHTML = `
            <h3 class="row-title">Daftar Saya</h3>
            <div class="row-container" id="mylist-row"></div>
        `;
        main.insertBefore(section, main.firstChild);
        row = document.getElementById('mylist-row');
    }
    
    const list = getMyList();
    if(list.length > 0) {
        row.parentElement.style.display = 'block';
        renderMovieRow('mylist-row', list.reverse());
    } else {
        row.parentElement.style.display = 'none';
    }
}

// ============================================================================
// INISIALISASI (IDLIX FIRST ARCHITECTURE)
// ============================================================================

async function mapTitlesToTMDB(titles) {
    const movies = [];
    for (let title of titles) {
        // Bersihkan judul dari tulisan berlebih seperti "Season X" atau "Episode Y"
        let cleanTitle = title.replace(/season \d+/i, '').replace(/episode \d+/i, '').trim();
        const searchData = await fetchTMDB(`/search/multi?query=${encodeURIComponent(cleanTitle)}`);
        // Pastikan bukan array kosong
        if (searchData.results && searchData.results.length > 0) {
            movies.push(searchData.results[0]);
        }
    }
    return movies;
}

async function initApp() {
    renderMyList();
    
    try {
        const response = await fetch(`${BACKEND_URL}/api/homepage`);
        const data = await response.json();
        if (data.titles && data.titles.length > 0) {
            const movies = await mapTitlesToTMDB(data.titles);
            if (movies.length > 0) {
                setHeroMovie(movies[0]);
                renderMovieRow('trending-row', movies.slice(1));
            }
        }
    } catch(err) {
        console.error("Gagal load homepage:", err);
    }
    
    // Sembunyikan Action Row karena Idlix homepage scraper hanya mengambil 1 list utama
    const actionRow = document.getElementById('action-row');
    if (actionRow) {
        actionRow.parentElement.style.display = 'none';
    }
    
    checkURLParams();
}

async function checkURLParams() {
    const params = new URLSearchParams(window.location.search);
    const playTitle = params.get('play');
    const playYear = params.get('year');
    if (playTitle) {
        // Cari di TMDB untuk dapet objek movie, lalu buka details modal
        const searchData = await fetchTMDB(`/search/multi?query=${encodeURIComponent(playTitle)}`);
        if (searchData.results && searchData.results.length > 0) {
            let matched = searchData.results[0];
            if (playYear) {
                const exact = searchData.results.find(m => {
                    const y = (m.release_date || m.first_air_date || "").substring(0, 4);
                    return y === playYear;
                });
                if (exact) matched = exact;
            }
            openDetails(matched);
        }
    }
}

document.addEventListener('DOMContentLoaded', () => {
    initApp();
    const searchInput = document.getElementById('search-input');
    let searchTimeout;
    searchInput.addEventListener('input', (e) => {
        clearTimeout(searchTimeout);
        const query = e.target.value.trim();
        searchTimeout = setTimeout(async () => {
            if (query.length > 2) {
                const loadingRow = document.querySelector('#trending-row');
                loadingRow.innerHTML = "<p style='color:white; padding: 20px;'>Mencari di Idlix (Menembus Cloudflare, harap tunggu)...</p>";
                
                try {
                    const response = await fetch(`${BACKEND_URL}/api/search?q=${encodeURIComponent(query)}`);
                    const data = await response.json();
                    
                    const rowTitle = document.querySelector('#trending-row').previousElementSibling;
                    rowTitle.textContent = `Hasil Pencarian: "${query}"`;
                    
                    if (data.titles && data.titles.length > 0) {
                        const movies = await mapTitlesToTMDB(data.titles);
                        renderMovieRow('trending-row', movies);
                    } else {
                        loadingRow.innerHTML = "<p style='color:#ccc; padding: 20px;'>Tidak ada hasil di Idlix untuk pencarian ini.</p>";
                    }
                } catch (error) {
                    loadingRow.innerHTML = "<p style='color:red; padding: 20px;'>Gagal menghubungi server pencarian.</p>";
                }
            } else if (query.length === 0) {
                const rowTitle = document.querySelector('#trending-row').previousElementSibling;
                rowTitle.textContent = "Sedang Hangat (Trending)";
                initApp();
            }
        }, 1000); 
    });
});
