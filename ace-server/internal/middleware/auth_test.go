package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nicroldan/ans/ace-server/internal/payment"
	"github.com/nicroldan/ans/ace-server/internal/store"
)

func TestDualAuth_APIKey(t *testing.T) {
	s := store.New()
	resp, key := s.CreateAPIKey("test-agent", []string{"catalog:read"})
	_ = resp

	validator := payment.NewMockValidator()
	handler := DualAuth(s, validator, true, []string{"mock"}, "USD", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-ACE-Key", key)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestDualAuth_PaymentToken(t *testing.T) {
	s := store.New()
	validator := payment.NewMockValidator()
	handler := DualAuth(s, validator, true, []string{"mock"}, "USD", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-ACE-Payment", "mock:test_token_123")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestDualAuth_NoAuth_Returns402(t *testing.T) {
	s := store.New()
	validator := payment.NewMockValidator()
	handler := DualAuth(s, validator, true, []string{"mock"}, "USD", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusPaymentRequired {
		t.Fatalf("expected 402, got %d", w.Code)
	}
}

func TestDualAuth_NoAuth_PaymentDisabled_Returns401(t *testing.T) {
	s := store.New()
	validator := payment.NewMockValidator()
	handler := DualAuth(s, validator, false, nil, "USD", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestDualAuth_InvalidPaymentToken(t *testing.T) {
	s := store.New()
	validator := payment.NewMockValidator()
	handler := DualAuth(s, validator, true, []string{"mock"}, "USD", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-ACE-Payment", "stripe:invalid_token") // mock validator rejects non-mock providers
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestDualAuth_BothHeaders_APIKeyWins(t *testing.T) {
	s := store.New()
	_, key := s.CreateAPIKey("test-agent", []string{"catalog:read"})
	validator := payment.NewMockValidator()
	handler := DualAuth(s, validator, true, []string{"mock"}, "USD", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actor, _ := GetActor(r)
		if actor != "test-agent" {
			t.Fatalf("expected actor test-agent, got %s", actor)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-ACE-Key", key)
	req.Header.Set("X-ACE-Payment", "mock:token")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
