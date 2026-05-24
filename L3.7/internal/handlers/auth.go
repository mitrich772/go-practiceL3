package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"warehousecontrol/internal/model"
)

// LoginRequest описывает payload для POST /login.
type LoginRequest struct {
	Username string     `json:"username"`
	Role     model.Role `json:"role"`
}

type LoginResponse struct {
	Token string         `json:"token"`
	User  model.AuthUser `json:"user"`
}

type UsersResponse struct {
	Users []model.User `json:"users"`
}

// UsersLister возвращает список демо-пользователей.
type UsersLister interface {
	ListUsers(ctx context.Context) ([]model.User, error)
}

type authClaims struct {
	Username string     `json:"username"`
	Role     model.Role `json:"role"`
	jwt.RegisteredClaims
}

// Login выдаёт JWT для выбранной роли.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" {
		req.Username = string(req.Role)
	}
	if !req.Role.Valid() {
		http.Error(w, errInvalidRole.Error(), http.StatusBadRequest)
		return
	}

	now := time.Now().UTC()
	claims := authClaims{
		Username: req.Username,
		Role:     req.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(h.tokenTTL)),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(h.jwtSecret)
	if err != nil {
		h.log.Error("failed to sign jwt", "err", err)
		http.Error(w, "failed to sign token", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, LoginResponse{
		Token: token,
		User: model.AuthUser{
			Username: req.Username,
			Role:     req.Role,
		},
	})
}

// ListUsers обрабатывает GET /users.
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.users.ListUsers(r.Context())
	if err != nil {
		h.log.Error("failed to list users", "err", err)
		http.Error(w, "failed to list users", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, UsersResponse{Users: users})
}

// AuthMiddleware проверяет JWT и кладёт пользователя в контекст.
func (h *Handler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := strings.TrimSpace(r.Header.Get("Authorization"))
		if header == "" {
			http.Error(w, errAuthorizationRequired.Error(), http.StatusUnauthorized)
			return
		}
		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
			http.Error(w, errInvalidToken.Error(), http.StatusUnauthorized)
			return
		}

		claims := authClaims{}
		token, err := jwt.ParseWithClaims(parts[1], &claims, func(token *jwt.Token) (any, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, errInvalidToken
			}
			return h.jwtSecret, nil
		})
		if err != nil || token == nil || !token.Valid || !claims.Role.Valid() || strings.TrimSpace(claims.Username) == "" {
			http.Error(w, errInvalidToken.Error(), http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r.WithContext(withUser(r.Context(), model.AuthUser{
			Username: claims.Username,
			Role:     claims.Role,
		})))
	})
}

func RequireWrite(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := requestUser(r)
		if !ok {
			http.Error(w, errUnauthorized.Error(), http.StatusUnauthorized)
			return
		}
		if !user.Role.CanWrite() {
			http.Error(w, errForbidden.Error(), http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func RequireDelete(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := requestUser(r)
		if !ok {
			http.Error(w, errUnauthorized.Error(), http.StatusUnauthorized)
			return
		}
		if !user.Role.CanDelete() {
			http.Error(w, errForbidden.Error(), http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func RequireHistory(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := requestUser(r)
		if !ok {
			http.Error(w, errUnauthorized.Error(), http.StatusUnauthorized)
			return
		}
		if !user.Role.CanViewHistory() {
			http.Error(w, errForbidden.Error(), http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
