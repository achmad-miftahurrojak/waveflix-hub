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

// POST /api/auth/register {email, username, password}
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
	writeAuth(w, id, body.Email, body.Username)
}

// POST /api/auth/login {email, password}
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
	var username, hash string
	err := db.QueryRow(
		"SELECT id, username, password_hash FROM users WHERE email = ?", body.Email,
	).Scan(&id, &username, &hash)
	if err != nil {
		httpError(w, http.StatusUnauthorized, "email atau password salah")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(body.Password)) != nil {
		httpError(w, http.StatusUnauthorized, "email atau password salah")
		return
	}
	writeAuth(w, id, body.Email, username)
}

// GET /api/auth/me  (butuh token)
func handleMe(w http.ResponseWriter, r *http.Request) {
	uid := r.Context().Value(userIDKey).(int64)
	var email, username string
	if err := db.QueryRow("SELECT email, username FROM users WHERE id = ?", uid).
		Scan(&email, &username); err != nil {
		httpError(w, http.StatusNotFound, "user tidak ditemukan")
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id": uid, "email": email, "username": username,
	})
}

func writeAuth(w http.ResponseWriter, id int64, email, username string) {
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
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func httpError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// requireAuth: middleware yang memvalidasi Bearer token dan menyuntik userID ke context.
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
