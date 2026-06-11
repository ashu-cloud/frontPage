package http

import (
	"log"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"
)

var requestSeq uint64

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// withLogging records the method, path, status and duration of a request.
func withLogging(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := strconv.FormatUint(atomic.AddUint64(&requestSeq, 1), 10)
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next(rec, r)

		log.Printf("req=%s %s %s status=%d dur=%s",
			id, r.Method, r.URL.Path, rec.status, time.Since(start).Round(time.Microsecond))
	}
}
