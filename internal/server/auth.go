package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/drive/drive/internal/model"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type authRequest struct {
	Username    string  `json:"username"`
	Password    string  `json:"password"`
	DisplayName *string `json:"display_name,omitempty"`
}

type authResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	ExpiresIn    int          `json:"expires_in"`
	User         userResponse `json:"user"`
}

type userResponse struct {
	ID          string  `json:"id"`
	Username    string  `json:"username"`
	DisplayName *string `json:"display_name,omitempty"`
	Role        string  `json:"role"`
	CreatedAt   string  `json:"created_at,omitempty"`
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	if !s.cfg.Auth.AllowRegistration {
		writeError(w, http.StatusForbidden, "REGISTRATION_DISABLED", "Registration is disabled")
		return
	}

	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	if len(req.Username) < 3 || len(req.Username) > 32 {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Username must be 3-32 characters")
		return
	}
	if len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Password must be at least 8 characters")
		return
	}

	existing, _ := s.userStore.FindByUsername(req.Username)
	if existing != nil {
		writeError(w, http.StatusConflict, "USERNAME_EXISTS", "Username is already taken")
		return
	}

	user, err := s.userStore.Create(req.Username, req.Password, model.RoleMember, req.DisplayName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create user")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"user": userResponse{
			ID:          user.ID,
			Username:    user.Username,
			DisplayName: user.DisplayName,
			Role:        string(user.Role),
			CreatedAt:   user.CreatedAt.Format(time.RFC3339),
		},
	})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	user, err := s.userStore.FindByUsername(req.Username)
	if err != nil || user == nil {
		writeError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid username or password")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		writeError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid username or password")
		return
	}

	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to generate token")
		return
	}

	duration := time.Duration(s.cfg.Auth.SessionDurationH) * time.Hour
	session, err := s.sessStore.Create(user.ID, time.Now().UTC().Add(duration))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create session")
		return
	}

	writeJSON(w, http.StatusOK, authResponse{
		AccessToken:  accessToken,
		RefreshToken: session.RefreshToken,
		ExpiresIn:    s.cfg.Auth.SessionDurationH * 3600,
		User: userResponse{
			ID:          user.ID,
			Username:    user.Username,
			DisplayName: user.DisplayName,
			Role:        string(user.Role),
		},
	})
}

func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	session, err := s.sessStore.FindByRefreshToken(req.RefreshToken)
	if err != nil || session == nil {
		writeError(w, http.StatusUnauthorized, "INVALID_TOKEN", "Invalid refresh token")
		return
	}

	if time.Now().UTC().After(session.ExpiresAt) {
		s.sessStore.Delete(session.ID)
		writeError(w, http.StatusUnauthorized, "TOKEN_EXPIRED", "Refresh token has expired")
		return
	}

	s.sessStore.Delete(session.ID)

	user, err := s.userStore.FindByID(session.UserID)
	if err != nil || user == nil {
		writeError(w, http.StatusUnauthorized, "USER_NOT_FOUND", "User not found")
		return
	}

	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to generate token")
		return
	}

	duration := time.Duration(s.cfg.Auth.SessionDurationH) * time.Hour
	newSession, err := s.sessStore.Create(user.ID, time.Now().UTC().Add(duration))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create session")
		return
	}

	writeJSON(w, http.StatusOK, authResponse{
		AccessToken:  accessToken,
		RefreshToken: newSession.RefreshToken,
		ExpiresIn:    s.cfg.Auth.SessionDurationH * 3600,
		User: userResponse{
			ID:          user.ID,
			Username:    user.Username,
			DisplayName: user.DisplayName,
			Role:        string(user.Role),
		},
	})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	s.sessStore.DeleteByRefreshToken(req.RefreshToken)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	user, err := s.userStore.FindByID(userID)
	if err != nil || user == nil {
		writeError(w, http.StatusNotFound, "USER_NOT_FOUND", "User not found")
		return
	}

	writeJSON(w, http.StatusOK, userResponse{
		ID:          user.ID,
		Username:    user.Username,
		DisplayName: user.DisplayName,
		Role:        string(user.Role),
		CreatedAt:   user.CreatedAt.Format(time.RFC3339),
	})
}

func (s *Server) generateAccessToken(user *model.User) (string, error) {
	claims := jwt.MapClaims{
		"user_id": user.ID,
		"role":    string(user.Role),
		"exp":     time.Now().UTC().Add(time.Duration(s.cfg.Auth.SessionDurationH) * time.Hour).Unix(),
		"iat":     time.Now().UTC().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.Auth.JWTSecret))
}
