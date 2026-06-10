package articles

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "modernc.org/sqlite"
)

type Article struct {
	Slug     string         `json:"slug"`
	Category string         `json:"category"`
	Date     string         `json:"date"`
	Title    string         `json:"title"`
	Excerpt  string         `json:"excerpt"`
	Tags     []string       `json:"tags"`
	Body     []string       `json:"body,omitempty"`
	Blocks   []ArticleBlock `json:"blocks"`
}

type ArticleBlock struct {
	Type     string        `json:"type"`
	Level    int           `json:"level,omitempty"`
	Text     string        `json:"text,omitempty"`
	Items    []string      `json:"items,omitempty"`
	Language string        `json:"language,omitempty"`
	Filename string        `json:"filename,omitempty"`
	Code     string        `json:"code,omitempty"`
	Src      string        `json:"src,omitempty"`
	Alt      string        `json:"alt,omitempty"`
	Caption  string        `json:"caption,omitempty"`
	Title    string        `json:"title,omitempty"`
	Links    []ArticleLink `json:"links,omitempty"`
}

type ArticleLink struct {
	Label string `json:"label"`
	Href  string `json:"href"`
	Note  string `json:"note,omitempty"`
}

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	store := &Store{db: db}
	if err := store.init(context.Background()); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) SyncContentDir(ctx context.Context, dir string) error {
	items, signature, err := LoadContentDir(dir)
	if err != nil {
		return err
	}
	return s.ReplaceAll(ctx, items, signature)
}

func LoadContentDir(dir string) ([]Article, string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, "", fmt.Errorf("read article content directory %q: %w", dir, err)
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".json") {
			continue
		}
		files = append(files, filepath.Join(dir, entry.Name()))
	}
	sort.Strings(files)
	if len(files) == 0 {
		return nil, "", fmt.Errorf("article content directory %q has no .json files", dir)
	}

	hasher := sha256.New()
	articles := make([]Article, 0, len(files))
	seenSlugs := make(map[string]string, len(files))
	for _, path := range files {
		payload, err := os.ReadFile(path)
		if err != nil {
			return nil, "", fmt.Errorf("read article content %q: %w", path, err)
		}
		payload = bytes.TrimPrefix(payload, []byte{0xef, 0xbb, 0xbf})

		relativePath, err := filepath.Rel(dir, path)
		if err != nil {
			return nil, "", fmt.Errorf("resolve article content path %q: %w", path, err)
		}

		hasher.Write([]byte(filepath.ToSlash(relativePath)))
		hasher.Write([]byte{0})
		hasher.Write(payload)
		hasher.Write([]byte{0})

		var article Article
		if err := json.Unmarshal(payload, &article); err != nil {
			return nil, "", fmt.Errorf("parse article content %q: %w", path, err)
		}
		if err := normalizeArticle(&article); err != nil {
			return nil, "", fmt.Errorf("validate article content %q: %w", path, err)
		}
		if previousPath, ok := seenSlugs[article.Slug]; ok {
			return nil, "", fmt.Errorf("duplicate article slug %q in %q and %q", article.Slug, previousPath, path)
		}
		seenSlugs[article.Slug] = path
		articles = append(articles, article)
	}

	return articles, hex.EncodeToString(hasher.Sum(nil)), nil
}

func normalizeArticle(article *Article) error {
	article.Slug = strings.TrimSpace(article.Slug)
	article.Category = strings.TrimSpace(article.Category)
	article.Date = strings.TrimSpace(article.Date)
	article.Title = strings.TrimSpace(article.Title)
	article.Excerpt = strings.TrimSpace(article.Excerpt)
	article.Tags = uniqueStrings(article.Tags)

	if article.Slug == "" {
		return errors.New("slug is required")
	}
	if article.Category == "" {
		return fmt.Errorf("article %q category is required", article.Slug)
	}
	if article.Date == "" {
		return fmt.Errorf("article %q date is required", article.Slug)
	}
	if article.Title == "" {
		return fmt.Errorf("article %q title is required", article.Slug)
	}
	if article.Excerpt == "" {
		return fmt.Errorf("article %q excerpt is required", article.Slug)
	}

	for index := range article.Blocks {
		normalizeBlock(&article.Blocks[index])
		if article.Blocks[index].Type == "" {
			return fmt.Errorf("article %q block %d type is required", article.Slug, index+1)
		}
	}

	if len(article.Blocks) == 0 {
		article.Blocks = blocksFromBody(article.Body)
	}
	if len(article.Blocks) == 0 {
		return fmt.Errorf("article %q must contain at least one block", article.Slug)
	}
	article.Body = bodyFromBlocks(article.Blocks)
	return nil
}

func Normalize(article *Article) error {
	return normalizeArticle(article)
}

func normalizeBlock(block *ArticleBlock) {
	block.Type = strings.TrimSpace(block.Type)
	block.Text = strings.TrimSpace(block.Text)
	block.Language = strings.TrimSpace(block.Language)
	block.Filename = strings.TrimSpace(block.Filename)
	block.Src = strings.TrimSpace(block.Src)
	block.Alt = strings.TrimSpace(block.Alt)
	block.Caption = strings.TrimSpace(block.Caption)
	block.Title = strings.TrimSpace(block.Title)
	block.Items = compactStrings(block.Items)

	for index := range block.Links {
		block.Links[index].Label = strings.TrimSpace(block.Links[index].Label)
		block.Links[index].Href = strings.TrimSpace(block.Links[index].Href)
		block.Links[index].Note = strings.TrimSpace(block.Links[index].Note)
	}
}

func (s *Store) ReplaceAll(ctx context.Context, articles []Article, signature string) error {
	if signature == "" {
		return errors.New("article content signature is required")
	}

	var currentSignature string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM article_metadata WHERE key = 'content_signature'`).Scan(&currentSignature)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("read article content signature: %w", err)
	}
	if currentSignature == signature {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin article sync transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM articles`); err != nil {
		return fmt.Errorf("clear article table: %w", err)
	}

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO articles (slug, category, published_at, title, excerpt, tags_json, body_json, blocks_json, sort_order)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("prepare article sync statement: %w", err)
	}
	defer stmt.Close()

	for index, article := range articles {
		if err := normalizeArticle(&article); err != nil {
			return err
		}

		tagsJSON, err := json.Marshal(article.Tags)
		if err != nil {
			return fmt.Errorf("marshal tags for %q: %w", article.Title, err)
		}

		bodyJSON, err := json.Marshal(article.Body)
		if err != nil {
			return fmt.Errorf("marshal body for %q: %w", article.Title, err)
		}

		blocksJSON, err := json.Marshal(article.Blocks)
		if err != nil {
			return fmt.Errorf("marshal blocks for %q: %w", article.Title, err)
		}

		if _, err := stmt.ExecContext(
			ctx,
			article.Slug,
			article.Category,
			article.Date,
			article.Title,
			article.Excerpt,
			string(tagsJSON),
			string(bodyJSON),
			string(blocksJSON),
			index,
		); err != nil {
			return fmt.Errorf("sync article %q: %w", article.Title, err)
		}
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO article_metadata (key, value)
		VALUES ('content_signature', ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = CURRENT_TIMESTAMP
	`, signature); err != nil {
		return fmt.Errorf("write article content signature: %w", err)
	}

	return tx.Commit()
}

func (s *Store) List(ctx context.Context) ([]Article, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT slug, category, published_at, title, excerpt, tags_json, body_json, blocks_json
		FROM articles
		ORDER BY sort_order ASC, published_at DESC, rowid ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("query articles: %w", err)
	}
	defer rows.Close()

	var result []Article
	for rows.Next() {
		article, err := scanArticle(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, article)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate articles: %w", err)
	}
	return result, nil
}

func (s *Store) Get(ctx context.Context, slug string) (Article, bool, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT slug, category, published_at, title, excerpt, tags_json, body_json, blocks_json
		FROM articles
		WHERE slug = ?
	`, slug)

	article, err := scanArticle(row)
	if err == nil {
		return article, true, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return Article{}, false, nil
	}
	return Article{}, false, err
}

func (s *Store) init(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, `PRAGMA journal_mode = WAL`); err != nil {
		return fmt.Errorf("set journal mode: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, `PRAGMA foreign_keys = ON`); err != nil {
		return fmt.Errorf("enable foreign keys: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS articles (
			slug TEXT PRIMARY KEY,
			category TEXT NOT NULL,
			published_at TEXT NOT NULL,
			title TEXT NOT NULL,
			excerpt TEXT NOT NULL,
			tags_json TEXT NOT NULL,
			body_json TEXT NOT NULL,
			blocks_json TEXT NOT NULL DEFAULT '[]',
			sort_order INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		return fmt.Errorf("create articles table: %w", err)
	}
	if err := s.ensureColumn(ctx, "articles", "blocks_json", "TEXT NOT NULL DEFAULT '[]'"); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS article_metadata (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		return fmt.Errorf("create article metadata table: %w", err)
	}
	return nil
}

func (s *Store) ensureColumn(ctx context.Context, tableName, columnName, definition string) error {
	rows, err := s.db.QueryContext(ctx, `PRAGMA table_info(`+tableName+`)`)
	if err != nil {
		return fmt.Errorf("inspect %s columns: %w", tableName, err)
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name string
		var columnType string
		var notNull int
		var defaultValue sql.NullString
		var primaryKey int
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			return fmt.Errorf("scan %s columns: %w", tableName, err)
		}
		if name == columnName {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate %s columns: %w", tableName, err)
	}

	if _, err := s.db.ExecContext(ctx, fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", tableName, columnName, definition)); err != nil {
		return fmt.Errorf("add %s.%s column: %w", tableName, columnName, err)
	}
	return nil
}

type articleScanner interface {
	Scan(dest ...any) error
}

func scanArticle(scanner articleScanner) (Article, error) {
	var article Article
	var tagsJSON string
	var bodyJSON string
	var blocksJSON string

	if err := scanner.Scan(
		&article.Slug,
		&article.Category,
		&article.Date,
		&article.Title,
		&article.Excerpt,
		&tagsJSON,
		&bodyJSON,
		&blocksJSON,
	); err != nil {
		return Article{}, err
	}

	if err := json.Unmarshal([]byte(tagsJSON), &article.Tags); err != nil {
		return Article{}, fmt.Errorf("unmarshal article tags %q: %w", article.Slug, err)
	}
	if err := json.Unmarshal([]byte(bodyJSON), &article.Body); err != nil {
		return Article{}, fmt.Errorf("unmarshal article body %q: %w", article.Slug, err)
	}
	if err := json.Unmarshal([]byte(blocksJSON), &article.Blocks); err != nil {
		return Article{}, fmt.Errorf("unmarshal article blocks %q: %w", article.Slug, err)
	}
	if len(article.Blocks) == 0 {
		article.Blocks = blocksFromBody(article.Body)
	}
	return article, nil
}

func bodyFromBlocks(blocks []ArticleBlock) []string {
	var body []string
	for _, block := range blocks {
		switch block.Type {
		case "paragraph", "callout":
			if block.Text != "" {
				body = append(body, block.Text)
			}
		case "heading":
			if block.Text != "" {
				body = append(body, block.Text)
			}
		case "list":
			body = append(body, block.Items...)
		case "code":
			if block.Code != "" {
				body = append(body, block.Code)
			}
		case "image":
			if block.Caption != "" {
				body = append(body, block.Caption)
			}
		case "links":
			for _, link := range block.Links {
				body = append(body, link.Label, link.Note)
			}
		}
	}
	return compactStrings(body)
}

func blocksFromBody(body []string) []ArticleBlock {
	blocks := make([]ArticleBlock, 0, len(body))
	for _, paragraph := range compactStrings(body) {
		blocks = append(blocks, ArticleBlock{Type: "paragraph", Text: paragraph})
	}
	return blocks
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range compactStrings(values) {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func compactStrings(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		result = append(result, value)
	}
	return result
}
