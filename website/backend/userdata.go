package main

import (
	"encoding/json"
	"net/http"
	"strconv"
)

func getProfileID(r *http.Request) int {
	pid, _ := strconv.Atoi(r.Header.Get("X-Profile-ID"))
	return pid
}

// Bentuk item yang dikirim/diterima frontend (mirip TmdbItem seperlunya).
type mediaItem struct {
	TmdbID      int64   `json:"tmdb_id"`
	MediaType   string  `json:"media_type"`
	Title       string  `json:"title"`
	PosterPath  string  `json:"poster_path"`
	VoteAverage float64 `json:"vote_average"`
	Season      int     `json:"season,omitempty"`
	Episode     int     `json:"episode,omitempty"`
	Runtime     int     `json:"runtime,omitempty"`
	Progress    int     `json:"progress,omitempty"`
}

func getList(w http.ResponseWriter, r *http.Request, table string) {
	uid := r.Context().Value(userIDKey).(int64)
	pid := getProfileID(r)
	rows, err := db.Query(
		"SELECT tmdb_id, media_type, title, poster_path, vote_average FROM "+
			table+" WHERE user_id = ? AND profile_id = ? ORDER BY added_at DESC", uid, pid)
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
	pid := getProfileID(r)
	if it.MediaType != "tv" {
		it.MediaType = "movie"
	}
	_, err := db.Exec(
		"INSERT OR IGNORE INTO "+table+
			"(user_id, profile_id, tmdb_id, media_type, title, poster_path, vote_average) VALUES(?,?,?,?,?,?,?)",
		uid, pid, it.TmdbID, it.MediaType, it.Title, it.PosterPath, it.VoteAverage)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "gagal simpan")
		return
	}
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func deleteList(w http.ResponseWriter, r *http.Request, table string) {
	uid := r.Context().Value(userIDKey).(int64)
	pid := getProfileID(r)
	q := r.URL.Query()
	idStr := getIDParam(q)
	tmdbID, _ := strconv.ParseInt(idStr, 10, 64)
	media := getMediaParam(q)
	if tmdbID == 0 {
		httpError(w, http.StatusBadRequest, "id tidak valid")
		return
	}
	if _, err := db.Exec(
		"DELETE FROM "+table+" WHERE user_id = ? AND profile_id = ? AND tmdb_id = ? AND media_type = ?",
		uid, pid, tmdbID, media); err != nil {
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
	pid := getProfileID(r)

	if r.Method == http.MethodDelete {
		tmdbIDStr := r.URL.Query().Get("tmdb_id")
		if tmdbIDStr != "" {
			tmdbID, _ := strconv.ParseInt(tmdbIDStr, 10, 64)
			db.Exec("DELETE FROM history WHERE user_id = ? AND profile_id = ? AND tmdb_id = ?", uid, pid, tmdbID)
		} else {
			db.Exec("DELETE FROM history WHERE user_id = ? AND profile_id = ?", uid, pid)
		}
		json.NewEncoder(w).Encode(map[string]bool{"ok": true})
		return
	}

	if r.Method == http.MethodPost {
		var it mediaItem
		if err := json.NewDecoder(r.Body).Decode(&it); err != nil || it.TmdbID == 0 {
			httpError(w, http.StatusBadRequest, "data tidak valid")
			return
		}
		if it.MediaType != "tv" {
			it.MediaType = "movie"
		}
		if _, err := db.Exec(
			`INSERT INTO history(user_id, profile_id, tmdb_id, media_type, title, poster_path, vote_average, season, episode, runtime, progress)
			 VALUES(?,?,?,?,?,?,?,?,?,?,?)`,
			uid, pid, it.TmdbID, it.MediaType, it.Title, it.PosterPath, it.VoteAverage, it.Season, it.Episode, it.Runtime, it.Progress); err != nil {
			httpError(w, http.StatusInternalServerError, "gagal simpan history")
			return
		}
		json.NewEncoder(w).Encode(map[string]bool{"ok": true})
		return
	}

	// GET: 50 tontonan terakhir (unik per judul, terbaru dulu)
	rows, err := db.Query(
		`SELECT tmdb_id, media_type, title, poster_path, vote_average, season, episode, runtime, progress, MAX(watched_at)
		 FROM history WHERE user_id = ? AND profile_id = ?
		 GROUP BY tmdb_id, media_type ORDER BY MAX(watched_at) DESC LIMIT 50`, uid, pid)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "gagal ambil history")
		return
	}
	defer rows.Close()

	items := []mediaItem{}
	for rows.Next() {
		var it mediaItem
		var ts string
		rows.Scan(&it.TmdbID, &it.MediaType, &it.Title, &it.PosterPath, &it.VoteAverage, &it.Season, &it.Episode, &it.Runtime, &it.Progress, &ts)
		items = append(items, it)
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"results": items})
}
