package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"path/filepath"

	"personal_webside/internal/articles"
	"personal_webside/internal/profile"
)

func main() {
	contentDir := flag.String("content", "content/articles", "article content directory")
	outDir := flag.String("out", "frontend/public/data", "static data output directory")
	flag.Parse()

	items, _, err := articles.LoadContentDir(*contentDir)
	if err != nil {
		log.Fatalf("load article content: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(*outDir, "articles"), 0755); err != nil {
		log.Fatalf("create output directory: %v", err)
	}

	writeJSON(filepath.Join(*outDir, "profile.json"), profile.Data())
	writeJSON(filepath.Join(*outDir, "articles.json"), items)
	for _, item := range items {
		writeJSON(filepath.Join(*outDir, "articles", item.Slug+".json"), item)
	}
}

func writeJSON(path string, value any) {
	file, err := os.Create(path)
	if err != nil {
		log.Fatalf("create %s: %v", path, err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		log.Fatalf("write %s: %v", path, err)
	}
}
