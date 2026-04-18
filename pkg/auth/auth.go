package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type ctxKey string

const userIDKey ctxKey = "kaas-user-id"

const (
	defaultTokenTTL = 7 * 24 * time.Hour
)

var (
	secretOnce sync.Once
	secret     []byte
)

var ErrUnauthorized = errors.New("unauthorized")

func Secret() []byte {
	secretOnce.Do(func() {
		if s := os.Getenv("JWT_SECRET"); s != "" {
			secret = []byte(s)
			return
		}
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			s := fmt.Sprintf("kaas-fallback-%d", time.Now().UnixNano())
			b = []byte(s)
		}
		secret = []byte(hex.EncodeToString(b))
		slog.Warn("JWT_SECRET not set; generated ephemeral secret. Tokens will be invalidated on restart.")
	})
	return secret
}

func HashPassword(password string) (string, error) {
	if password == "" {
		return "", errors.New("password must not be empty")
	}
	h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(h), nil
}

func CheckPassword(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

type Claims struct {
	UserID string `json:"sub"`
	jwt.RegisteredClaims
}

func GenerateToken(userID string) (string, error) {
	if userID == "" {
		return "", errors.New("userID required")
	}
	now := time.Now()
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(defaultTokenTTL)),
			Issuer:    "kaas",
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString(Secret())
}

func ParseToken(s string) (string, error) {
	if s == "" {
		return "", ErrUnauthorized
	}
	claims := &Claims{}
	tok, err := jwt.ParseWithClaims(s, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return Secret(), nil
	})
	if err != nil || !tok.Valid {
		return "", ErrUnauthorized
	}
	if claims.UserID == "" {
		return "", ErrUnauthorized
	}
	return claims.UserID, nil
}

func UserIDFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(userIDKey).(string)
	return v
}

func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

func Required(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := r.Header.Get("Authorization")
		if h == "" {
			writeUnauthorized(w, "missing authorization header")
			return
		}
		parts := strings.SplitN(h, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			writeUnauthorized(w, "invalid authorization header")
			return
		}
		userID, err := ParseToken(strings.TrimSpace(parts[1]))
		if err != nil {
			writeUnauthorized(w, "invalid or expired token")
			return
		}
		ctx := WithUserID(r.Context(), userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func writeUnauthorized(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	fmt.Fprintf(w, `{"error":%q}`, msg)
}
