package service

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io"
	"strings"
	"testing"
	"time"

	"pack_mate/internal/config"
	"pack_mate/internal/domain"
	"pack_mate/internal/dto/request"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"golang.org/x/crypto/bcrypt"
)

type fakeUserRepository struct {
	created   *domain.User
	got       *domain.User
	updated   *domain.User
	err       error
	updateErr error
}

func (r *fakeUserRepository) Create(_ context.Context, user *domain.User) error {
	if r.err != nil {
		return r.err
	}
	r.created = user
	r.got = user
	return nil
}

func (r *fakeUserRepository) GetByID(_ context.Context, _ bson.ObjectID) (*domain.User, error) {
	if r.err != nil {
		return nil, r.err
	}
	if r.got == nil {
		return nil, mongo.ErrNoDocuments
	}
	return r.got, nil
}

func (r *fakeUserRepository) Update(_ context.Context, user *domain.User) error {
	if r.updateErr != nil {
		return r.updateErr
	}
	if r.err != nil {
		return r.err
	}
	r.updated = user
	r.got = user
	return nil
}

type fakeAuthIdentityRepository struct {
	created        *domain.AuthIdentity
	got            *domain.AuthIdentity
	disabledUserID bson.ObjectID
	err            error
}

func (r *fakeAuthIdentityRepository) Create(_ context.Context, identity *domain.AuthIdentity) error {
	if r.err != nil {
		return r.err
	}
	r.created = identity
	r.got = identity
	return nil
}

func (r *fakeAuthIdentityRepository) GetByProviderAndIdentifier(_ context.Context, _ domain.AuthProvider, _ string) (*domain.AuthIdentity, error) {
	if r.err != nil {
		return nil, r.err
	}
	if r.got == nil {
		return nil, mongo.ErrNoDocuments
	}
	return r.got, nil
}

func (r *fakeAuthIdentityRepository) DisableByUserID(_ context.Context, userID bson.ObjectID, _ time.Time) error {
	if r.err != nil {
		return r.err
	}
	r.disabledUserID = userID
	return nil
}

type fakePhoneCodeRepository struct {
	created     *domain.PhoneVerificationCode
	got         *domain.PhoneVerificationCode
	consumed    bson.ObjectID
	attempted   bson.ObjectID
	recentCount int64
	err         error
}

func (r *fakePhoneCodeRepository) Create(_ context.Context, code *domain.PhoneVerificationCode) error {
	if r.err != nil {
		return r.err
	}
	r.created = code
	r.got = code
	return nil
}

func (r *fakePhoneCodeRepository) GetLatestActive(_ context.Context, _ string, _ domain.PhoneVerificationPurpose, _ time.Time) (*domain.PhoneVerificationCode, error) {
	if r.err != nil {
		return nil, r.err
	}
	if r.got == nil {
		return nil, mongo.ErrNoDocuments
	}
	return r.got, nil
}

func (r *fakePhoneCodeRepository) MarkConsumed(_ context.Context, codeID bson.ObjectID, _ time.Time) error {
	if r.err != nil {
		return r.err
	}
	r.consumed = codeID
	return nil
}

func (r *fakePhoneCodeRepository) IncrementAttempt(_ context.Context, codeID bson.ObjectID) error {
	if r.err != nil {
		return r.err
	}
	r.attempted = codeID
	return nil
}

func (r *fakePhoneCodeRepository) CountRecent(_ context.Context, _ string, _ time.Time) (int64, error) {
	if r.err != nil {
		return 0, r.err
	}
	return r.recentCount, nil
}

type fakeRefreshTokenRepository struct {
	created       *domain.RefreshToken
	got           *domain.RefreshToken
	revoked       string
	revokedUserID bson.ObjectID
	err           error
}

func (r *fakeRefreshTokenRepository) Create(_ context.Context, token *domain.RefreshToken) error {
	if r.err != nil {
		return r.err
	}
	r.created = token
	r.got = token
	return nil
}

func (r *fakeRefreshTokenRepository) GetByTokenHash(_ context.Context, _ string) (*domain.RefreshToken, error) {
	if r.err != nil {
		return nil, r.err
	}
	if r.got == nil {
		return nil, mongo.ErrNoDocuments
	}
	return r.got, nil
}

func (r *fakeRefreshTokenRepository) Revoke(_ context.Context, tokenHash string, _ time.Time) error {
	if r.err != nil {
		return r.err
	}
	r.revoked = tokenHash
	return nil
}

func (r *fakeRefreshTokenRepository) RevokeByUserID(_ context.Context, userID bson.ObjectID, _ time.Time) error {
	if r.err != nil {
		return r.err
	}
	r.revokedUserID = userID
	return nil
}

type fakeUserPasswordCredentialRepository struct {
	created                *domain.UserPasswordCredential
	got                    *domain.UserPasswordCredential
	failedAttemptCount     int
	lockedUntil            *time.Time
	recordFailureUpdatedAt time.Time
	recordSuccessUsedAt    time.Time
	updatedPasswordHash    string
	updatedPasswordAlgo    string
	updatePasswordAt       time.Time
	err                    error
	recordErr              error
	updateErr              error
}

func (r *fakeUserPasswordCredentialRepository) Create(_ context.Context, credential *domain.UserPasswordCredential) error {
	if r.err != nil {
		return r.err
	}
	r.created = credential
	r.got = credential
	return nil
}

func (r *fakeUserPasswordCredentialRepository) GetByUserID(_ context.Context, _ bson.ObjectID) (*domain.UserPasswordCredential, error) {
	if r.err != nil {
		return nil, r.err
	}
	if r.got == nil {
		return nil, mongo.ErrNoDocuments
	}
	return r.got, nil
}

func (r *fakeUserPasswordCredentialRepository) RecordFailure(_ context.Context, _ bson.ObjectID, failedAttemptCount int, lockedUntil *time.Time, updatedAt time.Time) error {
	if r.recordErr != nil {
		return r.recordErr
	}
	r.failedAttemptCount = failedAttemptCount
	r.lockedUntil = lockedUntil
	r.recordFailureUpdatedAt = updatedAt
	if r.got != nil {
		r.got.FailedAttemptCount = failedAttemptCount
		r.got.LockedUntil = lockedUntil
		r.got.UpdatedAt = updatedAt
	}
	return nil
}

func (r *fakeUserPasswordCredentialRepository) RecordSuccess(_ context.Context, _ bson.ObjectID, usedAt time.Time) error {
	if r.recordErr != nil {
		return r.recordErr
	}
	r.recordSuccessUsedAt = usedAt
	if r.got != nil {
		r.got.FailedAttemptCount = 0
		r.got.LockedUntil = nil
		r.got.LastUsedAt = &usedAt
		r.got.UpdatedAt = usedAt
	}
	return nil
}

func (r *fakeUserPasswordCredentialRepository) UpdatePassword(_ context.Context, _ bson.ObjectID, passwordHash string, passwordAlgo string, updatedAt time.Time) error {
	if r.updateErr != nil {
		return r.updateErr
	}
	r.updatedPasswordHash = passwordHash
	r.updatedPasswordAlgo = passwordAlgo
	r.updatePasswordAt = updatedAt
	if r.got != nil {
		r.got.PasswordHash = passwordHash
		r.got.PasswordAlgo = passwordAlgo
		r.got.FailedAttemptCount = 0
		r.got.LockedUntil = nil
		r.got.UpdatedAt = updatedAt
	}
	return nil
}

type recordingSMSService struct {
	phone string
	code  string
	err   error
}

type fakeAuthUploadService struct {
	objectKey  string
	objectKeys []string
	err        error
}

func (s *fakeAuthUploadService) Upload(_ context.Context, objectKey string, _ string, _ io.Reader) error {
	if s.err != nil {
		return s.err
	}
	s.objectKey = objectKey
	s.objectKeys = append(s.objectKeys, objectKey)
	return nil
}

func testImageReader() *bytes.Reader {
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	var body bytes.Buffer
	if err := png.Encode(&body, img); err != nil {
		panic(err)
	}
	return bytes.NewReader(body.Bytes())
}

func (s *recordingSMSService) SendLoginCode(_ context.Context, phone string, code string) error {
	if s.err != nil {
		return s.err
	}
	s.phone = phone
	s.code = code
	return nil
}

func validAuthConfig() config.AuthConfig {
	return config.AuthConfig{
		AccessTokenSecret:      "test-secret",
		AccessTokenTTLSeconds:  7200,
		RefreshTokenTTLSeconds: 2592000,
		PhoneCode: config.PhoneCodeConfig{
			TTLSeconds:            300,
			MaxAttempts:           5,
			ResendIntervalSeconds: 60,
			DailySendLimit:        10,
			UseDevFixedCode:       true,
			DevFixedCode:          "123456",
		},
		Password: config.PasswordConfig{
			BcryptCost:          bcrypt.MinCost,
			MaxFailedAttempts:   5,
			LockDurationSeconds: 900,
		},
	}
}

func newTestAuthService(users *fakeUserRepository, identities *fakeAuthIdentityRepository, codes *fakePhoneCodeRepository, refreshTokens *fakeRefreshTokenRepository, sms *recordingSMSService) AuthService {
	cfg := validAuthConfig()
	return NewAuthService(
		users,
		identities,
		codes,
		refreshTokens,
		&fakeUserPasswordCredentialRepository{},
		&fakeAuthUploadService{},
		sms,
		NewTokenService(cfg.AccessTokenSecret, time.Duration(cfg.AccessTokenTTLSeconds)*time.Second),
		cfg,
	)
}

func TestSendPhoneCodeStoresAndSendsCode(t *testing.T) {
	t.Parallel()

	codes := &fakePhoneCodeRepository{}
	sms := &recordingSMSService{}
	svc := newTestAuthService(&fakeUserRepository{}, &fakeAuthIdentityRepository{}, codes, &fakeRefreshTokenRepository{}, sms)

	if err := svc.SendPhoneCode(context.Background(), request.SendPhoneCodeInput{Phone: "13800138000"}); err != nil {
		t.Fatalf("SendPhoneCode returned error: %v", err)
	}
	if codes.created == nil || codes.created.Phone != "+8613800138000" {
		t.Fatalf("expected normalized code to be stored, got %+v", codes.created)
	}
	if sms.phone != "+8613800138000" || sms.code != "123456" {
		t.Fatalf("unexpected sms send: phone=%s code=%s", sms.phone, sms.code)
	}
}

func TestSendPhoneCodeRejectsTooFrequentRequests(t *testing.T) {
	t.Parallel()

	codes := &fakePhoneCodeRepository{recentCount: int64(validAuthConfig().PhoneCode.DailySendLimit)}
	svc := newTestAuthService(&fakeUserRepository{}, &fakeAuthIdentityRepository{}, codes, &fakeRefreshTokenRepository{}, &recordingSMSService{})

	err := svc.SendPhoneCode(context.Background(), request.SendPhoneCodeInput{Phone: "13800138000"})
	if err == nil || !strings.Contains(err.Error(), "phone code send too frequently") {
		t.Fatalf("expected too frequent error, got %v", err)
	}
}

func TestLoginWithPhoneCreatesUserOnFirstLogin(t *testing.T) {
	t.Parallel()

	codeID := bson.NewObjectID()
	codes := &fakePhoneCodeRepository{
		got: &domain.PhoneVerificationCode{
			ID:        codeID,
			Phone:     "+8613800138000",
			CodeHash:  hashPhoneCode("+8613800138000", "123456"),
			Purpose:   domain.PhoneVerificationPurposeLogin,
			ExpiresAt: time.Now().Add(time.Minute),
			CreatedAt: time.Now().Add(-time.Minute),
		},
	}
	users := &fakeUserRepository{}
	identities := &fakeAuthIdentityRepository{}
	refreshTokens := &fakeRefreshTokenRepository{}
	svc := newTestAuthService(users, identities, codes, refreshTokens, &recordingSMSService{})

	result, err := svc.LoginWithPhone(context.Background(), request.PhoneLoginInput{
		Phone: "13800138000",
		Code:  "123456",
	})
	if err != nil {
		t.Fatalf("LoginWithPhone returned error: %v", err)
	}
	if users.created == nil {
		t.Fatalf("expected user to be created")
	}
	if users.created.Profile.Username != "user8613800138000" {
		t.Fatalf("expected default username user8613800138000, got %+v", users.created.Profile)
	}
	if users.created.Profile.AvatarObjectKey != defaultUserAvatarObjectKey {
		t.Fatalf("expected default avatar object key %s, got %s", defaultUserAvatarObjectKey, users.created.Profile.AvatarObjectKey)
	}
	if users.created.Profile.Gender != "" || users.created.Profile.Birthday != nil {
		t.Fatalf("expected empty gender and birthday, got %+v", users.created.Profile)
	}
	if identities.created == nil || identities.created.UserID != users.created.ID {
		t.Fatalf("expected identity to be created for user, got %+v", identities.created)
	}
	if codes.attempted != codeID || codes.consumed != codeID {
		t.Fatalf("expected code attempt and consume, got attempted=%s consumed=%s", codes.attempted.Hex(), codes.consumed.Hex())
	}
	if result.AccessToken == "" || result.RefreshToken == "" || refreshTokens.created == nil {
		t.Fatalf("expected tokens to be created, got result=%+v stored=%+v", result, refreshTokens.created)
	}
}

func TestLoginWithPhoneReusesExistingUser(t *testing.T) {
	t.Parallel()

	userID := bson.NewObjectID()
	user := &domain.User{ID: userID, Profile: domain.UserProfile{Username: "用户8000", AvatarObjectKey: defaultUserAvatarObjectKey}, Status: domain.UserStatusCreated}
	codes := &fakePhoneCodeRepository{
		got: &domain.PhoneVerificationCode{
			ID:       bson.NewObjectID(),
			Phone:    "+8613800138000",
			CodeHash: hashPhoneCode("+8613800138000", "123456"),
		},
	}
	users := &fakeUserRepository{got: user}
	identities := &fakeAuthIdentityRepository{got: &domain.AuthIdentity{UserID: userID, Provider: domain.AuthProviderPhone, Identifier: "+8613800138000"}}
	svc := newTestAuthService(users, identities, codes, &fakeRefreshTokenRepository{}, &recordingSMSService{})

	result, err := svc.LoginWithPhone(context.Background(), request.PhoneLoginInput{Phone: "13800138000", Code: "123456"})
	if err != nil {
		t.Fatalf("LoginWithPhone returned error: %v", err)
	}
	if users.created != nil || identities.created != nil {
		t.Fatalf("expected existing user to be reused")
	}
	if result.User.ID != userID {
		t.Fatalf("expected user %s, got %s", userID.Hex(), result.User.ID.Hex())
	}
}

func TestLoginWithPhonePasswordReturnsTokens(t *testing.T) {
	t.Parallel()

	userID := bson.NewObjectID()
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("Trip2026Pass"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("GenerateFromPassword returned error: %v", err)
	}
	users := &fakeUserRepository{got: &domain.User{ID: userID, Profile: domain.UserProfile{Username: "用户8000", AvatarObjectKey: defaultUserAvatarObjectKey}, Status: domain.UserStatusCreated}}
	identities := &fakeAuthIdentityRepository{got: &domain.AuthIdentity{UserID: userID, Provider: domain.AuthProviderPhone, Identifier: "+8613800138000", Status: domain.AuthIdentityStatusActive}}
	passwords := &fakeUserPasswordCredentialRepository{got: &domain.UserPasswordCredential{UserID: userID, PasswordHash: string(passwordHash), PasswordAlgo: passwordAlgorithmBcrypt, FailedAttemptCount: 2}}
	refreshTokens := &fakeRefreshTokenRepository{}
	cfg := validAuthConfig()
	svc := NewAuthService(users, identities, &fakePhoneCodeRepository{}, refreshTokens, passwords, &fakeAuthUploadService{}, &recordingSMSService{}, NewTokenService(cfg.AccessTokenSecret, time.Duration(cfg.AccessTokenTTLSeconds)*time.Second), cfg)

	result, err := svc.LoginWithPhonePassword(context.Background(), request.PhonePasswordLoginInput{
		Phone:    "13800138000",
		Password: "Trip2026Pass",
	})
	if err != nil {
		t.Fatalf("LoginWithPhonePassword returned error: %v", err)
	}
	if result.AccessToken == "" || result.RefreshToken == "" || refreshTokens.created == nil {
		t.Fatalf("expected tokens to be created, got result=%+v stored=%+v", result, refreshTokens.created)
	}
	if passwords.recordSuccessUsedAt.IsZero() || passwords.got.FailedAttemptCount != 0 || passwords.got.LockedUntil != nil || passwords.got.LastUsedAt == nil {
		t.Fatalf("expected password success state to be recorded, got %+v", passwords.got)
	}
}

func TestLoginWithPhonePasswordRecordsFailure(t *testing.T) {
	t.Parallel()

	userID := bson.NewObjectID()
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("Trip2026Pass"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("GenerateFromPassword returned error: %v", err)
	}
	users := &fakeUserRepository{got: &domain.User{ID: userID, Profile: domain.UserProfile{Username: "用户8000", AvatarObjectKey: defaultUserAvatarObjectKey}, Status: domain.UserStatusCreated}}
	identities := &fakeAuthIdentityRepository{got: &domain.AuthIdentity{UserID: userID, Provider: domain.AuthProviderPhone, Identifier: "+8613800138000", Status: domain.AuthIdentityStatusActive}}
	passwords := &fakeUserPasswordCredentialRepository{got: &domain.UserPasswordCredential{UserID: userID, PasswordHash: string(passwordHash), PasswordAlgo: passwordAlgorithmBcrypt, FailedAttemptCount: 1}}
	cfg := validAuthConfig()
	svc := NewAuthService(users, identities, &fakePhoneCodeRepository{}, &fakeRefreshTokenRepository{}, passwords, &fakeAuthUploadService{}, &recordingSMSService{}, NewTokenService(cfg.AccessTokenSecret, time.Duration(cfg.AccessTokenTTLSeconds)*time.Second), cfg)

	_, err = svc.LoginWithPhonePassword(context.Background(), request.PhonePasswordLoginInput{
		Phone:    "13800138000",
		Password: "Wrong2026Pass",
	})
	if err == nil || !strings.Contains(err.Error(), "phone or password is invalid") {
		t.Fatalf("expected invalid password error, got %v", err)
	}
	if passwords.failedAttemptCount != 2 || passwords.lockedUntil != nil || passwords.recordFailureUpdatedAt.IsZero() {
		t.Fatalf("expected password failure state to be recorded, got count=%d locked=%v updated=%v", passwords.failedAttemptCount, passwords.lockedUntil, passwords.recordFailureUpdatedAt)
	}
}

func TestLoginWithPhonePasswordLocksAfterMaxFailures(t *testing.T) {
	t.Parallel()

	userID := bson.NewObjectID()
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("Trip2026Pass"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("GenerateFromPassword returned error: %v", err)
	}
	users := &fakeUserRepository{got: &domain.User{ID: userID, Profile: domain.UserProfile{Username: "用户8000", AvatarObjectKey: defaultUserAvatarObjectKey}, Status: domain.UserStatusCreated}}
	identities := &fakeAuthIdentityRepository{got: &domain.AuthIdentity{UserID: userID, Provider: domain.AuthProviderPhone, Identifier: "+8613800138000", Status: domain.AuthIdentityStatusActive}}
	passwords := &fakeUserPasswordCredentialRepository{got: &domain.UserPasswordCredential{UserID: userID, PasswordHash: string(passwordHash), PasswordAlgo: passwordAlgorithmBcrypt, FailedAttemptCount: 4}}
	cfg := validAuthConfig()
	svc := NewAuthService(users, identities, &fakePhoneCodeRepository{}, &fakeRefreshTokenRepository{}, passwords, &fakeAuthUploadService{}, &recordingSMSService{}, NewTokenService(cfg.AccessTokenSecret, time.Duration(cfg.AccessTokenTTLSeconds)*time.Second), cfg)

	_, err = svc.LoginWithPhonePassword(context.Background(), request.PhonePasswordLoginInput{
		Phone:    "13800138000",
		Password: "Wrong2026Pass",
	})
	if err == nil || !strings.Contains(err.Error(), "phone or password is invalid") {
		t.Fatalf("expected invalid password error, got %v", err)
	}
	if passwords.failedAttemptCount != 5 || passwords.lockedUntil == nil {
		t.Fatalf("expected password to be locked, got count=%d locked=%v", passwords.failedAttemptCount, passwords.lockedUntil)
	}
}

func TestLoginWithPhonePasswordResetsFailuresAfterLockExpires(t *testing.T) {
	t.Parallel()

	userID := bson.NewObjectID()
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("Trip2026Pass"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("GenerateFromPassword returned error: %v", err)
	}
	expiredLockedUntil := time.Now().UTC().Add(-time.Minute)
	users := &fakeUserRepository{got: &domain.User{ID: userID, Profile: domain.UserProfile{Username: "用户8000", AvatarObjectKey: defaultUserAvatarObjectKey}, Status: domain.UserStatusCreated}}
	identities := &fakeAuthIdentityRepository{got: &domain.AuthIdentity{UserID: userID, Provider: domain.AuthProviderPhone, Identifier: "+8613800138000", Status: domain.AuthIdentityStatusActive}}
	passwords := &fakeUserPasswordCredentialRepository{got: &domain.UserPasswordCredential{UserID: userID, PasswordHash: string(passwordHash), PasswordAlgo: passwordAlgorithmBcrypt, FailedAttemptCount: 5, LockedUntil: &expiredLockedUntil}}
	cfg := validAuthConfig()
	svc := NewAuthService(users, identities, &fakePhoneCodeRepository{}, &fakeRefreshTokenRepository{}, passwords, &fakeAuthUploadService{}, &recordingSMSService{}, NewTokenService(cfg.AccessTokenSecret, time.Duration(cfg.AccessTokenTTLSeconds)*time.Second), cfg)

	_, err = svc.LoginWithPhonePassword(context.Background(), request.PhonePasswordLoginInput{
		Phone:    "13800138000",
		Password: "Wrong2026Pass",
	})
	if err == nil || !strings.Contains(err.Error(), "phone or password is invalid") {
		t.Fatalf("expected invalid password error, got %v", err)
	}
	if passwords.failedAttemptCount != 1 || passwords.lockedUntil != nil || passwords.got.LockedUntil != nil {
		t.Fatalf("expected expired lock to reset failure state, got count=%d locked=%v credential_locked=%v", passwords.failedAttemptCount, passwords.lockedUntil, passwords.got.LockedUntil)
	}
}

func TestLoginWithPhonePasswordRejectsLockedCredential(t *testing.T) {
	t.Parallel()

	userID := bson.NewObjectID()
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("Trip2026Pass"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("GenerateFromPassword returned error: %v", err)
	}
	lockedUntil := time.Now().UTC().Add(15 * time.Minute)
	users := &fakeUserRepository{got: &domain.User{ID: userID, Profile: domain.UserProfile{Username: "用户8000", AvatarObjectKey: defaultUserAvatarObjectKey}, Status: domain.UserStatusCreated}}
	identities := &fakeAuthIdentityRepository{got: &domain.AuthIdentity{UserID: userID, Provider: domain.AuthProviderPhone, Identifier: "+8613800138000", Status: domain.AuthIdentityStatusActive}}
	passwords := &fakeUserPasswordCredentialRepository{got: &domain.UserPasswordCredential{UserID: userID, PasswordHash: string(passwordHash), PasswordAlgo: passwordAlgorithmBcrypt, FailedAttemptCount: 5, LockedUntil: &lockedUntil}}
	cfg := validAuthConfig()
	svc := NewAuthService(users, identities, &fakePhoneCodeRepository{}, &fakeRefreshTokenRepository{}, passwords, &fakeAuthUploadService{}, &recordingSMSService{}, NewTokenService(cfg.AccessTokenSecret, time.Duration(cfg.AccessTokenTTLSeconds)*time.Second), cfg)

	_, err = svc.LoginWithPhonePassword(context.Background(), request.PhonePasswordLoginInput{
		Phone:    "13800138000",
		Password: "Trip2026Pass",
	})
	if err == nil || !strings.Contains(err.Error(), "password is locked") {
		t.Fatalf("expected locked password error, got %v", err)
	}
	if !passwords.recordSuccessUsedAt.IsZero() || passwords.failedAttemptCount != 0 {
		t.Fatalf("did not expect password state update, got success=%v failure=%d", passwords.recordSuccessUsedAt, passwords.failedAttemptCount)
	}
}

func TestLoginWithPhonePasswordRejectsMissingCredential(t *testing.T) {
	t.Parallel()

	userID := bson.NewObjectID()
	users := &fakeUserRepository{got: &domain.User{ID: userID, Profile: domain.UserProfile{Username: "用户8000", AvatarObjectKey: defaultUserAvatarObjectKey}, Status: domain.UserStatusCreated}}
	identities := &fakeAuthIdentityRepository{got: &domain.AuthIdentity{UserID: userID, Provider: domain.AuthProviderPhone, Identifier: "+8613800138000", Status: domain.AuthIdentityStatusActive}}
	cfg := validAuthConfig()
	svc := NewAuthService(users, identities, &fakePhoneCodeRepository{}, &fakeRefreshTokenRepository{}, &fakeUserPasswordCredentialRepository{}, &fakeAuthUploadService{}, &recordingSMSService{}, NewTokenService(cfg.AccessTokenSecret, time.Duration(cfg.AccessTokenTTLSeconds)*time.Second), cfg)

	_, err := svc.LoginWithPhonePassword(context.Background(), request.PhonePasswordLoginInput{
		Phone:    "13800138000",
		Password: "Trip2026Pass",
	})
	if err == nil || !strings.Contains(err.Error(), "phone or password is invalid") {
		t.Fatalf("expected invalid password login error, got %v", err)
	}
}

func TestSetupPasswordCreatesCredential(t *testing.T) {
	t.Parallel()

	userID := bson.NewObjectID()
	users := &fakeUserRepository{got: &domain.User{ID: userID, Profile: domain.UserProfile{Username: "用户8000", AvatarObjectKey: defaultUserAvatarObjectKey}, Status: domain.UserStatusCreated}}
	identities := &fakeAuthIdentityRepository{got: &domain.AuthIdentity{UserID: userID, Provider: domain.AuthProviderPhone, Identifier: "+8613800138000", Status: domain.AuthIdentityStatusActive}}
	passwords := &fakeUserPasswordCredentialRepository{}
	cfg := validAuthConfig()
	svc := NewAuthService(users, identities, &fakePhoneCodeRepository{}, &fakeRefreshTokenRepository{}, passwords, &fakeAuthUploadService{}, &recordingSMSService{}, NewTokenService(cfg.AccessTokenSecret, time.Duration(cfg.AccessTokenTTLSeconds)*time.Second), cfg)

	err := svc.SetupPassword(context.Background(), request.SetupPasswordInput{
		UserID:   userID.Hex(),
		Phone:    "13800138000",
		Password: "Trip2026Pass",
	})
	if err != nil {
		t.Fatalf("SetupPassword returned error: %v", err)
	}
	if passwords.created == nil || passwords.created.UserID != userID {
		t.Fatalf("expected password credential to be created, got %+v", passwords.created)
	}
	if passwords.created.PasswordHash == "Trip2026Pass" || passwords.created.PasswordAlgo != passwordAlgorithmBcrypt {
		t.Fatalf("unexpected password credential: %+v", passwords.created)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(passwords.created.PasswordHash), []byte("Trip2026Pass")); err != nil {
		t.Fatalf("expected password hash to match: %v", err)
	}
	if passwords.created.FailedAttemptCount != 0 || passwords.created.LockedUntil != nil || passwords.created.LastUsedAt != nil {
		t.Fatalf("unexpected credential state: %+v", passwords.created)
	}
}

func TestSetupPasswordAllowsConfiguredSpecialCharacters(t *testing.T) {
	t.Parallel()

	userID := bson.NewObjectID()
	users := &fakeUserRepository{got: &domain.User{ID: userID, Profile: domain.UserProfile{Username: "用户8000", AvatarObjectKey: defaultUserAvatarObjectKey}, Status: domain.UserStatusCreated}}
	identities := &fakeAuthIdentityRepository{got: &domain.AuthIdentity{UserID: userID, Provider: domain.AuthProviderPhone, Identifier: "+8613800138000", Status: domain.AuthIdentityStatusActive}}
	passwords := &fakeUserPasswordCredentialRepository{}
	cfg := validAuthConfig()
	svc := NewAuthService(users, identities, &fakePhoneCodeRepository{}, &fakeRefreshTokenRepository{}, passwords, &fakeAuthUploadService{}, &recordingSMSService{}, NewTokenService(cfg.AccessTokenSecret, time.Duration(cfg.AccessTokenTTLSeconds)*time.Second), cfg)

	err := svc.SetupPassword(context.Background(), request.SetupPasswordInput{
		UserID:   userID.Hex(),
		Phone:    "13800138000",
		Password: "TripPass!",
	})
	if err != nil {
		t.Fatalf("SetupPassword returned error: %v", err)
	}
	if passwords.created == nil {
		t.Fatalf("expected password credential to be created")
	}
}

func TestSetupPasswordRejectsExistingCredential(t *testing.T) {
	t.Parallel()

	userID := bson.NewObjectID()
	users := &fakeUserRepository{got: &domain.User{ID: userID, Profile: domain.UserProfile{Username: "用户8000", AvatarObjectKey: defaultUserAvatarObjectKey}, Status: domain.UserStatusCreated}}
	identities := &fakeAuthIdentityRepository{got: &domain.AuthIdentity{UserID: userID, Provider: domain.AuthProviderPhone, Identifier: "+8613800138000", Status: domain.AuthIdentityStatusActive}}
	passwords := &fakeUserPasswordCredentialRepository{got: &domain.UserPasswordCredential{UserID: userID, PasswordHash: "hash"}}
	cfg := validAuthConfig()
	svc := NewAuthService(users, identities, &fakePhoneCodeRepository{}, &fakeRefreshTokenRepository{}, passwords, &fakeAuthUploadService{}, &recordingSMSService{}, NewTokenService(cfg.AccessTokenSecret, time.Duration(cfg.AccessTokenTTLSeconds)*time.Second), cfg)

	err := svc.SetupPassword(context.Background(), request.SetupPasswordInput{
		UserID:   userID.Hex(),
		Phone:    "13800138000",
		Password: "Trip2026Pass",
	})
	if err == nil || !strings.Contains(err.Error(), "password already setup") {
		t.Fatalf("expected password already setup error, got %v", err)
	}
}

func TestSetupPasswordRejectsMismatchedPhone(t *testing.T) {
	t.Parallel()

	userID := bson.NewObjectID()
	otherUserID := bson.NewObjectID()
	users := &fakeUserRepository{got: &domain.User{ID: userID, Profile: domain.UserProfile{Username: "用户8000", AvatarObjectKey: defaultUserAvatarObjectKey}, Status: domain.UserStatusCreated}}
	identities := &fakeAuthIdentityRepository{got: &domain.AuthIdentity{UserID: otherUserID, Provider: domain.AuthProviderPhone, Identifier: "+8613800138000", Status: domain.AuthIdentityStatusActive}}
	cfg := validAuthConfig()
	svc := NewAuthService(users, identities, &fakePhoneCodeRepository{}, &fakeRefreshTokenRepository{}, &fakeUserPasswordCredentialRepository{}, &fakeAuthUploadService{}, &recordingSMSService{}, NewTokenService(cfg.AccessTokenSecret, time.Duration(cfg.AccessTokenTTLSeconds)*time.Second), cfg)

	err := svc.SetupPassword(context.Background(), request.SetupPasswordInput{
		UserID:   userID.Hex(),
		Phone:    "13800138000",
		Password: "Trip2026Pass",
	})
	if err == nil || !strings.Contains(err.Error(), "phone does not match current user") {
		t.Fatalf("expected mismatched phone error, got %v", err)
	}
}

func TestSetupPasswordRejectsWeakPassword(t *testing.T) {
	t.Parallel()

	userID := bson.NewObjectID()
	users := &fakeUserRepository{got: &domain.User{ID: userID, Profile: domain.UserProfile{Username: "用户8000", AvatarObjectKey: defaultUserAvatarObjectKey}, Status: domain.UserStatusCreated}}
	identities := &fakeAuthIdentityRepository{got: &domain.AuthIdentity{UserID: userID, Provider: domain.AuthProviderPhone, Identifier: "+8613800138000", Status: domain.AuthIdentityStatusActive}}
	cfg := validAuthConfig()
	svc := NewAuthService(users, identities, &fakePhoneCodeRepository{}, &fakeRefreshTokenRepository{}, &fakeUserPasswordCredentialRepository{}, &fakeAuthUploadService{}, &recordingSMSService{}, NewTokenService(cfg.AccessTokenSecret, time.Duration(cfg.AccessTokenTTLSeconds)*time.Second), cfg)

	err := svc.SetupPassword(context.Background(), request.SetupPasswordInput{
		UserID:   userID.Hex(),
		Phone:    "13800138000",
		Password: "12345678",
	})
	if err == nil || !strings.Contains(err.Error(), "password is too weak") {
		t.Fatalf("expected weak password error, got %v", err)
	}
}

func TestSetupPasswordRejectsLongPassword(t *testing.T) {
	t.Parallel()

	userID := bson.NewObjectID()
	users := &fakeUserRepository{got: &domain.User{ID: userID, Profile: domain.UserProfile{Username: "用户8000", AvatarObjectKey: defaultUserAvatarObjectKey}, Status: domain.UserStatusCreated}}
	identities := &fakeAuthIdentityRepository{got: &domain.AuthIdentity{UserID: userID, Provider: domain.AuthProviderPhone, Identifier: "+8613800138000", Status: domain.AuthIdentityStatusActive}}
	cfg := validAuthConfig()
	svc := NewAuthService(users, identities, &fakePhoneCodeRepository{}, &fakeRefreshTokenRepository{}, &fakeUserPasswordCredentialRepository{}, &fakeAuthUploadService{}, &recordingSMSService{}, NewTokenService(cfg.AccessTokenSecret, time.Duration(cfg.AccessTokenTTLSeconds)*time.Second), cfg)

	err := svc.SetupPassword(context.Background(), request.SetupPasswordInput{
		UserID:   userID.Hex(),
		Phone:    "13800138000",
		Password: "Trip2026PasswordLongTrip2026Password",
	})
	if err == nil || !strings.Contains(err.Error(), "password is too long") {
		t.Fatalf("expected long password error, got %v", err)
	}
}

func TestSetupPasswordRejectsInvalidPasswordCharacters(t *testing.T) {
	t.Parallel()

	userID := bson.NewObjectID()
	users := &fakeUserRepository{got: &domain.User{ID: userID, Profile: domain.UserProfile{Username: "用户8000", AvatarObjectKey: defaultUserAvatarObjectKey}, Status: domain.UserStatusCreated}}
	identities := &fakeAuthIdentityRepository{got: &domain.AuthIdentity{UserID: userID, Provider: domain.AuthProviderPhone, Identifier: "+8613800138000", Status: domain.AuthIdentityStatusActive}}
	cfg := validAuthConfig()
	svc := NewAuthService(users, identities, &fakePhoneCodeRepository{}, &fakeRefreshTokenRepository{}, &fakeUserPasswordCredentialRepository{}, &fakeAuthUploadService{}, &recordingSMSService{}, NewTokenService(cfg.AccessTokenSecret, time.Duration(cfg.AccessTokenTTLSeconds)*time.Second), cfg)

	err := svc.SetupPassword(context.Background(), request.SetupPasswordInput{
		UserID:   userID.Hex(),
		Phone:    "13800138000",
		Password: "Trip2026.",
	})
	if err == nil || !strings.Contains(err.Error(), "password has invalid characters") {
		t.Fatalf("expected invalid password characters error, got %v", err)
	}
}

func TestChangePasswordUpdatesCredentialAndRevokesRefreshTokens(t *testing.T) {
	t.Parallel()

	userID := bson.NewObjectID()
	oldPasswordHash, err := bcrypt.GenerateFromPassword([]byte("Trip2026Pass"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("GenerateFromPassword returned error: %v", err)
	}
	lastUsedAt := time.Now().UTC().Add(-time.Hour)
	lockedUntil := time.Now().UTC().Add(15 * time.Minute)
	users := &fakeUserRepository{got: &domain.User{ID: userID, Profile: domain.UserProfile{Username: "用户8000", AvatarObjectKey: defaultUserAvatarObjectKey}, Status: domain.UserStatusCreated}}
	passwords := &fakeUserPasswordCredentialRepository{got: &domain.UserPasswordCredential{
		UserID:             userID,
		PasswordHash:       string(oldPasswordHash),
		PasswordAlgo:       passwordAlgorithmBcrypt,
		FailedAttemptCount: 3,
		LockedUntil:        &lockedUntil,
		LastUsedAt:         &lastUsedAt,
	}}
	refreshTokens := &fakeRefreshTokenRepository{}
	cfg := validAuthConfig()
	svc := NewAuthService(users, &fakeAuthIdentityRepository{}, &fakePhoneCodeRepository{}, refreshTokens, passwords, &fakeAuthUploadService{}, &recordingSMSService{}, NewTokenService(cfg.AccessTokenSecret, time.Duration(cfg.AccessTokenTTLSeconds)*time.Second), cfg)

	err = svc.ChangePassword(context.Background(), request.ChangePasswordInput{
		UserID:      userID.Hex(),
		OldPassword: "Trip2026Pass",
		NewPassword: "NewTrip2027",
	})
	if err != nil {
		t.Fatalf("ChangePassword returned error: %v", err)
	}
	if passwords.updatedPasswordHash == "" || passwords.updatedPasswordHash == string(oldPasswordHash) || passwords.updatedPasswordAlgo != passwordAlgorithmBcrypt {
		t.Fatalf("expected password credential to be updated, got hash=%q algo=%q", passwords.updatedPasswordHash, passwords.updatedPasswordAlgo)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(passwords.updatedPasswordHash), []byte("NewTrip2027")); err != nil {
		t.Fatalf("expected updated password hash to match: %v", err)
	}
	if passwords.got.FailedAttemptCount != 0 || passwords.got.LockedUntil != nil || passwords.got.LastUsedAt == nil || !passwords.got.LastUsedAt.Equal(lastUsedAt) {
		t.Fatalf("unexpected password credential state: %+v", passwords.got)
	}
	if refreshTokens.revokedUserID != userID {
		t.Fatalf("expected refresh tokens to be revoked for %s, got %s", userID.Hex(), refreshTokens.revokedUserID.Hex())
	}
}

func TestChangePasswordRejectsInvalidOldPassword(t *testing.T) {
	t.Parallel()

	userID := bson.NewObjectID()
	oldPasswordHash, err := bcrypt.GenerateFromPassword([]byte("Trip2026Pass"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("GenerateFromPassword returned error: %v", err)
	}
	users := &fakeUserRepository{got: &domain.User{ID: userID, Profile: domain.UserProfile{Username: "用户8000", AvatarObjectKey: defaultUserAvatarObjectKey}, Status: domain.UserStatusCreated}}
	passwords := &fakeUserPasswordCredentialRepository{got: &domain.UserPasswordCredential{UserID: userID, PasswordHash: string(oldPasswordHash), PasswordAlgo: passwordAlgorithmBcrypt}}
	refreshTokens := &fakeRefreshTokenRepository{}
	cfg := validAuthConfig()
	svc := NewAuthService(users, &fakeAuthIdentityRepository{}, &fakePhoneCodeRepository{}, refreshTokens, passwords, &fakeAuthUploadService{}, &recordingSMSService{}, NewTokenService(cfg.AccessTokenSecret, time.Duration(cfg.AccessTokenTTLSeconds)*time.Second), cfg)

	err = svc.ChangePassword(context.Background(), request.ChangePasswordInput{
		UserID:      userID.Hex(),
		OldPassword: "Wrong2026Pass",
		NewPassword: "NewTrip2027",
	})
	if err == nil || !strings.Contains(err.Error(), "password is invalid") {
		t.Fatalf("expected invalid password error, got %v", err)
	}
	if passwords.updatedPasswordHash != "" || refreshTokens.revokedUserID != bson.NilObjectID {
		t.Fatalf("did not expect password update or token revoke")
	}
}

func TestChangePasswordRejectsMissingCredential(t *testing.T) {
	t.Parallel()

	userID := bson.NewObjectID()
	users := &fakeUserRepository{got: &domain.User{ID: userID, Profile: domain.UserProfile{Username: "用户8000", AvatarObjectKey: defaultUserAvatarObjectKey}, Status: domain.UserStatusCreated}}
	cfg := validAuthConfig()
	svc := NewAuthService(users, &fakeAuthIdentityRepository{}, &fakePhoneCodeRepository{}, &fakeRefreshTokenRepository{}, &fakeUserPasswordCredentialRepository{}, &fakeAuthUploadService{}, &recordingSMSService{}, NewTokenService(cfg.AccessTokenSecret, time.Duration(cfg.AccessTokenTTLSeconds)*time.Second), cfg)

	err := svc.ChangePassword(context.Background(), request.ChangePasswordInput{
		UserID:      userID.Hex(),
		OldPassword: "Trip2026Pass",
		NewPassword: "NewTrip2027",
	})
	if err == nil || !strings.Contains(err.Error(), "password credential not found") {
		t.Fatalf("expected password credential not found error, got %v", err)
	}
}

func TestChangePasswordRejectsSamePassword(t *testing.T) {
	t.Parallel()

	userID := bson.NewObjectID()
	oldPasswordHash, err := bcrypt.GenerateFromPassword([]byte("Trip2026Pass"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("GenerateFromPassword returned error: %v", err)
	}
	users := &fakeUserRepository{got: &domain.User{ID: userID, Profile: domain.UserProfile{Username: "用户8000", AvatarObjectKey: defaultUserAvatarObjectKey}, Status: domain.UserStatusCreated}}
	passwords := &fakeUserPasswordCredentialRepository{got: &domain.UserPasswordCredential{UserID: userID, PasswordHash: string(oldPasswordHash), PasswordAlgo: passwordAlgorithmBcrypt}}
	refreshTokens := &fakeRefreshTokenRepository{}
	cfg := validAuthConfig()
	svc := NewAuthService(users, &fakeAuthIdentityRepository{}, &fakePhoneCodeRepository{}, refreshTokens, passwords, &fakeAuthUploadService{}, &recordingSMSService{}, NewTokenService(cfg.AccessTokenSecret, time.Duration(cfg.AccessTokenTTLSeconds)*time.Second), cfg)

	err = svc.ChangePassword(context.Background(), request.ChangePasswordInput{
		UserID:      userID.Hex(),
		OldPassword: "Trip2026Pass",
		NewPassword: "Trip2026Pass",
	})
	if err == nil || !strings.Contains(err.Error(), "new password must be different") {
		t.Fatalf("expected same password error, got %v", err)
	}
	if passwords.updatedPasswordHash != "" || refreshTokens.revokedUserID != bson.NilObjectID {
		t.Fatalf("did not expect password update or token revoke")
	}
}

func TestLoginWithPhoneRejectsWrongCode(t *testing.T) {
	t.Parallel()

	codes := &fakePhoneCodeRepository{
		got: &domain.PhoneVerificationCode{
			ID:       bson.NewObjectID(),
			Phone:    "+8613800138000",
			CodeHash: hashPhoneCode("+8613800138000", "123456"),
		},
	}
	svc := newTestAuthService(&fakeUserRepository{}, &fakeAuthIdentityRepository{}, codes, &fakeRefreshTokenRepository{}, &recordingSMSService{})

	_, err := svc.LoginWithPhone(context.Background(), request.PhoneLoginInput{Phone: "13800138000", Code: "000000"})
	if err == nil || !strings.Contains(err.Error(), "phone code is invalid") {
		t.Fatalf("expected invalid code error, got %v", err)
	}
}

func TestRefreshRejectsRevokedToken(t *testing.T) {
	t.Parallel()

	cfg := validAuthConfig()
	tokenService := NewTokenService(cfg.AccessTokenSecret, time.Hour)
	plain, hash, err := tokenService.CreateRefreshToken()
	if err != nil {
		t.Fatalf("CreateRefreshToken returned error: %v", err)
	}
	revokedAt := time.Now()
	refreshTokens := &fakeRefreshTokenRepository{got: &domain.RefreshToken{
		UserID:    bson.NewObjectID(),
		TokenHash: hash,
		ExpiresAt: time.Now().Add(time.Hour),
		RevokedAt: &revokedAt,
	}}
	svc := NewAuthService(&fakeUserRepository{}, &fakeAuthIdentityRepository{}, &fakePhoneCodeRepository{}, refreshTokens, &fakeUserPasswordCredentialRepository{}, &fakeAuthUploadService{}, &recordingSMSService{}, tokenService, cfg)

	_, err = svc.Refresh(context.Background(), request.RefreshTokenInput{RefreshToken: plain})
	if err == nil || !strings.Contains(err.Error(), "refresh token is revoked") {
		t.Fatalf("expected revoked token error, got %v", err)
	}
}

func TestRefreshReturnsAccessTokenWithoutRotatingRefreshToken(t *testing.T) {
	t.Parallel()

	cfg := validAuthConfig()
	tokenService := NewTokenService(cfg.AccessTokenSecret, time.Hour)
	plain, hash, err := tokenService.CreateRefreshToken()
	if err != nil {
		t.Fatalf("CreateRefreshToken returned error: %v", err)
	}
	userID := bson.NewObjectID()
	users := &fakeUserRepository{got: &domain.User{ID: userID, Profile: domain.UserProfile{Username: "用户8000", AvatarObjectKey: defaultUserAvatarObjectKey}, Status: domain.UserStatusCreated}}
	refreshTokens := &fakeRefreshTokenRepository{got: &domain.RefreshToken{
		UserID:    userID,
		TokenHash: hash,
		ExpiresAt: time.Now().Add(time.Hour),
	}}
	svc := NewAuthService(users, &fakeAuthIdentityRepository{}, &fakePhoneCodeRepository{}, refreshTokens, &fakeUserPasswordCredentialRepository{}, &fakeAuthUploadService{}, &recordingSMSService{}, tokenService, cfg)

	result, err := svc.Refresh(context.Background(), request.RefreshTokenInput{RefreshToken: plain})
	if err != nil {
		t.Fatalf("Refresh returned error: %v", err)
	}
	if refreshTokens.revoked != "" {
		t.Fatalf("expected refresh token to remain active")
	}
	if result.AccessToken == "" {
		t.Fatalf("expected a new access token, got %+v", result)
	}
}

func TestLogoutRevokesRefreshToken(t *testing.T) {
	t.Parallel()

	cfg := validAuthConfig()
	tokenService := NewTokenService(cfg.AccessTokenSecret, time.Hour)
	refreshTokens := &fakeRefreshTokenRepository{}
	svc := NewAuthService(&fakeUserRepository{}, &fakeAuthIdentityRepository{}, &fakePhoneCodeRepository{}, refreshTokens, &fakeUserPasswordCredentialRepository{}, &fakeAuthUploadService{}, &recordingSMSService{}, tokenService, cfg)

	if err := svc.Logout(context.Background(), request.LogoutInput{RefreshToken: "refresh"}); err != nil {
		t.Fatalf("Logout returned error: %v", err)
	}
	if refreshTokens.revoked == "" {
		t.Fatalf("expected refresh token to be revoked")
	}
}

func TestDeleteMeDeletesUserAndRevokesAuthState(t *testing.T) {
	t.Parallel()

	userID := bson.NewObjectID()
	users := &fakeUserRepository{got: &domain.User{
		ID: userID,
		Profile: domain.UserProfile{
			Username:        "用户8000",
			AvatarObjectKey: defaultUserAvatarObjectKey,
		},
		Status: domain.UserStatusCreated,
	}}
	identities := &fakeAuthIdentityRepository{}
	refreshTokens := &fakeRefreshTokenRepository{}
	svc := newTestAuthService(users, identities, &fakePhoneCodeRepository{}, refreshTokens, &recordingSMSService{})

	if err := svc.DeleteMe(context.Background(), userID.Hex()); err != nil {
		t.Fatalf("DeleteMe returned error: %v", err)
	}
	if users.updated == nil || users.updated.Status != domain.UserStatusDeleted {
		t.Fatalf("expected user to be marked deleted, got %+v", users.updated)
	}
	if identities.disabledUserID != userID {
		t.Fatalf("expected identities to be disabled for %s, got %s", userID.Hex(), identities.disabledUserID.Hex())
	}
	if refreshTokens.revokedUserID != userID {
		t.Fatalf("expected refresh tokens to be revoked for %s, got %s", userID.Hex(), refreshTokens.revokedUserID.Hex())
	}
}

func TestDeleteMeRejectsInvalidUserID(t *testing.T) {
	t.Parallel()

	svc := newTestAuthService(&fakeUserRepository{}, &fakeAuthIdentityRepository{}, &fakePhoneCodeRepository{}, &fakeRefreshTokenRepository{}, &recordingSMSService{})

	err := svc.DeleteMe(context.Background(), "invalid")
	if err == nil || !strings.Contains(err.Error(), "invalid input") {
		t.Fatalf("expected invalid input error, got %v", err)
	}
}

func TestDeleteMeReturnsUserNotFoundForDeletedUser(t *testing.T) {
	t.Parallel()

	userID := bson.NewObjectID()
	users := &fakeUserRepository{got: &domain.User{ID: userID, Status: domain.UserStatusDeleted}}
	svc := newTestAuthService(users, &fakeAuthIdentityRepository{}, &fakePhoneCodeRepository{}, &fakeRefreshTokenRepository{}, &recordingSMSService{})

	err := svc.DeleteMe(context.Background(), userID.Hex())
	if err == nil || !strings.Contains(err.Error(), "user not found") {
		t.Fatalf("expected user not found error, got %v", err)
	}
}

func TestDeleteMeReturnsDeleteUserFailure(t *testing.T) {
	t.Parallel()

	userID := bson.NewObjectID()
	users := &fakeUserRepository{
		got:       &domain.User{ID: userID, Status: domain.UserStatusCreated},
		updateErr: errors.New("write failed"),
	}
	svc := newTestAuthService(users, &fakeAuthIdentityRepository{}, &fakePhoneCodeRepository{}, &fakeRefreshTokenRepository{}, &recordingSMSService{})

	err := svc.DeleteMe(context.Background(), userID.Hex())
	if err == nil || !strings.Contains(err.Error(), "delete user failed") {
		t.Fatalf("expected delete user failure, got %v", err)
	}
}

func TestDeleteMeReturnsDisableIdentityFailure(t *testing.T) {
	t.Parallel()

	userID := bson.NewObjectID()
	users := &fakeUserRepository{got: &domain.User{ID: userID, Status: domain.UserStatusCreated}}
	identities := &fakeAuthIdentityRepository{err: errors.New("write failed")}
	svc := newTestAuthService(users, identities, &fakePhoneCodeRepository{}, &fakeRefreshTokenRepository{}, &recordingSMSService{})

	err := svc.DeleteMe(context.Background(), userID.Hex())
	if err == nil || !strings.Contains(err.Error(), "disable auth identities failed") {
		t.Fatalf("expected disable auth identities failure, got %v", err)
	}
}

func TestDeleteMeReturnsRevokeRefreshTokenFailure(t *testing.T) {
	t.Parallel()

	userID := bson.NewObjectID()
	users := &fakeUserRepository{got: &domain.User{ID: userID, Status: domain.UserStatusCreated}}
	refreshTokens := &fakeRefreshTokenRepository{err: errors.New("write failed")}
	svc := newTestAuthService(users, &fakeAuthIdentityRepository{}, &fakePhoneCodeRepository{}, refreshTokens, &recordingSMSService{})

	err := svc.DeleteMe(context.Background(), userID.Hex())
	if err == nil || !strings.Contains(err.Error(), "revoke refresh tokens failed") {
		t.Fatalf("expected revoke refresh tokens failure, got %v", err)
	}
}

func TestMeReturnsUser(t *testing.T) {
	t.Parallel()

	userID := bson.NewObjectID()
	users := &fakeUserRepository{got: &domain.User{ID: userID, Profile: domain.UserProfile{Username: "用户8000", AvatarObjectKey: defaultUserAvatarObjectKey}, Status: domain.UserStatusCreated}}
	svc := newTestAuthService(users, &fakeAuthIdentityRepository{}, &fakePhoneCodeRepository{}, &fakeRefreshTokenRepository{}, &recordingSMSService{})

	user, err := svc.Me(context.Background(), userID.Hex())
	if err != nil {
		t.Fatalf("Me returned error: %v", err)
	}
	if user.ID != userID {
		t.Fatalf("expected user %s, got %s", userID.Hex(), user.ID.Hex())
	}
}

func TestUpdateMyProfileUpdatesUser(t *testing.T) {
	t.Parallel()

	userID := bson.NewObjectID()
	users := &fakeUserRepository{got: &domain.User{ID: userID, Profile: domain.UserProfile{Username: "旧用户名", AvatarObjectKey: defaultUserAvatarObjectKey}, Status: domain.UserStatusCreated}}
	svc := newTestAuthService(users, &fakeAuthIdentityRepository{}, &fakePhoneCodeRepository{}, &fakeRefreshTokenRepository{}, &recordingSMSService{})

	user, err := svc.UpdateMyProfile(context.Background(), userID.Hex(), request.UpdateMyProfileInput{
		Username: "新用户",
		Gender:   "female",
		Birthday: "1998-08-20",
		File:     testImageReader(),
		FileName: "avatar.png",
	})
	if err != nil {
		t.Fatalf("UpdateMyProfile returned error: %v", err)
	}
	if users.updated == nil {
		t.Fatalf("expected user to be updated")
	}
	expectedAvatarObjectKey := "users/user_" + userID.Hex() + "/profile/avatar/source.png"
	expectedAvatarDisplayObjectKey := "users/user_" + userID.Hex() + "/profile/avatar/display.jpg"
	if user.Profile.Username != "新用户" || user.Profile.Gender != domain.UserGenderFemale ||
		user.Profile.AvatarObjectKey != expectedAvatarObjectKey || user.Profile.AvatarDisplayObjectKey != expectedAvatarDisplayObjectKey {
		t.Fatalf("unexpected updated user: %+v", user.Profile)
	}
	if user.Profile.Birthday == nil || user.Profile.Birthday.Format("2006-01-02") != "1998-08-20" {
		t.Fatalf("unexpected birthday: %+v", user.Profile.Birthday)
	}
}

func TestUpdateMyProfileRejectsInvalidGender(t *testing.T) {
	t.Parallel()

	userID := bson.NewObjectID()
	users := &fakeUserRepository{got: &domain.User{ID: userID, Profile: domain.UserProfile{Username: "旧用户名", AvatarObjectKey: defaultUserAvatarObjectKey}, Status: domain.UserStatusCreated}}
	svc := newTestAuthService(users, &fakeAuthIdentityRepository{}, &fakePhoneCodeRepository{}, &fakeRefreshTokenRepository{}, &recordingSMSService{})

	_, err := svc.UpdateMyProfile(context.Background(), userID.Hex(), request.UpdateMyProfileInput{
		Username: "新用户",
		Gender:   "robot",
		Birthday: "1998-08-20",
		File:     testImageReader(),
		FileName: "avatar.png",
	})
	if err == nil || !strings.Contains(err.Error(), "gender is invalid") {
		t.Fatalf("expected invalid gender error, got %v", err)
	}
}

func TestUpdateMyProfileKeepsDefaultAvatarWhenFileMissing(t *testing.T) {
	t.Parallel()

	userID := bson.NewObjectID()
	users := &fakeUserRepository{got: &domain.User{ID: userID, Profile: domain.UserProfile{Username: "旧用户名", AvatarObjectKey: defaultUserAvatarObjectKey}, Status: domain.UserStatusCreated}}
	svc := newTestAuthService(users, &fakeAuthIdentityRepository{}, &fakePhoneCodeRepository{}, &fakeRefreshTokenRepository{}, &recordingSMSService{})

	user, err := svc.UpdateMyProfile(context.Background(), userID.Hex(), request.UpdateMyProfileInput{
		Username: "新用户",
		Gender:   "private",
		Birthday: "",
	})
	if err != nil {
		t.Fatalf("UpdateMyProfile returned error: %v", err)
	}
	if user.Profile.AvatarObjectKey != defaultUserAvatarObjectKey {
		t.Fatalf("expected default avatar object key to be kept, got %s", user.Profile.AvatarObjectKey)
	}
	if user.Profile.Gender != domain.UserGenderPrivate {
		t.Fatalf("expected private gender, got %s", user.Profile.Gender)
	}
	if user.Profile.Birthday != nil {
		t.Fatalf("expected birthday to be cleared, got %+v", user.Profile.Birthday)
	}
}

func TestMeRejectsDeletedUser(t *testing.T) {
	t.Parallel()

	userID := bson.NewObjectID()
	users := &fakeUserRepository{got: &domain.User{ID: userID, Profile: domain.UserProfile{Username: "用户8000", AvatarObjectKey: defaultUserAvatarObjectKey}, Status: domain.UserStatusDeleted}}
	svc := newTestAuthService(users, &fakeAuthIdentityRepository{}, &fakePhoneCodeRepository{}, &fakeRefreshTokenRepository{}, &recordingSMSService{})

	_, err := svc.Me(context.Background(), userID.Hex())
	if err == nil || !strings.Contains(err.Error(), "user not found") {
		t.Fatalf("expected user not found, got %v", err)
	}
}

func TestMeRejectsDisabledUser(t *testing.T) {
	t.Parallel()

	userID := bson.NewObjectID()
	users := &fakeUserRepository{got: &domain.User{ID: userID, Profile: domain.UserProfile{Username: "用户8000", AvatarObjectKey: defaultUserAvatarObjectKey}, Status: domain.UserStatusDisabled}}
	svc := newTestAuthService(users, &fakeAuthIdentityRepository{}, &fakePhoneCodeRepository{}, &fakeRefreshTokenRepository{}, &recordingSMSService{})

	_, err := svc.Me(context.Background(), userID.Hex())
	if err == nil || !strings.Contains(err.Error(), "user is disabled") {
		t.Fatalf("expected user is disabled, got %v", err)
	}
}

func TestLoginWithPhoneWrapsCreateUserFailure(t *testing.T) {
	t.Parallel()

	codes := &fakePhoneCodeRepository{
		got: &domain.PhoneVerificationCode{
			ID:       bson.NewObjectID(),
			Phone:    "+8613800138000",
			CodeHash: hashPhoneCode("+8613800138000", "123456"),
		},
	}
	users := &fakeUserRepository{err: errors.New("database down")}
	svc := newTestAuthService(users, &fakeAuthIdentityRepository{}, codes, &fakeRefreshTokenRepository{}, &recordingSMSService{})

	_, err := svc.LoginWithPhone(context.Background(), request.PhoneLoginInput{Phone: "13800138000", Code: "123456"})
	if err == nil || !strings.Contains(err.Error(), "create user failed") {
		t.Fatalf("expected create user failure, got %v", err)
	}
}
