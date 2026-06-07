package server

import (
	"context"
	"net/http"
	"strings"

	"github.com/drive/drive/internal/model"
	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const (
	contextKeyUserID         contextKey = "user_id"
	contextKeyUserRole       contextKey = "user_role"
	contextKeyFolderUnlocked contextKey = "folder_unlocked"
	contextKeyShareID        contextKey = "share_id"
	contextKeySharePerm      contextKey = "share_permission"
)

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authorization header")
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid authorization format")
			return
		}

		tokenStr := parts[1]
		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(s.cfg.Auth.JWTSecret), nil
		})
		if err != nil {
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid or expired token")
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok || !token.Valid {
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid token claims")
			return
		}

		userID, _ := claims["user_id"].(string)
		role, _ := claims["role"].(string)

		if userID == "" || role == "" {
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid token payload")
			return
		}

		ctx := context.WithValue(r.Context(), contextKeyUserID, userID)
		ctx = context.WithValue(ctx, contextKeyUserRole, role)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) adminMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role, _ := r.Context().Value(contextKeyUserRole).(string)
		if role != string(model.RoleAdmin) {
			writeError(w, http.StatusForbidden, "FORBIDDEN", "Admin access required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func getUserID(r *http.Request) string {
	id, _ := r.Context().Value(contextKeyUserID).(string)
	return id
}

func getUserRole(r *http.Request) string {
	role, _ := r.Context().Value(contextKeyUserRole).(string)
	return role
}

func getShareID(r *http.Request) string {
	id, _ := r.Context().Value(contextKeyShareID).(string)
	return id
}

func getSharePermission(r *http.Request) string {
	perm, _ := r.Context().Value(contextKeySharePerm).(string)
	return perm
}

func (s *Server) IsRegistrationAllowed() bool {
	return s.auth.IsRegistrationAllowed()
}
