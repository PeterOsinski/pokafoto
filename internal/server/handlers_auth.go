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

func (c *AuthCtl) IsRegistrationAllowed() bool {
	if c.SettingStore != nil {
		if val, err := c.SettingStore.Get("allow_registration"); err == nil && val != "" {
			return val == "true"
		}
	}
	return c.Cfg.Auth.AllowRegistration
}

func (c *AuthCtl) HandleAuthConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"allow_registration": c.IsRegistrationAllowed(),
	})
}

func (c *AuthCtl) HandleRegister(w http.ResponseWriter, r *http.Request) {
	if !c.IsRegistrationAllowed() {
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

	existing, _ := c.UserStore.FindByUsername(req.Username)
	if existing != nil {
		writeError(w, http.StatusConflict, "USERNAME_EXISTS", "Username is already taken")
		return
	}

	user, err := c.UserStore.Create(req.Username, req.Password, model.RoleMember, req.DisplayName)
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

func (c *AuthCtl) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	user, err := c.UserStore.FindByUsername(req.Username)
	if err != nil || user == nil {
		writeError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid username or password")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		writeError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid username or password")
		return
	}

	accessToken, err := c.generateAccessToken(user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to generate token")
		return
	}

	duration := time.Duration(c.Cfg.Auth.SessionDurationH) * time.Hour
	session, err := c.SessionStore.Create(user.ID, time.Now().UTC().Add(duration))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create session")
		return
	}

	writeJSON(w, http.StatusOK, authResponse{
		AccessToken:  accessToken,
		RefreshToken: session.RefreshToken,
		ExpiresIn:    c.Cfg.Auth.SessionDurationH * 3600,
		User: userResponse{
			ID:          user.ID,
			Username:    user.Username,
			DisplayName: user.DisplayName,
			Role:        string(user.Role),
		},
	})
}

func (c *AuthCtl) HandleRefresh(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	session, err := c.SessionStore.FindByRefreshToken(req.RefreshToken)
	if err != nil || session == nil {
		writeError(w, http.StatusUnauthorized, "INVALID_TOKEN", "Invalid refresh token")
		return
	}

	if time.Now().UTC().After(session.ExpiresAt) {
		c.SessionStore.Delete(session.ID)
		writeError(w, http.StatusUnauthorized, "TOKEN_EXPIRED", "Refresh token has expired")
		return
	}

	c.SessionStore.Delete(session.ID)

	user, err := c.UserStore.FindByID(session.UserID)
	if err != nil || user == nil {
		writeError(w, http.StatusUnauthorized, "USER_NOT_FOUND", "User not found")
		return
	}

	accessToken, err := c.generateAccessToken(user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to generate token")
		return
	}

	duration := time.Duration(c.Cfg.Auth.SessionDurationH) * time.Hour
	newSession, err := c.SessionStore.Create(user.ID, time.Now().UTC().Add(duration))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create session")
		return
	}

	writeJSON(w, http.StatusOK, authResponse{
		AccessToken:  accessToken,
		RefreshToken: newSession.RefreshToken,
		ExpiresIn:    c.Cfg.Auth.SessionDurationH * 3600,
		User: userResponse{
			ID:          user.ID,
			Username:    user.Username,
			DisplayName: user.DisplayName,
			Role:        string(user.Role),
		},
	})
}

func (c *AuthCtl) HandleLogout(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	c.SessionStore.DeleteByRefreshToken(req.RefreshToken)
	w.WriteHeader(http.StatusNoContent)
}

func (c *AuthCtl) HandleMe(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	user, err := c.UserStore.FindByID(userID)
	if err != nil || user == nil {
		writeError(w, http.StatusNotFound, "USER_NOT_FOUND", "User not found")
		return
	}

	resp := map[string]interface{}{
		"id":       user.ID,
		"username": user.Username,
		"role":     string(user.Role),
	}
	if user.DisplayName != nil {
		resp["display_name"] = *user.DisplayName
	}
	if user.SpaceQuota != nil {
		resp["space_quota"] = *user.SpaceQuota
	}
	writeJSON(w, http.StatusOK, resp)
}

func (c *AuthCtl) generateAccessToken(user *model.User) (string, error) {
	claims := jwt.MapClaims{
		"user_id": user.ID,
		"role":    string(user.Role),
		"exp":     time.Now().UTC().Add(time.Duration(c.Cfg.Auth.SessionDurationH) * time.Hour).Unix(),
		"iat":     time.Now().UTC().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(c.Cfg.Auth.JWTSecret))
}
