package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"
)

// ipEntry tracks the request count and the window start time for a single IP.
type ipEntry struct {
	mu        sync.Mutex
	count     int
	windowEnd time.Time
}

// RateLimit returns middleware that enforces a token-bucket style limit of
// requestsPerMinute per client IP address. When the limit is exceeded the
// handler responds with 429 Too Many Requests and a Retry-After header.
func RateLimit(requestsPerMinute int) func(http.Handler) http.Handler {
	var store sync.Map

	// Background goroutine that removes stale entries every minute.
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			now := time.Now()
			store.Range(func(key, value interface{}) bool {
				entry := value.(*ipEntry)
				entry.mu.Lock()
				expired := now.After(entry.windowEnd)
				entry.mu.Unlock()
				if expired {
					store.Delete(key)
				}
				return true
			})
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := clientIP(r)

			val, _ := store.LoadOrStore(ip, &ipEntry{
				count:     0,
				windowEnd: time.Now().Add(time.Minute),
			})
			entry := val.(*ipEntry)

			entry.mu.Lock()
			now := time.Now()
			if now.After(entry.windowEnd) {
				// Start a new window.
				entry.count = 0
				entry.windowEnd = now.Add(time.Minute)
			}
			entry.count++
			count := entry.count
			retryAfter := int(time.Until(entry.windowEnd).Seconds()) + 1
			entry.mu.Unlock()

			if count > requestsPerMinute {
				w.Header().Set("Retry-After", http.TimeFormat)
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", formatSeconds(retryAfter))
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`{"error":{"code":"rate_limit_exceeded","message":"too many requests"}}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// clientIP extracts the real client IP, honouring X-Forwarded-For when present.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first (leftmost) address.
		if idx := len(xff); idx > 0 {
			for i, ch := range xff {
				if ch == ',' {
					return xff[:i]
				}
			}
			return xff
		}
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

func formatSeconds(s int) string {
	if s < 0 {
		s = 0
	}
	return http.TimeFormat[:0] + itoa(s)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := [20]byte{}
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[pos:])
}
