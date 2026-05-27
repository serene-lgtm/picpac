package service

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/tencentyun/cos-go-sdk-v5"
)

// UploadService defines object upload behavior.
type UploadService interface {
	Upload(ctx context.Context, objectKey string, contentType string, body io.Reader) (string, error)
}

// COSUploadService uploads objects to Tencent COS.
type COSUploadService struct {
	client    *cos.Client
	bucketURL *url.URL
}

// NewCOSUploadService creates a COS-backed upload service.
func NewCOSUploadService(client *cos.Client, bucketURL *url.URL) *COSUploadService {
	return &COSUploadService{
		client:    client,
		bucketURL: bucketURL,
	}
}

// Upload uploads an object to COS and returns its URL.
func (s *COSUploadService) Upload(ctx context.Context, objectKey string, contentType string, body io.Reader) (string, error) {
	if s == nil || s.client == nil || s.bucketURL == nil {
		return "", fmt.Errorf("upload service is not configured")
	}

	_, err := s.client.Object.Put(ctx, objectKey, body, &cos.ObjectPutOptions{
		ObjectPutHeaderOptions: &cos.ObjectPutHeaderOptions{
			ContentType: strings.TrimSpace(contentType),
		},
	})
	if err != nil {
		return "", err
	}

	baseURL := strings.TrimRight(s.bucketURL.String(), "/")
	return baseURL + "/" + strings.TrimLeft(objectKey, "/"), nil
}
