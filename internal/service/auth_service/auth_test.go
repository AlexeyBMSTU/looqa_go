package authservice

import (
	"context"
	"errors"
	"looqa/internal/auth"
	"looqa/model"
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// mockRefresh — подмена RefreshRepository: запоминает вызовы и возвращает
// заданные результаты.
type mockRefresh struct {
	added     []model.RefreshToken // всё, что прошло через AddRefresh
	rotated   model.RefreshToken   // что вернуть из RotateRefresh
	rotateErr error
	addErr    error
	revokeErr error

	rotCalls    int
	gotRotate   string    // хеш, переданный в RotateRefresh
	gotRevoke   uuid.UUID // family_id, переданный в RevokeFamily
	revokeCalls int
}

func (m *mockRefresh) AddRefresh(_ context.Context, token model.RefreshToken) error {
	m.added = append(m.added, token)
	return m.addErr
}

func (m *mockRefresh) RotateRefresh(_ context.Context, hash string) (model.RefreshToken, error) {
	m.rotCalls++
	m.gotRotate = hash
	return m.rotated, m.rotateErr
}

func (m *mockRefresh) RevokeFamily(_ context.Context, familyID uuid.UUID) error {
	m.revokeCalls++
	m.gotRevoke = familyID
	return m.revokeErr
}

// mockUsers — подмена UserRepository.
type mockUsers struct {
	askedEmail string
	cred       model.Credential
	credErr    error

	createdUser model.User
	createdCred model.Credential
	createErr   error
	createCalls int
}

func (m *mockUsers) CreateUser(_ context.Context, user model.User, cred model.Credential) error {
	m.createCalls++
	m.createdUser = user
	m.createdCred = cred
	return m.createErr
}

func (m *mockUsers) GetCredentialByEmail(_ context.Context, email string) (model.Credential, error) {
	m.askedEmail = email
	return m.cred, m.credErr
}

// hash пароля для тестов (MinCost — быстро).
func hash(t *testing.T, password string) string {
	t.Helper()
	h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}
	return string(h)
}

// newTestService собирает сервис с настоящим выпускальщиком токенов
// (ключи Ed25519 генерируются мгновенно) и моками хранилищ.
func newTestService(refresh RefreshRepository, users UserRepository, refreshTTL time.Duration) (*Service, *auth.Issuer) {
	issuer, err := auth.NewIssuer(time.Minute)
	if err != nil {
		panic(err)
	}
	return NewService(refresh, users, issuer, refreshTTL), issuer
}

func TestRegister(t *testing.T) {
	tests := []struct {
		name         string
		email        string
		password     string
		mock         *mockUsers
		wantErr      error
		wantNoCreate bool // CreateUser не должен вызываться (ошибка валидации)
	}{
		{name: "invalid email", email: "not-an-email", password: "secret123", mock: &mockUsers{}, wantErr: model.ErrInvalid, wantNoCreate: true},
		{name: "email with display name", email: "Ivan <ivan@x.ru>", password: "secret123", mock: &mockUsers{}, wantErr: model.ErrInvalid, wantNoCreate: true},
		{name: "short password", email: "ivan@looqa.dev", password: "short", mock: &mockUsers{}, wantErr: model.ErrInvalid, wantNoCreate: true},
		{name: "email taken", email: "ivan@looqa.dev", password: "secret123", mock: &mockUsers{createErr: model.ErrConflict}, wantErr: model.ErrConflict},
		{name: "store internal error", email: "ivan@looqa.dev", password: "secret123", mock: &mockUsers{createErr: model.ErrInternal}, wantErr: model.ErrInternal},
		{name: "ok", email: "  IVAN@Looqa.DEV  ", password: "secret123", mock: &mockUsers{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _ := newTestService(&mockRefresh{}, tt.mock, time.Hour)

			user, err := svc.Register(context.Background(), model.User{FirstName: "Ivan", LastName: "Petrov"}, tt.email, tt.password)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("ожидали %v, получили: %v", tt.wantErr, err)
				}
				if tt.wantNoCreate && tt.mock.createCalls != 0 {
					t.Fatalf("пользователь не должен создаваться, а создан (%d вызовов)", tt.mock.createCalls)
				}
				return
			}
			if err != nil {
				t.Fatalf("не ожидали ошибку: %v", err)
			}

			if user.Id == uuid.Nil {
				t.Error("id пользователя не сгенерирован")
			}
			if tt.mock.createdUser != user {
				t.Errorf("в хранилище положили %+v вместо %+v", tt.mock.createdUser, user)
			}
			if tt.mock.createdCred.Email != "ivan@looqa.dev" {
				t.Errorf("email не нормализован: %q", tt.mock.createdCred.Email)
			}
			if tt.mock.createdCred.UserID != user.Id {
				t.Errorf("учётные данные привязаны к %s вместо %s", tt.mock.createdCred.UserID, user.Id)
			}
			if err := bcrypt.CompareHashAndPassword([]byte(tt.mock.createdCred.PasswordHash), []byte(tt.password)); err != nil {
				t.Errorf("в хранилище не bcrypt-хеш пароля: %v", err)
			}
		})
	}
}

func TestLogin(t *testing.T) {
	userID := uuid.New()

	t.Run("unknown email", func(t *testing.T) {
		refresh := &mockRefresh{}
		users := &mockUsers{credErr: model.ErrNotFound}
		svc, _ := newTestService(refresh, users, time.Hour)

		if _, err := svc.Login(context.Background(), "ivan@looqa.dev", "secret123"); !errors.Is(err, model.ErrUnauthorized) {
			t.Fatalf("ожидали ErrUnauthorized, получили: %v", err)
		}
		if len(refresh.added) != 0 {
			t.Fatalf("refresh не должен выдаваться, а выдан (%d вызовов)", len(refresh.added))
		}
	})

	t.Run("wrong password", func(t *testing.T) {
		refresh := &mockRefresh{}
		users := &mockUsers{cred: model.Credential{UserID: userID, Email: "ivan@looqa.dev", PasswordHash: hash(t, "other-pass")}}
		svc, _ := newTestService(refresh, users, time.Hour)

		if _, err := svc.Login(context.Background(), "IVAN@looqa.dev ", "secret123"); !errors.Is(err, model.ErrUnauthorized) {
			t.Fatalf("ожидали ErrUnauthorized, получили: %v", err)
		}
		if users.askedEmail != "ivan@looqa.dev" {
			t.Errorf("email не нормализован: %q", users.askedEmail)
		}
		if len(refresh.added) != 0 {
			t.Fatalf("refresh не должен выдаваться, а выдан (%d вызовов)", len(refresh.added))
		}
	})

	t.Run("internal error", func(t *testing.T) {
		users := &mockUsers{credErr: model.ErrInternal}
		svc, _ := newTestService(&mockRefresh{}, users, time.Hour)

		if _, err := svc.Login(context.Background(), "ivan@looqa.dev", "secret123"); !errors.Is(err, model.ErrInternal) {
			t.Fatalf("ожидали ErrInternal, получили: %v", err)
		}
	})

	t.Run("ok", func(t *testing.T) {
		before := time.Now()
		refresh := &mockRefresh{}
		users := &mockUsers{cred: model.Credential{UserID: userID, Email: "ivan@looqa.dev", PasswordHash: hash(t, "secret123")}}
		svc, issuer := newTestService(refresh, users, time.Hour)

		pair, err := svc.Login(context.Background(), "ivan@looqa.dev", "secret123")
		if err != nil {
			t.Fatalf("не ожидали ошибку: %v", err)
		}

		// access — валидный JWT нужного пользователя
		claims, err := issuer.Verify(pair.AccessToken)
		if err != nil {
			t.Fatalf("access-токен не проверяется: %v", err)
		}
		if claims.UserID != userID {
			t.Errorf("токен выдан пользователю %s вместо %s", claims.UserID, userID)
		}

		// refresh — сохранён хешем, не сам токен, и живёт refreshTTL
		if len(refresh.added) != 1 {
			t.Fatalf("ожидали один AddRefresh, получили %d", len(refresh.added))
		}
		stored := refresh.added[0]
		if stored.Hash == pair.RefreshToken {
			t.Error("в хранилище лежит сырой refresh-токен вместо хеша")
		}
		if stored.UserID != userID {
			t.Errorf("refresh привязан к %s вместо %s", stored.UserID, userID)
		}
		if stored.FamilyID != claims.SessionID {
			t.Errorf("refresh-семейство %s не совпадает с sid токена %s", stored.FamilyID, claims.SessionID)
		}
		if d := stored.ExpiresAt.Sub(before); d <= 0 || d > time.Hour+time.Second {
			t.Errorf("срок жизни refresh %v вне интервала (0, ttl]", d)
		}
		if pair.TokenType != "Bearer" || pair.ExpiresIn <= 0 {
			t.Errorf("некорректная пара: %+v", pair)
		}
	})
}

func TestRefresh(t *testing.T) {
	familyID := uuid.New()
	userID := uuid.New()
	rotated := model.RefreshToken{Hash: "abc", FamilyID: familyID, UserID: userID, ExpiresAt: time.Now().Add(time.Hour)}

	t.Run("ok", func(t *testing.T) {
		refresh := &mockRefresh{rotated: rotated}
		svc, issuer := newTestService(refresh, &mockUsers{}, time.Hour)

		pair, err := svc.Refresh(context.Background(), "raw-token")
		if err != nil {
			t.Fatalf("не ожидали ошибку: %v", err)
		}

		// новая пара — та же сессия и тот же пользователь
		claims, err := issuer.Verify(pair.AccessToken)
		if err != nil {
			t.Fatalf("access-токен не проверяется: %v", err)
		}
		if claims.UserID != userID || claims.SessionID != familyID {
			t.Errorf("после ротации claims %+v, ожидали пользователя %s и сессию %s", claims, userID, familyID)
		}
		if len(refresh.added) != 1 || refresh.added[0].FamilyID != familyID {
			t.Errorf("новый refresh должен лечь в то же семейство: %+v", refresh.added)
		}
		if refresh.gotRotate == "" || refresh.gotRotate == "raw-token" {
			t.Errorf("ротировать должны по хешу, а не по сырому токену: %q", refresh.gotRotate)
		}
	})

	t.Run("unknown token", func(t *testing.T) {
		refresh := &mockRefresh{rotateErr: model.ErrNotFound}
		svc, _ := newTestService(refresh, &mockUsers{}, time.Hour)

		if _, err := svc.Refresh(context.Background(), "garbage"); !errors.Is(err, model.ErrUnauthorized) {
			t.Fatalf("ожидали ErrUnauthorized, получили: %v", err)
		}
		if refresh.revokeCalls != 0 {
			t.Errorf("неизвестный токен не должен отзываться, а отозван (%d вызовов)", refresh.revokeCalls)
		}
	})

	t.Run("reuse revokes family", func(t *testing.T) {
		refresh := &mockRefresh{rotated: rotated, rotateErr: model.ErrConflict}
		svc, _ := newTestService(refresh, &mockUsers{}, time.Hour)

		if _, err := svc.Refresh(context.Background(), "stolen-token"); !errors.Is(err, model.ErrUnauthorized) {
			t.Fatalf("ожидали ErrUnauthorized, получили: %v", err)
		}
		if refresh.gotRevoke != familyID {
			t.Errorf("после переиспользования должны отозвать семейство %s, отозвали %s", familyID, refresh.gotRevoke)
		}
		if len(refresh.added) != 0 {
			t.Errorf("новая пара не должна выдаваться при переиспользовании: %+v", refresh.added)
		}
	})

	t.Run("internal error", func(t *testing.T) {
		refresh := &mockRefresh{rotateErr: model.ErrInternal}
		svc, _ := newTestService(refresh, &mockUsers{}, time.Hour)

		if _, err := svc.Refresh(context.Background(), "raw-token"); !errors.Is(err, model.ErrInternal) {
			t.Fatalf("ожидали ErrInternal, получили: %v", err)
		}
	})
}

func TestLogout(t *testing.T) {
	sessionID := uuid.New()

	tests := []struct {
		name      string
		revokeErr error
		wantErr   error
	}{
		{name: "ok"},
		{name: "internal", revokeErr: model.ErrInternal, wantErr: model.ErrInternal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			refresh := &mockRefresh{revokeErr: tt.revokeErr}
			svc, _ := newTestService(refresh, &mockUsers{}, time.Hour)

			err := svc.Logout(context.Background(), sessionID)
			if refresh.gotRevoke != sessionID {
				t.Errorf("отзывают семейство %s вместо %s", refresh.gotRevoke, sessionID)
			}
			if tt.wantErr == nil && err != nil {
				t.Fatalf("не ожидали ошибку: %v", err)
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Fatalf("ожидали %v, получили: %v", tt.wantErr, err)
			}
		})
	}
}
