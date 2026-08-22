package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"
)

var (
	vidlinkCache = make(map[string]bool)
	vidlinkMu    sync.RWMutex
	
	providerCache = make(map[string]bool)
	providerMu    sync.RWMutex
	
	// Netflix(8), Prime(119), Disney(337), Apple(350), HBO(384), Curiosity(190)
	// Asian/Global Drama platforms: Viu(158), Vidio(489), WeTV(623), Rakuten Viki(344), wavve(356), iQIYI(198, 199)
	majorProvidersRegex = regexp.MustCompile(`"provider_id"\s*:\s*(8|119|337|350|384|190|158|489|623|344|356|198|199)\b`)
)

func checkVidlinkAvailability(mediaType string, tmdbId string) bool {
	if tmdbId == "" || mediaType == "" {
		return false
	}
	cacheKey := mediaType + ":" + tmdbId
	vidlinkMu.RLock()
	if available, exists := vidlinkCache[cacheKey]; exists {
		vidlinkMu.RUnlock()
		return available
	}
	vidlinkMu.RUnlock()

	var targetUrl string
	if mediaType == "tv" {
		targetUrl = fmt.Sprintf("https://vidlink.pro/tv/%s/1/1", tmdbId)
	} else {
		targetUrl = fmt.Sprintf("https://vidlink.pro/movie/%s", tmdbId)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "curl", "-I", "-s", "-A", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36", targetUrl)
	out, err := cmd.CombinedOutput()
	
	// A 200 OK means the movie exists. 500 means it doesn't.
	available := err == nil && strings.Contains(string(out), "200 OK")

	vidlinkMu.Lock()
	vidlinkCache[cacheKey] = available
	vidlinkMu.Unlock()

	return available
}

func checkMajorProvider(mediaType string, tmdbId string) bool {
	if tmdbId == "" || mediaType == "" {
		return false
	}
	cacheKey := mediaType + ":" + tmdbId
	providerMu.RLock()
	if available, exists := providerCache[cacheKey]; exists {
		providerMu.RUnlock()
		return available
	}
	providerMu.RUnlock()

	targetUrl := fmt.Sprintf("%s/%s/%s/watch/providers?api_key=%s", tmdbBaseUrl, mediaType, tmdbId, tmdbApiKey)
	resp, err := httpClient.Get(targetUrl)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	
	bodyBytes, _ := io.ReadAll(resp.Body)
	isMajor := majorProvidersRegex.Match(bodyBytes)
		
	providerMu.Lock()
	providerCache[cacheKey] = isMajor
	providerMu.Unlock()
	
	return isMajor
}

func filterSafeContent(results []interface{}, rootMedia string) []interface{} {
	var wg sync.WaitGroup
	sem := make(chan struct{}, 10)
	available := make([]bool, len(results))

	for i, v := range results {
		if item, ok := v.(map[string]interface{}); ok {
			idFloat, ok := item["id"].(float64)
			if !ok {
				continue
			}
			idStr := fmt.Sprintf("%.0f", idFloat)
			
			itemMedia := rootMedia
			if m, ok := item["media_type"].(string); ok && m != "" {
				itemMedia = m
			}
			if itemMedia == "" || itemMedia == "person" {
				if itemMedia == "person" {
					continue
				}
				itemMedia = "movie" // default fallback
			}

			wg.Add(1)
			go func(idx int, itemMap map[string]interface{}, mediaType string, tmdbId string) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				
				// 1. Strictly filter to only the 5 major platforms (Skip if it's a direct search, rootMedia == "multi")
				if rootMedia != "multi" && !checkMajorProvider(mediaType, tmdbId) {
					return
				}
				
				// 2. Cek apakah playable di Vidlink
				isAvail := checkVidlinkAvailability(mediaType, tmdbId)
				
				// Set flag vidlink_available ke map (aman karena map berbeda setiap index)
				itemMap["vidlink_available"] = isAvail
				
				// Tetap loloskan ke hasil akhir selama lolos filter provider (atau jika itu search)
				available[idx] = true
			}(i, item, itemMedia, idStr)
		}
	}
	wg.Wait()
	
	var orderedFiltered []interface{}
	for i, v := range results {
		if available[i] {
			orderedFiltered = append(orderedFiltered, v)
		}
	}

	return orderedFiltered
}

const tmdbBaseUrl = "https://api.themoviedb.org/3"
const traktBaseUrl = "https://api.trakt.tv"
const watchRegion = "ID" // Indonesia

var (
	tmdbApiKey        string
	traktClientID     string
	traktClientSecret string
)

func loadConfig() {
	loadDotEnv(".env")
	tmdbApiKey = getenv("TMDB_API_KEY", "")
	if tmdbApiKey == "" {
		log.Println("[config] WARNING: TMDB_API_KEY kosong. Call ke TMDB akan gagal. Isi di backend/.env")
	}
	traktClientID = getenv("TRAKT_CLIENT_ID", "")
	traktClientSecret = getenv("TRAKT_CLIENT_SECRET", "")

	if traktClientID == "" {
		log.Println("[config] TRAKT_CLIENT_ID kosong — endpoint Trakt belum aktif. Isi di backend/.env")
	} else {
		log.Println("[config] Trakt client_id termuat ✓")
	}
}

var countryLang = map[string]string{
	"ID": "id",
	"US": "en",
	"KR": "ko",
	"JP": "ja",
	"CN": "zh",
	"GB": "en",
	"TH": "th",
}

func normalizeMedia(m string) string {
	if m == "tv" {
		return "tv"
	}
	return "movie"
}

func getMediaParam(q url.Values) string {
	m := q.Get("media")
	if m == "" {
		m = q.Get("media_type")
	}
	return normalizeMedia(m)
}

func getIDParam(q url.Values) string {
	id := q.Get("id")
	if id == "" {
		id = q.Get("tmdb_id")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return ""
	}
	for _, c := range id {
		if c < '0' || c > '9' {
			return "" // invalid non-numeric ID
		}
	}
	return id
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

type cacheItem struct {
	data     []byte
	expireAt time.Time
}

var (
	memCache = make(map[string]cacheItem)
	cacheMu  sync.RWMutex
)

func init() {
	go func() {
		for {
			time.Sleep(10 * time.Minute)
			now := time.Now()
			cacheMu.Lock()
			for k, v := range memCache {
				if now.After(v.expireAt) {
					delete(memCache, k)
				}
			}
			cacheMu.Unlock()
		}
	}()
}

func getTmdbLang(q url.Values) string {
	lang := q.Get("lang")
	switch lang {
	case "id":
		return "id-ID"
	case "ja":
		return "ja-JP"
	case "en":
		return "en-US"
	default:
		return "en-US"
	}
}

func getImageLangs(q url.Values) string {
	lang := getTmdbLang(q)[:2]
	if lang == "en" {
		return "en,null"
	}
	return lang + ",en,null"
}

// contentBlacklist berisi TMDB ID yang salah klasifikasi oleh TMDB (adult=false tapi kontennya eksplisit).
// Tambahkan ID di sini kalau ada konten tidak pantas yang lolos filter otomatis.
var contentBlacklist = map[int]bool{
	269955: true, // Obsessed (2014, KR) — erotic film, TMDB flag adult=false (wrong)
}

func filterAdult(results []interface{}) []interface{} {
	var filtered []interface{}
	for _, v := range results {
		if item, ok := v.(map[string]interface{}); ok {
			// Block TMDB-flagged adult content
			if adultVal, ok := item["adult"].(bool); ok && adultVal {
				continue
			}
			// Block manually blacklisted IDs (TMDB miscategorized content)
			if idFloat, ok := item["id"].(float64); ok {
				if contentBlacklist[int(idFloat)] {
					continue
				}
			}
			filtered = append(filtered, v)
		}
	}
	return filtered
}


func sanitizeTMDBData(data map[string]interface{}, rootMedia string) {
	if results, ok := data["results"].([]interface{}); ok {
		data["results"] = filterSafeContent(filterAdult(results), rootMedia)
	}
	
	// Sanitize recommendations and similar
	for _, key := range []string{"recommendations", "similar"} {
		if nested, ok := data[key].(map[string]interface{}); ok {
			if results, ok := nested["results"].([]interface{}); ok {
				nested["results"] = filterSafeContent(filterAdult(results), rootMedia)
			}
		}
	}

	// Sanitize credits (cast and crew can contain movies/shows when fetching person details)
	for _, key := range []string{"combined_credits", "movie_credits", "tv_credits", "credits"} {
		if nested, ok := data[key].(map[string]interface{}); ok {
			if cast, ok := nested["cast"].([]interface{}); ok {
				if rootMedia == "person" {
					nested["cast"] = filterSafeContent(filterAdult(cast), rootMedia)
				} else {
					nested["cast"] = filterAdult(cast)
				}
			}
			if crew, ok := nested["crew"].([]interface{}); ok {
				if rootMedia == "person" {
					nested["crew"] = filterSafeContent(filterAdult(crew), rootMedia)
				} else {
					nested["crew"] = filterAdult(crew)
				}
			}
		}
	}
}

// ============================================================================
// TRENDING  ->  /api/trending?media=all|movie|tv&page=1  (global, dipakai fallback)
// ============================================================================
func handleTrending(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	media := q.Get("media")
	if media == "" {
		media = q.Get("media_type")
	}
	if media != "movie" && media != "tv" {
		media = "movie" // Fallback ke movie untuk discover support
	}
	page := firstNonEmpty(q.Get("page"), "1")

	p := url.Values{}
	p.Set("api_key", tmdbApiKey)
	p.Set("language", getTmdbLang(q))
	p.Set("include_image_language", getImageLangs(q))
	p.Set("watch_region", watchRegion)
	p.Set("page", page)
	p.Set("sort_by", "popularity.desc")
	p.Set("include_adult", "false")
	if v := q.Get("provider"); v != "" {
		p.Set("with_watch_providers", v)
	}

	proxyRequest(w, tmdbBaseUrl+"/discover/"+media+"?"+p.Encode(), media)
}

func handleHomepage(w http.ResponseWriter, r *http.Request) { handleTrending(w, r) }

// ============================================================================
// DISCOVER  ->  /api/discover?media=movie&provider=8&genre=28&year=2024
//
//	&country=US&sort_by=popularity.desc&page=1&released_before=...
//
// ============================================================================
func handleDiscover(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	media := getMediaParam(q)
	page := firstNonEmpty(q.Get("page"), "1")

	p := url.Values{}
	p.Set("api_key", tmdbApiKey)
	p.Set("language", getTmdbLang(q))
	p.Set("include_image_language", getImageLangs(q))
	p.Set("watch_region", watchRegion)
	p.Set("page", page)
	p.Set("sort_by", firstNonEmpty(q.Get("sort_by"), "popularity.desc"))
	p.Set("include_adult", "false")
	if v := q.Get("provider"); v != "" {
		p.Set("with_watch_providers", v)
	} else {
		// Strict filter for major platforms if no provider specified
		p.Set("with_watch_providers", "8|119|337|350|384|190")
	}

	if v := q.Get("genre"); v != "" {
		p.Set("with_genres", v)
	}
	if v := q.Get("without_genres"); v != "" {
		p.Set("without_genres", v)
	}
	if v := q.Get("country"); v != "" {
		p.Set("with_origin_country", v)
		p.Set("watch_region", v)
		if lang, ok := countryLang[v]; ok {
			p.Set("with_original_language", lang)
		}
		// Certification filter per negara asal — block rating dewasa, biarkan tanpa cert tetap lolos
		switch v {
		case "KR":
			p.Set("certification_country", "KR")
			p.Set("certification.lte", "18") // block KR:19
		case "JP":
			p.Set("certification_country", "JP")
			p.Set("certification.lte", "R15+") // block JP:R18+
		case "US", "GB":
			p.Set("certification_country", "US")
			p.Set("certification.lte", "R") // block NC-17
		}
	}
	if v := q.Get("year"); v != "" {
		if media == "tv" {
			p.Set("first_air_date_year", v)
		} else {
			p.Set("primary_release_year", v)
		}
	}
	if v := q.Get("released_before"); v != "" {
		if media == "tv" {
			p.Set("first_air_date.lte", v)
		} else {
			p.Set("primary_release_date.lte", v)
		}
	}
	if v := q.Get("released_after"); v != "" {
		if media == "tv" {
			p.Set("first_air_date.gte", v)
		} else {
			p.Set("primary_release_date.gte", v)
		}
	}

	proxyRequest(w, tmdbBaseUrl+"/discover/"+media+"?"+p.Encode(), media)
}

// ============================================================================
// DETAIL  ->  /api/detail?media=movie&id=123
// Menyertakan credits (cast). Kalau overview (en-US) kosong -> fallback en-US.
// ============================================================================
func handleDetail(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	media := getMediaParam(q)
	id := getIDParam(q)
	if id == "" {
		writeJSON(w, `{"error":"id kosong"}`)
		return
	}

	base := tmdbBaseUrl + "/" + media + "/" + id
	imageLangs := getImageLangs(q)
	data, err := fetchJSON(base + "?language=" + getTmdbLang(q) + "&append_to_response=credits,videos,recommendations,similar,images&include_image_language=" + imageLangs + "&api_key=" + tmdbApiKey, media)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		writeJSON(w, `{"error":"gagal ambil detail"}`)
		return
	}

	// Block blacklisted content (TMDB miscategorized adult content)
	if idFloat, ok := data["id"].(float64); ok && contentBlacklist[int(idFloat)] {
		w.WriteHeader(http.StatusForbidden)
		writeJSON(w, `{"error":"konten tidak tersedia"}`)
		return
	}
	if adultVal, _ := data["adult"].(bool); adultVal {
		w.WriteHeader(http.StatusForbidden)
		writeJSON(w, `{"error":"konten tidak tersedia"}`)
		return
	}

	if ov, _ := data["overview"].(string); ov == "" {
		if en, err := fetchJSON(base+"?language=en-US&api_key="+tmdbApiKey, media); err == nil {
			if enOv, ok := en["overview"].(string); ok && enOv != "" {
				data["overview"] = enOv
			}
		}
	}

	fetchSafeRecommendations(media, q, data)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func handleDetailBatch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	media := getMediaParam(q)
	idsStr := q.Get("ids")
	if idsStr == "" {
		writeJSON(w, `{"error":"ids kosong"}`)
		return
	}
	ids := strings.Split(idsStr, ",")

	type result struct {
		id   string
		data map[string]interface{}
		err  error
	}
	ch := make(chan result, len(ids))

	for _, id := range ids {
		go func(id string) {
			base := tmdbBaseUrl + "/" + media + "/" + id
			imageLangs := getImageLangs(q)
			targetUrl := base + "?language=" + getTmdbLang(q) + "&append_to_response=credits,videos,recommendations,similar,images&include_image_language=" + imageLangs + "&api_key=" + tmdbApiKey
			data, err := fetchJSON(targetUrl, media)
			if err != nil {
				ch <- result{id: id, err: err}
				return
			}

			if ov, _ := data["overview"].(string); ov == "" {
				if en, err := fetchJSON(base+"?language=en-US&api_key="+tmdbApiKey, media); err == nil {
					if enOv, ok := en["overview"].(string); ok && enOv != "" {
						data["overview"] = enOv
					}
				}
			}
			fetchSafeRecommendations(media, q, data)
			ch <- result{id: id, data: data}
		}(id)
	}

	out := make(map[string]interface{})
	for i := 0; i < len(ids); i++ {
		res := <-ch
		if res.err == nil {
			out[res.id] = res.data
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

// ============================================================================
// PERSON  ->  /api/person?id=123
// ============================================================================
func handlePerson(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	id := getIDParam(q)
	if id == "" {
		writeJSON(w, `{"error":"id kosong"}`)
		return
	}
	
	targetUrl := tmdbBaseUrl + "/person/" + id + "?language=" + getTmdbLang(q) + "&append_to_response=combined_credits&api_key=" + tmdbApiKey
	data, err := fetchJSON(targetUrl, "person")
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		writeJSON(w, `{"error":"gagal ambil person"}`)
		return
	}

	// Fetch en-US to get the original/untranslated name, as TMDB person endpoint doesn't return original_name
	if en, err := fetchJSON(tmdbBaseUrl+"/person/"+id+"?language=en-US&api_key="+tmdbApiKey, "person"); err == nil {
		if enName, ok := en["name"].(string); ok && enName != "" {
			data["original_name"] = enName
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// ============================================================================
// SEASON  ->  /api/season?id=123&season=1   (daftar episode satu musim)
// ============================================================================
func handleSeason(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	id := getIDParam(q)
	season := firstNonEmpty(q.Get("season"), "1")
	if id == "" {
		writeJSON(w, `{"episodes":[]}`)
		return
	}
	proxyRequest(w, tmdbBaseUrl+"/tv/"+id+"/season/"+season+"?language="+getTmdbLang(q)+"&api_key="+tmdbApiKey, "tv")
}

// ============================================================================
// IMAGES  ->  /api/images?media=movie&id=123   (logo hero, en+null)
// ============================================================================
func handleImages(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	media := getMediaParam(q)
	id := getIDParam(q)
	if id == "" {
		writeJSON(w, `{"logos":[]}`)
		return
	}
	imageLangs := getImageLangs(q)
	proxyRequest(w, tmdbBaseUrl+"/"+media+"/"+id+"/images?include_image_language="+imageLangs+"&api_key="+tmdbApiKey, media)
}

// ============================================================================
// SEARCH  ->  /api/search?q=...
// ============================================================================
func handleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		writeJSON(w, `{"results":[]}`)
		return
	}
	page := firstNonEmpty(r.URL.Query().Get("page"), "1")
	imageLangs := getImageLangs(r.URL.Query())
	targetUrl := tmdbBaseUrl + "/search/multi?query=" + url.QueryEscape(query) +
		"&language=" + getTmdbLang(r.URL.Query()) + "&include_image_language=" + imageLangs + "&page=" + page + "&include_adult=false&api_key=" + tmdbApiKey
	proxyRequest(w, targetUrl, "multi")
}

// ============================================================================
// STREAM (mock — kerangka scraper M3U8)
// ============================================================================
func handleStream(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, `{
		"stream_url": "https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/BigBuckBunny.mp4",
		"subtitle_url": "/dummy_sub.vtt"
	}`)
}

// ============================================================================
// HELPERS
// ============================================================================

func fetchSafeRecommendations(media string, q url.Values, data map[string]interface{}) {
	genres := ""
	if g, ok := data["genres"].([]interface{}); ok && len(g) > 0 {
		if firstG, ok := g[0].(map[string]interface{}); ok {
			if id, ok := firstG["id"].(float64); ok {
				genres = fmt.Sprintf("%.0f", id)
			}
		}
	}

	discoverUrl := tmdbBaseUrl + "/discover/" + media + "?api_key=" + tmdbApiKey + 
		"&language=" + getTmdbLang(q) + 
		"&watch_region=" + watchRegion +
		"&sort_by=popularity.desc" +
		"&include_adult=false" +
		"&with_watch_providers=8|119|337|350|384|190"
	
	if genres != "" {
		discoverUrl += "&with_genres=" + genres
	}

	if customRecs, err := fetchJSON(discoverUrl, media); err == nil {
		data["recommendations"] = customRecs
		data["similar"] = map[string]interface{}{"page": 1, "results": []interface{}{}, "total_pages": 0, "total_results": 0}
	}
}

var httpClient = http.Client{
	Timeout: 10 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
	},
}

func writeJSON(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(body))
}

func fetchJSON(targetUrl string, rootMedia string) (map[string]interface{}, error) {
	cacheKey := "json:" + targetUrl
	cacheMu.RLock()
	if item, exists := memCache[cacheKey]; exists && time.Now().Before(item.expireAt) {
		cacheMu.RUnlock()
		var out map[string]interface{}
		json.Unmarshal(item.data, &out)
		return out, nil
	}
	cacheMu.RUnlock()

	resp, err := httpClient.Get(targetUrl)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	cacheMu.Lock()
	memCache[cacheKey] = cacheItem{
		data:     bodyBytes,
		expireAt: time.Now().Add(15 * time.Minute),
	}
	cacheMu.Unlock()

	var out map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &out); err != nil {
		return nil, err
	}
	
	// Block explicit adult content or obscure content at the root level
	if adultVal, ok := out["adult"].(bool); ok && adultVal {
		return nil, fmt.Errorf("content restricted")
	}
	if vc, ok := out["vote_count"].(float64); ok && vc < 300 {
		// Removed vote_count filter
	}

	sanitizeTMDBData(out, rootMedia)
	return out, nil
}

func proxyRequest(w http.ResponseWriter, targetUrl string, rootMedia string) {
	cacheKey := "proxy:" + targetUrl
	cacheMu.RLock()
	if item, exists := memCache[cacheKey]; exists && time.Now().Before(item.expireAt) {
		cacheMu.RUnlock()
		w.Header().Set("Content-Type", "application/json")
		w.Write(item.data)
		return
	}
	cacheMu.RUnlock()

	resp, err := httpClient.Get(targetUrl)
	if err != nil {
		log.Printf("Gagal fetch TMDB: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		writeJSON(w, `{"error":"gagal menghubungi TMDB"}`)
		return
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)

	var data map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &data); err == nil {
		// Block explicit adult content or obscure content at the root level
		if adultVal, ok := data["adult"].(bool); ok && adultVal {
			w.WriteHeader(http.StatusForbidden)
			writeJSON(w, `{"error":"content restricted"}`)
			return
		}
		if vc, ok := data["vote_count"].(float64); ok && vc < 300 {
			// Removed vote_count filter
		}
		
		sanitizeTMDBData(data, rootMedia)
		if sanitizedBytes, err := json.Marshal(data); err == nil {
			bodyBytes = sanitizedBytes
		}
	}

	cacheMu.Lock()
	memCache[cacheKey] = cacheItem{
		data:     bodyBytes,
		expireAt: time.Now().Add(15 * time.Minute),
	}
	cacheMu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	w.Write(bodyBytes)
}

func main() {
	loadConfig()
	initJWTSecret()
	initDB()

	os.MkdirAll("./uploads", 0755)

	mux := http.NewServeMux()
	mux.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir("./uploads"))))
	mux.HandleFunc("/api/trending", handleTrending)
	mux.HandleFunc("/api/homepage", handleHomepage)
	mux.HandleFunc("/api/discover", handleDiscover)
	mux.HandleFunc("/api/detail", handleDetail)
	mux.HandleFunc("/api/detail-batch", handleDetailBatch)
	mux.HandleFunc("/api/person", handlePerson)
	mux.HandleFunc("/api/season", handleSeason)
	mux.HandleFunc("/api/images", handleImages)
	mux.HandleFunc("/api/search", handleSearch)
	mux.HandleFunc("/api/stream", handleStream)

	// Auth
	mux.HandleFunc("/api/auth/register", rateLimit(3, time.Minute, handleRegister))
	mux.HandleFunc("/api/auth/login", rateLimit(5, time.Minute, handleLogin))
	mux.HandleFunc("/api/auth/check-email", rateLimit(5, time.Minute, handleCheckEmail))
	mux.HandleFunc("/api/auth/me", requireAuth(handleMe))
	mux.HandleFunc("/api/auth/profile", requireAuth(handleUpdateProfile))
	mux.HandleFunc("/api/auth/password", requireAuth(handleChangePassword))
	mux.HandleFunc("/api/auth/avatar", requireAuth(uploadImage("avatar")))
	mux.HandleFunc("/api/auth/banner", requireAuth(uploadImage("banner")))

	// Data user
	mux.HandleFunc("/api/watchlist", requireAuth(handleWatchlist))
	mux.HandleFunc("/api/favorites", requireAuth(handleFavorites))
	mux.HandleFunc("/api/history", requireAuth(handleHistory))
	
	// Profiles
	mux.HandleFunc("/api/profiles", requireAuth(handleProfiles))
	mux.HandleFunc("/api/profiles/", requireAuth(handleProfileDetail))

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if err := db.Ping(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			writeJSON(w, `{"status":"error","database":"disconnected"}`)
			return
		}
		writeJSON(w, `{"status":"ok","database":"connected"}`)
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, `{"service":"waveflix-api","status":"ok"}`)
	})

	handler := enableCORS(mux)
	handler = secureHeaders(handler)
	handler = recoveryMiddleware(handler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Summer Tide API berjalan di http://localhost:%s\n", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatalf("Server gagal: %v", err)
	}
}

func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("[PANIC] %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				writeJSON(w, `{"error":"internal server error"}`)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func secureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		next.ServeHTTP(w, r)
	})
}

func enableCORS(next http.Handler) http.Handler {
	allowedOrigins := map[string]bool{
		"http://localhost:3000": true,
		"http://127.0.0.1:3000": true,
	}
	if envOrigin := os.Getenv("ALLOWED_ORIGIN"); envOrigin != "" {
		allowedOrigins[envOrigin] = true
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if allowedOrigins[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else if origin != "" && len(allowedOrigins) == 2 { // fallback untuk dev lokal jika belum set origin produksi
			w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Profile-ID")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}
