package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math/big"
	"strings"
	"time"
	"unicode"

	"pack_mate/internal/config"
	"pack_mate/internal/domain"
	"pack_mate/internal/dto/request"
	"pack_mate/internal/repository"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"golang.org/x/crypto/bcrypt"
)

// AuthResult contains the authenticated user and tokens.
type AuthResult struct {
	AccessToken  string
	RefreshToken string
	User         *domain.User
}

const (
	defaultUserAvatarObjectKey = "users/default/avatar.png"
	passwordAlgorithmBcrypt    = "bcrypt"
)

// RefreshResult contains a refreshed access token.
type RefreshResult struct {
	AccessToken string
}

// AuthService defines authentication behavior.
type AuthService interface {
	SendPhoneCode(ctx context.Context, input request.SendPhoneCodeInput) error
	LoginWithPhone(ctx context.Context, input request.PhoneLoginInput) (*AuthResult, error)
	LoginWithPhonePassword(ctx context.Context, input request.PhonePasswordLoginInput) (*AuthResult, error)
	Refresh(ctx context.Context, input request.RefreshTokenInput) (*RefreshResult, error)
	Logout(ctx context.Context, input request.LogoutInput) error
	SetupPassword(ctx context.Context, input request.SetupPasswordInput) error
	ChangePassword(ctx context.Context, input request.ChangePasswordInput) error
	Me(ctx context.Context, userID string) (*domain.User, error)
	UpdateMyProfile(ctx context.Context, userID string, input request.UpdateMyProfileInput) (*domain.User, error)
	DeleteMe(ctx context.Context, userID string) error
}

type authService struct {
	users           repository.UserRepository
	identities      repository.AuthIdentityRepository
	phoneCodes      repository.PhoneVerificationCodeRepository
	refreshTokens   repository.RefreshTokenRepository
	passwords       repository.UserPasswordCredentialRepository
	uploader        UploadService
	sms             SMSService
	tokens          TokenService
	phoneCodeConfig config.PhoneCodeConfig
	passwordConfig  config.PasswordConfig
	refreshTokenTTL time.Duration
}

// NewAuthService creates an auth service.
func NewAuthService(
	users repository.UserRepository,
	identities repository.AuthIdentityRepository,
	phoneCodes repository.PhoneVerificationCodeRepository,
	refreshTokens repository.RefreshTokenRepository,
	passwords repository.UserPasswordCredentialRepository,
	uploader UploadService,
	sms SMSService,
	tokens TokenService,
	authConfig config.AuthConfig,
) AuthService {
	return &authService{
		users:           users,
		identities:      identities,
		phoneCodes:      phoneCodes,
		refreshTokens:   refreshTokens,
		passwords:       passwords,
		uploader:        uploader,
		sms:             sms,
		tokens:          tokens,
		phoneCodeConfig: authConfig.PhoneCode,
		passwordConfig:  authConfig.Password,
		refreshTokenTTL: time.Duration(authConfig.RefreshTokenTTLSeconds) * time.Second,
	}
}

// SendPhoneCode sends a phone login code.
func (s *authService) SendPhoneCode(ctx context.Context, input request.SendPhoneCodeInput) error {
	phone, err := normalizePhone(input.Phone)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	recentCount, err := s.phoneCodes.CountRecent(ctx, phone, now.Add(-24*time.Hour))
	if err != nil {
		return fmt.Errorf("count phone code failed: %w", err)
	}
	if recentCount >= int64(s.phoneCodeConfig.DailySendLimit) {
		return fmt.Errorf("phone code send too frequently")
	}

	latest, err := s.phoneCodes.GetLatestActive(ctx, phone, domain.PhoneVerificationPurposeLogin, now)
	if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		return fmt.Errorf("get phone code failed: %w", err)
	}
	if latest != nil && latest.CreatedAt.After(now.Add(-time.Duration(s.phoneCodeConfig.ResendIntervalSeconds)*time.Second)) {
		return fmt.Errorf("phone code send too frequently")
	}

	code, err := s.newPhoneCode()
	if err != nil {
		return fmt.Errorf("generate phone code failed: %w", err)
	}

	verificationCode := &domain.PhoneVerificationCode{
		ID:        bson.NewObjectID(),
		Phone:     phone,
		CodeHash:  hashPhoneCode(phone, code),
		Purpose:   domain.PhoneVerificationPurposeLogin,
		ExpiresAt: now.Add(time.Duration(s.phoneCodeConfig.TTLSeconds) * time.Second),
		CreatedAt: now,
	}
	if err := s.phoneCodes.Create(ctx, verificationCode); err != nil {
		return fmt.Errorf("create phone code failed: %w", err)
	}
	if err := s.sms.SendLoginCode(ctx, phone, code); err != nil {
		return fmt.Errorf("send phone code failed: %w", err)
	}

	return nil
}

// LoginWithPhone logs in with a phone code, creating a user on first login.
func (s *authService) LoginWithPhone(ctx context.Context, input request.PhoneLoginInput) (*AuthResult, error) {
	phone, err := normalizePhone(input.Phone)
	if err != nil {
		return nil, err
	}
	code := strings.TrimSpace(input.Code)
	if code == "" {
		return nil, fmt.Errorf("phone code is required")
	}

	now := time.Now().UTC()
	verificationCode, err := s.phoneCodes.GetLatestActive(ctx, phone, domain.PhoneVerificationPurposeLogin, now)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("phone code is invalid")
		}
		return nil, fmt.Errorf("get phone code failed: %w", err)
	}
	if verificationCode.AttemptCount >= s.phoneCodeConfig.MaxAttempts {
		return nil, fmt.Errorf("phone code is invalid")
	}
	if err := s.phoneCodes.IncrementAttempt(ctx, verificationCode.ID); err != nil {
		return nil, fmt.Errorf("increment phone code attempt failed: %w", err)
	}
	if verificationCode.CodeHash != hashPhoneCode(phone, code) {
		return nil, fmt.Errorf("phone code is invalid")
	}
	if err := s.phoneCodes.MarkConsumed(ctx, verificationCode.ID, now); err != nil {
		return nil, fmt.Errorf("consume phone code failed: %w", err)
	}

	user, err := s.getOrCreatePhoneUser(ctx, phone)
	if err != nil {
		return nil, err
	}

	return s.newAuthResult(ctx, user)
}

// LoginWithPhonePassword logs in with a phone number and password.
func (s *authService) LoginWithPhonePassword(ctx context.Context, input request.PhonePasswordLoginInput) (*AuthResult, error) {
	phone, err := normalizePhone(input.Phone)
	if err != nil {
		return nil, err
	}
	if input.Password == "" {
		return nil, fmt.Errorf("password is required")
	}

	identity, err := s.identities.GetByProviderAndIdentifier(ctx, domain.AuthProviderPhone, phone)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("phone or password is invalid")
		}
		return nil, fmt.Errorf("get auth identity failed: %w", err)
	}
	user, err := s.getActiveUserByID(ctx, identity.UserID)
	if err != nil {
		return nil, err
	}

	if s.passwords == nil {
		return nil, fmt.Errorf("password credential repository is not configured")
	}
	credential, err := s.passwords.GetByUserID(ctx, user.ID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("phone or password is invalid")
		}
		return nil, fmt.Errorf("get password credential failed: %w", err)
	}

	now := time.Now().UTC()
	if credential.LockedUntil != nil && credential.LockedUntil.After(now) {
		return nil, fmt.Errorf("password is locked")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(credential.PasswordHash), []byte(input.Password)); err != nil {
		if recordErr := s.recordPasswordFailure(ctx, credential, now); recordErr != nil {
			return nil, recordErr
		}
		return nil, fmt.Errorf("phone or password is invalid")
	}
	if err := s.passwords.RecordSuccess(ctx, user.ID, now); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("phone or password is invalid")
		}
		return nil, fmt.Errorf("record password login success failed: %w", err)
	}

	return s.newAuthResult(ctx, user)
}

// Refresh refreshes access token.
func (s *authService) Refresh(ctx context.Context, input request.RefreshTokenInput) (*RefreshResult, error) {
	refreshToken := strings.TrimSpace(input.RefreshToken)
	if refreshToken == "" {
		return nil, fmt.Errorf("refresh token is required")
	}

	tokenHash := s.tokens.HashToken(refreshToken)
	storedToken, err := s.refreshTokens.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("refresh token is invalid")
		}
		return nil, fmt.Errorf("get refresh token failed: %w", err)
	}
	now := time.Now().UTC()
	if storedToken.RevokedAt != nil {
		return nil, fmt.Errorf("refresh token is revoked")
	}
	if !storedToken.ExpiresAt.After(now) {
		return nil, fmt.Errorf("refresh token is expired")
	}

	user, err := s.getActiveUserByID(ctx, storedToken.UserID)
	if err != nil {
		return nil, err
	}

	accessToken, err := s.tokens.CreateAccessToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("create access token failed: %w", err)
	}

	return &RefreshResult{AccessToken: accessToken}, nil
}

// Logout revokes a refresh token.
func (s *authService) Logout(ctx context.Context, input request.LogoutInput) error {
	refreshToken := strings.TrimSpace(input.RefreshToken)
	if refreshToken == "" {
		return fmt.Errorf("refresh token is required")
	}

	if err := s.refreshTokens.Revoke(ctx, s.tokens.HashToken(refreshToken), time.Now().UTC()); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return fmt.Errorf("refresh token is invalid")
		}
		return fmt.Errorf("revoke refresh token failed: %w", err)
	}

	return nil
}

// SetupPassword sets up the current user's first login password.
func (s *authService) SetupPassword(ctx context.Context, input request.SetupPasswordInput) error {
	userID, err := parseObjectID(input.UserID)
	if err != nil {
		return err
	}
	phone, err := normalizePhone(input.Phone)
	if err != nil {
		return err
	}
	if input.Password == "" {
		return fmt.Errorf("password is required")
	}
	if err := validatePasswordStrength(input.Password); err != nil {
		return err
	}

	identity, err := s.identities.GetByProviderAndIdentifier(ctx, domain.AuthProviderPhone, phone)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return fmt.Errorf("phone does not match current user")
		}
		return fmt.Errorf("get auth identity failed: %w", err)
	}
	if identity.UserID != userID {
		return fmt.Errorf("phone does not match current user")
	}
	if _, err := s.getActiveUserByID(ctx, userID); err != nil {
		return err
	}

	if s.passwords == nil {
		return fmt.Errorf("password credential repository is not configured")
	}
	if _, err := s.passwords.GetByUserID(ctx, userID); err == nil {
		return fmt.Errorf("password already setup")
	} else if !errors.Is(err, mongo.ErrNoDocuments) {
		return fmt.Errorf("get password credential failed: %w", err)
	}

	passwordHash, err := s.hashPassword(input.Password)
	if err != nil {
		return fmt.Errorf("hash password failed: %w", err)
	}

	now := time.Now().UTC()
	credential := &domain.UserPasswordCredential{
		ID:                 bson.NewObjectID(),
		UserID:             userID,
		PasswordHash:       passwordHash,
		PasswordAlgo:       passwordAlgorithmBcrypt,
		FailedAttemptCount: 0,
		LockedUntil:        nil,
		LastUsedAt:         nil,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if err := s.passwords.Create(ctx, credential); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return fmt.Errorf("password already setup")
		}
		return fmt.Errorf("create password credential failed: %w", err)
	}

	return nil
}

// ChangePassword changes the current user's login password.
func (s *authService) ChangePassword(ctx context.Context, input request.ChangePasswordInput) error {
	userID, err := parseObjectID(input.UserID)
	if err != nil {
		return err
	}
	if input.OldPassword == "" {
		return fmt.Errorf("old password is required")
	}
	if input.NewPassword == "" {
		return fmt.Errorf("new password is required")
	}
	if err := validatePasswordStrength(input.NewPassword); err != nil {
		return err
	}
	if _, err := s.getActiveUserByID(ctx, userID); err != nil {
		return err
	}
	if s.passwords == nil {
		return fmt.Errorf("password credential repository is not configured")
	}
	credential, err := s.passwords.GetByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return fmt.Errorf("password credential not found")
		}
		return fmt.Errorf("get password credential failed: %w", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(credential.PasswordHash), []byte(input.OldPassword)); err != nil {
		return fmt.Errorf("password is invalid")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(credential.PasswordHash), []byte(input.NewPassword)); err == nil {
		return fmt.Errorf("new password must be different")
	}

	passwordHash, err := s.hashPassword(input.NewPassword)
	if err != nil {
		return fmt.Errorf("hash password failed: %w", err)
	}
	now := time.Now().UTC()
	if err := s.passwords.UpdatePassword(ctx, userID, passwordHash, passwordAlgorithmBcrypt, now); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return fmt.Errorf("password credential not found")
		}
		return fmt.Errorf("update password credential failed: %w", err)
	}
	if err := s.refreshTokens.RevokeByUserID(ctx, userID, now); err != nil {
		return fmt.Errorf("revoke refresh tokens failed: %w", err)
	}

	return nil
}

// Me returns the current user.
func (s *authService) Me(ctx context.Context, userID string) (*domain.User, error) {
	objectID, err := parseObjectID(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid input")
	}

	return s.getActiveUserByID(ctx, objectID)
}

// UpdateMyProfile updates the current user profile.
func (s *authService) UpdateMyProfile(ctx context.Context, userID string, input request.UpdateMyProfileInput) (*domain.User, error) {
	objectID, err := parseObjectID(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid input")
	}

	user, err := s.getActiveUserByID(ctx, objectID)
	if err != nil {
		return nil, err
	}

	username := strings.TrimSpace(input.Username)
	if username == "" {
		return nil, fmt.Errorf("username is required")
	}
	if len([]rune(username)) > 32 {
		return nil, fmt.Errorf("username is too long")
	}

	gender, err := parseRequiredUserGender(input.Gender)
	if err != nil {
		return nil, err
	}

	birthday, err := parseOptionalBirthday(input.Birthday)
	if err != nil {
		return nil, err
	}

	avatarObjectKey := strings.TrimSpace(user.Profile.AvatarObjectKey)
	avatarDisplayObjectKey := strings.TrimSpace(user.Profile.AvatarDisplayObjectKey)
	if input.File != nil {
		avatarObjectKey, avatarDisplayObjectKey, err = s.uploadUserAvatar(ctx, user.ID, input.FileName, input.File)
		if err != nil {
			return nil, err
		}
	}
	if avatarObjectKey == "" {
		return nil, fmt.Errorf("avatar is required")
	}
	if avatarDisplayObjectKey == "" {
		avatarDisplayObjectKey = avatarObjectKey
	}

	user.Profile.Username = username
	user.Profile.Gender = gender
	user.Profile.Birthday = birthday
	user.Profile.AvatarObjectKey = avatarObjectKey
	user.Profile.AvatarDisplayObjectKey = avatarDisplayObjectKey
	user.UpdatedAt = time.Now().UTC()

	if err := s.users.Update(ctx, user); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("update user profile failed: %w", err)
	}

	return user, nil
}

// DeleteMe deletes the current user account logically.
func (s *authService) DeleteMe(ctx context.Context, userID string) error {
	objectID, err := parseObjectID(userID)
	if err != nil {
		return fmt.Errorf("invalid input")
	}

	user, err := s.getActiveUserByID(ctx, objectID)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	user.Status = domain.UserStatusDeleted
	user.UpdatedAt = now

	if err := s.users.Update(ctx, user); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return fmt.Errorf("user not found")
		}
		return fmt.Errorf("delete user failed: %w", err)
	}
	if err := s.identities.DisableByUserID(ctx, objectID, now); err != nil {
		return fmt.Errorf("disable auth identities failed: %w", err)
	}
	if err := s.refreshTokens.RevokeByUserID(ctx, objectID, now); err != nil {
		return fmt.Errorf("revoke refresh tokens failed: %w", err)
	}

	return nil
}

func (s *authService) getOrCreatePhoneUser(ctx context.Context, phone string) (*domain.User, error) {
	identity, err := s.identities.GetByProviderAndIdentifier(ctx, domain.AuthProviderPhone, phone)
	if err == nil {
		return s.getUserByIdentity(ctx, identity)
	}
	if !errors.Is(err, mongo.ErrNoDocuments) {
		return nil, fmt.Errorf("get auth identity failed: %w", err)
	}

	now := time.Now().UTC()
	user := &domain.User{
		ID: bson.NewObjectID(),
		Profile: domain.UserProfile{
			Username:        newDefaultUsername(phone),
			Gender:          "",
			Birthday:        nil,
			AvatarObjectKey: defaultUserAvatarObjectKey,
		},
		Status:    domain.UserStatusCreated,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.users.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("create user failed: %w", err)
	}

	identity = &domain.AuthIdentity{
		ID:         bson.NewObjectID(),
		UserID:     user.ID,
		Provider:   domain.AuthProviderPhone,
		Identifier: phone,
		Status:     domain.AuthIdentityStatusActive,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := s.identities.Create(ctx, identity); err != nil {
		// Another concurrent first login may have created the unique phone identity.
		// Re-read it so the request can still log in instead of failing on the race.
		existingIdentity, getErr := s.identities.GetByProviderAndIdentifier(ctx, domain.AuthProviderPhone, phone)
		if getErr == nil {
			return s.getUserByIdentity(ctx, existingIdentity)
		}
		return nil, fmt.Errorf("create auth identity failed: %w", err)
	}

	return user, nil
}

func (s *authService) getUserByIdentity(ctx context.Context, identity *domain.AuthIdentity) (*domain.User, error) {
	user, err := s.getActiveUserByID(ctx, identity.UserID)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *authService) getActiveUserByID(ctx context.Context, userID bson.ObjectID) (*domain.User, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("get user failed: %w", err)
	}
	if user.Status == domain.UserStatusDeleted {
		return nil, fmt.Errorf("user not found")
	}
	if user.Status == domain.UserStatusDisabled {
		return nil, fmt.Errorf("user is disabled")
	}
	return user, nil
}

func (s *authService) newAuthResult(ctx context.Context, user *domain.User) (*AuthResult, error) {
	accessToken, err := s.tokens.CreateAccessToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("create access token failed: %w", err)
	}

	refreshToken, refreshTokenHash, err := s.tokens.CreateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("create refresh token failed: %w", err)
	}

	now := time.Now().UTC()
	storedToken := &domain.RefreshToken{
		ID:        bson.NewObjectID(),
		UserID:    user.ID,
		TokenHash: refreshTokenHash,
		ExpiresAt: now.Add(s.refreshTokenTTL),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.refreshTokens.Create(ctx, storedToken); err != nil {
		return nil, fmt.Errorf("create refresh token failed: %w", err)
	}

	return &AuthResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user,
	}, nil
}

func (s *authService) newPhoneCode() (string, error) {
	if s.phoneCodeConfig.UseDevFixedCode {
		return strings.TrimSpace(s.phoneCodeConfig.DevFixedCode), nil
	}

	value, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", value.Int64()), nil
}

func normalizePhone(value string) (string, error) {
	phone := strings.TrimSpace(value)
	if phone == "" {
		return "", fmt.Errorf("phone is required")
	}
	if strings.HasPrefix(phone, "+") {
		if len(phone) < 9 || !allDigits(phone[1:]) {
			return "", fmt.Errorf("phone is invalid")
		}
		return phone, nil
	}
	if !allDigits(phone) {
		return "", fmt.Errorf("phone is invalid")
	}
	if len(phone) == 11 && strings.HasPrefix(phone, "1") {
		return "+86" + phone, nil
	}
	if len(phone) < 8 {
		return "", fmt.Errorf("phone is invalid")
	}
	return "+" + phone, nil
}

func allDigits(value string) bool {
	for _, r := range value {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

func hashPhoneCode(phone string, code string) string {
	sum := sha256.Sum256([]byte(phone + ":" + strings.TrimSpace(code)))
	return hex.EncodeToString(sum[:])
}

func newDefaultUsername(phone string) string {
	return "user" + digitsOnly(phone)
}

func (s *authService) hashPassword(password string) (string, error) {
	cost := s.passwordConfig.BcryptCost
	if cost <= 0 {
		cost = bcrypt.DefaultCost
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func (s *authService) recordPasswordFailure(ctx context.Context, credential *domain.UserPasswordCredential, now time.Time) error {
	failedAttemptCount := credential.FailedAttemptCount
	if credential.LockedUntil != nil && !credential.LockedUntil.After(now) {
		failedAttemptCount = 0
	}
	failedAttemptCount++
	var lockedUntil *time.Time
	if failedAttemptCount >= s.passwordMaxFailedAttempts() {
		value := now.Add(s.passwordLockDuration())
		lockedUntil = &value
	}
	if err := s.passwords.RecordFailure(ctx, credential.UserID, failedAttemptCount, lockedUntil, now); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return fmt.Errorf("phone or password is invalid")
		}
		return fmt.Errorf("record password login failure failed: %w", err)
	}
	return nil
}

func (s *authService) passwordMaxFailedAttempts() int {
	if s.passwordConfig.MaxFailedAttempts <= 0 {
		return 5
	}
	return s.passwordConfig.MaxFailedAttempts
}

func (s *authService) passwordLockDuration() time.Duration {
	if s.passwordConfig.LockDurationSeconds <= 0 {
		return 15 * time.Minute
	}
	return time.Duration(s.passwordConfig.LockDurationSeconds) * time.Second
}

func validatePasswordStrength(password string) error {
	runes := []rune(password)
	if len(runes) < 8 {
		return fmt.Errorf("password is too short")
	}
	if len(runes) > 32 {
		return fmt.Errorf("password is too long")
	}
	if strings.TrimSpace(password) != password {
		return fmt.Errorf("password has invalid spaces")
	}

	hasUpper := false
	hasLower := false
	hasDigit := false
	hasSpecial := false
	for _, r := range runes {
		switch {
		case r >= 'A' && r <= 'Z':
			hasUpper = true
		case r >= 'a' && r <= 'z':
			hasLower = true
		case r >= '0' && r <= '9':
			hasDigit = true
		case isAllowedPasswordSpecial(r):
			hasSpecial = true
		default:
			return fmt.Errorf("password has invalid characters")
		}
	}
	categoryCount := 0
	for _, ok := range []bool{hasUpper, hasLower, hasDigit, hasSpecial} {
		if ok {
			categoryCount++
		}
	}
	if categoryCount < 3 {
		return fmt.Errorf("password is too weak")
	}

	return nil
}

func isAllowedPasswordSpecial(r rune) bool {
	return strings.ContainsRune("_#!@$%^&*()+=-", r)
}

func parseRequiredUserGender(value string) (domain.UserGender, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", fmt.Errorf("gender is required")
	}

	switch domain.UserGender(trimmed) {
	case domain.UserGenderMale, domain.UserGenderFemale, domain.UserGenderPrivate:
		return domain.UserGender(trimmed), nil
	default:
		return "", fmt.Errorf("gender is invalid")
	}
}

func parseOptionalBirthday(value string) (*time.Time, error) {
	birthday := strings.TrimSpace(value)
	if birthday == "" {
		return nil, nil
	}

	parsed, err := time.Parse("2006-01-02", birthday)
	if err != nil {
		return nil, fmt.Errorf("birthday is invalid")
	}
	if parsed.After(time.Now().UTC()) {
		return nil, fmt.Errorf("birthday is invalid")
	}

	return &parsed, nil
}

func digitsOnly(value string) string {
	var builder strings.Builder
	builder.Grow(len(value))
	for _, r := range value {
		if unicode.IsDigit(r) {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

func (s *authService) uploadUserAvatar(ctx context.Context, userID bson.ObjectID, fileName string, file io.ReadSeeker) (string, string, error) {
	body, contentType, err := readUpload(file)
	if err != nil {
		return "", "", fmt.Errorf("invalid input")
	}
	displayBody, displayContentType, err := buildDisplayImage(body)
	if err != nil {
		return "", "", fmt.Errorf("invalid input")
	}

	objectKey := buildUserAvatarObjectKey(userID, fileName, contentType)
	if err := s.uploader.Upload(ctx, objectKey, contentType, bytes.NewReader(body)); err != nil {
		return "", "", fmt.Errorf("upload user avatar failed: %w", err)
	}
	displayObjectKey := buildUserAvatarDisplayObjectKey(userID)
	if err := s.uploader.Upload(ctx, displayObjectKey, displayContentType, bytes.NewReader(displayBody)); err != nil {
		return "", "", fmt.Errorf("upload user avatar failed: %w", err)
	}

	return objectKey, displayObjectKey, nil
}

func buildUserAvatarObjectKey(userID bson.ObjectID, fileName string, contentType string) string {
	ext := extensionForUpload(fileName, contentType)
	return fmt.Sprintf("users/user_%s/profile/avatar/source%s", userID.Hex(), ext)
}

func buildUserAvatarDisplayObjectKey(userID bson.ObjectID) string {
	return fmt.Sprintf("users/user_%s/profile/avatar/display.jpg", userID.Hex())
}
