package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Configuration defines the application runtime configuration loaded from config.json.
type Configuration struct {
	Server    ServerConfig    `json:"server"`
	COS       COSConfig       `json:"cos"`
	Dashscope DashscopeConfig `json:"dashscope"`
	CORS      CORSConfig      `json:"cors"`
	Mongo     MongoConfig     `json:"mongo"`
	Auth      AuthConfig      `json:"auth"`
}

// ServerConfig defines HTTP server settings.
type ServerConfig struct {
	Port string `json:"port"`
}

// COSConfig defines Tencent COS access settings.
type COSConfig struct {
	BucketURL string `json:"bucket_url"`
	SecretID  string `json:"secret_id"`
	SecretKey string `json:"secret_key"`
}

// DashscopeConfig defines DashScope access settings.
type DashscopeConfig struct {
	APIKey     string `json:"api_key"`
	ImageModel string `json:"image_model"`
}

// CORSConfig defines allowed origins.
type CORSConfig struct {
	AllowedOrigins []string `json:"allowed_origins"`
}

// MongoConfig defines MongoDB connection settings.
type MongoConfig struct {
	URI                   string `json:"uri"`
	Database              string `json:"database"`
	ConnectTimeoutSeconds int    `json:"connect_timeout_seconds"`
	MaxPoolSize           uint64 `json:"max_pool_size"`
}

// AuthConfig defines authentication settings.
type AuthConfig struct {
	AccessTokenSecret      string          `json:"access_token_secret"`
	AccessTokenTTLSeconds  int             `json:"access_token_ttl_seconds"`
	RefreshTokenTTLSeconds int             `json:"refresh_token_ttl_seconds"`
	PhoneCode              PhoneCodeConfig `json:"phone_code"`
}

// PhoneCodeConfig defines phone verification code settings.
type PhoneCodeConfig struct {
	TTLSeconds            int    `json:"ttl_seconds"`
	MaxAttempts           int    `json:"max_attempts"`
	ResendIntervalSeconds int    `json:"resend_interval_seconds"`
	DailySendLimit        int    `json:"daily_send_limit"`
	UseDevFixedCode       bool   `json:"use_dev_fixed_code"`
	DevFixedCode          string `json:"dev_fixed_code"`
}

// Load reads and validates application configuration from a JSON file.
func Load(path string) (*Configuration, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var cfg Configuration
	if err := json.Unmarshal(body, &cfg); err != nil {
		return nil, fmt.Errorf("decode config file: %w", err)
	}

	if err := validate(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func validate(cfg *Configuration) error {
	switch {
	case strings.TrimSpace(cfg.Server.Port) == "":
		return fmt.Errorf("invalid config: server.port is required")
	case strings.TrimSpace(cfg.COS.BucketURL) == "":
		return fmt.Errorf("invalid config: cos.bucket_url is required")
	case strings.TrimSpace(cfg.COS.SecretID) == "":
		return fmt.Errorf("invalid config: cos.secret_id is required")
	case strings.TrimSpace(cfg.COS.SecretKey) == "":
		return fmt.Errorf("invalid config: cos.secret_key is required")
	case strings.TrimSpace(cfg.Dashscope.APIKey) == "":
		return fmt.Errorf("invalid config: dashscope.api_key is required")
	case strings.TrimSpace(cfg.Dashscope.ImageModel) == "":
		return fmt.Errorf("invalid config: dashscope.image_model is required")
	case strings.TrimSpace(cfg.Mongo.URI) == "":
		return fmt.Errorf("invalid config: mongo.uri is required")
	case strings.TrimSpace(cfg.Mongo.Database) == "":
		return fmt.Errorf("invalid config: mongo.database is required")
	}

	if cfg.Mongo.ConnectTimeoutSeconds <= 0 {
		cfg.Mongo.ConnectTimeoutSeconds = 10
	}
	if strings.TrimSpace(cfg.Auth.AccessTokenSecret) == "" {
		return fmt.Errorf("invalid config: auth.access_token_secret is required")
	}
	if cfg.Auth.AccessTokenTTLSeconds <= 0 {
		cfg.Auth.AccessTokenTTLSeconds = 7200
	}
	if cfg.Auth.RefreshTokenTTLSeconds <= 0 {
		cfg.Auth.RefreshTokenTTLSeconds = 2592000
	}
	if cfg.Auth.PhoneCode.TTLSeconds <= 0 {
		cfg.Auth.PhoneCode.TTLSeconds = 300
	}
	if cfg.Auth.PhoneCode.MaxAttempts <= 0 {
		cfg.Auth.PhoneCode.MaxAttempts = 5
	}
	if cfg.Auth.PhoneCode.ResendIntervalSeconds <= 0 {
		cfg.Auth.PhoneCode.ResendIntervalSeconds = 60
	}
	if cfg.Auth.PhoneCode.DailySendLimit <= 0 {
		cfg.Auth.PhoneCode.DailySendLimit = 10
	}
	if cfg.Auth.PhoneCode.UseDevFixedCode && strings.TrimSpace(cfg.Auth.PhoneCode.DevFixedCode) == "" {
		return fmt.Errorf("invalid config: auth.phone_code.dev_fixed_code is required when use_dev_fixed_code is true")
	}

	return nil
}
