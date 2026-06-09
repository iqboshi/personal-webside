package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"personal_webside/internal/articles"
	"personal_webside/internal/checkins"
	"personal_webside/internal/profile"
	"personal_webside/internal/todos"
)

const adminCookieName = "mengqing_admin_session"

func main() {
	if err := loadEnvFile(envOrDefault("ENV_FILE", "tmp/supabase.env")); err != nil {
		log.Printf("load env file: %v", err)
	}

	addr := flag.String("addr", defaultListenAddr(), "HTTP listen address")
	staticDir := flag.String("static", envOrDefault("STATIC_DIR", "frontend/dist"), "frontend build output directory")
	dbPath := flag.String("db", envOrDefault("DB_PATH", "data/homepage.db"), "SQLite database path")
	contentDir := flag.String("content", envOrDefault("CONTENT_DIR", "content/articles"), "article content directory")
	flag.Parse()

	articleStore, err := articles.Open(*dbPath)
	if err != nil {
		log.Fatalf("open article database: %v", err)
	}
	defer articleStore.Close()

	if err := articleStore.SyncContentDir(context.Background(), *contentDir); err != nil {
		log.Fatalf("sync article content: %v", err)
	}

	checkinStore, err := checkins.OpenFromEnv(*dbPath)
	if err != nil {
		log.Fatalf("open check-in database: %v", err)
	}
	defer checkinStore.Close()

	todoStore, err := todos.OpenFromEnv(*dbPath)
	if err != nil {
		log.Fatalf("open todo database: %v", err)
	}
	defer todoStore.Close()

	auth := newAdminAuth()
	cors := newCORSConfig()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", cors.wrap(handleHealth))
	mux.HandleFunc("/api/profile", cors.wrap(handleProfile))
	mux.HandleFunc("/api/articles", cors.wrap(handleArticles(articleStore)))
	mux.HandleFunc("/api/articles/", cors.wrap(handleArticle(articleStore)))
	mux.HandleFunc("/api/checkins", cors.wrap(handleCheckins(checkinStore)))
	mux.HandleFunc("/api/todos/today", cors.wrap(handleTodayTodos(todoStore)))
	mux.HandleFunc("/api/admin/session", cors.wrap(auth.handleSession))
	mux.HandleFunc("/api/admin/login", cors.wrap(auth.handleLogin))
	mux.HandleFunc("/api/admin/logout", cors.wrap(auth.handleLogout))
	mux.HandleFunc("/api/admin/checkins/", cors.wrap(auth.requireAdmin(handleAdminCheckin(checkinStore, todoStore))))
	mux.HandleFunc("/api/admin/todos/tasks", cors.wrap(auth.requireAdmin(handleAdminTodoTasks(todoStore))))
	mux.HandleFunc("/api/admin/todos/complete", cors.wrap(auth.requireAdmin(handleAdminTodoCompletion(todoStore))))
	mux.HandleFunc("/api/stream", cors.wrap(handleStream))

	if dirExists(*staticDir) {
		mux.Handle("/", spaHandler(*staticDir))
	}

	server := &http.Server{
		Addr:              *addr,
		Handler:           logRequests(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("personal homepage service listening on %s", *addr)
	if !auth.enabled {
		log.Printf("admin auth is disabled; set ADMIN_PASSWORD and SESSION_SECRET to enable write APIs")
	}
	if !dirExists(*staticDir) {
		log.Printf("static directory %q not found; API-only mode enabled", *staticDir)
	}
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func handleCheckins(store *checkins.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		items, err := store.List(r.Context())
		if err != nil {
			log.Printf("list checkins: %v", err)
			http.Error(w, "failed to load check-ins", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string][]checkins.Checkin{"items": items})
	}
}

func handleTodayTodos(store *todos.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		dateKey := strings.TrimSpace(r.URL.Query().Get("date"))
		if dateKey == "" {
			dateKey = store.TodayKey()
		}
		status, err := store.DayStatus(r.Context(), dateKey)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, status)
	}
}

func handleAdminCheckin(checkinStore *checkins.Store, todoStore *todos.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		dateKey := strings.TrimPrefix(r.URL.Path, "/api/admin/checkins/")
		if dateKey == "" || strings.Contains(dateKey, "/") {
			http.NotFound(w, r)
			return
		}

		var input struct {
			Note string `json:"note"`
		}
		if err := readJSONBody(r, &input, 4096); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		_, found, err := checkinStore.Get(r.Context(), dateKey)
		if err != nil {
			log.Printf("get check-in before save: %v", err)
			http.Error(w, "failed to load check-in", http.StatusInternalServerError)
			return
		}
		if !found && dateKey == todoStore.TodayKey() {
			unlocked, todoStatus, err := todoStore.CanUnlockCheckin(r.Context(), dateKey)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if !unlocked {
				writeJSON(w, http.StatusConflict, map[string]any{
					"error": "todo_not_completed",
					"todos": todoStatus,
				})
				return
			}
		}

		item, err := checkinStore.Upsert(r.Context(), dateKey, input.Note)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, item)
	}
}

func handleAdminTodoTasks(store *todos.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var input struct {
			Date  string            `json:"date"`
			Tasks []todos.TaskDraft `json:"tasks"`
		}
		if err := readJSONBody(r, &input, 8192); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if input.Date == "" {
			input.Date = store.TodayKey()
		}

		status, err := store.ReplaceTasks(r.Context(), input.Tasks, input.Date)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, status)
	}
}

func handleAdminTodoCompletion(store *todos.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var input struct {
			Date   string `json:"date"`
			TaskID string `json:"taskId"`
			Done   bool   `json:"done"`
		}
		if err := readJSONBody(r, &input, 4096); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if input.Date == "" {
			input.Date = store.TodayKey()
		}

		status, err := store.SetCompletion(r.Context(), input.Date, input.TaskID, input.Done)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, status)
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
		"module": "personal-homepage",
		"time":   time.Now().Format(time.RFC3339),
	})
}

func handleProfile(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, profile.Data())
}

func handleArticles(store *articles.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		items, err := store.List(r.Context())
		if err != nil {
			log.Printf("list articles: %v", err)
			http.Error(w, "failed to load articles", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, items)
	}
}

func handleArticle(store *articles.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		slug := strings.TrimPrefix(r.URL.Path, "/api/articles/")
		if slug == "" || strings.Contains(slug, "/") {
			http.NotFound(w, r)
			return
		}

		item, found, err := store.Get(r.Context(), slug)
		if err != nil {
			log.Printf("get article %q: %v", slug, err)
			http.Error(w, "failed to load article", http.StatusInternalServerError)
			return
		}
		if !found {
			http.NotFound(w, r)
			return
		}

		writeJSON(w, http.StatusOK, item)
	}
}

func handleStream(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	signals := []profile.Signal{
		{Level: "go-api", Label: "Profile API", Detail: "JSON site data served from Go net/http", Progress: 96},
		{Level: "cv", Label: "Vision Pipeline", Detail: "Remote sensing image tasks mapped to model outputs", Progress: 88},
		{Level: "rl-llm", Label: "Decision Replay", Detail: "Agent trajectory and reasoning states are structured for playback", Progress: 82},
		{Level: "workflow", Label: "Data Platform", Detail: "Assets, maps, workflows and model products stay connected", Progress: 91},
		{Level: "research", Label: "Paper Queue", Detail: "Agricultural AI readings and manuscripts remain in active iteration", Progress: 76},
	}

	ticker := time.NewTicker(1400 * time.Millisecond)
	defer ticker.Stop()

	index := 0
	for {
		if err := writeSignal(r.Context(), w, signals[index%len(signals)]); err != nil {
			return
		}
		flusher.Flush()
		index++

		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
		}
	}
}

func writeSignal(ctx context.Context, w http.ResponseWriter, signal profile.Signal) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	signal.Timestamp = time.Now().Format("15:04:05")
	payload, err := json.Marshal(signal)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(w, "event: signal\ndata: %s\n\n", payload)
	return err
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		log.Printf("write json: %v", err)
	}
}

func readJSONBody(r *http.Request, value any, limit int64) error {
	defer r.Body.Close()
	decoder := json.NewDecoder(io.LimitReader(r.Body, limit))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain one JSON object")
	}
	return nil
}

type corsConfig struct {
	origins map[string]struct{}
}

func newCORSConfig() corsConfig {
	raw := os.Getenv("CORS_ALLOWED_ORIGINS")
	if raw == "" {
		raw = os.Getenv("ADMIN_ALLOWED_ORIGINS")
	}
	if raw == "" {
		raw = "http://127.0.0.1:5173,http://localhost:5173,http://127.0.0.1:5174,http://localhost:5174,http://127.0.0.1:5175,http://localhost:5175,https://iqboshi.github.io"
	}

	origins := make(map[string]struct{})
	for _, item := range strings.Split(raw, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		origins[item] = struct{}{}
	}
	return corsConfig{origins: origins}
}

func (c corsConfig) wrap(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); origin != "" {
			if _, ok := c.origins[origin]; ok {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}

type adminAuth struct {
	username       string
	password       string
	passwordSHA256 string
	sessionSecret  []byte
	enabled        bool
}

type adminSession struct {
	Username  string `json:"username"`
	ExpiresAt int64  `json:"expiresAt"`
}

func newAdminAuth() *adminAuth {
	username := strings.TrimSpace(os.Getenv("ADMIN_USERNAME"))
	if username == "" {
		username = "admin"
	}

	password := os.Getenv("ADMIN_PASSWORD")
	passwordSHA256 := strings.TrimSpace(os.Getenv("ADMIN_PASSWORD_SHA256"))
	secret := os.Getenv("SESSION_SECRET")

	return &adminAuth{
		username:       username,
		password:       password,
		passwordSHA256: passwordSHA256,
		sessionSecret:  []byte(secret),
		enabled:        (password != "" || passwordSHA256 != "") && secret != "",
	}
}

func (a *adminAuth) handleSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	session, ok := a.sessionFromRequest(r)
	if !ok {
		session, ok = a.bearerSession(r)
	}
	if !ok {
		writeJSON(w, http.StatusOK, map[string]any{"authenticated": false})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"authenticated": true,
		"username":      session.Username,
	})
}

func (a *adminAuth) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !a.enabled {
		http.Error(w, "admin auth is not configured", http.StatusServiceUnavailable)
		return
	}

	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := readJSONBody(r, &input, 4096); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(input.Username) != a.username || !a.matchPassword(input.Password) {
		http.Error(w, "invalid username or password", http.StatusUnauthorized)
		return
	}

	token, err := a.signSession(adminSession{
		Username:  a.username,
		ExpiresAt: time.Now().Add(14 * 24 * time.Hour).Unix(),
	})
	if err != nil {
		log.Printf("sign admin session: %v", err)
		http.Error(w, "failed to create session", http.StatusInternalServerError)
		return
	}
	a.setSessionCookie(w, r, token, int((14 * 24 * time.Hour).Seconds()))
	writeJSON(w, http.StatusOK, map[string]any{
		"authenticated": true,
		"username":      a.username,
		"token":         token,
	})
}

func (a *adminAuth) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	a.setSessionCookie(w, r, "", -1)
	writeJSON(w, http.StatusOK, map[string]any{"authenticated": false})
}

func (a *adminAuth) requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := a.sessionFromRequest(r); !ok && !a.validBearerToken(r) {
			http.Error(w, "admin login required", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func (a *adminAuth) validBearerToken(r *http.Request) bool {
	_, ok := a.bearerSession(r)
	return ok
}

func (a *adminAuth) bearerSession(r *http.Request) (adminSession, bool) {
	if !a.enabled {
		return adminSession{}, false
	}

	header := strings.TrimSpace(r.Header.Get("Authorization"))
	token, ok := strings.CutPrefix(header, "Bearer ")
	if !ok || strings.TrimSpace(token) == "" {
		return adminSession{}, false
	}

	session, err := a.verifySession(strings.TrimSpace(token))
	if err != nil || session.ExpiresAt < time.Now().Unix() || session.Username != a.username {
		return adminSession{}, false
	}
	return session, true
}

func (a *adminAuth) matchPassword(value string) bool {
	if a.passwordSHA256 != "" {
		hash := sha256.Sum256([]byte(value))
		encoded := fmt.Sprintf("%x", hash)
		return subtle.ConstantTimeCompare([]byte(encoded), []byte(strings.ToLower(a.passwordSHA256))) == 1
	}
	return subtle.ConstantTimeCompare([]byte(value), []byte(a.password)) == 1
}

func (a *adminAuth) sessionFromRequest(r *http.Request) (adminSession, bool) {
	if !a.enabled {
		return adminSession{}, false
	}

	cookie, err := r.Cookie(adminCookieName)
	if err != nil || cookie.Value == "" {
		return adminSession{}, false
	}

	session, err := a.verifySession(cookie.Value)
	if err != nil || session.ExpiresAt < time.Now().Unix() || session.Username != a.username {
		return adminSession{}, false
	}
	return session, true
}

func (a *adminAuth) signSession(session adminSession) (string, error) {
	payload, err := json.Marshal(session)
	if err != nil {
		return "", err
	}
	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	signature := a.sign(encodedPayload)
	return encodedPayload + "." + signature, nil
}

func (a *adminAuth) verifySession(token string) (adminSession, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return adminSession{}, errors.New("invalid session token")
	}

	expectedSignature := a.sign(parts[0])
	if subtle.ConstantTimeCompare([]byte(parts[1]), []byte(expectedSignature)) != 1 {
		return adminSession{}, errors.New("invalid session signature")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return adminSession{}, err
	}

	var session adminSession
	if err := json.Unmarshal(payload, &session); err != nil {
		return adminSession{}, err
	}
	return session, nil
}

func (a *adminAuth) sign(value string) string {
	mac := hmac.New(sha256.New, a.sessionSecret)
	mac.Write([]byte(value))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (a *adminAuth) setSessionCookie(w http.ResponseWriter, r *http.Request, value string, maxAge int) {
	secure := r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
	sameSite := http.SameSiteLaxMode
	if secure {
		sameSite = http.SameSiteNoneMode
	}

	http.SetCookie(w, &http.Cookie{
		Name:     adminCookieName,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   secure,
		SameSite: sameSite,
	})
}

func spaHandler(staticDir string) http.Handler {
	fileServer := http.FileServer(http.Dir(staticDir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.ServeFile(w, r, filepath.Join(staticDir, "index.html"))
			return
		}

		cleanPath := filepath.Clean(strings.TrimPrefix(r.URL.Path, "/"))
		target := filepath.Join(staticDir, cleanPath)
		if info, err := os.Stat(target); err == nil && !info.IsDir() {
			fileServer.ServeHTTP(w, r)
			return
		}

		http.ServeFile(w, r, filepath.Join(staticDir, "index.html"))
	})
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func defaultListenAddr() string {
	if port := strings.TrimSpace(os.Getenv("PORT")); port != "" {
		return ":" + port
	}
	return envOrDefault("ADDR", ":8080")
}

func envOrDefault(key string, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func loadEnvFile(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}

	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}

	for index, rawLine := range strings.Split(string(content), "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return fmt.Errorf("%s:%d must be KEY=value", path, index+1)
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" {
			return fmt.Errorf("%s:%d has empty key", path, index+1)
		}
		if _, exists := os.LookupEnv(key); !exists {
			os.Setenv(key, strings.Trim(value, `"'`))
		}
	}
	return nil
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
	})
}
