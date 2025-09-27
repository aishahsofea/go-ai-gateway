package middleware

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/aishahsofea/go-ai-gateway/internal/models"
	"github.com/aishahsofea/go-ai-gateway/internal/utils"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type contextkey string

const (
	UserContextKey = contextkey("user")
)

func SetUser(r *http.Request, user *models.User) *http.Request {
	ctx := context.WithValue(r.Context(), UserContextKey, user)
	return r.WithContext(ctx)
}

func GetUser(r *http.Request) *models.User {
	user, ok := r.Context().Value(UserContextKey).(*models.User)
	if !ok {
		panic("missing user in request")
	}
	return user
}

func Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// Extract Bearer token from Authorization header
		w.Header().Add("Vary", "Authorization")
		authHeader := r.Header.Get("Authorization")

		if authHeader == "" {
			r = SetUser(r, models.AnonymousUser)
			next.ServeHTTP(w, r)
			return
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"error": "invalid authorization header format"})
			return
		}

		bearerToken := strings.TrimPrefix(authHeader, "Bearer ")

		// Pre-check for SECRET
		secret := os.Getenv("SECRET")
		if secret == "" {
			utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"error": "server misconfiguration: missing JWT secret"})
			return
		}

		// Validate JWT token
		token, err := jwt.Parse(bearerToken, func(token *jwt.Token) (interface{}, error) {
			_, ok := token.Method.(*jwt.SigningMethodHMAC)
			if !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"error": "invalid token"})
			return
		}

		// Extract user claims
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"error": "invalid token claims"})
			return
		}

		// Extract user ID from claims
		userIDString, ok := claims["sub"].(string)
		if !ok {
			utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"error": "invalid user ID in token"})
			return
		}

		userID, err := uuid.Parse(userIDString)
		if err != nil {
			utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"error": "invalid user ID format in token"})
			return
		}

		user := &models.User{
			ID: userID,
		}

		// Add user to request context
		r = SetUser(r, user)

		next.ServeHTTP(w, r)
	})
}
