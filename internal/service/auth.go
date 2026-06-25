package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
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
)

// AuthResult contains the authenticated user and tokens.
type AuthResult struct {
	AccessToken  string
	RefreshToken string
	User         *domain.User
}

// RefreshResult contains a refreshed access token.
type RefreshResult struct {
	AccessToken string
}

// AuthService defines authentication behavior.
type AuthService interface {
	SendPhoneCode(ctx context.Context, input request.SendPhoneCodeInput) error
	LoginWithPhone(ctx context.Context, input request.PhoneLoginInput) (*AuthResult, error)
	Refresh(ctx context.Context, input request.RefreshTokenInput) (*RefreshResult, error)
	Logout(ctx context.Context, input request.LogoutInput) error
	Me(ctx context.Context, userID string) (*domain.User, error)
}

type authService struct {
	users           repository.UserRepository
	identities      repository.AuthIdentityRepository
	phoneCodes      repository.PhoneVerificationCodeRepository
	refreshTokens   repository.RefreshTokenRepository
	sms             SMSService
	tokens          TokenService
	phoneCodeConfig config.PhoneCodeConfig
	refreshTokenTTL time.Duration
}

// NewAuthService creates an auth service.
func NewAuthService(
	users repository.UserRepository,
	identities repository.AuthIdentityRepository,
	phoneCodes repository.PhoneVerificationCodeRepository,
	refreshTokens repository.RefreshTokenRepository,
	sms SMSService,
	tokens TokenService,
	authConfig config.AuthConfig,
) AuthService {
	return &authService{
		users:           users,
		identities:      identities,
		phoneCodes:      phoneCodes,
		refreshTokens:   refreshTokens,
		sms:             sms,
		tokens:          tokens,
		phoneCodeConfig: authConfig.PhoneCode,
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

// Me returns the current user.
func (s *authService) Me(ctx context.Context, userID string) (*domain.User, error) {
	objectID, err := parseObjectID(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid input")
	}

	return s.getActiveUserByID(ctx, objectID)
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
		ID:          bson.NewObjectID(),
		DisplayName: newDefaultDisplayName(phone),
		Status:      domain.UserStatusCreated,
		CreatedAt:   now,
		UpdatedAt:   now,
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

func newDefaultDisplayName(phone string) string {
	if len(phone) <= 4 {
		return "用户" + phone
	}
	return "用户" + phone[len(phone)-4:]
}
