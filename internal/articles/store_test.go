package articles

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadContentDir(t *testing.T) {
	dir := t.TempDir()
	writeArticleFile(t, dir, "001-demo.json", `{
  "slug": "demo-project",
  "category": "项目",
  "date": "2026-06-04",
  "title": "Demo Project",
  "excerpt": "A short project note.",
  "tags": ["Go", "Go", "SQLite"],
  "blocks": [
    { "type": "heading", "level": 2, "text": "Overview" },
    { "type": "paragraph", "text": "Project body." }
  ]
}`)

	items, signature, err := LoadContentDir(dir)
	if err != nil {
		t.Fatalf("LoadContentDir() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("LoadContentDir() returned %d articles, want 1", len(items))
	}
	if signature == "" {
		t.Fatal("LoadContentDir() returned empty signature")
	}
	if got := items[0].Body; len(got) != 2 {
		t.Fatalf("article body len = %d, want 2", len(got))
	}
	if got := items[0].Tags; len(got) != 2 {
		t.Fatalf("deduped tag len = %d, want 2", len(got))
	}
}

func TestSyncContentDir(t *testing.T) {
	dir := t.TempDir()
	writeArticleFile(t, dir, "001-demo.json", `{
  "slug": "demo-project",
  "category": "项目",
  "date": "2026-06-04",
  "title": "Demo Project",
  "excerpt": "A short project note.",
  "tags": ["Go"],
  "blocks": [
    { "type": "heading", "level": 2, "text": "Overview" },
    { "type": "code", "language": "go", "filename": "main.go", "code": "package main" }
  ]
}`)

	store, err := Open(filepath.Join(t.TempDir(), "homepage.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()

	if err := store.SyncContentDir(context.Background(), dir); err != nil {
		t.Fatalf("SyncContentDir() error = %v", err)
	}

	item, ok, err := store.Get(context.Background(), "demo-project")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !ok {
		t.Fatal("Get() found = false, want true")
	}
	if got := len(item.Blocks); got != 2 {
		t.Fatalf("block len = %d, want 2", got)
	}
	if item.Body[1] != "package main" {
		t.Fatalf("body[1] = %q, want code text", item.Body[1])
	}
}

func writeArticleFile(t *testing.T, dir, name, payload string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(payload), 0644); err != nil {
		t.Fatalf("write article file: %v", err)
	}
}
