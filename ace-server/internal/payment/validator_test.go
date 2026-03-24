package payment

import (
	"context"
	"testing"
)

func TestMockValidator_AlwaysValid(t *testing.T) {
	v := NewMockValidator()
	result, err := v.Validate(context.Background(), "mock", "any_token", 1.50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Valid {
		t.Fatal("expected valid")
	}
	if result.TransactionID == "" {
		t.Fatal("expected transaction ID")
	}
}

func TestMockValidator_UnknownProvider(t *testing.T) {
	v := NewMockValidator()
	result, err := v.Validate(context.Background(), "stripe", "token", 1.50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Valid {
		t.Fatal("expected invalid for unknown provider")
	}
}

func TestParsePaymentHeader_Valid(t *testing.T) {
	provider, token, ok := ParsePaymentHeader("mock:my_token_123")
	if !ok {
		t.Fatal("expected ok")
	}
	if provider != "mock" || token != "my_token_123" {
		t.Fatalf("got provider=%q token=%q", provider, token)
	}
}

func TestParsePaymentHeader_ColonInToken(t *testing.T) {
	provider, token, ok := ParsePaymentHeader("x402:0x123:456:789")
	if !ok {
		t.Fatal("expected ok")
	}
	if provider != "x402" || token != "0x123:456:789" {
		t.Fatalf("got provider=%q token=%q", provider, token)
	}
}

func TestParsePaymentHeader_Invalid(t *testing.T) {
	_, _, ok := ParsePaymentHeader("no_colon")
	if ok {
		t.Fatal("expected not ok")
	}
	_, _, ok = ParsePaymentHeader("")
	if ok {
		t.Fatal("expected not ok for empty")
	}
}
