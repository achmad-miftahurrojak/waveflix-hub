package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

func jwtSecret() []byte {
	s := os.Getenv("JWT_SECRET")
	if s == "" {
		s = "summer-tide-dev-secret-ganti-di-produksi"
	}
	return []byte(s)
}

type ctxKey string

const userIDKey ctxKey = "userID"

type authResponse struct {
	Token string `json:"token"`
	User  struct {
		ID       int64  `json:"id"`
		Email    string `json:"email"`
		Username string `json:"username"`
		Avatar   string `json:"avatar"`
		Banner   string `json:"banner"`
	} `json:"user"`
}

func signToken(userID int64) (string, error) {
	claims := jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(30 * 24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret())
}

func decodeJSON(r *http.Request, dst interface{}) error {
	return json.NewDecoder(r.Body).Decode(dst)
}

func handleRegister(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &body); err != nil {
		httpError(w, http.StatusBadRequest, "data tidak valid")
		return
	}
	body.Email = strings.TrimSpace(strings.ToLower(body.Email))
	if body.Email == "" || len(body.Password) < 6 || body.Username == "" {
		httpError(w, http.StatusBadRequest, "email/username wajib, password minimal 6 karakter")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "gagal memproses password")
		return
	}

	res, err := db.Exec(
		"INSERT INTO users(email, username, password_hash) VALUES(?,?,?)",
		body.Email, body.Username, string(hash),
	)
	if err != nil {
		httpError(w, http.StatusConflict, "email sudah terdaftar")
		return
	}
	id, _ := res.LastInsertId()
	writeAuth(w, id, body.Email, body.Username, "", "")
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &body); err != nil {
		httpError(w, http.StatusBadRequest, "data tidak valid")
		return
	}
	body.Email = strings.TrimSpace(strings.ToLower(body.Email))

	var id int64
	var username, hash, avatar, banner string
	err := db.QueryRow(
		"SELECT id, username, password_hash, COALESCE(avatar,''), COALESCE(banner,'') FROM users WHERE email = ?", body.Email,
	).Scan(&id, &username, &hash, &avatar, &banner)
	if err != nil {
		httpError(w, http.StatusUnauthorized, "email atau password salah")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(body.Password)) != nil {
		httpError(w, http.StatusUnauthorized, "email atau password salah")
		return
	}
	writeAuth(w, id, body.Email, username, avatar, banner)
}

func handleCheckEmail(w http.ResponseWriter, r *http.Request) {
	email := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("email")))
	if email == "" {
		httpError(w, http.StatusBadRequest, "email wajib diisi")
		return
	}
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE email = ?)", email).Scan(&exists)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "gagal mengecek email")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"exists": exists})
}

func handleMe(w http.ResponseWriter, r *http.Request) {
	uid := r.Context().Value(userIDKey).(int64)
	var email, username, avatar, banner, bio, nameFont, joined string
	if err := db.QueryRow(
		`SELECT email, username, COALESCE(avatar,''), COALESCE(banner,''),
		        COALESCE(bio,''), COALESCE(name_font,''), COALESCE(created_at,'')
		 FROM users WHERE id = ?`, uid).
		Scan(&email, &username, &avatar, &banner, &bio, &nameFont, &joined); err != nil {
		httpError(w, http.StatusNotFound, "user tidak ditemukan")
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id": uid, "email": email, "username": username,
		"avatar": avatar, "banner": banner, "bio": bio,
		"name_font": nameFont, "joined": joined,
	})
}

func uploadImage(column string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpError(w, http.StatusMethodNotAllowed, "method tidak didukung")
			return
		}
		uid := r.Context().Value(userIDKey).(int64)
		var body struct {
			Image string `json:"image"`
		}
		if err := decodeJSON(r, &body); err != nil {
			httpError(w, http.StatusBadRequest, "data tidak valid")
			return
		}
		ok := strings.HasPrefix(body.Image, "data:image/png;base64,") ||
			strings.HasPrefix(body.Image, "data:image/jpeg;base64,") ||
			strings.HasPrefix(body.Image, "data:image/gif;base64,") ||
			body.Image == ""
		if !ok {
			httpError(w, http.StatusBadRequest, "format harus JPG, PNG, atau GIF")
			return
		}
		if len(body.Image) > 8_000_000 { // ~6MB file
			httpError(w, http.StatusRequestEntityTooLarge, "ukuran gambar terlalu besar (maks ~6MB)")
			return
		}
		if _, err := db.Exec("UPDATE users SET "+column+" = ? WHERE id = ?", body.Image, uid); err != nil {
			httpError(w, http.StatusInternalServerError, "gagal simpan gambar")
			return
		}
		handleMe(w, r) // kembalikan profil terbaru
	}
}

func handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch && r.Method != http.MethodPut {
		httpError(w, http.StatusMethodNotAllowed, "method tidak didukung")
		return
	}
	uid := r.Context().Value(userIDKey).(int64)
	var body struct {
		Username *string `json:"username"`
		Email    *string `json:"email"`
		Bio      *string `json:"bio"`
		NameFont *string `json:"name_font"`
	}
	if err := decodeJSON(r, &body); err != nil {
		httpError(w, http.StatusBadRequest, "data tidak valid")
		return
	}

	sets := []string{}
	args := []interface{}{}
	if body.Username != nil {
		u := strings.TrimSpace(*body.Username)
		if u == "" {
			httpError(w, http.StatusBadRequest, "username wajib diisi")
			return
		}
		sets = append(sets, "username = ?")
		args = append(args, u)
	}
	if body.Email != nil {
		e := strings.TrimSpace(strings.ToLower(*body.Email))
		if e == "" {
			httpError(w, http.StatusBadRequest, "email wajib diisi")
			return
		}
		sets = append(sets, "email = ?")
		args = append(args, e)
	}
	if body.Bio != nil {
		sets = append(sets, "bio = ?")
		args = append(args, strings.TrimSpace(*body.Bio))
	}
	if body.NameFont != nil {
		sets = append(sets, "name_font = ?")
		args = append(args, strings.TrimSpace(*body.NameFont))
	}
	if len(sets) == 0 {
		handleMe(w, r)
		return
	}

	args = append(args, uid)
	if _, err := db.Exec("UPDATE users SET "+strings.Join(sets, ", ")+" WHERE id = ?", args...); err != nil {
		httpError(w, http.StatusConflict, "gagal update (email mungkin sudah dipakai)")
		return
	}
	handleMe(w, r)
}

func handleChangePassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, http.StatusMethodNotAllowed, "method tidak didukung")
		return
	}
	uid := r.Context().Value(userIDKey).(int64)
	var body struct {
		Current string `json:"current_password"`
		New     string `json:"new_password"`
	}
	if err := decodeJSON(r, &body); err != nil {
		httpError(w, http.StatusBadRequest, "data tidak valid")
		return
	}
	if len(body.New) < 6 {
		httpError(w, http.StatusBadRequest, "password baru minimal 6 karakter")
		return
	}
	var hash string
	if err := db.QueryRow("SELECT password_hash FROM users WHERE id = ?", uid).Scan(&hash); err != nil {
		httpError(w, http.StatusNotFound, "user tidak ditemukan")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(body.Current)) != nil {
		httpError(w, http.StatusUnauthorized, "password saat ini salah")
		return
	}
	newHash, err := bcrypt.GenerateFromPassword([]byte(body.New), bcrypt.DefaultCost)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "gagal memproses password")
		return
	}
	if _, err := db.Exec("UPDATE users SET password_hash = ? WHERE id = ?", string(newHash), uid); err != nil {
		httpError(w, http.StatusInternalServerError, "gagal update password")
		return
	}
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func writeAuth(w http.ResponseWriter, id int64, email, username, avatar, banner string) {
	token, err := signToken(id)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "gagal membuat token")
		return
	}
	var resp authResponse
	resp.Token = token
	resp.User.ID = id
	resp.User.Email = email
	resp.User.Username = username
	resp.User.Avatar = avatar
	resp.User.Banner = banner
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func httpError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			httpError(w, http.StatusUnauthorized, "token tidak ada")
			return
		}
		token, err := jwt.Parse(parts[1], func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return jwtSecret(), nil
		})
		if err != nil || !token.Valid {
			httpError(w, http.StatusUnauthorized, "token tidak valid")
			return
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			httpError(w, http.StatusUnauthorized, "token tidak valid")
			return
		}
		sub, ok := claims["sub"].(float64)
		if !ok {
			httpError(w, http.StatusUnauthorized, "token tidak valid")
			return
		}
		ctx := context.WithValue(r.Context(), userIDKey, int64(sub))
		next(w, r.WithContext(ctx))
	}
}
