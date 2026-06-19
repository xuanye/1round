package middleware

import (
	"context"
	"net/http"
	"strings"

	jwtauth "github.com/xuanye/one-round/apps/server/internal/infra/auth"
)

type userIDKey struct{}

func Auth(tokens *jwtauth.JWTService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := bearerToken(r)
			if token == "" {
				token = r.URL.Query().Get("token")
			}
			userID, err := tokens.Verify(token)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userIDKey{}, userID)))
		})
	}
}

func UserID(ctx context.Context) string {
	v, _ := ctx.Value(userIDKey{}).(string)
	return v
}

func bearerToken(r *http.Request) string {
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
}
