package middleware

import (
	"betest/internal/response"
	"betest/internal/rbac"
	"context"
	"crypto/rsa"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-redis/redis/v8"
	"github.com/golang-jwt/jwt/v5"
)

// ContextKey type for context keys
type ContextKey string

const UserIDContextKey ContextKey = "user_id"

var (
	PublicKey *rsa.PublicKey // Will be set from main or auth
	Rdb       *redis.Client  // Will be set from main or auth
)

func JWTMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			response.SendError(w, http.StatusUnauthorized, "Missing token")
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return PublicKey, nil
		})

		if err != nil || !token.Valid {
			response.SendError(w, http.StatusUnauthorized, "Invalid token")
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			// Cek apakah token ada di blacklist
			if jti, ok := claims["jti"].(string); ok {
				blacklistKey := fmt.Sprintf("blacklist:access_token:%s", jti)
				val, err := Rdb.Get(context.Background(), blacklistKey).Result()
				if err == nil && val == "1" {
					// Token ada di blacklist (sudah logout)
					response.SendError(w, http.StatusUnauthorized, "Token has been revoked")
					return
				}
			}

			userID := int(claims["user_id"].(float64))
			ctx := context.WithValue(r.Context(), UserIDContextKey, userID)
			r = r.WithContext(ctx)
		} else {
			response.SendError(w, http.StatusUnauthorized, "Invalid token claims")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// GetUserID returns the authenticated user ID from request context, or 0 if not set.
func GetUserID(r *http.Request) int {
	v := r.Context().Value(UserIDContextKey)
	if v == nil {
		return 0
	}
	if id, ok := v.(int); ok {
		return id
	}
	return 0
}

// RequirePermission returns middleware that allows access only if the user has the given permission code.
// Must be used after JWTMiddleware.
func RequirePermission(permissionCode string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := GetUserID(r)
			if userID == 0 {
				response.SendError(w, http.StatusUnauthorized, "Unauthorized")
				return
			}
			ok, err := rbac.HasPermission(r.Context(), userID, permissionCode)
			if err != nil {
				response.SendError(w, http.StatusInternalServerError, "Error checking permission")
				return
			}
			if !ok {
				response.SendError(w, http.StatusForbidden, "Forbidden: insufficient permission")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
