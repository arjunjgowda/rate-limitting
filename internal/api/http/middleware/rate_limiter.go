package middleware

import (
	"net/http"
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
			if err != nil || (result != nil && !result.Allowed) {
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte("rate limit exceeded. please try again later"))
				return
			}

			// 4. Continue to the next handler
			next.ServeHTTP(w, r)
		})
	}
}
