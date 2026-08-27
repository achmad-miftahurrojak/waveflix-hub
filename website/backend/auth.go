package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const authCookieName = "waveflix_auth"

var jwtSecretValue []byte

func initJWTSecret() {
	s := os.Getenv("JWT_SECRET")
	if s == "" {
		log.Fatal("JWT_SECRET environment variable is required. Isi di backend/.env dengan string acak minimal 32 karakter.")
	}
	if len(s) < 32 {
		log.Fatal("JWT_SECRET harus berukuran minimal 32 karakter.")
	}
	jwtSecretValue = []byte(s)
}

func jwtSecret() []byte {
	return jwtSecretValue
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
		Language string `json:"language"`
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
	r.Body = http.MaxBytesReader(w, r.Body, 1_000_000) // 1MB max
	var body struct {
		Email    string `json:"email"`
		Username string `json:"username"`
		Password string `json:"password"`
		Code     string `json:"code"`
	}
	if err := decodeJSON(r, &body); err != nil {
		httpError(w, http.StatusBadRequest, "data tidak valid")
		return
	}
	body.Email = strings.TrimSpace(strings.ToLower(body.Email))
	body.Code = strings.TrimSpace(body.Code)
	if body.Email == "" || len(body.Password) < 8 || body.Username == "" || len([]rune(body.Email)) > 254 || len([]rune(body.Username)) > 80 {
		httpError(w, http.StatusBadRequest, "email/username wajib, password minimal 8 karakter")
		return
	}
	if !verifyEmailCode(body.Email, body.Code) {
		httpError(w, http.StatusForbidden, "kode verifikasi tidak valid atau kedaluwarsa")
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
		if strings.Contains(err.Error(), "UNIQUE") {
			httpError(w, http.StatusConflict, "email sudah terdaftar")
			return
		}
		httpError(w, http.StatusInternalServerError, "gagal mendaftarkan user")
		return
	}
	id, _ := res.LastInsertId()
	if _, err := db.Exec("INSERT INTO profiles (user_id, name) VALUES (?, ?)", id, body.Username); err != nil {
		httpError(w, http.StatusInternalServerError, "gagal membuat profil awal")
		return
	}
	writeAuth(w, id, body.Email, body.Username, "", "", "id")
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1_000_000) // 1MB max
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
	var username, hash, avatar, banner, language string
	err := db.QueryRow(
		"SELECT id, username, password_hash, COALESCE(avatar,''), COALESCE(banner,''), COALESCE(language,'id') FROM users WHERE email = ?", body.Email,
	).Scan(&id, &username, &hash, &avatar, &banner, &language)
	if err != nil {
		httpError(w, http.StatusUnauthorized, "email atau password salah")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(body.Password)) != nil {
		httpError(w, http.StatusUnauthorized, "email atau password salah")
		return
	}
	writeAuth(w, id, body.Email, username, avatar, banner, language)
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
	var email, username, avatar, banner, bio, nameFont, joined, language string
	if err := db.QueryRow(
		`SELECT email, username, COALESCE(avatar,''), COALESCE(banner,''),
		        COALESCE(bio,''), COALESCE(name_font,''), COALESCE(created_at,''), COALESCE(language,'id')
		 FROM users WHERE id = ?`, uid).
		Scan(&email, &username, &avatar, &banner, &bio, &nameFont, &joined, &language); err != nil {
		httpError(w, http.StatusNotFound, "user tidak ditemukan")
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id": uid, "email": email, "username": username,
		"avatar": avatar, "banner": banner, "bio": bio,
		"name_font": nameFont, "joined": joined, "language": language,
	})
}

func uploadImage(column string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpError(w, http.StatusMethodNotAllowed, "method tidak didukung")
			return
		}
		uid := r.Context().Value(userIDKey).(int64)
		r.Body = http.MaxBytesReader(w, r.Body, 8_000_000) // ~6MB file + overhead base64
		var body struct {
			Image string `json:"image"`
		}
		if err := decodeJSON(r, &body); err != nil {
			httpError(w, http.StatusBadRequest, "data tidak valid")
			return
		}

		if body.Image == "" {
			if _, err := db.Exec("UPDATE users SET "+column+" = ? WHERE id = ?", "", uid); err != nil {
				httpError(w, http.StatusInternalServerError, "gagal hapus gambar")
				return
			}
			handleMe(w, r)
			return
		}

		if len(body.Image) > 8_000_000 { // ~6MB file
			httpError(w, http.StatusRequestEntityTooLarge, "ukuran gambar terlalu besar (maks ~6MB)")
			return
		}

		ext, data, err := validateImageData(body.Image)
		if err != nil {
			httpError(w, http.StatusBadRequest, "gambar tidak valid atau terlalu besar")
			return
		}

		filename := fmt.Sprintf("%s_%d.%s", column, uid, ext)
		filepath := fmt.Sprintf("./uploads/%s", filename)

		if err := os.WriteFile(filepath, data, 0600); err != nil {
			httpError(w, http.StatusInternalServerError, "gagal simpan gambar ke disk")
			return
		}

		dbPath := fmt.Sprintf("/uploads/%s", filename)
		if _, err := db.Exec("UPDATE users SET "+column+" = ? WHERE id = ?", dbPath, uid); err != nil {
			httpError(w, http.StatusInternalServerError, "gagal simpan path gambar")
			return
		}

		handleMe(w, r)
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
		Language *string `json:"language"`
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
		if e == "" || len([]rune(e)) > 254 {
			httpError(w, http.StatusBadRequest, "email wajib diisi")
			return
		}
		sets = append(sets, "email = ?")
		args = append(args, e)
	}
	if body.Bio != nil {
		if len([]rune(*body.Bio)) > 500 {
			httpError(w, http.StatusBadRequest, "bio terlalu panjang")
			return
		}
		sets = append(sets, "bio = ?")
		args = append(args, strings.TrimSpace(*body.Bio))
	}
	if body.NameFont != nil {
		if len([]rune(*body.NameFont)) > 80 {
			httpError(w, http.StatusBadRequest, "font terlalu panjang")
			return
		}
		sets = append(sets, "name_font = ?")
		args = append(args, strings.TrimSpace(*body.NameFont))
	}
	if body.Language != nil {
		sets = append(sets, "language = ?")
		args = append(args, strings.TrimSpace(*body.Language))
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
	if len(body.New) < 8 {
		httpError(w, http.StatusBadRequest, "password baru minimal 8 karakter")
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

func writeAuth(w http.ResponseWriter, id int64, email, username, avatar, banner, language string) {
	token, err := signToken(id)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "gagal membuat token")
		return
	}
	secure := os.Getenv("AUTH_COOKIE_SECURE") == "1" || strings.EqualFold(os.Getenv("AUTH_COOKIE_SECURE"), "true")
	http.SetCookie(w, &http.Cookie{Name: authCookieName, Value: token, Path: "/", MaxAge: 30 * 24 * 60 * 60, HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode})
	var resp authResponse
	resp.Token = token
	resp.User.ID = id
	resp.User.Email = email
	resp.User.Username = username
	resp.User.Avatar = avatar
	resp.User.Banner = banner
	resp.User.Language = language
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	secure := os.Getenv("AUTH_COOKIE_SECURE") == "1" || strings.EqualFold(os.Getenv("AUTH_COOKIE_SECURE"), "true")
	http.SetCookie(w, &http.Cookie{Name: authCookieName, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode})
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func httpError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			if cookie, err := r.Cookie(authCookieName); err == nil {
				authHeader = "Bearer " + cookie.Value
			}
		}
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

func validateImageData(dataURL string) (string, []byte, error) {
	var ext, rawBase64 string
	switch {
	case strings.HasPrefix(dataURL, "data:image/png;base64,"):
		ext, rawBase64 = "png", strings.TrimPrefix(dataURL, "data:image/png;base64,")
	case strings.HasPrefix(dataURL, "data:image/jpeg;base64,"):
		ext, rawBase64 = "jpg", strings.TrimPrefix(dataURL, "data:image/jpeg;base64,")
	case strings.HasPrefix(dataURL, "data:image/gif;base64,"):
		ext, rawBase64 = "gif", strings.TrimPrefix(dataURL, "data:image/gif;base64,")
	default:
		return "", nil, fmt.Errorf("unsupported image format")
	}
	raw, err := base64.StdEncoding.DecodeString(rawBase64)
	if err != nil || len(raw) > 6_000_000 {
		return "", nil, fmt.Errorf("invalid image data")
	}
	img, format, err := image.Decode(bytes.NewReader(raw))
	if err != nil || (format != "png" && format != "jpeg" && format != "gif") {
		return "", nil, fmt.Errorf("invalid image signature")
	}
	if img.Bounds().Dx() > 2048 || img.Bounds().Dy() > 2048 {
		return "", nil, fmt.Errorf("image dimensions too large")
	}
	var encoded bytes.Buffer
	switch ext {
	case "png":
		err = png.Encode(&encoded, img)
	case "jpg":
		err = jpeg.Encode(&encoded, img, &jpeg.Options{Quality: 85})
	case "gif":
		err = gif.Encode(&encoded, img, nil)
	}
	if err != nil || encoded.Len() > 6_000_000 {
		return "", nil, fmt.Errorf("image encoding failed")
	}
	return ext, encoded.Bytes(), nil
}
