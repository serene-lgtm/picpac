package service

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// TokenService defines auth token behavior.
type TokenService interface {
	CreateAccessToken(userID bson.ObjectID) (string, error)
	ParseAccessToken(token string) (bson.ObjectID, error)
	CreateRefreshToken() (plain string, hash string, err error)
	HashToken(token string) string
}

type hmacTokenService struct {
	secret         []byte
	accessTokenTTL time.Duration
}

type jwtHeader struct {
	Algorithm string `json:"alg"`
	Type      string `json:"typ"`
}

type jwtClaims struct {
	Subject   string `json:"sub"`
	ExpiresAt int64  `json:"exp"`
	IssuedAt  int64  `json:"iat"`
}

// NewTokenService creates an HMAC-backed token service.
func NewTokenService(secret string, accessTokenTTL time.Duration) TokenService {
	return &hmacTokenService{
		secret:         []byte(secret),
		accessTokenTTL: accessTokenTTL,
	}
}

// CreateAccessToken creates a signed access token.
func (s *hmacTokenService) CreateAccessToken(userID bson.ObjectID) (string, error) {
	now := time.Now().UTC()
	header := jwtHeader{Algorithm: "HS256", Type: "JWT"}
	claims := jwtClaims{
		Subject:   userID.Hex(),
		ExpiresAt: now.Add(s.accessTokenTTL).Unix(),
		IssuedAt:  now.Unix(),
	}

	headerValue, err := encodeJWTPart(header)
	if err != nil {
		return "", err
	}
	claimsValue, err := encodeJWTPart(claims)
	if err != nil {
		return "", err
	}

	unsigned := headerValue + "." + claimsValue
	signature := s.sign(unsigned)
	return unsigned + "." + signature, nil
}

// ParseAccessToken parses and validates a signed access token.
func (s *hmacTokenService) ParseAccessToken(token string) (bson.ObjectID, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return bson.NilObjectID, fmt.Errorf("access token is invalid")
	}

	unsigned := parts[0] + "." + parts[1]
	expectedSignature := s.sign(unsigned)
	if !hmac.Equal([]byte(expectedSignature), []byte(parts[2])) {
		return bson.NilObjectID, fmt.Errorf("access token is invalid")
	}

	var claims jwtClaims
	if err := decodeJWTPart(parts[1], &claims); err != nil {
		return bson.NilObjectID, fmt.Errorf("access token is invalid")
	}
	if time.Now().UTC().Unix() >= claims.ExpiresAt {
		return bson.NilObjectID, fmt.Errorf("access token is expired")
	}

	userID, err := bson.ObjectIDFromHex(claims.Subject)
	if err != nil {
		return bson.NilObjectID, fmt.Errorf("access token is invalid")
	}

	return userID, nil
}

// CreateRefreshToken creates a refresh token and its hash.
func (s *hmacTokenService) CreateRefreshToken() (string, string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", err
	}
	token := base64.RawURLEncoding.EncodeToString(bytes)
	return token, s.HashToken(token), nil
}

// HashToken hashes a token for storage.
func (s *hmacTokenService) HashToken(token string) string {
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(token))
	return hex.EncodeToString(mac.Sum(nil))
}

func (s *hmacTokenService) sign(value string) string {
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(value))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func encodeJWTPart(value any) (string, error) {
	body, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(body), nil
}

func decodeJWTPart(value string, target any) error {
	body, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, target)
}
