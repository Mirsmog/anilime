package handlers

import (
	"net/http"

	"google.golang.org/grpc/metadata"

	authv1 "github.com/example/anime-platform/gen/auth/v1"
	"github.com/example/anime-platform/internal/platform/api"
	"github.com/example/anime-platform/internal/platform/auth"
)

// Me handles GET /v1/me — returns the authenticated user's profile,
// optionally enriched with email/username from the auth service.
func Me(authClient authv1.AuthServiceClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uid, _ := auth.UserIDFromContext(r.Context())
		resp := map[string]any{"user_id": uid}

		if authz := r.Header.Get("Authorization"); authz != "" {
			md := metadata.New(map[string]string{"authorization": authz})
			ctx := metadata.NewOutgoingContext(r.Context(), md)
			me, err := authClient.Me(ctx, &authv1.MeRequest{})
			if err == nil {
				if me.GetEmail() != "" {
					resp["email"] = me.GetEmail()
				}
				if me.GetUsername() != "" {
					resp["username"] = me.GetUsername()
				}
			}
		}

		api.WriteJSON(w, http.StatusOK, resp)
	}
}
