package httpserver

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type ctxKeyRequestID struct{}

func RequestIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(ctxKeyRequestID{}).(string)
	return v
}

func RequestIDMiddleware(headerName string) func(next http.Handler) http.Handler {
	if strings.TrimSpace(headerName) == "" {
		headerName = "X-Request-Id"
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rid := strings.TrimSpace(r.Header.Get(headerName))
			if rid == "" {
				rid = uuid.NewString()
			}
			w.Header().Set(headerName, rid)
			ctx := context.WithValue(r.Context(), ctxKeyRequestID{}, rid)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
