package payment

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"
)

// PaymentResult holds the result of a payment validation.
type PaymentResult struct {
	Valid            bool
	TransactionID    string
	BalanceRemaining *float64
}

// Validator validates payment tokens from different providers.
type Validator interface {
	Validate(ctx context.Context, provider, token string, price float64) (PaymentResult, error)
}

// MockValidator accepts any token for the "mock" provider.
type MockValidator struct{}

// NewMockValidator creates a new MockValidator.
func NewMockValidator() *MockValidator {
	return &MockValidator{}
}

// Validate checks a payment token. Mock provider always succeeds.
func (v *MockValidator) Validate(_ context.Context, provider, token string, price float64) (PaymentResult, error) {
	if provider != "mock" {
		return PaymentResult{Valid: false}, nil
	}
	b := make([]byte, 8)
	rand.Read(b)
	return PaymentResult{
		Valid:         true,
		TransactionID: "txn_mock_" + hex.EncodeToString(b),
	}, nil
}

// ParsePaymentHeader parses "provider:token" from the X-ACE-Payment header.
// Splits on the first colon only — tokens may contain colons.
func ParsePaymentHeader(header string) (provider, token string, ok bool) {
	if header == "" {
		return "", "", false
	}
	idx := strings.Index(header, ":")
	if idx <= 0 || idx == len(header)-1 {
		return "", "", false
	}
	return header[:idx], header[idx+1:], true
}
