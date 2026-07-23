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

// WatchlistItem is one symbol a user follows, joined to its current price.
type WatchlistItem struct {
	Symbol  string  `json:"symbol"`
	Price   float64 `json:"price"`
	AddedAt string  `json:"added_at"`
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

// GetQuoteBySymbol returns one quote by ticker symbol.
func (s *Store) GetQuoteBySymbol(symbol string) (*Quote, error) {
	var q Quote
	err := s.db.QueryRow(`
		SELECT id, symbol, name, price, prev_close, updated_at
		FROM quotes WHERE UPPER(symbol) = UPPER(?)`, symbol).
		Scan(&q.ID, &q.Symbol, &q.Name, &q.Price, &q.PrevClose, &q.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &q, nil
}

// ListQuotes returns up to limit quotes, most recently updated first.
func (s *Store) ListQuotes(limit int) ([]Quote, error) {
	rows, err := s.db.Query(`
		SELECT id, symbol, name, price, prev_close, updated_at
		FROM quotes
		ORDER BY updated_at DESC, id ASC
		LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Quote{}
	for rows.Next() {
		var q Quote
		if err := rows.Scan(&q.ID, &q.Symbol, &q.Name, &q.Price, &q.PrevClose, &q.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, q)
	}
	return out, rows.Err()
}

// GetWatchlist returns everything a user follows, with current prices.
func (s *Store) GetWatchlist(userID int64) ([]WatchlistItem, error) {
	rows, err := s.db.Query(`
		SELECT w.symbol, q.price, w.created_at
		FROM watchlist w
		JOIN quotes q ON UPPER(q.symbol) = UPPER(w.symbol)
		WHERE w.user_id = ?
		ORDER BY w.created_at ASC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []WatchlistItem{}
	for rows.Next() {
		var it WatchlistItem
		if err := rows.Scan(&it.Symbol, &it.Price, &it.AddedAt); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// AddToWatchlist records that a user is following a symbol.
func (s *Store) AddToWatchlist(userID int64, symbol string) error {
	_, err := s.db.Exec(`
		INSERT INTO watchlist (user_id, symbol, created_at)
		VALUES (?, ?, ?)`,
		userID, symbol, time.Now().UTC().Format(time.RFC3339))
	return err
}
