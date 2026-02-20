package grpcapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/metadata"

	authv1 "github.com/example/anime-platform/gen/auth/v1"
	"github.com/example/anime-platform/services/auth/internal/config"
	"github.com/example/anime-platform/services/auth/internal/domain"
	"github.com/example/anime-platform/services/auth/internal/store"
	"github.com/example/anime-platform/services/auth/internal/tokens"
)

type AuthService struct {
	authv1.UnimplementedAuthServiceServer
	Store  store.Store
	Tokens tokens.Service
	Cfg    config.AuthConfig
}

func (s *AuthService) Register(ctx context.Context, req *authv1.RegisterRequest) (*authv1.RegisterResponse, error) {
	email := strings.TrimSpace(req.GetEmail())
	username := strings.TrimSpace(req.GetUsername())
	password := req.GetPassword()

	if !isValidEmail(email) {
		return nil, errInvalidArgument("VALIDATION_EMAIL", "Invalid email", map[string]string{"email": "invalid"})
	}
	if !isValidUsername(username) {
		return nil, errInvalidArgument("VALIDATION_USERNAME", "Invalid username", map[string]string{"username": "invalid"})
	}
	if len(password) < 8 {
		return nil, errInvalidArgument("VALIDATION_PASSWORD", "Password too short", map[string]string{"password": "min length 8"})
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errInternal("INTERNAL", "Internal error")
	}

	u, err := s.Store.CreateUser(ctx, store.CreateUserParams{Email: email, Username: username, PasswordHash: string(hash)})
	if err != nil {
		if errors.Is(err, store.ErrConflict) {
			return nil, errAlreadyExists("USER_ALREADY_EXISTS", "User already exists")
		}
		return nil, errInternal("INTERNAL", "Internal error")
	}

	// If bootstrap admin username matches, promote this user immediately.
	if strings.EqualFold(strings.TrimSpace(s.Cfg.BootstrapAdminUsername), u.Username) && s.Cfg.BootstrapAdminUsername != "" {
		// best-effort
		if id, err := uuid.Parse(u.ID); err == nil {
			_ = s.Store.SetUserRoleByID(ctx, id, "admin")
			u.Role = "admin"
		}
	}

	resp, err := s.issueTokens(ctx, u, clientIPFromMD(ctx), userAgentFromMD(ctx))
	if err != nil {
		return nil, errInternal("INTERNAL", "Internal error")
	}
	return resp, nil
}

func (s *AuthService) Login(ctx context.Context, req *authv1.LoginRequest) (*authv1.LoginResponse, error) {
	login := strings.TrimSpace(req.GetLogin())
	if login == "" {
		return nil, errInvalidArgument("VALIDATION_LOGIN", "Login is required", map[string]string{"login": "required"})
	}
	if req.GetPassword() == "" {
		return nil, errInvalidArgument("VALIDATION_PASSWORD", "Password is required", map[string]string{"password": "required"})
	}

	row, err := s.Store.FindUserByLogin(ctx, login)
	if err != nil {
		return nil, errUnauthenticated("AUTH_INVALID_CREDENTIALS", "Invalid credentials")
	}
	if bcrypt.CompareHashAndPassword([]byte(row.PasswordHash), []byte(req.GetPassword())) != nil {
		return nil, errUnauthenticated("AUTH_INVALID_CREDENTIALS", "Invalid credentials")
	}

	resp, err := s.issueTokens(ctx, row.User, clientIPFromMD(ctx), userAgentFromMD(ctx))
	if err != nil {
		return nil, errInternal("INTERNAL", "Internal error")
	}

	return &authv1.LoginResponse{User: resp.User, AccessToken: resp.AccessToken, RefreshToken: resp.RefreshToken, ExpiresIn: resp.ExpiresIn}, nil
}

func (s *AuthService) Refresh(ctx context.Context, req *authv1.RefreshRequest) (*authv1.RefreshResponse, error) {
	raw := strings.TrimSpace(req.GetRefreshToken())
	if raw == "" {
		return nil, errInvalidArgument("VALIDATION_REFRESH_TOKEN", "refresh_token is required", map[string]string{"refresh_token": "required"})
	}

	sess, err := s.Store.GetRefreshSessionByHash(ctx, sha256Hex(raw))
	if err != nil {
		return nil, errUnauthenticated("AUTH_INVALID_REFRESH", "Invalid refresh token")
	}
	now := time.Now().UTC()
	if sess.RevokedAt != nil || now.After(sess.ExpiresAt) {
		return nil, errUnauthenticated("AUTH_INVALID_REFRESH", "Invalid refresh token")
	}

	u, err := s.Store.GetUserByID(ctx, sess.UserID.String())
	if err != nil {
		return nil, errInternal("INTERNAL", "Internal error")
	}

	access, exp, err := s.Tokens.NewAccessToken(sess.UserID.String(), u.Role, now)
	if err != nil {
		return nil, errInternal("INTERNAL", "Internal error")
	}
	newRaw, newHash, err := tokens.NewRefreshToken()
	if err != nil {
		return nil, errInternal("INTERNAL", "Internal error")
	}
	newID := uuid.New()
	if err := s.Store.ReplaceRefreshSession(ctx, sess.ID, newID, now); err != nil {
		return nil, errInternal("INTERNAL", "Internal error")
	}
	if err := s.Store.CreateRefreshSession(ctx, store.CreateRefreshSessionParams{
		SessionID: newID,
		UserID:    sess.UserID,
		TokenHash: newHash,
		ExpiresAt: now.Add(s.Cfg.RefreshTokenTTL),
		UserAgent: userAgentFromMD(ctx),
		IP:        clientIPFromMD(ctx),
		Now:       now,
	}); err != nil {
		return nil, errInternal("INTERNAL", "Internal error")
	}

	return &authv1.RefreshResponse{
		User:         toPBUser(u),
		AccessToken:  access,
		RefreshToken: newRaw,
		ExpiresIn:    int64(time.Until(exp).Seconds()),
	}, nil
}

func (s *AuthService) Logout(ctx context.Context, req *authv1.LogoutRequest) (*authv1.LogoutResponse, error) {
	raw := strings.TrimSpace(req.GetRefreshToken())
	if raw == "" {
		return nil, errInvalidArgument("VALIDATION_REFRESH_TOKEN", "refresh_token is required", map[string]string{"refresh_token": "required"})
	}
	sess, err := s.Store.GetRefreshSessionByHash(ctx, sha256Hex(raw))
	if err == nil {
		_ = s.Store.RevokeRefreshSession(ctx, sess.ID, time.Now().UTC())
	}
	return &authv1.LogoutResponse{}, nil
}

func (s *AuthService) Me(ctx context.Context, _ *authv1.MeRequest) (*authv1.MeResponse, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	authz := first(md.Get("authorization"))
	if authz == "" {
		authz = first(md.Get("Authorization"))
	}
	authz = strings.TrimSpace(authz)
	if authz == "" {
		return nil, errUnauthenticated("AUTH_MISSING", "Missing bearer token")
	}
	parts := strings.SplitN(authz, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return nil, errUnauthenticated("AUTH_INVALID", "Invalid bearer token")
	}
	claims, err := s.Tokens.ParseAccessToken(strings.TrimSpace(parts[1]))
	if err != nil {
		return nil, errUnauthenticated("AUTH_INVALID", "Invalid token")
	}
	if strings.TrimSpace(claims.Subject) == "" {
		return nil, errUnauthenticated("AUTH_INVALID", "Invalid token")
	}
	u, err := s.Store.GetUserByID(ctx, claims.Subject)
	if err != nil {
		return &authv1.MeResponse{UserId: claims.Subject}, nil
	}
	return &authv1.MeResponse{UserId: u.ID, Email: u.Email, Username: u.Username}, nil
}

func (s *AuthService) issueTokens(ctx context.Context, u domain.User, ip net.IP, userAgent string) (*authv1.RegisterResponse, error) {
	now := time.Now().UTC()
	access, exp, err := s.Tokens.NewAccessToken(u.ID, u.Role, now)
	if err != nil {
		return nil, err
	}
	refreshRaw, refreshHash, err := tokens.NewRefreshToken()
	if err != nil {
		return nil, err
	}
	sessionID := uuid.New()
	userID, _ := uuid.Parse(u.ID)
	if err := s.Store.CreateRefreshSession(ctx, store.CreateRefreshSessionParams{
		SessionID: sessionID,
		UserID:    userID,
		TokenHash: refreshHash,
		ExpiresAt: now.Add(s.Cfg.RefreshTokenTTL),
		UserAgent: userAgent,
		IP:        ip,
		Now:       now,
	}); err != nil {
		return nil, err
	}

	return &authv1.RegisterResponse{
		User:         toPBUser(u),
		AccessToken:  access,
		RefreshToken: refreshRaw,
		ExpiresIn:    int64(time.Until(exp).Seconds()),
	}, nil
}

func toPBUser(u domain.User) *authv1.User {
	return &authv1.User{Id: u.ID, Email: u.Email, Username: u.Username, CreatedAtRfc3339: u.CreatedAt.UTC().Format(time.RFC3339)}
}

var usernameRe = regexp.MustCompile(`^[a-zA-Z0-9_]{3,32}$`)

func isValidUsername(s string) bool {
	s = strings.TrimSpace(s)
	return usernameRe.MatchString(s)
}

var emailRe = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

func isValidEmail(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) > 254 {
		return false
	}
	return emailRe.MatchString(s)
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func clientIPFromMD(ctx context.Context) net.IP {
	md, _ := metadata.FromIncomingContext(ctx)
	// For now trust x-forwarded-for from edge/bff if provided; otherwise empty.
	xff := first(md.Get("x-forwarded-for"))
	if xff == "" {
		return nil
	}
	parts := strings.Split(xff, ",")
	ip := strings.TrimSpace(parts[0])
	return net.ParseIP(ip)
}

func userAgentFromMD(ctx context.Context) string {
	md, _ := metadata.FromIncomingContext(ctx)
	ua := first(md.Get("user-agent"))
	if ua == "" {
		ua = first(md.Get("User-Agent"))
	}
	return ua
}

func first(v []string) string {
	if len(v) == 0 {
		return ""
	}
	return v[0]
}
