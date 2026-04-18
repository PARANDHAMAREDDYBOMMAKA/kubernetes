package kaas

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/parandhamareddybommaka/kube/pkg/auth"
	"github.com/parandhamareddybommaka/kube/pkg/db"
	"github.com/parandhamareddybommaka/kube/pkg/models"
)

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	Token string       `json:"token"`
	User  *models.User `json:"user"`
}

func genID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (s *Server) Register(w http.ResponseWriter, r *http.Request) {
	if !requireDB(w) {
		return
	}
	var req registerRequest
	if !readJSON(w, r, &req) {
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if req.Email == "" || req.Password == "" {
		writeErr(w, http.StatusBadRequest, "email and password are required")
		return
	}

	ctx := r.Context()
	var existing models.User
	err := db.Users().FindOne(ctx, bson.M{"email": req.Email}).Decode(&existing)
	if err == nil {
		writeErr(w, http.StatusConflict, "email already registered")
		return
	}
	if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		writeErr(w, http.StatusInternalServerError, "database error")
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	user := &models.User{
		ID:           genID(),
		Email:        req.Email,
		PasswordHash: hash,
		Name:         req.Name,
		CreatedAt:    nowUTC(),
	}
	if _, err := db.Users().InsertOne(ctx, user); err != nil {
		s.Logger.Error("insert user", "err", err)
		writeErr(w, http.StatusInternalServerError, "failed to create user")
		return
	}

	token, err := auth.GenerateToken(user.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to issue token")
		return
	}
	user.PasswordHash = ""
	writeJSON(w, http.StatusCreated, authResponse{Token: token, User: user})
}

// Login authenticates an existing user.
func (s *Server) Login(w http.ResponseWriter, r *http.Request) {
	if !requireDB(w) {
		return
	}
	var req loginRequest
	if !readJSON(w, r, &req) {
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if req.Email == "" || req.Password == "" {
		writeErr(w, http.StatusBadRequest, "email and password are required")
		return
	}

	ctx := r.Context()
	user, err := lookupByEmail(ctx, req.Email)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "invalid email or password")
		return
	}
	if err := auth.CheckPassword(user.PasswordHash, req.Password); err != nil {
		writeErr(w, http.StatusUnauthorized, "invalid email or password")
		return
	}
	token, err := auth.GenerateToken(user.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to issue token")
		return
	}
	user.PasswordHash = ""
	writeJSON(w, http.StatusOK, authResponse{Token: token, User: user})
}

// Me returns the authenticated user.
func (s *Server) Me(w http.ResponseWriter, r *http.Request) {
	if !requireDB(w) {
		return
	}
	user, err := currentUser(r.Context())
	if err != nil {
		writeErr(w, http.StatusUnauthorized, err.Error())
		return
	}
	user.PasswordHash = ""
	writeJSON(w, http.StatusOK, map[string]any{"user": user})
}

func lookupByEmail(ctx context.Context, email string) (*models.User, error) {
	var u models.User
	if err := db.Users().FindOne(ctx, bson.M{"email": email}).Decode(&u); err != nil {
		return nil, err
	}
	return &u, nil
}
