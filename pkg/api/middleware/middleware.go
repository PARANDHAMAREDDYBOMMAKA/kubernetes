package middleware

import (
	"fmt"
	"net"
	"net/http"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

type Middleware func(http.Handler) http.Handler

func Chain(h http.Handler, middlewares ...Middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &responseWriter{ResponseWriter: w, status: 200}

		next.ServeHTTP(wrapped, r)

		fmt.Printf("%s %s %d %s\n", r.Method, r.URL.Path, wrapped.status, time.Since(start))
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (w *responseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				debug.PrintStack()
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func ContentType(contentType string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", contentType)
			next.ServeHTTP(w, r)
		})
	}
}

func RateLimit(rps int) Middleware {
	tokens := make(chan struct{}, rps)
	go func() {
		ticker := time.NewTicker(time.Second / time.Duration(rps))
		defer ticker.Stop()
		for range ticker.C {
			select {
			case tokens <- struct{}{}:
			default:
			}
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			select {
			case <-tokens:
				next.ServeHTTP(w, r)
			default:
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			}
		})
	}
}

type ipBucket struct {
	tokens     float64
	lastRefill time.Time
}

func PerIPRateLimit(capacity int, refillPerMinute int) Middleware {
	var mu sync.Mutex
	buckets := map[string]*ipBucket{}
	refillPerSec := float64(refillPerMinute) / 60.0

	go func() {
		t := time.NewTicker(5 * time.Minute)
		defer t.Stop()
		for range t.C {
			mu.Lock()
			cutoff := time.Now().Add(-30 * time.Minute)
			for ip, b := range buckets {
				if b.lastRefill.Before(cutoff) {
					delete(buckets, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := clientIP(r)
			mu.Lock()
			b, ok := buckets[ip]
			now := time.Now()
			if !ok {
				b = &ipBucket{tokens: float64(capacity), lastRefill: now}
				buckets[ip] = b
			} else {
				elapsed := now.Sub(b.lastRefill).Seconds()
				b.tokens += elapsed * refillPerSec
				if b.tokens > float64(capacity) {
					b.tokens = float64(capacity)
				}
				b.lastRefill = now
			}
			allow := b.tokens >= 1
			if allow {
				b.tokens -= 1
			}
			mu.Unlock()
			if !allow {
				w.Header().Set("Retry-After", "60")
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if comma := strings.IndexByte(xff, ','); comma >= 0 {
			xff = xff[:comma]
		}
		return strings.TrimSpace(xff)
	}
	if xr := r.Header.Get("X-Real-IP"); xr != "" {
		return strings.TrimSpace(xr)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
