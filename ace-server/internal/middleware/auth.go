package middleware

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/nicroldan/ans/ace-server/internal/payment"
	"github.com/nicroldan/ans/ace-server/internal/store"
	"github.com/nicroldan/ans/shared/ace"
)

type contextKey string

const (
	ActorKey     contextKey = "actor"
	ActorTypeKey contextKey = "actor_type"
)

// GetActor extracts actor info from request context.
func GetActor(r *http.Request) (actor, actorType string) {
	if v, ok := r.Context().Value(ActorKey).(string); ok {
		actor = v
	}
	if v, ok := r.Context().Value(ActorTypeKey).(string); ok {
		actorType = v
	}
	if actor == "" {
		actor = "anonymous"
	}
	if actorType == "" {
		actorType = "agent"
	}
	return
}

// ACEAuth validates the X-ACE-Key header against stored API keys.
func ACEAuth(s *store.MemoryStore, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("X-ACE-Key")
		if key == "" {
			writeAuthError(w, http.StatusUnauthorized, "auth_required", "X-ACE-Key header is required")
			return
		}
		apiKey, valid := s.ValidateAPIKey(key)
		if !valid {
			writeAuthError(w, http.StatusUnauthorized, "invalid_key", "Invalid API key")
			return
		}
		ctx := context.WithValue(r.Context(), ActorKey, apiKey.Name)
		ctx = context.WithValue(ctx, ActorTypeKey, "agent")
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// DualAuth accepts either X-ACE-Key or X-ACE-Payment headers.
// If paymentEnabled is false, only API keys are accepted (401 on missing).
// If paymentEnabled is true and neither header is present, returns 402.
func DualAuth(s *store.MemoryStore, pv payment.Validator, paymentEnabled bool, providers []string, currency string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Priority 1: API Key
		if key := r.Header.Get("X-ACE-Key"); key != "" {
			apiKey, valid := s.ValidateAPIKey(key)
			if !valid {
				writeAuthError(w, http.StatusUnauthorized, "invalid_key", "Invalid API key")
				return
			}
			ctx := context.WithValue(r.Context(), ActorKey, apiKey.Name)
			ctx = context.WithValue(ctx, ActorTypeKey, "agent")
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// Priority 2: Payment Token
		if paymentHeader := r.Header.Get("X-ACE-Payment"); paymentHeader != "" {
			if !paymentEnabled {
				writeAuthError(w, http.StatusUnauthorized, "payment_not_supported", "This store does not accept payment auth")
				return
			}
			provider, token, ok := payment.ParsePaymentHeader(paymentHeader)
			if !ok {
				writeAuthError(w, http.StatusBadRequest, "invalid_payment", "Invalid X-ACE-Payment format, expected provider:token")
				return
			}
			result, err := pv.Validate(r.Context(), provider, token, 0)
			if err != nil {
				writeAuthError(w, http.StatusInternalServerError, "payment_error", "Payment validation failed")
				return
			}
			if !result.Valid {
				writeAuthError(w, http.StatusUnauthorized, "payment_rejected", "Payment token rejected")
				return
			}
			ctx := context.WithValue(r.Context(), ActorKey, "payment:"+result.TransactionID)
			ctx = context.WithValue(ctx, ActorTypeKey, "agent")
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// No auth provided
		if paymentEnabled {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusPaymentRequired)
			resp := ace.PaymentRequiredResponse{
				Error: "Payment or API key required",
				Code:  "payment_required",
				Pricing: ace.PricingInfo{
					Price:             0,
					Currency:          currency,
					AcceptedProviders: providers,
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		writeAuthError(w, http.StatusUnauthorized, "auth_required", "X-ACE-Key header is required")
	})
}

// AdminAuth validates a static admin token from the ADMIN_TOKEN env var.
func AdminAuth(adminToken string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token == "" {
			writeAuthError(w, http.StatusUnauthorized, "auth_required", "Authorization header is required")
			return
		}
		// Accept "Bearer <token>" or raw token
		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}
		if token != adminToken {
			writeAuthError(w, http.StatusUnauthorized, "invalid_token", "Invalid admin token")
			return
		}
		ctx := context.WithValue(r.Context(), ActorKey, "admin")
		ctx = context.WithValue(ctx, ActorTypeKey, "human")
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func writeAuthError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	resp := ace.ErrorResponse{Error: message, Code: code}
	b, _ := json.Marshal(resp)
	w.Write(b)
}
