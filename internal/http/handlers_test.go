package http

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/frontpage/quotesvc/internal/store"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	st := store.New(db)
	if err := st.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return NewServer(st)
}

func TestGetQuote(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/quotes/1", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var q store.Quote
	if err := json.NewDecoder(rec.Body).Decode(&q); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if q.Symbol != "AAPL" {
		t.Errorf("symbol = %q, want AAPL", q.Symbol)
	}
}

func TestGetQuoteNotFound(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/quotes/99999", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestListQuotesRejectsBadLimit(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/quotes?limit=abc", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestGetWatchlist(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/watchlist/42", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body struct {
		Symbols []store.WatchlistItem `json:"symbols"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Symbols) != 4 {
		t.Errorf("got %d symbols, want 4", len(body.Symbols))
	}
	for _, sym := range body.Symbols {
		if sym.Delisted {
			t.Errorf("expected symbol %s to not be delisted", sym.Symbol)
		}
	}
}

func TestGetWatchlistWithDelisted(t *testing.T) {
	srv := newTestServer(t)

	// User 7 has MSFT, AMZN, and delisted TWTR
	req := httptest.NewRequest(http.MethodGet, "/watchlist/7", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body struct {
		Symbols []store.WatchlistItem `json:"symbols"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Symbols) != 3 {
		t.Fatalf("got %d symbols, want 3", len(body.Symbols))
	}

	foundTWTR := false
	for _, sym := range body.Symbols {
		if sym.Symbol == "TWTR" {
			foundTWTR = true
			if !sym.Delisted {
				t.Errorf("expected TWTR to be marked as delisted")
			}
		} else {
			if sym.Delisted {
				t.Errorf("expected %s to not be delisted", sym.Symbol)
			}
		}
	}
	if !foundTWTR {
		t.Errorf("expected to find TWTR in user 7's watchlist")
	}
}


func TestFindQuoteBySymbol(t *testing.T) {
	srv := newTestServer(t)

	// Test lowercase symbol (should be uppercased to fetch)
	req := httptest.NewRequest(http.MethodGet, "/quote/meta", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var q store.Quote
	if err := json.NewDecoder(rec.Body).Decode(&q); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if q.ID != 7 || q.Symbol != "META" || q.Name != "Meta Platforms, Inc." {
		t.Errorf("unexpected quote: %+v", q)
	}

	// Test uppercase symbol
	req = httptest.NewRequest(http.MethodGet, "/quote/META", nil)
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestFindQuoteByName(t *testing.T) {
	srv := newTestServer(t)

	// Exact name with URL encoding
	req := httptest.NewRequest(http.MethodGet, "/quote/Meta%20Platforms,%20Inc.", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var q store.Quote
	if err := json.NewDecoder(rec.Body).Decode(&q); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// Verify untampered data
	if q.ID != 7 || q.Symbol != "META" || q.Name != "Meta Platforms, Inc." || q.Price != 512.6 {
		t.Errorf("unexpected quote: %+v", q)
	}

	// Lowercased with leading and trailing spaces
	req = httptest.NewRequest(http.MethodGet, "/quote/%20meta%20platforms,%20inc.%20", nil)
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var q2 store.Quote
	if err := json.NewDecoder(rec.Body).Decode(&q2); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if q2.Symbol != "META" || q2.Name != "Meta Platforms, Inc." {
		t.Errorf("unexpected quote: %+v", q2)
	}
}

func TestFindQuoteNotFound(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/quote/DOESNOTEXIST", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestFindQuoteDelisted(t *testing.T) {
	srv := newTestServer(t)

	// TWTR was delisted in migration 005
	req := httptest.NewRequest(http.MethodGet, "/quote/twtr", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 for delisted stock", rec.Code)
	}
}

func TestFindQuoteMethodNotAllowed(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/quote/meta", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}
}

