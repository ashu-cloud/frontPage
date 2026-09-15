package store

import (
	"database/sql"
	"embed"
	"fmt"
	"sort"
	"strings"
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
	Symbol   string  `json:"symbol"`
	Price    float64 `json:"price"`
	AddedAt  string  `json:"added_at"`
	Delisted bool    `json:"delisted"`
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
		FROM quotes WHERE id = ? AND delisted = 0`, id).
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
		FROM quotes WHERE symbol = ? AND delisted = 0`, symbol).
		Scan(&q.ID, &q.Symbol, &q.Name, &q.Price, &q.PrevClose, &q.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &q, nil
}

// FindQuote searches for a quote by ticker symbol (uppercased) or by company name
// (spaces stripped and lowercased). Returns the untampered quote data from the database.
func (s *Store) FindQuote(query string) (*Quote, error) {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return nil, nil
	}

	upper := strings.ToUpper(trimmed)
	lower := strings.ToLower(trimmed)

	// 1. Try matching by ticker symbol (uppercased)
	var q Quote
	err := s.db.QueryRow(`
		SELECT id, symbol, name, price, prev_close, updated_at
		FROM quotes
		WHERE symbol = ? AND delisted = 0`, upper).
		Scan(&q.ID, &q.Symbol, &q.Name, &q.Price, &q.PrevClose, &q.UpdatedAt)
	if err == nil {
		return &q, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	// 2. Try matching by company name (trimmed and lowercased, or with all spaces stripped)
	err = s.db.QueryRow(`
		SELECT id, symbol, name, price, prev_close, updated_at
		FROM quotes
		WHERE delisted = 0 AND (
			LOWER(TRIM(name)) = ? OR
			REPLACE(LOWER(name), ' ', '') = REPLACE(?, ' ', '')
		)
		LIMIT 1`, lower, lower).
		Scan(&q.ID, &q.Symbol, &q.Name, &q.Price, &q.PrevClose, &q.UpdatedAt)
	if err == nil {
		return &q, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	// 3. Fallback: partial match on company name
	err = s.db.QueryRow(`
		SELECT id, symbol, name, price, prev_close, updated_at
		FROM quotes
		WHERE delisted = 0 AND LOWER(name) LIKE ?
		ORDER BY LENGTH(name) ASC
		LIMIT 1`, "%"+lower+"%").
		Scan(&q.ID, &q.Symbol, &q.Name, &q.Price, &q.PrevClose, &q.UpdatedAt)
	if err == nil {
		return &q, nil
	}
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return nil, err
}

// ListQuotes returns up to limit quotes, most recently updated first.
func (s *Store) ListQuotes(limit int) ([]Quote, error) {
	rows, err := s.db.Query(`
		SELECT id, symbol, name, price, prev_close, updated_at
		FROM quotes
		WHERE delisted = 0
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

// GetWatchlist returns everything a user follows, with current prices and delisted status.
func (s *Store) GetWatchlist(userID int64) ([]WatchlistItem, error) {
	rows, err := s.db.Query(`
		SELECT w.symbol, COALESCE(q.price, 0), w.created_at, COALESCE(q.delisted, 0)
		FROM watchlist w
		LEFT JOIN quotes q ON q.symbol = w.symbol
		WHERE w.user_id = ?
		ORDER BY w.created_at ASC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []WatchlistItem{}
	for rows.Next() {
		var it WatchlistItem
		var delisted int
		if err := rows.Scan(&it.Symbol, &it.Price, &it.AddedAt, &delisted); err != nil {
			return nil, err
		}
		it.Delisted = delisted == 1
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
