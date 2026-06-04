package store

import (
	"database/sql"
	"embed"
	"fmt"
	"sort"
	"time"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// Quote is a single tradable instrument and its latest price.
type Quote struct {
	ID        int64   `json:"id"`
	Symbol    string  `json:"symbol"`
	Name      string  `json:"name"`
	Price     float64 `json:"price"`
	PrevClose float64 `json:"prev_close"`
	UpdatedAt string  `json:"updated_at"`
}

type Store struct {
	db *sql.DB
}

func New(db *sql.DB) *Store {
	return &Store{db: db}
}

// Migrate applies any migration files that have not been applied yet.
func (s *Store) Migrate() error {
	_, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		name TEXT PRIMARY KEY,
		applied_at TEXT NOT NULL
	)`)
	if err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)

	for _, name := range names {
		var seen string
		err := s.db.QueryRow(`SELECT name FROM schema_migrations WHERE name = ?`, name).Scan(&seen)
		if err == nil {
			continue
		}
		if err != sql.ErrNoRows {
			return fmt.Errorf("check migration %s: %w", name, err)
		}

		body, err := migrationFS.ReadFile("migrations/" + name)
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}
		if _, err := s.db.Exec(string(body)); err != nil {
			return fmt.Errorf("apply %s: %w", name, err)
		}
		_, err = s.db.Exec(`INSERT INTO schema_migrations (name, applied_at) VALUES (?, ?)`,
			name, time.Now().UTC().Format(time.RFC3339))
		if err != nil {
			return fmt.Errorf("record %s: %w", name, err)
		}
	}
	return nil
}

// GetQuote returns one quote by its id.
func (s *Store) GetQuote(id int64) (*Quote, error) {
	var q Quote
	err := s.db.QueryRow(`
		SELECT id, symbol, name, price, prev_close, updated_at
		FROM quotes WHERE id = ?`, id).
		Scan(&q.ID, &q.Symbol, &q.Name, &q.Price, &q.PrevClose, &q.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &q, nil
}
