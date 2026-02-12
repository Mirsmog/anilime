package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"

	jwt "github.com/golang-jwt/jwt/v5"
)

type ctxKeyUserID struct{}
type ctxKeyRole struct{}

func UserIDFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ctxKeyUserID{}).(string)
	return v, ok
}

// WithUserID injects user_id into context. Useful for testing.
func WithUserID(ctx context.Context, uid string) context.Context {
	return context.WithValue(ctx, ctxKeyUserID{}, uid)
}

func RoleFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ctxKeyRole{}).(string)
	return v, ok
}

type Claims struct {
	jwt.RegisteredClaims
	Role string `json:"role"`
}

type JWTVerifier struct {
	Secret []byte
}

func (v JWTVerifier) Parse(tokenString string) (*Claims, error) {
	parsed, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, errors.New("unexpected signing method")
		}
		return v.Secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := parsed.Claims.(*Claims)
	if !ok || !parsed.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

// RequireUser middleware validates Bearer token and injects user_id into context.
func RequireUser(verifier JWTVerifier) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authz := strings.TrimSpace(r.Header.Get("Authorization"))
			if authz == "" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			parts := strings.SplitN(authz, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			claims, err := verifier.Parse(strings.TrimSpace(parts[1]))
			if err != nil || strings.TrimSpace(claims.Subject) == "" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), ctxKeyUserID{}, claims.Subject)
			if strings.TrimSpace(claims.Role) != "" {
				ctx = context.WithValue(ctx, ctxKeyRole{}, claims.Role)
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
