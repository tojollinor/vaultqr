
package main

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	qrcode "github.com/skip2/go-qrcode"
)

type config struct {
	Port        string
	Secret      string
	DefaultSize int
	MaxSize     int
	SessionTTL  time.Duration
}

type qrEntry struct {
	Content  string
	ShowText bool
	Size     int
	Expires  time.Time
}

type store struct {
	mu      sync.RWMutex
	entries map[string]qrEntry
}

type pageData struct {
	Content  string
	ShowText bool
	Image    string
	Size     int
}

var page = template.Must(template.New("page").Parse(`<!doctype html>
<html lang="de">
<head>
	<meta charset="utf-8">
	<meta name="viewport" content="width=device-width,initial-scale=1">
	<title>VaultQR</title>
	<style>
		:root {
			color-scheme: light dark;
			--bg: #111318;
			--card: #1b1e25;
			--text: #f4f6fb;
			--muted: #aeb6c5;
			--border: #313745;
			--button: #2d65d8;
			--button-hover: #3c75eb;
		}
		* { box-sizing: border-box; }
		body {
			margin: 0;
			min-height: 100vh;
			display: grid;
			place-items: center;
			padding: 24px;
			background: var(--bg);
			color: var(--text);
			font-family: system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
		}
		main {
			width: min(720px, 100%);
			background: var(--card);
			border: 1px solid var(--border);
			border-radius: 16px;
			padding: 24px;
			box-shadow: 0 18px 60px rgb(0 0 0 / 25%);
		}
		h1 { margin: 0 0 20px; font-size: 1.5rem; }
		.qr-wrap {
			display: flex;
			justify-content: center;
			overflow: auto;
			padding: 16px;
			background: #fff;
			border-radius: 12px;
		}
		.qr-wrap img {
			display: block;
			width: min({{.Size}}px, 100%);
			height: auto;
			image-rendering: pixelated;
		}
		.label {
			margin: 22px 0 8px;
			color: var(--muted);
			font-size: .9rem;
		}
		pre {
			margin: 0;
			padding: 14px;
			white-space: pre-wrap;
			overflow-wrap: anywhere;
			background: rgb(0 0 0 / 20%);
			border: 1px solid var(--border);
			border-radius: 10px;
			font: inherit;
		}
		.actions {
			display: flex;
			flex-wrap: wrap;
			gap: 10px;
			margin-top: 14px;
		}
		button, a.button {
			border: 0;
			border-radius: 9px;
			padding: 10px 14px;
			background: var(--button);
			color: #fff;
			font: inherit;
			text-decoration: none;
			cursor: pointer;
		}
		button:hover, a.button:hover { background: var(--button-hover); }
		.status { min-height: 1.4em; margin-top: 10px; color: var(--muted); }
	</style>
</head>
<body>
	<main>
		<h1>VaultQR</h1>

		<div class="qr-wrap">
			<img id="qr" src="{{.Image}}" alt="QR-Code">
		</div>

		{{if .ShowText}}
		<div class="label">Inhalt</div>
		<pre id="content">{{.Content}}</pre>
		{{end}}

		<div class="actions">
			{{if .ShowText}}<button id="copy" type="button">Text kopieren</button>{{end}}
			<a class="button" href="{{.Image}}" download="vaultqr.png">PNG speichern</a>
		</div>
		<div class="status" id="status" aria-live="polite"></div>
	</main>

	{{if .ShowText}}
	<script>
		const text = document.getElementById("content").textContent;
		const status = document.getElementById("status");

		document.getElementById("copy").addEventListener("click", async () => {
			try {
				await navigator.clipboard.writeText(text);
				status.textContent = "Text wurde kopiert.";
			} catch {
				const area = document.createElement("textarea");
				area.value = text;
				document.body.appendChild(area);
				area.select();
				document.execCommand("copy");
				area.remove();
				status.textContent = "Text wurde kopiert.";
			}
		});
	</script>
	{{end}}
</body>
</html>`))

func main() {
	cfg := loadConfig()
	s := &store{entries: make(map[string]qrEntry)}

	go s.cleanupLoop()

	mux := http.NewServeMux()
	mux.HandleFunc("/", createHandler(cfg, s))
	mux.HandleFunc("/view", viewHandler(cfg, s))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           requestLogger(mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("VaultQR startet auf Port %s, Standardgröße %d px, Maximalgröße %d px, Secret-Schutz: %t",
		cfg.Port, cfg.DefaultSize, cfg.MaxSize, cfg.Secret != "")

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func loadConfig() config {
	maxSize := envInt("MAX_SIZE", 1024)
	if maxSize < 64 {
		maxSize = 64
	}

	defaultSize := envInt("SIZE", 400)
	if defaultSize < 64 {
		defaultSize = 64
	}
	if defaultSize > maxSize {
		defaultSize = maxSize
	}

	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = "8080"
	}

	ttlMinutes := envInt("SESSION_TTL_MINUTES", 30)
	if ttlMinutes < 1 {
		ttlMinutes = 1
	}

	return config{
		Port:        port,
		Secret:      os.Getenv("SECRET"),
		DefaultSize: defaultSize,
		MaxSize:     maxSize,
		SessionTTL:  time.Duration(ttlMinutes) * time.Minute,
	}
}

func envInt(name string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		log.Printf("%s=%q ist ungültig, verwende %d", name, value, fallback)
		return fallback
	}
	return n
}

func createHandler(cfg config, s *store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		query := r.URL.Query()
		if len(query) == 0 {
			http.Redirect(w, r, "/view", http.StatusSeeOther)
			return
		}

		if cfg.Secret != "" && !secretMatches(cfg.Secret, query.Get("secret")) {
			http.Error(w, "403 Forbidden", http.StatusForbidden)
			return
		}

		content := query.Get("content")
		showText := false

		if content == "" {
			content = query.Get("text")
			showText = content != ""
		}

		if content == "" {
			http.Error(w, "Parameter 'content' oder 'text' fehlt", http.StatusBadRequest)
			return
		}

		size := cfg.DefaultSize
		if requested := strings.TrimSpace(query.Get("size")); requested != "" {
			parsed, err := strconv.Atoi(requested)
			if err != nil || parsed < 64 {
				http.Error(w, "Ungültige Größe: mindestens 64 Pixel", http.StatusBadRequest)
				return
			}
			size = parsed
		}
		if size > cfg.MaxSize {
			http.Error(w, fmt.Sprintf("Größe überschreitet MAX_SIZE=%d", cfg.MaxSize), http.StatusBadRequest)
			return
		}

		token, err := randomToken()
		if err != nil {
			http.Error(w, "Interner Fehler", http.StatusInternalServerError)
			return
		}

		s.set(token, qrEntry{
			Content:  content,
			ShowText: showText,
			Size:     size,
			Expires:  time.Now().Add(cfg.SessionTTL),
		})

		http.SetCookie(w, &http.Cookie{
			Name:     "vaultqr_session",
			Value:    token,
			Path:     "/",
			MaxAge:   int(cfg.SessionTTL.Seconds()),
			HttpOnly: true,
			Secure:   requestIsHTTPS(r),
			SameSite: http.SameSiteStrictMode,
		})

		http.Redirect(w, r, "/view", http.StatusSeeOther)
	}
}

func viewHandler(_ config, s *store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("vaultqr_session")
		if err != nil {
			http.Error(w, "Kein gültiger QR-Code-Aufruf vorhanden", http.StatusBadRequest)
			return
		}

		entry, ok := s.get(cookie.Value)
		if !ok || time.Now().After(entry.Expires) {
			http.Error(w, "QR-Code-Sitzung ist abgelaufen", http.StatusGone)
			return
		}

		png, err := qrcode.Encode(entry.Content, qrcode.Medium, entry.Size)
		if err != nil {
			http.Error(w, "QR-Code konnte nicht erzeugt werden", http.StatusInternalServerError)
			log.Printf("QR-Erzeugung fehlgeschlagen: %v", err)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy",
			"default-src 'none'; img-src data:; style-src 'unsafe-inline'; script-src 'unsafe-inline'; base-uri 'none'; form-action 'none'")

		data := pageData{
			Content:  entry.Content,
			ShowText: entry.ShowText,
			Image:    "data:image/png;base64," + base64.StdEncoding.EncodeToString(png),
			Size:     entry.Size,
		}
		if err := page.Execute(w, data); err != nil {
			log.Printf("Template-Ausgabe fehlgeschlagen: %v", err)
		}
	}
}

func randomToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func requestIsHTTPS(r *http.Request) bool {
	return r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

func secretMatches(expected, supplied string) bool {
	if len(expected) != len(supplied) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(expected), []byte(supplied)) == 1
}

func (s *store) set(token string, entry qrEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries[token] = entry
}

func (s *store) get(token string) (qrEntry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, ok := s.entries[token]
	return entry, ok
}

func (s *store) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		s.mu.Lock()
		for token, entry := range s.entries {
			if now.After(entry.Expires) {
				delete(s.entries, token)
			}
		}
		s.mu.Unlock()
	}
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s von %s in %s", r.Method, r.URL.Path, r.RemoteAddr, time.Since(start))
	})
}
