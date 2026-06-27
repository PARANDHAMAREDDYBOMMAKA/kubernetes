package auth

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// Secret() memoizes via sync.Once on first use, so pin a deterministic secret
// before any token is generated or parsed.
func TestMain(m *testing.M) {
	os.Setenv("JWT_SECRET", "test-secret-do-not-use-in-prod")
	os.Exit(m.Run())
}

func TestHashAndCheckPassword(t *testing.T) {
	hash, err := HashPassword("hunter2")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if hash == "hunter2" {
		t.Fatal("hash must not equal plaintext")
	}
	if err := CheckPassword(hash, "hunter2"); err != nil {
		t.Fatalf("CheckPassword should accept correct password: %v", err)
	}
	if err := CheckPassword(hash, "wrong"); err == nil {
		t.Fatal("CheckPassword should reject wrong password")
	}
}

func TestHashPasswordEmpty(t *testing.T) {
	if _, err := HashPassword(""); err == nil {
		t.Fatal("expected error for empty password")
	}
}

func TestGenerateAndParseToken(t *testing.T) {
	tok, err := GenerateToken("user-123")
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	uid, err := ParseToken(tok)
	if err != nil {
		t.Fatalf("ParseToken: %v", err)
	}
	if uid != "user-123" {
		t.Fatalf("got user id %q, want user-123", uid)
	}
}

func TestGenerateTokenEmptyUser(t *testing.T) {
	if _, err := GenerateToken(""); err == nil {
		t.Fatal("expected error for empty userID")
	}
}

func TestParseTokenInvalid(t *testing.T) {
	cases := map[string]string{
		"empty":    "",
		"garbage":  "not-a-jwt",
		"tampered": "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ4In0.bad-signature",
	}
	for name, tok := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseToken(tok); err == nil {
				t.Fatalf("expected error for %s token", name)
			}
		})
	}
}

func TestRequiredMiddleware(t *testing.T) {
	tok, err := GenerateToken("u-1")
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	var seenUser string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenUser = UserIDFromCtx(r.Context())
		w.WriteHeader(http.StatusOK)
	})
	handler := Required(next)

	tests := []struct {
		name       string
		header     string
		wantStatus int
		wantUser   string
	}{
		{"valid bearer", "Bearer " + tok, http.StatusOK, "u-1"},
		{"case-insensitive scheme", "bearer " + tok, http.StatusOK, "u-1"},
		{"missing header", "", http.StatusUnauthorized, ""},
		{"no bearer prefix", tok, http.StatusUnauthorized, ""},
		{"bad token", "Bearer not-a-token", http.StatusUnauthorized, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			seenUser = ""
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if seenUser != tc.wantUser {
				t.Fatalf("ctx user = %q, want %q", seenUser, tc.wantUser)
			}
		})
	}
}

func TestWithAndFromContext(t *testing.T) {
	ctx := WithUserID(t.Context(), "abc")
	if got := UserIDFromCtx(ctx); got != "abc" {
		t.Fatalf("UserIDFromCtx = %q, want abc", got)
	}
	if got := UserIDFromCtx(t.Context()); got != "" {
		t.Fatalf("empty context should yield empty string, got %q", got)
	}
}
