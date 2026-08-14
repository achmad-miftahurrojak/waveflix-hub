// ============================================================================
// KONFIGURASI API
// ============================================================================
const TMDB_API_KEY = "cff0f315183dd0830f0ef2ef924ae25c";
const TMDB_BASE_URL = "https://api.themoviedb.org/3";
const TMDB_IMAGE_URL = "https://image.tmdb.org/t/p/original";
const TMDB_POSTER_URL = "https://image.tmdb.org/t/p/w500";
const BACKEND_URL = "http://localhost:8080"; // URL Golang Backend

// ============================================================================
// UI LOGIC
// ============================================================================

// Navbar Scroll Effect
window.addEventListener('scroll', () => {
    const navbar = document.getElementById('navbar');
    if (window.scrollY > 50) {
        navbar.classList.add('scrolled');
    } else {
        navbar.classList.remove('scrolled');
    }
});

// Modal Player Logic
const modal = document.getElementById('player-modal');
const iframe = document.getElementById('video-frame');
let currentHeroMovie = null;

function closePlayer() {
    modal.classList.remove('active');
    iframe.src = ""; // Stop video
    document.body.style.overflow = 'auto';
}

modal.addEventListener('click', (e) => {
    if (e.target === modal) {
        closePlayer();
    }
});

// ============================================================================
// API LOGIC (TMDB & GOLANG)
// ============================================================================

async function fetchTMDB(endpoint) {
    if (TMDB_API_KEY === "ISI_DENGAN_API_KEY_TMDB_ANDA") {
        console.error("API Key TMDB belum diisi!");
        return { results: [] };
    }
    
    try {
        const response = await fetch(`${TMDB_BASE_URL}${endpoint}?api_key=${TMDB_API_KEY}&language=id-ID`);
        return await response.json();
    } catch (error) {
        console.error("Gagal mengambil data dari TMDB:", error);
        return { results: [] };
    }
}

// Fitur Putar Video (Menghubungi Golang Scraper)
async function playMovie(title, year) {
    alert(`Mencari video untuk: ${title} (${year})...\nProses ini memakan waktu beberapa detik karena menembus proteksi Cloudflare.`);
    
    try {
        // Panggil Golang Backend
        const response = await fetch(`${BACKEND_URL}/api/play?title=${encodeURIComponent(title)}&year=${year}`);
        const data = await response.json();
        
        if (data.iframeUrl) {
            iframe.src = data.iframeUrl;
            modal.classList.add('active');
            document.body.style.overflow = 'hidden';
        } else {
            alert("Maaf, video tidak ditemukan di server idlix.");
        }
    } catch (error) {
        console.error("Gagal menghubungi backend Golang:", error);
        alert("Gagal menghubungi server Summer Tide (Golang Backend). Pastikan server menyala.");
    }
}

function playHeroMovie() {
    if (currentHeroMovie) {
        const year = currentHeroMovie.release_date ? currentHeroMovie.release_date.substring(0, 4) : "";
        playMovie(currentHeroMovie.title, year);
    }
}

// ============================================================================
// RENDER LOGIC
// ============================================================================

function setHeroMovie(movie) {
    currentHeroMovie = movie;
    
    const heroSection = document.getElementById('hero-section');
    heroSection.style.backgroundImage = `url('${TMDB_IMAGE_URL}${movie.backdrop_path}')`;
    
    document.getElementById('hero-title').textContent = movie.title || movie.original_title;
    
    // Potong deskripsi jika terlalu panjang
    let desc = movie.overview || "Tidak ada deskripsi tersedia.";
    if (desc.length > 200) desc = desc.substring(0, 200) + "...";
    document.getElementById('hero-desc').textContent = desc;
    
    // Update Meta
    const year = movie.release_date ? movie.release_date.substring(0, 4) : "";
    const match = Math.round(movie.vote_average * 10) + "% Match";
    
    document.querySelector('.hero-meta .year').textContent = year;
    document.querySelector('.hero-meta .match').textContent = match;
}

function renderMovieRow(containerId, movies) {
    const container = document.getElementById(containerId);
    container.innerHTML = ""; // Bersihkan kontainer
    
    movies.forEach(movie => {
        if (!movie.poster_path) return; // Skip jika tidak ada poster
        
        const card = document.createElement('div');
        card.className = 'movie-card';
        
        const year = movie.release_date ? movie.release_date.substring(0, 4) : "";
        const rating = movie.vote_average ? movie.vote_average.toFixed(1) : "N/A";
        
        // Ketika di-klik, panggil Golang untuk mencari videonya
        card.onclick = () => playMovie(movie.title, year);
        
        card.innerHTML = `
            <div class="poster-container">
                <img src="${TMDB_POSTER_URL}${movie.poster_path}" alt="${movie.title}">
                <span class="quality-badge">HD</span>
                <span class="rating-badge"><i class="fas fa-star"></i> ${rating}</span>
            </div>
            <div class="movie-info">
                <h4>${movie.title}</h4>
                <span class="year-text">${year}</span>
            </div>
        `;
        
        container.appendChild(card);
    });
}

// ============================================================================
// INISIALISASI
// ============================================================================

async function initApp() {
    // Ambil data Film Trending dari TMDB
    const trendingData = await fetchTMDB('/trending/movie/week');
    
    if (trendingData.results && trendingData.results.length > 0) {
        // Set film pertama sebagai Hero
        setHeroMovie(trendingData.results[0]);
        
        // Sisa film masuk ke baris Trending
        renderMovieRow('trending-row', trendingData.results.slice(1, 15));
    }
    
    // Ambil data Film Action (Genre ID 28)
    const actionData = await fetchTMDB('/discover/movie?with_genres=28&sort_by=popularity.desc');
    if (actionData.results) {
        renderMovieRow('action-row', actionData.results);
    }
}

// Mulai aplikasi
document.addEventListener('DOMContentLoaded', () => {
    initApp();

    // Event Listener untuk Pencarian
    const searchInput = document.getElementById('search-input');
    let searchTimeout;

    searchInput.addEventListener('input', (e) => {
        clearTimeout(searchTimeout);
        const query = e.target.value.trim();

        searchTimeout = setTimeout(async () => {
            if (query.length > 2) {
                // Sembunyikan baris lain dan buat baris hasil pencarian
                const searchData = await fetchTMDB(`/search/movie?query=${encodeURIComponent(query)}`);
                
                // Ubah judul baris trending menjadi Hasil Pencarian
                const rowTitle = document.querySelector('.movie-row:first-of-type .row-title');
                rowTitle.textContent = `Hasil Pencarian: "${query}"`;
                
                if (searchData.results) {
                    renderMovieRow('trending-row', searchData.results);
                }
            } else if (query.length === 0) {
                // Kembalikan ke awal jika kosong
                const rowTitle = document.querySelector('.movie-row:first-of-type .row-title');
                rowTitle.textContent = "Sedang Hangat (Trending)";
                initApp();
            }
        }, 800); // Debounce 800ms
    });
});
