package checkins

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

const MaxNoteLength = 240

var dateKeyPattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

type Checkin struct {
	Date      string `json:"date"`
	Note      string `json:"note"`
	CheckedAt string `json:"checkedAt"`
	UpdatedAt string `json:"updatedAt"`
}

type Store struct {
	db       *sql.DB
	location *time.Location
	dialect  storeDialect
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

	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		location = time.FixedZone("Asia/Shanghai", 8*60*60)
	}

	store := &Store{db: db, location: location, dialect: sqliteDialect}
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

	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		location = time.FixedZone("Asia/Shanghai", 8*60*60)
	}

	store := &Store{db: db, location: location, dialect: postgresDialect}
	if err := store.init(context.Background()); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) List(ctx context.Context) ([]Checkin, error) {
	query := `
		SELECT date_key, note, checked_at, updated_at
		FROM checkins
		ORDER BY date_key DESC
		LIMIT 500
	`
	if s.dialect == postgresDialect {
		query = `
			SELECT
				to_char(date_key, 'YYYY-MM-DD') AS date_key,
				note,
				to_char(checked_at AT TIME ZONE 'Asia/Shanghai', 'YYYY-MM-DD"T"HH24:MI:SS"+08:00"') AS checked_at,
				to_char(updated_at AT TIME ZONE 'Asia/Shanghai', 'YYYY-MM-DD"T"HH24:MI:SS"+08:00"') AS updated_at
			FROM checkins
			ORDER BY date_key DESC
			LIMIT 500
		`
	}

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query checkins: %w", err)
	}
	defer rows.Close()

	var result []Checkin
	for rows.Next() {
		var item Checkin
		if err := rows.Scan(&item.Date, &item.Note, &item.CheckedAt, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan checkin: %w", err)
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate checkins: %w", err)
	}
	return result, nil
}

func (s *Store) Upsert(ctx context.Context, dateKey string, note string) (Checkin, error) {
	dateKey, err := s.normalizeDateKey(dateKey)
	if err != nil {
		return Checkin{}, err
	}
	note, err = normalizeNote(note)
	if err != nil {
		return Checkin{}, err
	}

	now := time.Now().In(s.location).Format(time.RFC3339)
	if s.dialect == postgresDialect {
		if _, err := s.db.ExecContext(ctx, `
			INSERT INTO checkins (date_key, note, checked_at, updated_at)
			VALUES ($1::date, $2, now(), now())
			ON CONFLICT(date_key) DO UPDATE SET
				note = excluded.note,
				updated_at = excluded.updated_at
		`, dateKey, note); err != nil {
			return Checkin{}, fmt.Errorf("upsert checkin %q: %w", dateKey, err)
		}
	} else {
		if _, err := s.db.ExecContext(ctx, `
			INSERT INTO checkins (date_key, note, checked_at, updated_at)
			VALUES (?, ?, ?, ?)
			ON CONFLICT(date_key) DO UPDATE SET
				note = excluded.note,
				updated_at = excluded.updated_at
		`, dateKey, note, now, now); err != nil {
			return Checkin{}, fmt.Errorf("upsert checkin %q: %w", dateKey, err)
		}
	}

	item, found, err := s.Get(ctx, dateKey)
	if err != nil {
		return Checkin{}, err
	}
	if !found {
		return Checkin{}, fmt.Errorf("checkin %q was not found after upsert", dateKey)
	}
	return item, nil
}

func (s *Store) Get(ctx context.Context, dateKey string) (Checkin, bool, error) {
	query := `
		SELECT date_key, note, checked_at, updated_at
		FROM checkins
		WHERE date_key = ?
	`
	args := []any{dateKey}
	if s.dialect == postgresDialect {
		query = `
			SELECT
				to_char(date_key, 'YYYY-MM-DD') AS date_key,
				note,
				to_char(checked_at AT TIME ZONE 'Asia/Shanghai', 'YYYY-MM-DD"T"HH24:MI:SS"+08:00"') AS checked_at,
				to_char(updated_at AT TIME ZONE 'Asia/Shanghai', 'YYYY-MM-DD"T"HH24:MI:SS"+08:00"') AS updated_at
			FROM checkins
			WHERE date_key = $1::date
		`
	}

	row := s.db.QueryRowContext(ctx, query, args...)

	var item Checkin
	err := row.Scan(&item.Date, &item.Note, &item.CheckedAt, &item.UpdatedAt)
	if err == nil {
		return item, true, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return Checkin{}, false, nil
	}
	return Checkin{}, false, fmt.Errorf("get checkin %q: %w", dateKey, err)
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
	if _, err := s.db.ExecContext(ctx, `PRAGMA foreign_keys = ON`); err != nil {
		return fmt.Errorf("enable foreign keys: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS checkins (
			date_key TEXT PRIMARY KEY,
			note TEXT NOT NULL DEFAULT '',
			checked_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)
	`); err != nil {
		return fmt.Errorf("create checkins table: %w", err)
	}
	return nil
}

func (s *Store) initPostgres(ctx context.Context) error {
	if err := s.db.PingContext(ctx); err != nil {
		return fmt.Errorf("connect postgres: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS checkins (
			date_key DATE PRIMARY KEY,
			note TEXT NOT NULL DEFAULT '',
			checked_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)
	`); err != nil {
		return fmt.Errorf("create checkins table: %w", err)
	}
	return nil
}

func (s *Store) normalizeDateKey(value string) (string, error) {
	value = strings.TrimSpace(value)
	if !dateKeyPattern.MatchString(value) {
		return "", errors.New("date must use YYYY-MM-DD")
	}

	date, err := time.ParseInLocation("2006-01-02", value, s.location)
	if err != nil || date.Format("2006-01-02") != value {
		return "", errors.New("date is invalid")
	}

	today := time.Now().In(s.location)
	todayDate := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, s.location)
	if date.After(todayDate) {
		return "", errors.New("future dates cannot be checked in")
	}
	return value, nil
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

func normalizeNote(value string) (string, error) {
	value = strings.TrimSpace(value)
	if len([]rune(value)) > MaxNoteLength {
		return "", fmt.Errorf("note must be %d characters or fewer", MaxNoteLength)
	}
	return value, nil
}
