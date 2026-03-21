package ace

import "time"

// --- Discovery types ---

// WellKnownResponse represents the .well-known/agent-commerce response.
type WellKnownResponse struct {
	StoreID        string         `json:"store_id"`
	Name           string         `json:"name"`
	Version        string         `json:"version"`
	ACEBaseURL     string         `json:"ace_base_url"`
	Capabilities   []string       `json:"capabilities"`
	Auth           AuthConfig     `json:"auth"`
	Currencies     []string       `json:"currencies"`
	PoliciesPublic map[string]any `json:"policies_public,omitempty"`
}

// AuthConfig describes the authentication method for a store.
type AuthConfig struct {
	Type   string `json:"type"`
	Header string `json:"header"`
}

// --- Product/Catalog types ---

// Product represents an item available for purchase.
type Product struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Price       Money     `json:"price"`
	Variants    []Variant `json:"variants,omitempty"`
	Status      string    `json:"status"` // draft, published, unpublished
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Variant represents a specific SKU of a product.
type Variant struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	SKU        string            `json:"sku"`
	Price      Money             `json:"price"`
	Inventory  int               `json:"inventory"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

// Money represents a monetary amount in the smallest currency unit (cents).
type Money struct {
	Amount   int64  `json:"amount"`   // in cents
	Currency string `json:"currency"` // ISO 4217
}

// --- Cart types ---

// Cart represents a shopping cart.
type Cart struct {
	ID        string     `json:"id"`
	Items     []CartItem `json:"items"`
	Total     Money      `json:"total"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// CartItem represents a single line item in a cart.
type CartItem struct {
	ProductID string `json:"product_id"`
	VariantID string `json:"variant_id,omitempty"`
	Quantity  int    `json:"quantity"`
	Price     Money  `json:"price"`
}

// AddCartItemRequest is the request body for adding an item to a cart.
type AddCartItemRequest struct {
	ProductID string `json:"product_id"`
	VariantID string `json:"variant_id,omitempty"`
	Quantity  int    `json:"quantity"`
}

// --- Order types ---

// Order represents a placed order.
type Order struct {
	ID        string      `json:"id"`
	CartID    string      `json:"cart_id"`
	Items     []OrderItem `json:"items"`
	Total     Money       `json:"total"`
	Status    string      `json:"status"` // pending, paid, fulfilled, refunded, cancelled
	Payment   *Payment    `json:"payment,omitempty"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

// OrderItem represents a single line item in an order.
type OrderItem struct {
	ProductID   string `json:"product_id"`
	ProductName string `json:"product_name"`
	VariantID   string `json:"variant_id,omitempty"`
	Quantity    int    `json:"quantity"`
	Price       Money  `json:"price"`
}

// CreateOrderRequest is the request body for creating an order from a cart.
type CreateOrderRequest struct {
	CartID string `json:"cart_id"`
}

// --- Payment types ---

// Payment represents a payment associated with an order.
type Payment struct {
	ID         string    `json:"id"`
	OrderID    string    `json:"order_id"`
	Status     string    `json:"status"`   // pending, processing, completed, failed
	Provider   string    `json:"provider"` // stripe, mercadopago
	Amount     Money     `json:"amount"`
	ExternalID string    `json:"external_id,omitempty"`
	PaymentURL string    `json:"payment_url,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// InitiatePaymentRequest is the request body for initiating a payment.
type InitiatePaymentRequest struct {
	Provider string `json:"provider"` // stripe, mercadopago
}

// --- Shipping types ---

// ShippingQuoteRequest is the request body for getting shipping quotes.
type ShippingQuoteRequest struct {
	Items       []CartItem `json:"items"`
	Destination Address    `json:"destination"`
}

// Address represents a physical address.
type Address struct {
	Country    string `json:"country"`
	State      string `json:"state"`
	City       string `json:"city"`
	PostalCode string `json:"postal_code"`
	Line1      string `json:"line1"`
	Line2      string `json:"line2,omitempty"`
}

// ShippingQuote contains available shipping options.
type ShippingQuote struct {
	Options []ShippingOption `json:"options"`
}

// ShippingOption represents a single shipping method with price and delivery estimate.
type ShippingOption struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Price         Money  `json:"price"`
	EstimatedDays int    `json:"estimated_days"`
}

// --- Registry types ---

// StoreRegistration is the request body for registering a store in the registry.
type StoreRegistration struct {
	WellKnownURL string   `json:"well_known_url"`
	Categories   []string `json:"categories,omitempty"`
	Country      string   `json:"country,omitempty"`
}

// StoreEntry represents a store as stored in the registry.
type StoreEntry struct {
	ID           string    `json:"id"`
	WellKnownURL string    `json:"well_known_url"`
	Name         string    `json:"name"`
	Categories   []string  `json:"categories,omitempty"`
	Country      string    `json:"country,omitempty"`
	Currencies   []string  `json:"currencies,omitempty"`
	Capabilities []string  `json:"capabilities,omitempty"`
	HealthStatus string    `json:"health_status"` // healthy, degraded, down, unknown
	LastChecked  time.Time `json:"last_checked"`
	RegisteredAt time.Time `json:"registered_at"`
}

// --- Common types ---

// PaginatedResponse wraps a list of items with pagination metadata.
type PaginatedResponse[T any] struct {
	Data   []T `json:"data"`
	Total  int `json:"total"`
	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}

// ErrorResponse is the standard error payload returned by ACE endpoints.
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code"`
	Details string `json:"details,omitempty"`
}

// --- Policy/Trust types ---

// Policy represents an access control policy for a store action.
type Policy struct {
	ID     string `json:"id"`
	Action string `json:"action"` // product.publish, order.refund, etc.
	Effect string `json:"effect"` // allow, deny, approval
}

// Approval represents a pending or resolved approval request.
type Approval struct {
	ID          string     `json:"id"`
	Action      string     `json:"action"`
	Resource    string     `json:"resource"`
	Status      string     `json:"status"` // pending, approved, rejected
	RequestedBy string     `json:"requested_by"`
	ResolvedBy  string     `json:"resolved_by,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
}

// AuditEntry records a single auditable action in the system.
type AuditEntry struct {
	ID            string         `json:"id"`
	StoreID       string         `json:"store_id"`
	Action        string         `json:"action"`
	Actor         string         `json:"actor"`
	ActorType     string         `json:"actor_type"` // human, agent
	Resource      string         `json:"resource"`
	ResourceID    string         `json:"resource_id"`
	Details       map[string]any `json:"details,omitempty"`
	CorrelationID string         `json:"correlation_id"`
	Timestamp     time.Time      `json:"timestamp"`
}

// --- API Key types ---

// APIKey represents an API key (without the secret portion).
type APIKey struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	KeyPrefix string     `json:"key_prefix"` // first 8 chars for identification
	Scopes    []string   `json:"scopes"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// CreateAPIKeyRequest is the request body for creating a new API key.
type CreateAPIKeyRequest struct {
	Name   string   `json:"name"`
	Scopes []string `json:"scopes"`
}

// CreateAPIKeyResponse includes the full key, shown only once at creation time.
type CreateAPIKeyResponse struct {
	APIKey
	Key string `json:"key"` // full key, shown only once
}
