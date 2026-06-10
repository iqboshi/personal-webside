package sitecontent

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"

	"personal_webside/internal/articles"
	"personal_webside/internal/profile"
)

const (
	MaxProfileJSONBytes  = 256 * 1024
	MaxArticlesJSONBytes = 2 * 1024 * 1024
)

type Content struct {
	Profile   profile.SiteData   `json:"profile"`
	Articles  []articles.Article `json:"articles"`
	UpdatedAt string             `json:"updatedAt"`
}

type Draft struct {
	Profile  profile.SiteData   `json:"profile"`
	Articles []articles.Article `json:"articles"`
}

type Store struct {
	db      *sql.DB
	dialect storeDialect
}

type storeDialect string

const (
	sqliteDialect   storeDialect = "sqlite"
	postgresDialect storeDialect = "postgres"
)

func OpenFromEnv(sqlitePath string) (*Store, error) {
	dsn := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if dsn == "" {
		return Open(sqlitePath)
	}
	return OpenPostgres(dsn)
}

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	store := &Store{db: db, dialect: sqliteDialect}
	if err := store.init(context.Background()); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

func OpenPostgres(dsn string) (*Store, error) {
	db, err := sql.Open("pgx", normalizePostgresDSN(dsn))
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}

	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(30 * time.Minute)

	store := &Store{db: db, dialect: postgresDialect}
	if err := store.init(context.Background()); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) EnsureDefaults(ctx context.Context, defaultProfile profile.SiteData, defaultArticles []articles.Article) error {
	if err := validateProfile(defaultProfile); err != nil {
		return err
	}
	if err := validateArticles(defaultArticles); err != nil {
		return err
	}

	exists, err := s.exists(ctx)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	return s.Save(ctx, Draft{Profile: defaultProfile, Articles: defaultArticles})
}

func (s *Store) Get(ctx context.Context) (Content, error) {
	profileJSON, articlesJSON, updatedAt, err := s.readRaw(ctx)
	if err != nil {
		return Content{}, err
	}

	var content Content
	if err := json.Unmarshal([]byte(profileJSON), &content.Profile); err != nil {
		return Content{}, fmt.Errorf("parse profile content: %w", err)
	}
	if err := json.Unmarshal([]byte(articlesJSON), &content.Articles); err != nil {
		return Content{}, fmt.Errorf("parse article content: %w", err)
	}
	content.UpdatedAt = updatedAt
	return content, nil
}

func (s *Store) Save(ctx context.Context, draft Draft) error {
	if err := validateProfile(draft.Profile); err != nil {
		return err
	}
	if err := validateArticles(draft.Articles); err != nil {
		return err
	}

	profileJSON, err := json.Marshal(draft.Profile)
	if err != nil {
		return fmt.Errorf("marshal profile content: %w", err)
	}
	if len(profileJSON) > MaxProfileJSONBytes {
		return fmt.Errorf("profile content must be %d bytes or fewer", MaxProfileJSONBytes)
	}

	articlesJSON, err := json.Marshal(draft.Articles)
	if err != nil {
		return fmt.Errorf("marshal article content: %w", err)
	}
	if len(articlesJSON) > MaxArticlesJSONBytes {
		return fmt.Errorf("article content must be %d bytes or fewer", MaxArticlesJSONBytes)
	}

	if s.dialect == postgresDialect {
		_, err = s.db.ExecContext(ctx, `
			INSERT INTO site_content (id, profile_json, articles_json, updated_at)
			VALUES (1, $1::jsonb, $2::jsonb, now())
			ON CONFLICT(id) DO UPDATE SET
				profile_json = excluded.profile_json,
				articles_json = excluded.articles_json,
				updated_at = excluded.updated_at
		`, string(profileJSON), string(articlesJSON))
	} else {
		now := time.Now().Format(time.RFC3339)
		_, err = s.db.ExecContext(ctx, `
			INSERT INTO site_content (id, profile_json, articles_json, updated_at)
			VALUES (1, ?, ?, ?)
			ON CONFLICT(id) DO UPDATE SET
				profile_json = excluded.profile_json,
				articles_json = excluded.articles_json,
				updated_at = excluded.updated_at
		`, string(profileJSON), string(articlesJSON), now)
	}
	if err != nil {
		return fmt.Errorf("save site content: %w", err)
	}
	return nil
}

func (s *Store) exists(ctx context.Context) (bool, error) {
	var value int
	err := s.db.QueryRowContext(ctx, `SELECT 1 FROM site_content WHERE id = 1`).Scan(&value)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return false, fmt.Errorf("check site content: %w", err)
}

func (s *Store) readRaw(ctx context.Context) (string, string, string, error) {
	query := `SELECT profile_json, articles_json, updated_at FROM site_content WHERE id = 1`
	if s.dialect == postgresDialect {
		query = `
			SELECT
				profile_json::text,
				articles_json::text,
				to_char(updated_at AT TIME ZONE 'Asia/Shanghai', 'YYYY-MM-DD"T"HH24:MI:SS"+08:00"') AS updated_at
			FROM site_content
			WHERE id = 1
		`
	}

	var profileJSON string
	var articlesJSON string
	var updatedAt string
	err := s.db.QueryRowContext(ctx, query).Scan(&profileJSON, &articlesJSON, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", "", errors.New("site content is not initialized")
	}
	if err != nil {
		return "", "", "", fmt.Errorf("read site content: %w", err)
	}
	return profileJSON, articlesJSON, updatedAt, nil
}

func (s *Store) init(ctx context.Context) error {
	if s.dialect == postgresDialect {
		return s.initPostgres(ctx)
	}
	return s.initSQLite(ctx)
}

func (s *Store) initSQLite(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, `PRAGMA journal_mode = WAL`); err != nil {
		return fmt.Errorf("set journal mode: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS site_content (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			profile_json TEXT NOT NULL,
			articles_json TEXT NOT NULL,
			updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		return fmt.Errorf("create site content table: %w", err)
	}
	return nil
}

func (s *Store) initPostgres(ctx context.Context) error {
	if err := s.db.PingContext(ctx); err != nil {
		return fmt.Errorf("connect postgres: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS site_content (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			profile_json JSONB NOT NULL,
			articles_json JSONB NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)
	`); err != nil {
		return fmt.Errorf("create site content table: %w", err)
	}
	return nil
}

func validateProfile(value profile.SiteData) error {
	if strings.TrimSpace(value.Person.Name) == "" {
		return errors.New("profile person.name is required")
	}
	if strings.TrimSpace(value.Person.English) == "" {
		return errors.New("profile person.english is required")
	}
	if strings.TrimSpace(value.Hero.Headline) == "" {
		return errors.New("profile hero.headline is required")
	}
	return nil
}

func validateArticles(items []articles.Article) error {
	seen := make(map[string]struct{}, len(items))
	for index := range items {
		if err := articles.Normalize(&items[index]); err != nil {
			return err
		}
		if _, ok := seen[items[index].Slug]; ok {
			return fmt.Errorf("duplicate article slug %q", items[index].Slug)
		}
		seen[items[index].Slug] = struct{}{}
	}
	return nil
}

func normalizePostgresDSN(dsn string) string {
	if strings.Contains(dsn, "sslmode=") {
		return dsn
	}
	separator := "?"
	if strings.Contains(dsn, "?") {
		separator = "&"
	}
	return dsn + separator + "sslmode=require"
}
