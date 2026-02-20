package grpcapi

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	authv1 "github.com/example/anime-platform/gen/auth/v1"
	"github.com/example/anime-platform/services/auth/internal/config"
	"github.com/example/anime-platform/services/auth/internal/domain"
	"github.com/example/anime-platform/services/auth/internal/store"
	"github.com/example/anime-platform/services/auth/internal/tokens"
)

// ─── Mock Store ───────────────────────────────────────────────────────────────

type mockStore struct {
	users    map[string]domain.User
	byLogin  map[string]store.UserRow
	sessions map[string]store.RefreshSession

	createUserErr           error
	findUserByLoginErr      error
	getUserByIDErr          error
	createRefreshSessionErr error
	getRefreshSessionErr    error
}

func (m *mockStore) CreateUser(_ context.Context, p store.CreateUserParams) (domain.User, error) {
	if m.createUserErr != nil {
		return domain.User{}, m.createUserErr
	}
	u := domain.User{
		ID:        uuid.NewString(),
		Email:     p.Email,
		Username:  p.Username,
		Role:      "user",
		CreatedAt: time.Now().UTC(),
	}
	if m.users == nil {
		m.users = make(map[string]domain.User)
	}
	m.users[u.ID] = u
	return u, nil
}

func (m *mockStore) FindUserByLogin(_ context.Context, login string) (store.UserRow, error) {
	if m.findUserByLoginErr != nil {
		return store.UserRow{}, m.findUserByLoginErr
	}
	row, ok := m.byLogin[login]
	if !ok {
		return store.UserRow{}, store.ErrNotFound
	}
	return row, nil
}

func (m *mockStore) GetUserByID(_ context.Context, userID string) (domain.User, error) {
	if m.getUserByIDErr != nil {
		return domain.User{}, m.getUserByIDErr
	}
	u, ok := m.users[userID]
	if !ok {
		return domain.User{}, store.ErrNotFound
	}
	return u, nil
}

func (m *mockStore) SetUserRoleByID(_ context.Context, userID uuid.UUID, role string) error {
	if u, ok := m.users[userID.String()]; ok {
		u.Role = role
		m.users[userID.String()] = u
	}
	return nil
}

func (m *mockStore) CreateRefreshSession(_ context.Context, p store.CreateRefreshSessionParams) error {
	if m.createRefreshSessionErr != nil {
		return m.createRefreshSessionErr
	}
	if m.sessions == nil {
		m.sessions = make(map[string]store.RefreshSession)
	}
	m.sessions[p.TokenHash] = store.RefreshSession{
		ID:        p.SessionID,
		UserID:    p.UserID,
		TokenHash: p.TokenHash,
		ExpiresAt: p.ExpiresAt,
	}
	return nil
}

func (m *mockStore) GetRefreshSessionByHash(_ context.Context, tokenHash string) (store.RefreshSession, error) {
	if m.getRefreshSessionErr != nil {
		return store.RefreshSession{}, m.getRefreshSessionErr
	}
	sess, ok := m.sessions[tokenHash]
	if !ok {
		return store.RefreshSession{}, store.ErrNotFound
	}
	return sess, nil
}

func (m *mockStore) RevokeRefreshSession(_ context.Context, sessionID uuid.UUID, now time.Time) error {
	for hash, sess := range m.sessions {
		if sess.ID == sessionID {
			sess.RevokedAt = &now
			m.sessions[hash] = sess
		}
	}
	return nil
}

func (m *mockStore) ReplaceRefreshSession(_ context.Context, oldID, _ uuid.UUID, now time.Time) error {
	for hash, sess := range m.sessions {
		if sess.ID == oldID {
			t := now
			sess.RevokedAt = &t
			m.sessions[hash] = sess
		}
	}
	return nil
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func newTestAuthService(ms *mockStore) *AuthService {
	return &AuthService{
		Store:  ms,
		Tokens: tokens.Service{Secret: []byte("test-secret"), AccessTokenTTL: 15 * time.Minute, RefreshTokenTTL: 30 * 24 * time.Hour},
		Cfg:    config.AuthConfig{RefreshTokenTTL: 30 * 24 * time.Hour},
	}
}

func grpcCode(err error) codes.Code {
	if err == nil {
		return codes.OK
	}
	return status.Code(err)
}

func userRowWithPassword(email, username, password string) store.UserRow {
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	uid := uuid.NewString()
	return store.UserRow{
		User:         domain.User{ID: uid, Email: email, Username: username, Role: "user", CreatedAt: time.Now()},
		PasswordHash: string(hash),
	}
}

// ─── Register ─────────────────────────────────────────────────────────────────

func TestRegister_OK(t *testing.T) {
	svc := newTestAuthService(&mockStore{})
	resp, err := svc.Register(context.Background(), &authv1.RegisterRequest{
		Email: "user@example.com", Username: "testuser", Password: "password123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetAccessToken() == "" {
		t.Fatal("expected non-empty access token")
	}
	if resp.GetUser().GetEmail() != "user@example.com" {
		t.Fatalf("expected email user@example.com, got %s", resp.GetUser().GetEmail())
	}
}

func TestRegister_InvalidEmail(t *testing.T) {
	svc := newTestAuthService(&mockStore{})
	_, err := svc.Register(context.Background(), &authv1.RegisterRequest{
		Email: "notanemail", Username: "testuser", Password: "password123",
	})
	if grpcCode(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", grpcCode(err))
	}
}

func TestRegister_InvalidUsername(t *testing.T) {
	svc := newTestAuthService(&mockStore{})
	_, err := svc.Register(context.Background(), &authv1.RegisterRequest{
		Email: "user@example.com", Username: "a", Password: "password123",
	})
	if grpcCode(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", grpcCode(err))
	}
}

func TestRegister_PasswordTooShort(t *testing.T) {
	svc := newTestAuthService(&mockStore{})
	_, err := svc.Register(context.Background(), &authv1.RegisterRequest{
		Email: "user@example.com", Username: "testuser", Password: "short",
	})
	if grpcCode(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", grpcCode(err))
	}
}

func TestRegister_Duplicate(t *testing.T) {
	svc := newTestAuthService(&mockStore{createUserErr: store.ErrConflict})
	_, err := svc.Register(context.Background(), &authv1.RegisterRequest{
		Email: "user@example.com", Username: "testuser", Password: "password123",
	})
	if grpcCode(err) != codes.AlreadyExists {
		t.Fatalf("expected AlreadyExists, got %v", grpcCode(err))
	}
}

// ─── Login ────────────────────────────────────────────────────────────────────

func TestLogin_OK(t *testing.T) {
	row := userRowWithPassword("user@example.com", "testuser", "password123")
	ms := &mockStore{
		users:   map[string]domain.User{row.User.ID: row.User},
		byLogin: map[string]store.UserRow{"testuser": row},
	}
	svc := newTestAuthService(ms)
	resp, err := svc.Login(context.Background(), &authv1.LoginRequest{Login: "testuser", Password: "password123"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetAccessToken() == "" {
		t.Fatal("expected non-empty access token")
	}
}

func TestLogin_EmptyLogin(t *testing.T) {
	svc := newTestAuthService(&mockStore{})
	_, err := svc.Login(context.Background(), &authv1.LoginRequest{Login: "", Password: "password123"})
	if grpcCode(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", grpcCode(err))
	}
}

func TestLogin_EmptyPassword(t *testing.T) {
	svc := newTestAuthService(&mockStore{})
	_, err := svc.Login(context.Background(), &authv1.LoginRequest{Login: "user@example.com", Password: ""})
	if grpcCode(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", grpcCode(err))
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	svc := newTestAuthService(&mockStore{byLogin: map[string]store.UserRow{}})
	_, err := svc.Login(context.Background(), &authv1.LoginRequest{Login: "ghost@example.com", Password: "password123"})
	if grpcCode(err) != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated, got %v", grpcCode(err))
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	row := userRowWithPassword("user@example.com", "testuser", "correctpassword")
	ms := &mockStore{
		users:   map[string]domain.User{row.User.ID: row.User},
		byLogin: map[string]store.UserRow{"testuser": row},
	}
	svc := newTestAuthService(ms)
	_, err := svc.Login(context.Background(), &authv1.LoginRequest{Login: "testuser", Password: "wrongpassword"})
	if grpcCode(err) != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated, got %v", grpcCode(err))
	}
}

// ─── Refresh ──────────────────────────────────────────────────────────────────

func TestRefresh_EmptyToken(t *testing.T) {
	svc := newTestAuthService(&mockStore{})
	_, err := svc.Refresh(context.Background(), &authv1.RefreshRequest{RefreshToken: ""})
	if grpcCode(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", grpcCode(err))
	}
}

func TestRefresh_SessionNotFound(t *testing.T) {
	svc := newTestAuthService(&mockStore{sessions: map[string]store.RefreshSession{}})
	_, err := svc.Refresh(context.Background(), &authv1.RefreshRequest{RefreshToken: "unknown-token"})
	if grpcCode(err) != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated, got %v", grpcCode(err))
	}
}

func TestRefresh_ExpiredSession(t *testing.T) {
	raw, hash, _ := tokens.NewRefreshToken()
	userID := uuid.New()
	ms := &mockStore{
		sessions: map[string]store.RefreshSession{
			hash: {ID: uuid.New(), UserID: userID, TokenHash: hash, ExpiresAt: time.Now().Add(-time.Hour)},
		},
	}
	svc := newTestAuthService(ms)
	_, err := svc.Refresh(context.Background(), &authv1.RefreshRequest{RefreshToken: raw})
	if grpcCode(err) != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated, got %v", grpcCode(err))
	}
}

func TestRefresh_RevokedSession(t *testing.T) {
	raw, hash, _ := tokens.NewRefreshToken()
	userID := uuid.New()
	revokedAt := time.Now().Add(-time.Minute)
	ms := &mockStore{
		sessions: map[string]store.RefreshSession{
			hash: {ID: uuid.New(), UserID: userID, TokenHash: hash, ExpiresAt: time.Now().Add(time.Hour), RevokedAt: &revokedAt},
		},
	}
	svc := newTestAuthService(ms)
	_, err := svc.Refresh(context.Background(), &authv1.RefreshRequest{RefreshToken: raw})
	if grpcCode(err) != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated, got %v", grpcCode(err))
	}
}

func TestRefresh_OK(t *testing.T) {
	raw, hash, _ := tokens.NewRefreshToken()
	userID := uuid.New()
	u := domain.User{ID: userID.String(), Email: "u@example.com", Username: "uname", Role: "user", CreatedAt: time.Now()}
	ms := &mockStore{
		users: map[string]domain.User{userID.String(): u},
		sessions: map[string]store.RefreshSession{
			hash: {ID: uuid.New(), UserID: userID, TokenHash: hash, ExpiresAt: time.Now().Add(time.Hour)},
		},
	}
	svc := newTestAuthService(ms)
	resp, err := svc.Refresh(context.Background(), &authv1.RefreshRequest{RefreshToken: raw})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetAccessToken() == "" {
		t.Fatal("expected non-empty access token")
	}
	if resp.GetRefreshToken() == raw {
		t.Fatal("new refresh token must differ from the old one")
	}
}

// ─── Logout ───────────────────────────────────────────────────────────────────

func TestLogout_EmptyToken(t *testing.T) {
	svc := newTestAuthService(&mockStore{})
	_, err := svc.Logout(context.Background(), &authv1.LogoutRequest{RefreshToken: ""})
	if grpcCode(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", grpcCode(err))
	}
}

func TestLogout_SessionNotFound_StillOK(t *testing.T) {
	// Logout is best-effort: missing session still returns success.
	svc := newTestAuthService(&mockStore{sessions: map[string]store.RefreshSession{}})
	_, err := svc.Logout(context.Background(), &authv1.LogoutRequest{RefreshToken: "unknown-token"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLogout_OK(t *testing.T) {
	raw, hash, _ := tokens.NewRefreshToken()
	sessID := uuid.New()
	userID := uuid.New()
	ms := &mockStore{
		sessions: map[string]store.RefreshSession{
			hash: {ID: sessID, UserID: userID, TokenHash: hash, ExpiresAt: time.Now().Add(time.Hour)},
		},
	}
	svc := newTestAuthService(ms)
	_, err := svc.Logout(context.Background(), &authv1.LogoutRequest{RefreshToken: raw})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ms.sessions[hash].RevokedAt == nil {
		t.Fatal("session should be revoked after logout")
	}
}

// ─── Me ───────────────────────────────────────────────────────────────────────

func TestMe_MissingToken(t *testing.T) {
	svc := newTestAuthService(&mockStore{})
	_, err := svc.Me(context.Background(), &authv1.MeRequest{})
	if grpcCode(err) != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated, got %v", grpcCode(err))
	}
}

func TestMe_InvalidBearerFormat(t *testing.T) {
	svc := newTestAuthService(&mockStore{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "NotBearer bad"))
	_, err := svc.Me(ctx, &authv1.MeRequest{})
	if grpcCode(err) != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated, got %v", grpcCode(err))
	}
}

func TestMe_InvalidToken(t *testing.T) {
	svc := newTestAuthService(&mockStore{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer notavalidjwt"))
	_, err := svc.Me(ctx, &authv1.MeRequest{})
	if grpcCode(err) != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated, got %v", grpcCode(err))
	}
}

func TestMe_OK(t *testing.T) {
	tokSvc := tokens.Service{Secret: []byte("test-secret"), AccessTokenTTL: 15 * time.Minute}
	userID := uuid.NewString()
	u := domain.User{ID: userID, Email: "u@example.com", Username: "uname", Role: "user", CreatedAt: time.Now()}
	access, _, err := tokSvc.NewAccessToken(userID, "user", time.Now())
	if err != nil {
		t.Fatalf("failed to create access token: %v", err)
	}
	ms := &mockStore{users: map[string]domain.User{userID: u}}
	svc := newTestAuthService(ms)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+access))
	resp, err := svc.Me(ctx, &authv1.MeRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetUserId() != userID {
		t.Fatalf("expected user_id %s, got %s", userID, resp.GetUserId())
	}
	if resp.GetEmail() != "u@example.com" {
		t.Fatalf("expected email u@example.com, got %s", resp.GetEmail())
	}
}
