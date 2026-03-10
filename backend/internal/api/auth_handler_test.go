package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/rakutao/collection-gateway/internal/auth"
	"github.com/rakutao/collection-gateway/internal/domain"
	"github.com/rakutao/collection-gateway/internal/repo"
	"golang.org/x/crypto/bcrypt"
)

// --- mocks ---

type mockUserStore struct {
	createFn     func(ctx context.Context, email, nickname, passwordHash string) (*domain.User, error)
	getByEmailFn func(ctx context.Context, email string) (*domain.User, error)
}

func (m *mockUserStore) Create(ctx context.Context, email, nickname, passwordHash string) (*domain.User, error) {
	return m.createFn(ctx, email, nickname, passwordHash)
}
func (m *mockUserStore) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return m.getByEmailFn(ctx, email)
}

type mockCacheStore struct {
	store map[string][]byte
}

func newMockCache() *mockCacheStore {
	return &mockCacheStore{store: make(map[string][]byte)}
}
func (m *mockCacheStore) Get(_ context.Context, key string) ([]byte, error) {
	v, ok := m.store[key]
	if !ok {
		return nil, nil
	}
	return v, nil
}
func (m *mockCacheStore) Set(_ context.Context, key string, data []byte, _ time.Duration) error {
	m.store[key] = data
	return nil
}
func (m *mockCacheStore) Del(_ context.Context, key string) error {
	delete(m.store, key)
	return nil
}

type mockEmailSender struct {
	lastTo   string
	lastCode string
}

func (m *mockEmailSender) SendVerifyCode(to, code string) error {
	m.lastTo = to
	m.lastCode = code
	return nil
}

func authReq(method, url, body string) *http.Request {
	if body != "" {
		return httptest.NewRequest(method, url, strings.NewReader(body))
	}
	return httptest.NewRequest(method, url, nil)
}

// --- tests ---

func TestHandleLogin_Success(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.MinCost)
	users := &mockUserStore{
		getByEmailFn: func(_ context.Context, email string) (*domain.User, error) {
			return &domain.User{ID: 1, Email: email, Nickname: "Nick", PasswordHash: string(hash)}, nil
		},
	}
	jm := auth.NewJWTManager("test-secret", 72*time.Hour)
	h := NewAuthHandler(users, jm, newMockCache(), &mockEmailSender{})

	req := authReq(http.MethodPost, "/auth/login", `{"email":"a@b.com","password":"secret123"}`)
	rec := httptest.NewRecorder()
	h.HandleLogin(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var resp APIResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Code != 0 {
		t.Errorf("code = %d, want 0", resp.Code)
	}
	data := resp.Data.(map[string]interface{})
	if data["token"] == nil || data["token"] == "" {
		t.Error("expected non-empty token")
	}
}

func TestHandleLogin_InvalidCredentials(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.MinCost)
	users := &mockUserStore{
		getByEmailFn: func(_ context.Context, email string) (*domain.User, error) {
			return &domain.User{ID: 1, Email: email, PasswordHash: string(hash)}, nil
		},
	}
	jm := auth.NewJWTManager("test-secret", 72*time.Hour)
	h := NewAuthHandler(users, jm, newMockCache(), &mockEmailSender{})

	req := authReq(http.MethodPost, "/auth/login", `{"email":"a@b.com","password":"wrong"}`)
	rec := httptest.NewRecorder()
	h.HandleLogin(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestHandleLogin_UserNotFound(t *testing.T) {
	users := &mockUserStore{
		getByEmailFn: func(_ context.Context, _ string) (*domain.User, error) {
			return nil, repo.ErrUserNotFound
		},
	}
	jm := auth.NewJWTManager("test-secret", 72*time.Hour)
	h := NewAuthHandler(users, jm, newMockCache(), &mockEmailSender{})

	req := authReq(http.MethodPost, "/auth/login", `{"email":"a@b.com","password":"secret123"}`)
	rec := httptest.NewRecorder()
	h.HandleLogin(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestHandleLogin_MissingFields(t *testing.T) {
	jm := auth.NewJWTManager("test-secret", 72*time.Hour)
	h := NewAuthHandler(&mockUserStore{}, jm, newMockCache(), &mockEmailSender{})

	req := authReq(http.MethodPost, "/auth/login", `{"email":"","password":""}`)
	rec := httptest.NewRecorder()
	h.HandleLogin(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestHandleRegister_Success(t *testing.T) {
	users := &mockUserStore{
		createFn: func(_ context.Context, email, nickname, _ string) (*domain.User, error) {
			return &domain.User{ID: 5, Email: email, Nickname: nickname}, nil
		},
	}
	jm := auth.NewJWTManager("test-secret", 72*time.Hour)
	cache := newMockCache()
	cache.store["verify:a@b.com"] = []byte("123456")

	h := NewAuthHandler(users, jm, cache, &mockEmailSender{})

	req := authReq(http.MethodPost, "/auth/register", `{"email":"a@b.com","password":"secret123","nickname":"Nick","code":"123456"}`)
	rec := httptest.NewRecorder()
	h.HandleRegister(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var resp APIResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Code != 0 {
		t.Errorf("code = %d, want 0", resp.Code)
	}
	// Verify code was consumed
	if _, exists := cache.store["verify:a@b.com"]; exists {
		t.Error("expected verify code to be deleted after use")
	}
}

func TestHandleSendCode_Success(t *testing.T) {
	jm := auth.NewJWTManager("test-secret", 72*time.Hour)
	cache := newMockCache()
	emailMock := &mockEmailSender{}
	h := NewAuthHandler(&mockUserStore{}, jm, cache, emailMock)

	req := authReq(http.MethodPost, "/auth/send-code", `{"email":"user@example.com"}`)
	rec := httptest.NewRecorder()
	h.HandleSendCode(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if emailMock.lastTo != "user@example.com" {
		t.Errorf("email sent to %q, want user@example.com", emailMock.lastTo)
	}
	if emailMock.lastCode == "" {
		t.Error("expected non-empty verification code")
	}
	// Verify code stored in cache
	stored, _ := cache.Get(context.Background(), "verify:user@example.com")
	if stored == nil {
		t.Error("expected verification code in cache")
	}
	// Verify cooldown set
	cd, _ := cache.Get(context.Background(), "verify_cd:user@example.com")
	if cd == nil {
		t.Error("expected cooldown key in cache")
	}
}

func TestHandleSendCode_Cooldown(t *testing.T) {
	jm := auth.NewJWTManager("test-secret", 72*time.Hour)
	cache := newMockCache()
	cache.store["verify_cd:user@example.com"] = []byte("1")
	h := NewAuthHandler(&mockUserStore{}, jm, cache, &mockEmailSender{})

	req := authReq(http.MethodPost, "/auth/send-code", `{"email":"user@example.com"}`)
	rec := httptest.NewRecorder()
	h.HandleSendCode(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("status = %d, want 429", rec.Code)
	}
}

func TestHandleSendCode_InvalidEmail(t *testing.T) {
	jm := auth.NewJWTManager("test-secret", 72*time.Hour)
	h := NewAuthHandler(&mockUserStore{}, jm, newMockCache(), &mockEmailSender{})

	req := authReq(http.MethodPost, "/auth/send-code", `{"email":"notanemail"}`)
	rec := httptest.NewRecorder()
	h.HandleSendCode(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestHandleRegister_MissingCode(t *testing.T) {
	jm := auth.NewJWTManager("test-secret", 72*time.Hour)
	h := NewAuthHandler(&mockUserStore{}, jm, newMockCache(), &mockEmailSender{})

	req := authReq(http.MethodPost, "/auth/register", `{"email":"a@b.com","password":"secret123","nickname":"Nick"}`)
	rec := httptest.NewRecorder()
	h.HandleRegister(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
	var resp APIResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Code != 40006 {
		t.Errorf("code = %d, want 40006", resp.Code)
	}
}

func TestHandleRegister_WrongCode(t *testing.T) {
	jm := auth.NewJWTManager("test-secret", 72*time.Hour)
	cache := newMockCache()
	cache.store["verify:a@b.com"] = []byte("123456")

	h := NewAuthHandler(&mockUserStore{}, jm, cache, &mockEmailSender{})

	req := authReq(http.MethodPost, "/auth/register", `{"email":"a@b.com","password":"secret123","nickname":"Nick","code":"000000"}`)
	rec := httptest.NewRecorder()
	h.HandleRegister(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestHandleRegister_EmailExists(t *testing.T) {
	users := &mockUserStore{
		createFn: func(_ context.Context, _, _, _ string) (*domain.User, error) {
			return nil, repo.ErrEmailExists
		},
	}
	jm := auth.NewJWTManager("test-secret", 72*time.Hour)
	cache := newMockCache()
	cache.store["verify:a@b.com"] = []byte("123456")

	h := NewAuthHandler(users, jm, cache, &mockEmailSender{})

	req := authReq(http.MethodPost, "/auth/register", `{"email":"a@b.com","password":"secret123","nickname":"Nick","code":"123456"}`)
	rec := httptest.NewRecorder()
	h.HandleRegister(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("status = %d, want 409", rec.Code)
	}
}
