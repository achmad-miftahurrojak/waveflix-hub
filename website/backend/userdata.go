package main

import (
	"encoding/json"
	"net/http"
	"strconv"
)

// Bentuk item yang dikirim/diterima frontend (mirip TmdbItem seperlunya).
type mediaItem struct {
	TmdbID      int64   `json:"tmdb_id"`
	MediaType   string  `json:"media_type"`
	Title       string  `json:"title"`
	PosterPath  string  `json:"poster_path"`
	VoteAverage float64 `json:"vote_average"`
	Season      int     `json:"season,omitempty"`
	Episode     int     `json:"episode,omitempty"`
}

// getList: GET daftar (watchlist/favorites) — `table` adalah nama tabel (literal).
func getList(w http.ResponseWriter, r *http.Request, table string) {
	uid := r.Context().Value(userIDKey).(int64)
	rows, err := db.Query(
		"SELECT tmdb_id, media_type, title, poster_path, vote_average FROM "+
			table+" WHERE user_id = ? ORDER BY added_at DESC", uid)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "gagal ambil "+table)
		return
	}
	defer rows.Close()

	items := []mediaItem{}
	for rows.Next() {
		var it mediaItem
		rows.Scan(&it.TmdbID, &it.MediaType, &it.Title, &it.PosterPath, &it.VoteAverage)
		items = append(items, it)
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"results": items})
}

// addList: POST {tmdb_id, media_type, title, poster_path, vote_average}
func addList(w http.ResponseWriter, r *http.Request, table string) {
	uid := r.Context().Value(userIDKey).(int64)
	var it mediaItem
	if err := json.NewDecoder(r.Body).Decode(&it); err != nil || it.TmdbID == 0 {
		httpError(w, http.StatusBadRequest, "data tidak valid")
		return
	}
	if it.MediaType != "tv" {
		it.MediaType = "movie"
	}
	_, err := db.Exec(
		"INSERT OR IGNORE INTO "+table+
			"(user_id, tmdb_id, media_type, title, poster_path, vote_average) VALUES(?,?,?,?,?,?)",
		uid, it.TmdbID, it.MediaType, it.Title, it.PosterPath, it.VoteAverage)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "gagal simpan")
		return
	}
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

// deleteList: DELETE ?tmdb_id=..&media_type=..
func deleteList(w http.ResponseWriter, r *http.Request, table string) {
	uid := r.Context().Value(userIDKey).(int64)
	tmdbID, _ := strconv.ParseInt(r.URL.Query().Get("tmdb_id"), 10, 64)
	media := normalizeMedia(r.URL.Query().Get("media_type"))
	if _, err := db.Exec(
		"DELETE FROM "+table+" WHERE user_id = ? AND tmdb_id = ? AND media_type = ?",
		uid, tmdbID, media); err != nil {
		httpError(w, http.StatusInternalServerError, "gagal hapus")
		return
	}
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

// listHandler membuat router GET/POST/DELETE untuk satu tabel daftar.
func listHandler(table string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			getList(w, r, table)
		case http.MethodPost:
			addList(w, r, table)
		case http.MethodDelete:
			deleteList(w, r, table)
		default:
			httpError(w, http.StatusMethodNotAllowed, "method tidak didukung")
		}
	}
}

// Router /api/watchlist & /api/favorites.
var handleWatchlist = listHandler("watchlist")
var handleFavorites = listHandler("favorites")

// GET /api/history  &  POST /api/history
func handleHistory(w http.ResponseWriter, r *http.Request) {
	uid := r.Context().Value(userIDKey).(int64)

	if r.Method == http.MethodPost {
		var it mediaItem
		if err := json.NewDecoder(r.Body).Decode(&it); err != nil || it.TmdbID == 0 {
			httpError(w, http.StatusBadRequest, "data tidak valid")
			return
		}
		if it.MediaType != "tv" {
			it.MediaType = "movie"
		}
		db.Exec(
			`INSERT INTO history(user_id, tmdb_id, media_type, title, poster_path, vote_average, season, episode)
			 VALUES(?,?,?,?,?,?,?,?)`,
			uid, it.TmdbID, it.MediaType, it.Title, it.PosterPath, it.VoteAverage, it.Season, it.Episode)
		json.NewEncoder(w).Encode(map[string]bool{"ok": true})
		return
	}

	// GET: 50 tontonan terakhir (unik per judul, terbaru dulu)
	rows, err := db.Query(
		`SELECT tmdb_id, media_type, title, poster_path, vote_average, MAX(watched_at)
		 FROM history WHERE user_id = ?
		 GROUP BY tmdb_id, media_type ORDER BY MAX(watched_at) DESC LIMIT 50`, uid)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "gagal ambil history")
		return
	}
	defer rows.Close()

	items := []mediaItem{}
	for rows.Next() {
		var it mediaItem
		var ts string
		rows.Scan(&it.TmdbID, &it.MediaType, &it.Title, &it.PosterPath, &it.VoteAverage, &ts)
		items = append(items, it)
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"results": items})
}
