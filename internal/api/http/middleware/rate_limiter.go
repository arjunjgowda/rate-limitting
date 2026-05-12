package middleware

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/arjunjgowda/rate-limitting/pkg/ratelimit"
)

// RateLimiter returns a standard net/http middleware.
// This ensures compatibility with GoFr's UseMiddleware and other middlewares in the project.
func RateLimiter(rl ratelimit.RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. Prepare the request
			// We use the RemoteAddr or X-Forwarded-For for the IP
			req := ratelimit.RateLimitRequest{
				Key:         r.RemoteAddr,
				Cost:        1,
				CurrentTime: time.Now(),
			}

			// 2. Ask the RateLimiter if we can proceed
			// Note: rl.Allow expects context.Context, which r.Context() provides
			result, err := rl.Allow(r.Context(), req)

			// 3. Handle Errors or Rejections
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"error":   "Internal Server Error",
					"message": "An error occurred while processing your request",
				})
				return
			}

			if result != nil && !result.Allowed {
				// Standard headers for rate limiting
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-RateLimit-Retry-After", strconv.FormatInt(int64(result.RetryAfter.Seconds()), 10))
				w.Header().Set("X-RateLimit-Limit", strconv.FormatUint(uint64(result.Limit), 10))
				w.Header().Set("X-RateLimit-Remaining", "0")
				w.Header().Set("X-RateLimit-Retry-Window-Unit", "seconds")

				w.WriteHeader(http.StatusTooManyRequests)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"error":           "Too Many Requests",
					"message":         "Rate limit exceeded. Please try again later.",
					"retryAfter":      result.RetryAfter.Seconds(),
					"retryWindowUnit": "seconds",
					"limit":           result.Limit,
					"currentTime":     time.Now().Format(time.RFC3339),
				})
				return
			}

			// 4. Continue to the next handler
			next.ServeHTTP(w, r)
		})
	}
}
