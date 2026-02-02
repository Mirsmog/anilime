package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	authv1 "github.com/example/anime-platform/gen/auth/v1"
	"github.com/example/anime-platform/internal/platform/api"
	"github.com/example/anime-platform/internal/platform/httpserver"
)

type registerRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type userResponse struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Username  string `json:"username"`
	CreatedAt string `json:"created_at"`
}

type authResponse struct {
	User         userResponse `json:"user"`
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	ExpiresIn    int64        `json:"expires_in"`
}

func Register(c authv1.AuthServiceClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := withForwardedMD(r)
		rid := httpserver.RequestIDFromContext(r.Context())

		var req registerRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
			api.BadRequest(w, "INVALID_JSON", "Invalid JSON", rid, nil)
			return
		}

		resp, err := c.Register(ctx, &authv1.RegisterRequest{Email: strings.TrimSpace(req.Email), Username: strings.TrimSpace(req.Username), Password: req.Password})
		if err != nil {
			writeGRPCError(w, rid, err)
			return
		}

		api.WriteJSON(w, http.StatusCreated, toAuthResponse(resp.GetUser(), resp.GetAccessToken(), resp.GetRefreshToken(), resp.GetExpiresIn()))
	}
}

func Login(c authv1.AuthServiceClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := withForwardedMD(r)
		rid := httpserver.RequestIDFromContext(r.Context())

		var req loginRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
			api.BadRequest(w, "INVALID_JSON", "Invalid JSON", rid, nil)
			return
		}

		resp, err := c.Login(ctx, &authv1.LoginRequest{Login: strings.TrimSpace(req.Login), Password: req.Password})
		if err != nil {
			writeGRPCError(w, rid, err)
			return
		}

		api.WriteJSON(w, http.StatusOK, toAuthResponse(resp.GetUser(), resp.GetAccessToken(), resp.GetRefreshToken(), resp.GetExpiresIn()))
	}
}

func Refresh(c authv1.AuthServiceClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := withForwardedMD(r)
		rid := httpserver.RequestIDFromContext(r.Context())

		var req refreshRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
			api.BadRequest(w, "INVALID_JSON", "Invalid JSON", rid, nil)
			return
		}

		resp, err := c.Refresh(ctx, &authv1.RefreshRequest{RefreshToken: strings.TrimSpace(req.RefreshToken)})
		if err != nil {
			writeGRPCError(w, rid, err)
			return
		}
		api.WriteJSON(w, http.StatusOK, toAuthResponse(resp.GetUser(), resp.GetAccessToken(), resp.GetRefreshToken(), resp.GetExpiresIn()))
	}
}

func Logout(c authv1.AuthServiceClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := withForwardedMD(r)
		rid := httpserver.RequestIDFromContext(r.Context())

		var req refreshRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
			api.BadRequest(w, "INVALID_JSON", "Invalid JSON", rid, nil)
			return
		}

		_, err := c.Logout(ctx, &authv1.LogoutRequest{RefreshToken: strings.TrimSpace(req.RefreshToken)})
		if err != nil {
			writeGRPCError(w, rid, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func toAuthResponse(u *authv1.User, access, refresh string, expires int64) authResponse {
	ur := userResponse{}
	if u != nil {
		ur = userResponse{ID: u.GetId(), Email: u.GetEmail(), Username: u.GetUsername(), CreatedAt: u.GetCreatedAtRfc3339()}
	}
	return authResponse{User: ur, AccessToken: access, RefreshToken: refresh, ExpiresIn: expires}
}

func withForwardedMD(r *http.Request) context.Context {
	md := metadata.New(nil)
	if authz := strings.TrimSpace(r.Header.Get("Authorization")); authz != "" {
		md.Set("authorization", authz)
	}
	if ua := strings.TrimSpace(r.UserAgent()); ua != "" {
		md.Set("user-agent", ua)
	}
	return metadata.NewOutgoingContext(r.Context(), md)
}

func writeGRPCError(w http.ResponseWriter, requestID string, err error) {
	st, ok := status.FromError(err)
	if !ok {
		api.Internal(w, requestID)
		return
	}

	code := "INTERNAL"
	details := map[string]any{}

	// Try to extract structured details
	for _, d := range st.Details() {
		switch v := d.(type) {
		case *errdetails.ErrorInfo:
			if v.GetReason() != "" {
				code = v.GetReason()
			}
		case *errdetails.BadRequest:
			for _, fv := range v.GetFieldViolations() {
				if fv.GetField() != "" {
					details[fv.GetField()] = fv.GetDescription()
				}
			}
		}
	}
	if len(details) == 0 {
		details = nil
	}

	switch st.Code() {
	case codes.InvalidArgument:
		api.BadRequest(w, code, st.Message(), requestID, details)
	case codes.Unauthenticated:
		api.Unauthorized(w, code, st.Message(), requestID)
	case codes.PermissionDenied:
		api.Forbidden(w, code, st.Message(), requestID)
	case codes.NotFound:
		api.NotFound(w, code, st.Message(), requestID)
	case codes.AlreadyExists:
		api.Conflict(w, code, st.Message(), requestID, details)
	case codes.ResourceExhausted:
		api.RateLimited(w, code, st.Message(), requestID, details)
	default:
		api.Internal(w, requestID)
	}
}
