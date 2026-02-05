package auth

import (
	"net/http"
	"strings"
)

// RequireAdmin allows request only if RequireUser already injected role=admin into context.
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role, _ := RoleFromContext(r.Context())
		if strings.ToLower(strings.TrimSpace(role)) != "admin" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
