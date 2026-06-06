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

	store := &Store{db: db, location: location}
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
	rows, err := s.db.QueryContext(ctx, `
		SELECT date_key, note, checked_at, updated_at
		FROM checkins
		ORDER BY date_key DESC
		LIMIT 500
	`)
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
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO checkins (date_key, note, checked_at, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(date_key) DO UPDATE SET
			note = excluded.note,
			updated_at = excluded.updated_at
	`, dateKey, note, now, now); err != nil {
		return Checkin{}, fmt.Errorf("upsert checkin %q: %w", dateKey, err)
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
	row := s.db.QueryRowContext(ctx, `
		SELECT date_key, note, checked_at, updated_at
		FROM checkins
		WHERE date_key = ?
	`, dateKey)

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

func normalizeNote(value string) (string, error) {
	value = strings.TrimSpace(value)
	if len([]rune(value)) > MaxNoteLength {
		return "", fmt.Errorf("note must be %d characters or fewer", MaxNoteLength)
	}
	return value, nil
}
