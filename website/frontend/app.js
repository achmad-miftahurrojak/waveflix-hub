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

function getIframeUrl(tmdbId, type, season, episode) {
    if (type === 'tv') {
        return `https://vidsrc.me/embed/tv?tmdb=${tmdbId}&season=${season}&episode=${episode}`;
    }
    return `https://vidsrc.me/embed/movie?tmdb=${tmdbId}`;
}

async function playMovie(tmdbId, type = 'movie', season = '', episode = '') {
    const loadingOverlay = document.getElementById('loading-overlay');
    loadingOverlay.classList.add('active');
    
    currentPlaying = { tmdbId, type, season, episode };
    
    setTimeout(() => {
        loadingOverlay.classList.remove('active');
        
        iframe.src = getIframeUrl(tmdbId, type, season, episode);
        playerModal.classList.add('active');
        document.body.classList.add('modal-open');
    }, 500);
}



function playHeroMovie() {
    if (currentHeroMovie) {
        playMovie(currentHeroMovie.id, currentHeroMovie.media_type || 'movie');
    }
}

document.getElementById('details-play-btn').addEventListener('click', () => {
    if (currentDetailsMovie) {
        const isTv = currentDetailsMovie.media_type === 'tv' || currentDetailsMovie.first_air_date;
        const type = isTv ? 'tv' : 'movie';
        
        if (isTv) {
            // Default mainkan S1 E1 jika klik putar utama
            playMovie(currentDetailsMovie.id, type, 1, 1);
        } else {
            playMovie(currentDetailsMovie.id, type);
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
                playMovie(currentDetailsMovie.id, 'tv', seasonNumber, epNum);
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

function renderMovieRow(containerId, movies, append = false) {
    const container = document.getElementById(containerId);
    if(!container) return;
    if(!append) container.innerHTML = ""; 
    
    // [UI FIX] Inject carousel buttons if they don't exist
    const section = container.parentElement;
    if (!section.querySelector('.carousel-btn.left')) {
        const leftBtn = document.createElement('button');
        leftBtn.className = 'carousel-btn left';
        leftBtn.innerHTML = '<i class="fas fa-chevron-left"></i>';
        leftBtn.onclick = () => { container.scrollBy({ left: -800, behavior: 'smooth' }) };
        
        const rightBtn = document.createElement('button');
        rightBtn.className = 'carousel-btn right';
        rightBtn.innerHTML = '<i class="fas fa-chevron-right"></i>';
        rightBtn.onclick = () => { container.scrollBy({ left: 800, behavior: 'smooth' }) };
        
        section.appendChild(leftBtn);
        section.appendChild(rightBtn);
        
        // Munculkan panah hanya saat kursor di atas barisan film
        section.addEventListener('mouseenter', () => {
            leftBtn.style.display = 'block';
            rightBtn.style.display = 'block';
        });
        section.addEventListener('mouseleave', () => {
            leftBtn.style.display = 'none';
            rightBtn.style.display = 'none';
        });
    }

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



const genres = [
    { id: 'trending', endpoint: '/api/homepage' },
    { id: 'action', endpoint: '/api/discover?genre=28' },
    { id: 'drama', endpoint: '/api/discover?genre=18' },
    { id: 'animation', endpoint: '/api/discover?genre=16' },
    { id: 'horror', endpoint: '/api/discover?genre=27' }
];
let genrePages = { trending: 1, action: 1, drama: 1, animation: 1, horror: 1 };
let isLoadingGenre = { trending: false, action: false, drama: false, animation: false, horror: false };

async function loadGenre(genreId, page) {
    if (isLoadingGenre[genreId]) return;
    isLoadingGenre[genreId] = true;
    
    const genreObj = genres.find(g => g.id === genreId);
    let url = `${BACKEND_URL}${genreObj.endpoint}`;
    if (url.includes('?')) {
        url += `&page=${page}`;
    } else {
        url += `?page=${page}`;
    }

    try {
        const response = await fetch(url);
        const data = await response.json();
        if (data.results && data.results.length > 0) {
            const movies = data.results;
            
            if (genreId === 'trending' && page === 1 && movies.length > 0) {
                setHeroMovie(movies[0]);
            }
            
            // Tampilkan section jika ini page 1
            if (page === 1) {
                const section = document.getElementById(`${genreId}-section`);
                if (section) section.style.display = 'block';
            }
            
            const append = page > 1;
            // Hindari hero movie di row trending
            const moviesToRender = (genreId === 'trending' && page === 1) ? movies.slice(1) : movies;
            renderMovieRow(`${genreId}-row`, moviesToRender, append);
            
            setupInfiniteScroll(`${genreId}-row`, genreId);
        }
    } catch(err) {
        console.error(`Gagal load ${genreId}:`, err);
    }
    isLoadingGenre[genreId] = false;
}

function setupInfiniteScroll(rowId, genreId) {
    const row = document.getElementById(rowId);
    if (!row) return;
    
    row.addEventListener('scroll', () => {
        // Jika scroll mendekati akhir (sisa 300px)
        if (row.scrollLeft + row.clientWidth >= row.scrollWidth - 300) {
            if (!isLoadingGenre[genreId]) {
                genrePages[genreId]++;
                loadGenre(genreId, genrePages[genreId]);
            }
        }
    });
}

async function initApp() {
    renderMyList();
    
    // Load halaman 1 untuk semua genre secara paralel
    genres.forEach(g => {
        genrePages[g.id] = 1;
        loadGenre(g.id, 1);
    });
    
    const loadBtn = document.getElementById('load-more-btn');
    if(loadBtn) loadBtn.style.display = 'none'; // Sembunyikan karena sudah pakai infinite scroll
    
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
                loadingRow.innerHTML = "<p style='color:white; padding: 20px;'>Mencari (harap tunggu)...</p>";
                
                try {
                    const response = await fetch(`${BACKEND_URL}/api/search?q=${encodeURIComponent(query)}`);
                    const data = await response.json();
                    
                    const rowTitle = document.querySelector('#trending-row').previousElementSibling;
                    rowTitle.textContent = `Hasil Pencarian: "${query}"`;
                    
                    if (data.results && data.results.length > 0) {
                        const movies = data.results;
                        renderMovieRow('trending-row', movies);
                    } else {
                        loadingRow.innerHTML = "<p style='color:#ccc; padding: 20px;'>Tidak ada hasil untuk pencarian ini.</p>";
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
