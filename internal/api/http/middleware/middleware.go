package middleware

import (
	"net/http"
	"time"

	"github.com/arjunjgowda/rate-limitting/internal/api/common"
	"gofr.dev/pkg/gofr/logging"
)

// RequestLogger returns a middleware that uses GoFr's logging.Logger.
func RequestLogger(logger logging.Logger) func(http.Handler) http.Handler {
	return func(inner http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			inner.ServeHTTP(w, r)

			// Pass the official GoFr logging.Logger to the shared logic
			common.LogRequest(logger, r.Method, r.URL.Path, time.Since(start))
		})
	}
}
