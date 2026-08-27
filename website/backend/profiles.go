package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type Profile struct {
	ID        int    `json:"id"`
	UserID    int    `json:"user_id"`
	Name      string `json:"name"`
	Avatar    string `json:"avatar"`
	Banner    string `json:"banner"`
	Bio       string `json:"bio"`
	CreatedAt string `json:"created_at"`
}

func handleProfiles(w http.ResponseWriter, r *http.Request) {
	uid, ok := r.Context().Value(userIDKey).(int64)
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	userID := int(uid)

	switch r.Method {
	case "GET":
		rows, err := db.Query("SELECT id, user_id, name, COALESCE(avatar,''), COALESCE(banner,''), COALESCE(bio,''), created_at FROM profiles WHERE user_id = ? ORDER BY id ASC", userID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			writeJSON(w, `{"error":"failed to get profiles"}`)
			return
		}
		defer rows.Close()

		var profiles []Profile = []Profile{}
		for rows.Next() {
			var p Profile
			if err := rows.Scan(&p.ID, &p.UserID, &p.Name, &p.Avatar, &p.Banner, &p.Bio, &p.CreatedAt); err != nil {
				continue
			}
			profiles = append(profiles, p)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(profiles)

	case "POST":
		r.Body = http.MaxBytesReader(w, r.Body, 64_000)
		var p Profile
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			writeJSON(w, `{"error":"invalid request"}`)
			return
		}

		if p.Name == "" {
			w.WriteHeader(http.StatusBadRequest)
			writeJSON(w, `{"error":"name is required"}`)
			return
		}
		if len([]rune(p.Name)) > 80 || len([]rune(p.Bio)) > 500 {
			w.WriteHeader(http.StatusBadRequest)
			writeJSON(w, `{"error":"profile fields terlalu panjang"}`)
			return
		}

		// limit check (max 4 per account)
		var count int
		db.QueryRow("SELECT COUNT(*) FROM profiles WHERE user_id = ?", userID).Scan(&count)
		if count >= 4 {
			w.WriteHeader(http.StatusBadRequest)
			writeJSON(w, `{"error":"maksimum 4 profil per akun"}`)
			return
		}

		res, err := db.Exec("INSERT INTO profiles (user_id, name, avatar, banner, bio) VALUES (?, ?, ?, ?, ?)", userID, p.Name, p.Avatar, p.Banner, p.Bio)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			writeJSON(w, `{"error":"failed to create profile"}`)
			return
		}

		id, _ := res.LastInsertId()
		p.ID = int(id)
		p.UserID = userID

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(p)

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func handleProfileDetail(w http.ResponseWriter, r *http.Request) {
	uid, ok := r.Context().Value(userIDKey).(int64)
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	userID := int(uid)

	// /api/profiles/123
	path := r.URL.Path
	parts := strings.Split(path, "/")
	if len(parts) < 4 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	profileID, err := strconv.Atoi(parts[3])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Verify ownership
	var ownerID int
	err = db.QueryRow("SELECT user_id FROM profiles WHERE id = ?", profileID).Scan(&ownerID)
	if err != nil || ownerID != userID {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	switch r.Method {
	case "GET":
		var p Profile
		err := db.QueryRow("SELECT id, user_id, name, COALESCE(avatar,''), COALESCE(banner,''), COALESCE(bio,''), created_at FROM profiles WHERE id = ?", profileID).
			Scan(&p.ID, &p.UserID, &p.Name, &p.Avatar, &p.Banner, &p.Bio, &p.CreatedAt)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(p)

	case "PUT", "PATCH":
		var p Profile
		r.Body = http.MaxBytesReader(w, r.Body, 15_000_000) // limit to ~15MB
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			writeJSON(w, `{"error":"invalid request"}`)
			return
		}
		if len([]rune(p.Name)) > 80 || len([]rune(p.Bio)) > 500 {
			w.WriteHeader(http.StatusBadRequest)
			writeJSON(w, `{"error":"profile fields terlalu panjang"}`)
			return
		}

		if strings.HasPrefix(p.Avatar, "data:image/") {
			p.Avatar = processProfileImage(p.Avatar, "avatar", profileID)
		}
		if strings.HasPrefix(p.Banner, "data:image/") {
			p.Banner = processProfileImage(p.Banner, "banner", profileID)
		}

		// Update all provided fields
		db.Exec(
			"UPDATE profiles SET name = CASE WHEN ? != '' THEN ? ELSE name END, avatar = CASE WHEN ? != '' THEN ? ELSE avatar END, banner = CASE WHEN ? != '' THEN ? ELSE banner END, bio = ? WHERE id = ?",
			p.Name, p.Name,
			p.Avatar, p.Avatar,
			p.Banner, p.Banner,
			p.Bio,
			profileID,
		)

		// Return updated profile
		var updated Profile
		db.QueryRow("SELECT id, user_id, name, COALESCE(avatar,''), COALESCE(banner,''), COALESCE(bio,''), created_at FROM profiles WHERE id = ?", profileID).
			Scan(&updated.ID, &updated.UserID, &updated.Name, &updated.Avatar, &updated.Banner, &updated.Bio, &updated.CreatedAt)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(updated)

	case "DELETE":
		db.Exec("DELETE FROM profiles WHERE id = ?", profileID)
		// Delete related records
		db.Exec("DELETE FROM history WHERE profile_id = ?", profileID)
		db.Exec("DELETE FROM watchlist WHERE profile_id = ?", profileID)
		db.Exec("DELETE FROM favorites WHERE profile_id = ?", profileID)
		writeJSON(w, `{"status":"ok"}`)

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func processProfileImage(data, prefix string, id int) string {
	if data == "" {
		return ""
	}
	ext, decoded, err := validateImageData(data)
	if err != nil {
		return ""
	}

	filename := fmt.Sprintf("profile_%s_%d_%d.%s", prefix, id, time.Now().Unix(), ext)
	filepath := fmt.Sprintf("./uploads/%s", filename)

	if err := os.WriteFile(filepath, decoded, 0600); err != nil {
		return ""
	}
	return fmt.Sprintf("/uploads/%s", filename)
}
