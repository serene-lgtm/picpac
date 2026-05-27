package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"pack_mate/internal/config"
	mongodb "pack_mate/internal/repository/mongodb"

	"github.com/tencentyun/cos-go-sdk-v5"
	"github.com/tencentyun/cos-go-sdk-v5/debug"
)

func main() {
	appCfg, err := loadConfig()
	if err != nil {
		panic(err)
	}

	client, bucketURL, err := newCOSClient(appCfg)
	if err != nil {
		panic(err)
	}

	mongoConn, err := mongodb.New(appCfg.Mongo)
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = mongoConn.Close(context.Background())
	}()

	port := strings.TrimSpace(appCfg.Server.Port)
	if port == "" {
		port = "8080"
	}

	router := newRouter(appCfg, client, bucketURL, mongoConn.Database)

	if err := router.Run(":" + port); err != nil {
		panic(err)
	}
}

func loadConfig() (*config.Configuration, error) {
	candidates := []string{}
	if path := strings.TrimSpace(os.Getenv("PICPAC_CONFIG")); path != "" {
		candidates = append(candidates, path)
	}
	candidates = append(candidates, "config.json", filepath.Join("..", "config.json"))

	if executablePath, err := os.Executable(); err == nil {
		executableDir := filepath.Dir(executablePath)
		candidates = append(candidates,
			filepath.Join(executableDir, "config.json"),
			filepath.Join(executableDir, "..", "config.json"),
		)
	}

	var lastErr error
	seen := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		cleanPath := filepath.Clean(candidate)
		if _, ok := seen[cleanPath]; ok {
			continue
		}
		seen[cleanPath] = struct{}{}

		cfg, err := config.Load(cleanPath)
		if err == nil {
			return cfg, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		lastErr = err
	}

	return nil, fmt.Errorf("load config.json: %w", lastErr)
}

func newCOSClient(cfg *config.Configuration) (*cos.Client, *url.URL, error) {
	bucketURLRaw := strings.TrimSpace(cfg.COS.BucketURL)
	secretID := strings.TrimSpace(cfg.COS.SecretID)
	secretKey := strings.TrimSpace(cfg.COS.SecretKey)

	switch {
	case bucketURLRaw == "":
		return nil, nil, errors.New("missing COS_BUCKET_URL")
	case secretID == "":
		return nil, nil, errors.New("missing SECRETID")
	case secretKey == "":
		return nil, nil, errors.New("missing SECRETKEY")
	}

	bucketURL, err := url.Parse(bucketURLRaw)
	if err != nil {
		return nil, nil, fmt.Errorf("parse COS_BUCKET_URL: %w", err)
	}

	baseURL := &cos.BaseURL{BucketURL: bucketURL}
	client := cos.NewClient(baseURL, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  secretID,
			SecretKey: secretKey,
			Transport: &debug.DebugRequestTransport{
				RequestHeader:  true,
				RequestBody:    false,
				ResponseHeader: true,
				ResponseBody:   false,
			},
		},
	})

	return client, bucketURL, nil
}
