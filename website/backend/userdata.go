package main

import (
	"encoding/json"
	"net/http"
	"strconv"
)

func getProfileID(r *http.Request, uid int64) int {
	pid, _ := strconv.Atoi(r.Header.Get("X-Profile-ID"))
	if pid == 0 {
		db.QueryRow("SELECT id FROM profiles WHERE user_id = $1 AND is_default = true LIMIT 1", uid).Scan(&pid)
	}
	if pid == 0 {
		db.QueryRow("SELECT id FROM profiles WHERE user_id = $1 ORDER BY id ASC LIMIT 1", uid).Scan(&pid)
	}
	return pid
}

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
	pid := getProfileID(r, uid)
	rows, err := db.Query(
		"SELECT tmdb_id, media_type, title, poster_path, vote_average FROM "+
			table+" WHERE user_id = $1 AND profile_id = $2 ORDER BY added_at DESC",
		uid, pid,
	)
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

func addList(w http.ResponseWriter, r *http.Request, table string) {
	uid := r.Context().Value(userIDKey).(int64)
	r.Body = http.MaxBytesReader(w, r.Body, 64_000)
	var it mediaItem
	if err := json.NewDecoder(r.Body).Decode(&it); err != nil || it.TmdbID == 0 {
		httpError(w, http.StatusBadRequest, "data tidak valid")
		return
	}
	pid := getProfileID(r, uid)
	if it.MediaType != "tv" {
		it.MediaType = "movie"
	}

	_, err := db.Exec(
		"INSERT INTO "+table+
			"(user_id, profile_id, tmdb_id, media_type, title, poster_path) VALUES($1,$2,$3,$4,$5,$6)"+
			" ON CONFLICT (profile_id, tmdb_id, media_type) DO NOTHING",
		uid, pid, it.TmdbID, it.MediaType, it.Title, it.PosterPath,
	)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "gagal simpan")
		return
	}
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func deleteList(w http.ResponseWriter, r *http.Request, table string) {
	uid := r.Context().Value(userIDKey).(int64)
	pid := getProfileID(r, uid)
	q := r.URL.Query()
	idStr := getIDParam(q)
	tmdbID, _ := strconv.ParseInt(idStr, 10, 64)
	media := getMediaParam(q)
	if tmdbID == 0 {
		httpError(w, http.StatusBadRequest, "id tidak valid")
		return
	}
	if _, err := db.Exec(
		"DELETE FROM "+table+" WHERE user_id = $1 AND profile_id = $2 AND tmdb_id = $3 AND media_type = $4",
		uid, pid, tmdbID, media,
	); err != nil {
		httpError(w, http.StatusInternalServerError, "gagal hapus")
		return
	}
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

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

var handleWatchlist = listHandler("watchlist")
var handleFavorites = listHandler("favorites")

func handleHistory(w http.ResponseWriter, r *http.Request) {
	uid := r.Context().Value(userIDKey).(int64)
	pid := getProfileID(r, uid)

	if r.Method == http.MethodDelete {
		tmdbIDStr := r.URL.Query().Get("tmdb_id")
		if tmdbIDStr != "" {
			tmdbID, _ := strconv.ParseInt(tmdbIDStr, 10, 64)
			db.Exec("DELETE FROM history WHERE user_id = $1 AND profile_id = $2 AND tmdb_id = $3", uid, pid, tmdbID)
		} else {
			db.Exec("DELETE FROM history WHERE user_id = $1 AND profile_id = $2", uid, pid)
		}
		json.NewEncoder(w).Encode(map[string]bool{"ok": true})
		return
	}

	if r.Method == http.MethodPost {
		r.Body = http.MaxBytesReader(w, r.Body, 64_000)
		var it mediaItem
		if err := json.NewDecoder(r.Body).Decode(&it); err != nil || it.TmdbID == 0 {
			httpError(w, http.StatusBadRequest, "data tidak valid")
			return
		}
		if it.MediaType != "tv" {
			it.MediaType = "movie"
		}
		db.Exec("DELETE FROM history WHERE user_id = $1 AND profile_id = $2 AND tmdb_id = $3 AND media_type = $4", uid, pid, it.TmdbID, it.MediaType)

		if _, err := db.Exec(
			`INSERT INTO history(user_id, profile_id, tmdb_id, media_type, title, poster_path, season_number, episode_number, progress_seconds, duration_seconds, watched_at)
			 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,NOW())`,
			uid, pid, it.TmdbID, it.MediaType, it.Title, it.PosterPath, it.Season, it.Episode, it.Progress, it.Runtime,
		); err != nil {
			httpError(w, http.StatusInternalServerError, "gagal simpan history")
			return
		}
		json.NewEncoder(w).Encode(map[string]bool{"ok": true})
		return
	}

	rows, err := db.Query(
		`SELECT tmdb_id, media_type, title, poster_path, season_number, episode_number, progress_seconds, duration_seconds, watched_at
		 FROM history 
		 WHERE user_id = $1 AND profile_id = $2
		 ORDER BY watched_at DESC
		 LIMIT 50`,
		uid, pid,
	)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "gagal ambil history")
		return
	}
	defer rows.Close()

	items := []mediaItem{}
	for rows.Next() {
		var it mediaItem
		var ts string
		rows.Scan(&it.TmdbID, &it.MediaType, &it.Title, &it.PosterPath, &it.Season, &it.Episode, &it.Progress, &it.Runtime, &ts)
		items = append(items, it)
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"results": items})
}
