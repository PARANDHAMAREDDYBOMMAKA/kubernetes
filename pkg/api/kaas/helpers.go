// Package kaas exposes the KaaS HTTP API under /api/v1.
package kaas

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/parandhamareddybommaka/kube/pkg/auth"
	"github.com/parandhamareddybommaka/kube/pkg/db"
	"github.com/parandhamareddybommaka/kube/pkg/lbmgr"
	"github.com/parandhamareddybommaka/kube/pkg/models"
	"github.com/parandhamareddybommaka/kube/pkg/provisioner"
)

// Server bundles the dependencies the KaaS handlers need.
type Server struct {
	Prov   provisioner.Provisioner
	LB     *lbmgr.Manager
	Logger *slog.Logger
}

// NewServer constructs a Server with the given deps. Logger defaults to slog.Default.
func NewServer(p provisioner.Provisioner, lb *lbmgr.Manager, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	return &Server{Prov: p, LB: lb, Logger: logger}
}

// writeJSON writes a JSON response with the given status.
func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if body == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(body)
}

// writeErr writes a JSON error envelope.
func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// readJSON decodes a request body into v. Returns true on success.
func readJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return false
	}
	return true
}

// requireDB returns false and writes 503 if Mongo isn't connected.
func requireDB(w http.ResponseWriter) bool {
	if !db.IsConnected() {
		writeErr(w, http.StatusServiceUnavailable, "database unavailable")
		return false
	}
	return true
}

// currentUser returns the authenticated user's full record or 401.
func currentUser(ctx context.Context) (*models.User, error) {
	userID := auth.UserIDFromCtx(ctx)
	if userID == "" {
		return nil, errors.New("not authenticated")
	}
	var user models.User
	if err := db.Users().FindOne(ctx, bson.M{"_id": userID}).Decode(&user); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}

// nowUTC returns the current time in UTC.
func nowUTC() time.Time { return time.Now().UTC() }
