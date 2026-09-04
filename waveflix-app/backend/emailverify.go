package main

import (
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/smtp"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	codeTTL         = 10 * time.Minute
	maxCodeAttempts = 5
)

func handleSendCode(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1_000)
	var body struct {
		Email string `json:"email"`
	}
	if err := decodeJSON(r, &body); err != nil {
		httpError(w, http.StatusBadRequest, "data tidak valid")
		return
	}
	email := strings.TrimSpace(strings.ToLower(body.Email))
	if !strings.Contains(email, "@") || strings.ContainsAny(email, " \t\r\n") {
		httpError(w, http.StatusBadRequest, "email tidak valid")
		return
	}

	var exists bool
	if err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)", email).Scan(&exists); err != nil {
		httpError(w, http.StatusInternalServerError, "gagal mengecek email")
		return
	}
	if exists {
		httpError(w, http.StatusConflict, "email sudah terdaftar")
		return
	}

	code, err := generateCode()
	if err != nil {
		httpError(w, http.StatusInternalServerError, "gagal membuat kode")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(code+email), bcrypt.DefaultCost)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "gagal memproses kode")
		return
	}

	if _, err := db.Exec(
		`INSERT INTO email_verifications(token, expires_at)
		 VALUES($1, $2)`,

		string(hash)+"|||"+email,
		time.Now().Add(codeTTL),
	); err != nil {
		httpError(w, http.StatusInternalServerError, "gagal menyimpan kode")
		return
	}

	if smtpConfigured() {
		if err := sendVerificationEmail(email, code); err != nil {
			httpError(w, http.StatusBadGateway, "gagal mengirim email verifikasi")
			return
		}
	} else {

		log.Printf("[auth] SMTP belum dikonfigurasi — kode verifikasi %s untuk %s", code, email)
	}
	if os.Getenv("E2E_TEST_MODE") == "1" {
		writeJSON(w, fmt.Sprintf(`{"ok":true,"test_code":%q}`, code))
		return
	}
	writeJSON(w, `{"ok":true}`)
}

func verifyEmailCode(email, code string) bool {
	tx, err := db.Begin()
	if err != nil {
		return false
	}
	defer tx.Rollback()

	var id int
	var tokenField string
	var expiresAt time.Time
	err = tx.QueryRow(
		"SELECT id, token, expires_at FROM email_verifications WHERE token LIKE $1 ORDER BY expires_at DESC LIMIT 1",
		"%|||"+email,
	).Scan(&id, &tokenField, &expiresAt)
	if err != nil {
		return false
	}
	if time.Now().After(expiresAt) {
		return false
	}

	parts := strings.SplitN(tokenField, "|||", 2)
	if len(parts) != 2 || parts[1] != email {
		return false
	}
	storedHash := parts[0]

	if bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(code+email)) != nil {
		return false
	}

	if _, err := tx.Exec("DELETE FROM email_verifications WHERE id = $1", id); err != nil {
		return false
	}
	return tx.Commit() == nil
}

func generateCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func smtpConfigured() bool {
	return os.Getenv("SMTP_HOST") != "" && os.Getenv("SMTP_FROM") != ""
}

func sendVerificationEmail(to, code string) error {
	host := os.Getenv("SMTP_HOST")
	port := getenv("SMTP_PORT", "587")
	user := os.Getenv("SMTP_USER")
	pass := os.Getenv("SMTP_PASS")
	from := getenv("SMTP_FROM", user)

	msg := []byte("From: " + from + "\r\n" +
		"To: " + to + "\r\n" +
		"Subject: Kode verifikasi Waveflix\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n" +
		"\r\n" +
		"Kode verifikasi kamu: " + code + "\r\n\r\n" +
		"Berlaku 10 menit. Jangan bagikan kode ini ke siapa pun.\r\n")

	addr := net.JoinHostPort(host, port)
	auth := smtp.PlainAuth("", user, pass, host)

	if port == "465" {
		conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: host})
		if err != nil {
			return err
		}
		c, err := smtp.NewClient(conn, host)
		if err != nil {
			return err
		}
		defer c.Close()
		if err := c.Auth(auth); err != nil {
			return err
		}
		if err := c.Mail(from); err != nil {
			return err
		}
		if err := c.Rcpt(to); err != nil {
			return err
		}
		w, err := c.Data()
		if err != nil {
			return err
		}
		if _, err := w.Write(msg); err != nil {
			return err
		}
		if err := w.Close(); err != nil {
			return err
		}
		return c.Quit()
	}

	return smtp.SendMail(addr, auth, from, []string{to}, msg)
}
