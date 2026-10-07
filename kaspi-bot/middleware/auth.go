package middleware

import (
    "context"
    "net/http"
    "strings"
    "time"

    "kaspi-bot/config"
    "encoding/json"
    "github.com/golang-jwt/jwt/v5"
)


type contextKey string

const UserContextKey contextKey = "user_id"

// JWTMiddleware validates JWT tokens and extracts user_id
func JWTMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Get token from Authorization header
        authHeader := r.Header.Get("Authorization")
        if authHeader == "" {
            sendError(w, "Authorization header required", http.StatusUnauthorized)
            return
        }

        // Extract token from "Bearer <token>"
        parts := strings.Split(authHeader, " ")
        if len(parts) != 2 || parts[0] != "Bearer" {
            sendError(w, "Invalid authorization header format", http.StatusUnauthorized)
            return
        }

        tokenString := parts[1]

        // Parse and validate token
        token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
            return []byte(config.JWTSecret), nil
        })

        if err != nil {
            sendError(w, "Invalid token: "+err.Error(), http.StatusUnauthorized)
            return
        }

        claims, ok := token.Claims.(*Claims)
        if !ok || !token.Valid {
            sendError(w, "Invalid token claims", http.StatusUnauthorized)
            return
        }

        // Check if token is expired
        if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
            sendError(w, "Token expired", http.StatusUnauthorized)
            return
        }

        // Add user_id to request context
        ctx := context.WithValue(r.Context(), UserContextKey, claims.UserID)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// GetUserIDFromContext extracts user_id from request context
func GetUserIDFromContext(r *http.Request) (string, bool) {
    userID, ok := r.Context().Value(UserContextKey).(string)
    return userID, ok
}

// sendError helper function
func sendError(w http.ResponseWriter, message string, code int) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(code)
    json.NewEncoder(w).Encode(map[string]string{"error": message})
}
