package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/PratSins/FrameVerse-Backend/pkg/authclient"
)

type contextKey string

const (
	UserClaimsContextKey contextKey = "user_claims"
	UserIDContextKey     contextKey = "user_id"
	EmailContextKey      contextKey = "email"
	TierContextKey       contextKey = "tier"
)

func RequireAuth(client *authclient.Client) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenString := extractToken(r)
			if tokenString == "" {
				http.Error(w, `{"error":"missing or invalid authorization header"}`, http.StatusUnauthorized)
				return
			}

			claims, err := client.ValidateToken(tokenString)
			if err != nil {
				http.Error(w, `{"error":"invalid or expired token"}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), UserClaimsContextKey, claims)
			ctx = context.WithValue(ctx, UserIDContextKey, claims.UserID)
			ctx = context.WithValue(ctx, EmailContextKey, claims.Email)
			ctx = context.WithValue(ctx, TierContextKey, claims.Tier)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func OptionalAuth(client *authclient.Client) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenString := extractToken(r)
			if tokenString != "" {
				if claims, err := client.ValidateToken(tokenString); err == nil {
					ctx := context.WithValue(r.Context(), UserClaimsContextKey, claims)
					ctx = context.WithValue(ctx, UserIDContextKey, claims.UserID)
					ctx = context.WithValue(ctx, EmailContextKey, claims.Email)
					ctx = context.WithValue(ctx, TierContextKey, claims.Tier)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

func WebSocketAuth(client *authclient.Client, r *http.Request) (*authclient.UserClaims, error) {
	token := r.URL.Query().Get("token")
	if token == "" {
		token = extractToken(r)
	}

	if token == "" {
		return nil, nil // unauthenticated guest
	}

	return client.ValidateToken(token)
}

func extractToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return strings.TrimSpace(parts[1])
		}
	}

	// Fallback to query param
	if token := r.URL.Query().Get("token"); token != "" {
		return strings.TrimSpace(token)
	}

	return ""
}

func GetUserID(ctx context.Context) string {
	if val, ok := ctx.Value(UserIDContextKey).(string); ok {
		return val
	}
	return ""
}

func GetEmail(ctx context.Context) string {
	if val, ok := ctx.Value(EmailContextKey).(string); ok {
		return val
	}
	return ""
}

func GetTier(ctx context.Context) string {
	if val, ok := ctx.Value(TierContextKey).(string); ok {
		return val
	}
	return ""
}

func GetUserClaims(ctx context.Context) *authclient.UserClaims {
	if val, ok := ctx.Value(UserClaimsContextKey).(*authclient.UserClaims); ok {
		return val
	}
	return nil
}
