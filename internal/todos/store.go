package todos

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
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

const (
	MaxTasks       = 18
	MaxTitleLength = 48
)

var (
	dateKeyPattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	taskIDPattern  = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{7,63}$`)
)

type Task struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	SortOrder int    `json:"sortOrder"`
}

type DayTask struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Done        bool   `json:"done"`
	CompletedAt string `json:"completedAt,omitempty"`
	SortOrder   int    `json:"sortOrder"`
}

type DayStatus struct {
	Date      string    `json:"date"`
	Tasks     []DayTask `json:"tasks"`
	Total     int       `json:"total"`
	Completed int       `json:"completed"`
	AllDone   bool      `json:"allDone"`
}

type TaskDraft struct {
	ID    string `json:"id,omitempty"`
	Title string `json:"title"`
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

	store := &Store{db: db, location: shanghaiLocation(), dialect: sqliteDialect}
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

	store := &Store{db: db, location: shanghaiLocation(), dialect: postgresDialect}
	if err := store.init(context.Background()); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) TodayKey() string {
	return time.Now().In(s.location).Format("2006-01-02")
}

func (s *Store) DayStatus(ctx context.Context, dateKey string) (DayStatus, error) {
	dateKey, err := s.normalizeDateKey(dateKey)
	if err != nil {
		return DayStatus{}, err
	}

	query := `
		SELECT t.id, t.title, t.sort_order, c.completed_at
		FROM todo_tasks t
		LEFT JOIN todo_completions c ON c.task_id = t.id AND c.date_key = ?
		WHERE t.active = 1
		ORDER BY t.sort_order ASC, t.created_at ASC
	`
	args := []any{dateKey}
	if s.dialect == postgresDialect {
		query = `
			SELECT
				t.id,
				t.title,
				t.sort_order,
				to_char(c.completed_at AT TIME ZONE 'Asia/Shanghai', 'YYYY-MM-DD"T"HH24:MI:SS"+08:00"') AS completed_at
			FROM todo_tasks t
			LEFT JOIN todo_completions c ON c.task_id = t.id AND c.date_key = $1::date
			WHERE t.active = TRUE
			ORDER BY t.sort_order ASC, t.created_at ASC
		`
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return DayStatus{}, fmt.Errorf("query todo status: %w", err)
	}
	defer rows.Close()

	status := DayStatus{Date: dateKey, Tasks: make([]DayTask, 0)}
	for rows.Next() {
		var task DayTask
		var completedAt sql.NullString
		if err := rows.Scan(&task.ID, &task.Title, &task.SortOrder, &completedAt); err != nil {
			return DayStatus{}, fmt.Errorf("scan todo task: %w", err)
		}
		if completedAt.Valid && strings.TrimSpace(completedAt.String) != "" {
			task.Done = true
			task.CompletedAt = completedAt.String
			status.Completed++
		}
		status.Tasks = append(status.Tasks, task)
	}
	if err := rows.Err(); err != nil {
		return DayStatus{}, fmt.Errorf("iterate todo status: %w", err)
	}

	status.Total = len(status.Tasks)
	status.AllDone = status.Completed == status.Total
	return status, nil
}

func (s *Store) ReplaceTasks(ctx context.Context, drafts []TaskDraft, dateKey string) (DayStatus, error) {
	dateKey, err := s.normalizeDateKey(dateKey)
	if err != nil {
		return DayStatus{}, err
	}

	tasks, err := normalizeDrafts(drafts)
	if err != nil {
		return DayStatus{}, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return DayStatus{}, fmt.Errorf("begin todo task transaction: %w", err)
	}
	defer tx.Rollback()

	now := time.Now().In(s.location).Format(time.RFC3339)
	if s.dialect == postgresDialect {
		if _, err := tx.ExecContext(ctx, `UPDATE todo_tasks SET active = FALSE, updated_at = now() WHERE active = TRUE`); err != nil {
			return DayStatus{}, fmt.Errorf("disable old todo tasks: %w", err)
		}
		for _, task := range tasks {
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO todo_tasks (id, title, sort_order, active, created_at, updated_at)
				VALUES ($1, $2, $3, TRUE, now(), now())
				ON CONFLICT(id) DO UPDATE SET
					title = excluded.title,
					sort_order = excluded.sort_order,
					active = TRUE,
					updated_at = excluded.updated_at
			`, task.ID, task.Title, task.SortOrder); err != nil {
				return DayStatus{}, fmt.Errorf("save todo task %q: %w", task.Title, err)
			}
		}
	} else {
		if _, err := tx.ExecContext(ctx, `UPDATE todo_tasks SET active = 0, updated_at = ? WHERE active = 1`, now); err != nil {
			return DayStatus{}, fmt.Errorf("disable old todo tasks: %w", err)
		}
		for _, task := range tasks {
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO todo_tasks (id, title, sort_order, active, created_at, updated_at)
				VALUES (?, ?, ?, 1, ?, ?)
				ON CONFLICT(id) DO UPDATE SET
					title = excluded.title,
					sort_order = excluded.sort_order,
					active = 1,
					updated_at = excluded.updated_at
			`, task.ID, task.Title, task.SortOrder, now, now); err != nil {
				return DayStatus{}, fmt.Errorf("save todo task %q: %w", task.Title, err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return DayStatus{}, fmt.Errorf("commit todo tasks: %w", err)
	}
	return s.DayStatus(ctx, dateKey)
}

func (s *Store) SetCompletion(ctx context.Context, dateKey string, taskID string, done bool) (DayStatus, error) {
	dateKey, err := s.normalizeDateKey(dateKey)
	if err != nil {
		return DayStatus{}, err
	}
	taskID = strings.TrimSpace(taskID)
	if !taskIDPattern.MatchString(taskID) {
		return DayStatus{}, errors.New("task id is invalid")
	}

	if err := s.ensureActiveTask(ctx, taskID); err != nil {
		return DayStatus{}, err
	}

	now := time.Now().In(s.location).Format(time.RFC3339)
	if done {
		if s.dialect == postgresDialect {
			if _, err := s.db.ExecContext(ctx, `
				INSERT INTO todo_completions (date_key, task_id, completed_at)
				VALUES ($1::date, $2, now())
				ON CONFLICT(date_key, task_id) DO UPDATE SET completed_at = excluded.completed_at
			`, dateKey, taskID); err != nil {
				return DayStatus{}, fmt.Errorf("complete todo task: %w", err)
			}
		} else {
			if _, err := s.db.ExecContext(ctx, `
				INSERT INTO todo_completions (date_key, task_id, completed_at)
				VALUES (?, ?, ?)
				ON CONFLICT(date_key, task_id) DO UPDATE SET completed_at = excluded.completed_at
			`, dateKey, taskID, now); err != nil {
				return DayStatus{}, fmt.Errorf("complete todo task: %w", err)
			}
		}
	} else {
		query := `DELETE FROM todo_completions WHERE date_key = ? AND task_id = ?`
		args := []any{dateKey, taskID}
		if s.dialect == postgresDialect {
			query = `DELETE FROM todo_completions WHERE date_key = $1::date AND task_id = $2`
		}
		if _, err := s.db.ExecContext(ctx, query, args...); err != nil {
			return DayStatus{}, fmt.Errorf("uncomplete todo task: %w", err)
		}
	}

	return s.DayStatus(ctx, dateKey)
}

func (s *Store) CanUnlockCheckin(ctx context.Context, dateKey string) (bool, DayStatus, error) {
	status, err := s.DayStatus(ctx, dateKey)
	if err != nil {
		return false, DayStatus{}, err
	}
	return status.AllDone, status, nil
}

func (s *Store) ensureActiveTask(ctx context.Context, taskID string) error {
	var exists int
	query := `SELECT 1 FROM todo_tasks WHERE id = ? AND active = 1`
	args := []any{taskID}
	if s.dialect == postgresDialect {
		query = `SELECT 1 FROM todo_tasks WHERE id = $1 AND active = TRUE`
	}
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&exists)
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return errors.New("task is not active")
	}
	return fmt.Errorf("check todo task: %w", err)
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
		CREATE TABLE IF NOT EXISTS todo_tasks (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			sort_order INTEGER NOT NULL DEFAULT 0,
			active INTEGER NOT NULL DEFAULT 1,
			created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		return fmt.Errorf("create todo tasks table: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS todo_completions (
			date_key TEXT NOT NULL,
			task_id TEXT NOT NULL,
			completed_at TEXT NOT NULL,
			PRIMARY KEY (date_key, task_id),
			FOREIGN KEY (task_id) REFERENCES todo_tasks(id) ON DELETE CASCADE
		)
	`); err != nil {
		return fmt.Errorf("create todo completions table: %w", err)
	}
	return nil
}

func (s *Store) initPostgres(ctx context.Context) error {
	if err := s.db.PingContext(ctx); err != nil {
		return fmt.Errorf("connect postgres: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS todo_tasks (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			sort_order INTEGER NOT NULL DEFAULT 0,
			active BOOLEAN NOT NULL DEFAULT TRUE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)
	`); err != nil {
		return fmt.Errorf("create todo tasks table: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS todo_completions (
			date_key DATE NOT NULL,
			task_id TEXT NOT NULL REFERENCES todo_tasks(id) ON DELETE CASCADE,
			completed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			PRIMARY KEY (date_key, task_id)
		)
	`); err != nil {
		return fmt.Errorf("create todo completions table: %w", err)
	}
	return nil
}

func (s *Store) normalizeDateKey(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		value = s.TodayKey()
	}
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
		return "", errors.New("future dates cannot be edited")
	}
	return value, nil
}

func normalizeDrafts(drafts []TaskDraft) ([]Task, error) {
	if len(drafts) > MaxTasks {
		return nil, fmt.Errorf("todo tasks must be %d or fewer", MaxTasks)
	}

	seenIDs := make(map[string]struct{}, len(drafts))
	seenTitles := make(map[string]struct{}, len(drafts))
	tasks := make([]Task, 0, len(drafts))
	for _, draft := range drafts {
		title := strings.Join(strings.Fields(draft.Title), " ")
		if title == "" {
			continue
		}
		if len([]rune(title)) > MaxTitleLength {
			return nil, fmt.Errorf("todo task must be %d characters or fewer", MaxTitleLength)
		}

		titleKey := strings.ToLower(title)
		if _, ok := seenTitles[titleKey]; ok {
			return nil, fmt.Errorf("duplicate todo task %q", title)
		}
		seenTitles[titleKey] = struct{}{}

		id := strings.TrimSpace(strings.ToLower(draft.ID))
		if id == "" {
			id = newTaskID()
		}
		if !taskIDPattern.MatchString(id) {
			return nil, fmt.Errorf("todo task id %q is invalid", draft.ID)
		}
		if _, ok := seenIDs[id]; ok {
			return nil, fmt.Errorf("duplicate todo task id %q", id)
		}
		seenIDs[id] = struct{}{}

		tasks = append(tasks, Task{ID: id, Title: title, SortOrder: len(tasks)})
	}
	return tasks, nil
}

func newTaskID() string {
	var buffer [6]byte
	if _, err := rand.Read(buffer[:]); err != nil {
		return fmt.Sprintf("task-%d", time.Now().UnixNano())
	}
	return "task-" + hex.EncodeToString(buffer[:])
}

func shanghaiLocation() *time.Location {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.FixedZone("Asia/Shanghai", 8*60*60)
	}
	return location
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
