package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"personal_webside/internal/articles"
	"personal_webside/internal/profile"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	staticDir := flag.String("static", "frontend/dist", "frontend build output directory")
	dbPath := flag.String("db", "data/homepage.db", "SQLite database path")
	contentDir := flag.String("content", "content/articles", "article content directory")
	flag.Parse()

	articleStore, err := articles.Open(*dbPath)
	if err != nil {
		log.Fatalf("open article database: %v", err)
	}
	defer articleStore.Close()

	if err := articleStore.SyncContentDir(context.Background(), *contentDir); err != nil {
		log.Fatalf("sync article content: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", withCORS(handleHealth))
	mux.HandleFunc("/api/profile", withCORS(handleProfile))
	mux.HandleFunc("/api/articles", withCORS(handleArticles(articleStore)))
	mux.HandleFunc("/api/articles/", withCORS(handleArticle(articleStore)))
	mux.HandleFunc("/api/stream", withCORS(handleStream))

	if dirExists(*staticDir) {
		mux.Handle("/", spaHandler(*staticDir))
	}

	server := &http.Server{
		Addr:              *addr,
		Handler:           logRequests(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("personal homepage service listening on %s", *addr)
	if !dirExists(*staticDir) {
		log.Printf("static directory %q not found; API-only mode enabled", *staticDir)
	}
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
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
		{Level: "go-api", Label: "Profile API", Detail: "JSON resume graph served from Go net/http", Progress: 96},
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

func withCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
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

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
	})
}
