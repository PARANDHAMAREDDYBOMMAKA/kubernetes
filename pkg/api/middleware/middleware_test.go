package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestClientIP(t *testing.T) {
	cases := []struct {
		name       string
		xff        string
		xrealip    string
		remoteAddr string
		want       string
	}{
		{"x-forwarded-for single", "203.0.113.5", "", "10.0.0.1:1234", "203.0.113.5"},
		{"x-forwarded-for list", "203.0.113.5, 70.0.0.1", "", "10.0.0.1:1234", "203.0.113.5"},
		{"x-real-ip", "", "198.51.100.7", "10.0.0.1:1234", "198.51.100.7"},
		{"remote addr fallback", "", "", "192.0.2.9:5678", "192.0.2.9"},
		{"remote addr without port", "", "", "192.0.2.9", "192.0.2.9"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.RemoteAddr = tc.remoteAddr
			if tc.xff != "" {
				r.Header.Set("X-Forwarded-For", tc.xff)
			}
			if tc.xrealip != "" {
				r.Header.Set("X-Real-IP", tc.xrealip)
			}
			if got := clientIP(r); got != tc.want {
				t.Errorf("clientIP = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestPerIPRateLimit(t *testing.T) {
	// Capacity 2, negligible refill so the bucket doesn't replenish mid-test.
	handler := PerIPRateLimit(2, 1)(okHandler())

	do := func(ip string) int {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set("X-Real-IP", ip)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, r)
		return rec.Code
	}

	if c := do("1.1.1.1"); c != http.StatusOK {
		t.Fatalf("req 1: status %d, want 200", c)
	}
	if c := do("1.1.1.1"); c != http.StatusOK {
		t.Fatalf("req 2: status %d, want 200", c)
	}
	if c := do("1.1.1.1"); c != http.StatusTooManyRequests {
		t.Fatalf("req 3: status %d, want 429", c)
	}
	// A different IP has its own bucket and is unaffected.
	if c := do("2.2.2.2"); c != http.StatusOK {
		t.Fatalf("other IP: status %d, want 200", c)
	}
}

func TestCORSPreflight(t *testing.T) {
	handler := CORS(okHandler())
	r := httptest.NewRequest(http.MethodOptions, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, r)

	if rec.Code != http.StatusOK {
		t.Errorf("preflight status = %d, want 200", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("missing CORS allow-origin header")
	}
}

func TestRecovery(t *testing.T) {
	panicky := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	})
	handler := Recovery(panicky)
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, r) // must not panic out of the handler
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
}

func TestChainOrder(t *testing.T) {
	var order []string
	mw := func(name string) Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, name)
				next.ServeHTTP(w, r)
			})
		}
	}
	final := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "handler")
	})

	Chain(final, mw("a"), mw("b")).ServeHTTP(
		httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	want := []string{"a", "b", "handler"}
	if len(order) != len(want) {
		t.Fatalf("order = %v, want %v", order, want)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("order = %v, want %v", order, want)
		}
	}
}

func TestLoggerCapturesStatus(t *testing.T) {
	handler := Logger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusTeapot {
		t.Errorf("status = %d, want 418", rec.Code)
	}
}
