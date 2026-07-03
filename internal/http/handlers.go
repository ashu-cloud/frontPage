package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/frontpage/quotesvc/internal/store"
)

type Server struct {
	store *store.Store
	mux   *http.ServeMux
}

func NewServer(s *store.Store) *Server {
	srv := &Server{store: s, mux: http.NewServeMux()}
	srv.routes()
	return srv
}

func (s *Server) routes() {
	s.mux.HandleFunc("/quotes/", withLogging(s.handleGetQuote))
	s.mux.HandleFunc("/watchlist/", s.handleGetWatchlist)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// GET /quotes/{id}
func (s *Server) handleGetQuote(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	raw := strings.TrimPrefix(r.URL.Path, "/quotes/")
	if raw == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id < 1 {
		writeError(w, http.StatusBadRequest, "id must be a positive integer")
		return
	}

	q, err := s.store.GetQuote(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load quote")
		return
	}
	if q == nil {
		writeError(w, http.StatusNotFound, "quote not found")
		return
	}
	writeJSON(w, http.StatusOK, q)
}

// GET /watchlist/{userID}
func (s *Server) handleGetWatchlist(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	raw := strings.TrimPrefix(r.URL.Path, "/watchlist/")
	userID, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || userID < 1 {
		writeError(w, http.StatusBadRequest, "userID must be a positive integer")
		return
	}

	items, err := s.store.GetWatchlist(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load watchlist")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"user_id": userID,
		"symbols": items,
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
