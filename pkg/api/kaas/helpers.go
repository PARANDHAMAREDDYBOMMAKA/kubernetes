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

type Server struct {
	Prov   provisioner.Provisioner
	LB     *lbmgr.Manager
	Logger *slog.Logger
}

func NewServer(p provisioner.Provisioner, lb *lbmgr.Manager, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	return &Server{Prov: p, LB: lb, Logger: logger}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if body == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(body)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func readJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return false
	}
	return true
}

func requireDB(w http.ResponseWriter) bool {
	if !db.IsConnected() {
		writeErr(w, http.StatusServiceUnavailable, "database unavailable")
		return false
	}
	return true
}

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

func nowUTC() time.Time { return time.Now().UTC() }
