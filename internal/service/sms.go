package service

import "context"

// SMSService defines SMS sending behavior.
type SMSService interface {
	SendLoginCode(ctx context.Context, phone string, code string) error
}

type fakeSMSService struct{}

// NewFakeSMSService creates a fake SMS sender for development.
func NewFakeSMSService() SMSService {
	return &fakeSMSService{}
}

// SendLoginCode pretends to send a phone login code.
func (s *fakeSMSService) SendLoginCode(_ context.Context, _ string, _ string) error {
	return nil
}
