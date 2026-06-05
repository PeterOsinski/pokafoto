package server

import (
	"context"
	"net/http"
	"strings"
	"time"

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

func (s *Server) folderUnlockSecret() string {
	return s.cfg.Auth.JWTSecret + ":folder_unlock"
}

func (s *Server) parseFolderUnlockToken(tokenStr string) (string, bool) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(s.folderUnlockSecret()), nil
	})
	if err != nil || !token.Valid {
		return "", false
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", false
	}

	sub, _ := claims["sub"].(string)
	if sub != "folder_unlock" {
		return "", false
	}

	folderID, _ := claims["folder_id"].(string)
	return folderID, folderID != ""
}

func (s *Server) generateFolderUnlockToken(folderID string, expiryTime time.Time) (string, error) {
	claims := jwt.MapClaims{
		"sub":       "folder_unlock",
		"folder_id": folderID,
		"iat":       time.Now().Unix(),
		"exp":       expiryTime.Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.folderUnlockSecret()))
}

func (s *Server) parseShareSessionToken(tokenStr string) (string, model.SharePermission, bool) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(s.cfg.Auth.JWTSecret), nil
	})
	if err != nil || !token.Valid {
		return "", "", false
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", "", false
	}

	sub, _ := claims["sub"].(string)
	if sub != "share_session" {
		return "", "", false
	}

	shareID, _ := claims["share_id"].(string)
	permissions, _ := claims["permissions"].(string)

	return shareID, model.SharePermission(permissions), shareID != ""
}

func (s *Server) generateShareSessionToken(shareID, folderID string, permissions model.SharePermission, expiryTime time.Time) (string, error) {
	claims := jwt.MapClaims{
		"sub":         "share_session",
		"share_id":    shareID,
		"folder_id":   folderID,
		"permissions": string(permissions),
		"iat":         time.Now().Unix(),
		"exp":         expiryTime.Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.Auth.JWTSecret))
}

func (s *Server) shareAccessMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-Share-Session-Token")
		if token == "" {
			token = r.URL.Query().Get("share_session_token")
		}
		if token == "" {
			writeError(w, http.StatusUnauthorized, "SHARE_TOKEN_REQUIRED", "Share session token required")
			return
		}

		shareID, permissions, ok := s.parseShareSessionToken(token)
		if !ok {
			writeError(w, http.StatusUnauthorized, "UNLOCK_EXPIRED", "Share session token is invalid or expired")
			return
		}

		ctx := context.WithValue(r.Context(), contextKeyShareID, shareID)
		ctx = context.WithValue(ctx, contextKeySharePerm, string(permissions))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) checkFolderAccess(folderID, userID string, r *http.Request) bool {
	fp, err := s.file.FolderPwStore.FindByFolderID(folderID)
	if err != nil {
		return true
	}

	now := time.Now().UTC()
	if now.After(fp.ExpiresAt) {
		s.file.FolderPwStore.DeleteByFolderID(folderID)
		return true
	}

	if _, err := s.file.FolderStore.FindByID(folderID); err != nil {
		return false
	}

	unlockToken := r.Header.Get("X-Folder-Unlock-Token")
	if unlockToken == "" {
		unlockToken = r.URL.Query().Get("folder_unlock_token")
	}
	if unlockToken != "" {
		unlockedFolderID, ok := s.parseFolderUnlockToken(unlockToken)
		if ok && unlockedFolderID == folderID {
			return true
		}
	}

	return false
}

func (s *Server) checkShareAccess(r *http.Request, permission model.SharePermission) (*model.FolderShare, bool) {
	token := r.Header.Get("X-Share-Session-Token")
	if token == "" {
		token = r.URL.Query().Get("share_session_token")
	}
	if token == "" {
		return nil, false
	}

	shareID, perms, ok := s.parseShareSessionToken(token)
	if !ok {
		return nil, false
	}

	share, err := s.share.FolderShareStore.FindByID(shareID)
	if err != nil {
		return nil, false
	}

	if share.ExpiresAt != nil && time.Now().UTC().After(*share.ExpiresAt) {
		return nil, false
	}

	switch permission {
	case model.ShareRead:
		return share, true
	case model.ShareReadUpload:
		if perms == model.ShareReadUpload || perms == model.ShareReadWrite {
			return share, true
		}
	case model.ShareReadWrite:
		if perms == model.ShareReadWrite {
			return share, true
		}
	}

	return nil, false
}

func (s *Server) folderPasswordExpiryDuration() time.Duration {
	minutes := 30
	if s.cfg.Auth.FolderPasswordExpiryMinutes > 0 {
		minutes = s.cfg.Auth.FolderPasswordExpiryMinutes
	}
	return time.Duration(minutes) * time.Minute
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
