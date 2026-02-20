package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	authv1 "github.com/example/anime-platform/gen/auth/v1"
	"github.com/example/anime-platform/internal/platform/auth"
)

// ─── Stub auth client ─────────────────────────────────────────────────────────

type stubAuthClient struct {
	authv1.AuthServiceClient
	registerResp *authv1.RegisterResponse
	registerErr  error
	loginResp    *authv1.LoginResponse
	loginErr     error
	refreshResp  *authv1.RefreshResponse
	refreshErr   error
	logoutResp   *authv1.LogoutResponse
	logoutErr    error
	meResp       *authv1.MeResponse
	meErr        error
}

func (s *stubAuthClient) Register(_ context.Context, _ *authv1.RegisterRequest, _ ...grpc.CallOption) (*authv1.RegisterResponse, error) {
	return s.registerResp, s.registerErr
}
func (s *stubAuthClient) Login(_ context.Context, _ *authv1.LoginRequest, _ ...grpc.CallOption) (*authv1.LoginResponse, error) {
	return s.loginResp, s.loginErr
}
func (s *stubAuthClient) Refresh(_ context.Context, _ *authv1.RefreshRequest, _ ...grpc.CallOption) (*authv1.RefreshResponse, error) {
	return s.refreshResp, s.refreshErr
}
func (s *stubAuthClient) Logout(_ context.Context, _ *authv1.LogoutRequest, _ ...grpc.CallOption) (*authv1.LogoutResponse, error) {
	return s.logoutResp, s.logoutErr
}
func (s *stubAuthClient) Me(_ context.Context, _ *authv1.MeRequest, _ ...grpc.CallOption) (*authv1.MeResponse, error) {
	return s.meResp, s.meErr
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func jsonBody(v any) *bytes.Buffer {
	b, _ := json.Marshal(v)
	return bytes.NewBuffer(b)
}

func postJSON(url string, body *bytes.Buffer) *http.Request {
	req := httptest.NewRequest(http.MethodPost, url, body)
	req.Header.Set("Content-Type", "application/json")
	return req
}

func testUser() *authv1.User {
	return &authv1.User{Id: "user-1", Email: "u@example.com", Username: "uname"}
}

// ─── Register handler ─────────────────────────────────────────────────────────

func TestRegisterHandler_OK(t *testing.T) {
	stub := &stubAuthClient{
		registerResp: &authv1.RegisterResponse{User: testUser(), AccessToken: "tok", RefreshToken: "ref", ExpiresIn: 900},
	}
	req := postJSON("/v1/auth/register", jsonBody(map[string]string{"email": "u@example.com", "username": "uname", "password": "pass1234"}))
	rr := httptest.NewRecorder()
	Register(stub, nil).ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["access_token"] != "tok" {
		t.Fatalf("expected access_token 'tok', got %v", resp["access_token"])
	}
}

func TestRegisterHandler_GRPCError(t *testing.T) {
	stub := &stubAuthClient{registerErr: status.Error(codes.AlreadyExists, "user already exists")}
	req := postJSON("/v1/auth/register", jsonBody(map[string]string{"email": "u@example.com", "username": "uname", "password": "pass1234"}))
	rr := httptest.NewRecorder()
	Register(stub, nil).ServeHTTP(rr, req)
	if rr.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rr.Code)
	}
}

func TestRegisterHandler_InvalidBody(t *testing.T) {
	stub := &stubAuthClient{}
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/register", bytes.NewBufferString("notjson"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	Register(stub, nil).ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

// ─── Login handler ────────────────────────────────────────────────────────────

func TestLoginHandler_OK(t *testing.T) {
	stub := &stubAuthClient{
		loginResp: &authv1.LoginResponse{User: testUser(), AccessToken: "access", RefreshToken: "refresh", ExpiresIn: 900},
	}
	req := postJSON("/v1/auth/login", jsonBody(map[string]string{"login": "uname", "password": "pass1234"}))
	rr := httptest.NewRecorder()
	Login(stub, nil).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["access_token"] != "access" {
		t.Fatalf("unexpected access_token: %v", resp["access_token"])
	}
}

func TestLoginHandler_InvalidCredentials(t *testing.T) {
	stub := &stubAuthClient{loginErr: status.Error(codes.Unauthenticated, "invalid credentials")}
	req := postJSON("/v1/auth/login", jsonBody(map[string]string{"login": "uname", "password": "wrong"}))
	rr := httptest.NewRecorder()
	Login(stub, nil).ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

// ─── Refresh handler ──────────────────────────────────────────────────────────

func TestRefreshHandler_OK(t *testing.T) {
	stub := &stubAuthClient{
		refreshResp: &authv1.RefreshResponse{User: testUser(), AccessToken: "newaccess", RefreshToken: "newrefresh", ExpiresIn: 900},
	}
	req := postJSON("/v1/auth/refresh", jsonBody(map[string]string{"refresh_token": "oldtoken"}))
	rr := httptest.NewRecorder()
	Refresh(stub).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestRefreshHandler_InvalidToken(t *testing.T) {
	stub := &stubAuthClient{refreshErr: status.Error(codes.Unauthenticated, "invalid refresh token")}
	req := postJSON("/v1/auth/refresh", jsonBody(map[string]string{"refresh_token": "bad"}))
	rr := httptest.NewRecorder()
	Refresh(stub).ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

// ─── Logout handler ───────────────────────────────────────────────────────────

func TestLogoutHandler_OK(t *testing.T) {
	stub := &stubAuthClient{logoutResp: &authv1.LogoutResponse{}}
	req := postJSON("/v1/auth/logout", jsonBody(map[string]string{"refresh_token": "sometoken"}))
	rr := httptest.NewRecorder()
	Logout(stub).ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rr.Code)
	}
}

// ─── Me handler ───────────────────────────────────────────────────────────────

func TestMeHandler_AuthenticatedWithHeader(t *testing.T) {
	stub := &stubAuthClient{
		meResp: &authv1.MeResponse{UserId: "user-1", Email: "u@example.com", Username: "uname"},
	}
	req := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	req.Header.Set("Authorization", "Bearer sometoken")
	req = req.WithContext(auth.WithUserID(req.Context(), "user-1"))
	rr := httptest.NewRecorder()
	Me(stub).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["email"] != "u@example.com" {
		t.Fatalf("expected email in response, got %v", resp["email"])
	}
}

func TestMeHandler_NoAuthHeader(t *testing.T) {
	stub := &stubAuthClient{}
	req := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	req = req.WithContext(auth.WithUserID(req.Context(), "user-1"))
	rr := httptest.NewRecorder()
	Me(stub).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var resp map[string]any
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["user_id"] != "user-1" {
		t.Fatalf("expected user_id in response, got %v", resp["user_id"])
	}
	if resp["email"] != nil {
		t.Fatal("email should not be present without auth header")
	}
}
