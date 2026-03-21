package store

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/nicroldan/ans/shared/ace"
)

// MemoryStore provides thread-safe in-memory storage for all ACE entities.
type MemoryStore struct {
	mu         sync.RWMutex
	products   map[string]*ace.Product
	carts      map[string]*ace.Cart
	orders     map[string]*ace.Order
	payments   map[string]*ace.Payment
	policies   map[string]*ace.Policy // keyed by action
	approvals  map[string]*ace.Approval
	auditLog   []*ace.AuditEntry
	apiKeys    map[string]*StoredAPIKey
	counters   map[string]int
}

// StoredAPIKey holds an API key with its hash for validation.
type StoredAPIKey struct {
	ace.APIKey
	KeyHash string
}

// New creates a new MemoryStore.
func New() *MemoryStore {
	return &MemoryStore{
		products:  make(map[string]*ace.Product),
		carts:     make(map[string]*ace.Cart),
		orders:    make(map[string]*ace.Order),
		payments:  make(map[string]*ace.Payment),
		policies:  make(map[string]*ace.Policy),
		approvals: make(map[string]*ace.Approval),
		apiKeys:   make(map[string]*StoredAPIKey),
		counters:  make(map[string]int),
	}
}

func (s *MemoryStore) nextID(prefix string) string {
	s.counters[prefix]++
	return fmt.Sprintf("%s%d", prefix, s.counters[prefix])
}

func generateRandomKey() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func hashKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}

// --- Products ---

func (s *MemoryStore) CreateProduct(p *ace.Product) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p.ID = s.nextID("prod_")
	now := time.Now()
	p.CreatedAt = now
	p.UpdatedAt = now
	if p.Status == "" {
		p.Status = "draft"
	}
	s.products[p.ID] = p
}

func (s *MemoryStore) GetProduct(id string) (*ace.Product, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.products[id]
	return p, ok
}

func (s *MemoryStore) UpdateProduct(id string, updates map[string]any) (*ace.Product, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.products[id]
	if !ok {
		return nil, false
	}
	if name, ok := updates["name"].(string); ok {
		p.Name = name
	}
	if desc, ok := updates["description"].(string); ok {
		p.Description = desc
	}
	if status, ok := updates["status"].(string); ok {
		p.Status = status
	}
	if price, ok := updates["price"].(ace.Money); ok {
		p.Price = price
	}
	if variants, ok := updates["variants"].([]ace.Variant); ok {
		p.Variants = variants
	}
	p.UpdatedAt = time.Now()
	return p, true
}

func (s *MemoryStore) DeleteProduct(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.products[id]; !ok {
		return false
	}
	delete(s.products, id)
	return true
}

func (s *MemoryStore) ListProducts(status, category, query string, offset, limit int) ([]ace.Product, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var filtered []ace.Product
	for _, p := range s.products {
		if status != "" && p.Status != status {
			continue
		}
		if query != "" {
			q := strings.ToLower(query)
			if !strings.Contains(strings.ToLower(p.Name), q) &&
				!strings.Contains(strings.ToLower(p.Description), q) {
				continue
			}
		}
		if category != "" {
			// Check if any variant attribute matches category, or product name/description contains it
			found := false
			for _, v := range p.Variants {
				if cat, ok := v.Attributes["category"]; ok && strings.EqualFold(cat, category) {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		filtered = append(filtered, *p)
	}

	total := len(filtered)
	if offset >= total {
		return []ace.Product{}, total
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return filtered[offset:end], total
}

func (s *MemoryStore) SetProductStatus(id, status string) (*ace.Product, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.products[id]
	if !ok {
		return nil, false
	}
	p.Status = status
	p.UpdatedAt = time.Now()
	return p, true
}

// UpdateVariantInventory updates the inventory for a specific variant.
func (s *MemoryStore) UpdateVariantInventory(variantID string, inventory int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, p := range s.products {
		for i, v := range p.Variants {
			if v.ID == variantID {
				p.Variants[i].Inventory = inventory
				p.UpdatedAt = time.Now()
				return true
			}
		}
	}
	return false
}

// DecrementInventory reduces inventory for given product/variant by qty. Returns false if insufficient.
func (s *MemoryStore) DecrementInventory(productID, variantID string, qty int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.products[productID]
	if !ok {
		return false
	}
	if variantID != "" {
		for i, v := range p.Variants {
			if v.ID == variantID {
				if v.Inventory < qty {
					return false
				}
				p.Variants[i].Inventory -= qty
				return true
			}
		}
		return false
	}
	// No variant specified: decrement first variant
	if len(p.Variants) > 0 {
		if p.Variants[0].Inventory < qty {
			return false
		}
		p.Variants[0].Inventory -= qty
		return true
	}
	return false
}

// --- Carts ---

func (s *MemoryStore) CreateCart() *ace.Cart {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	c := &ace.Cart{
		ID:        s.nextID("cart_"),
		Items:     []ace.CartItem{},
		Total:     ace.Money{Amount: 0, Currency: "USD"},
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.carts[c.ID] = c
	return c
}

func (s *MemoryStore) GetCart(id string) (*ace.Cart, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.carts[id]
	return c, ok
}

func (s *MemoryStore) AddCartItem(cartID string, item ace.CartItem) (*ace.Cart, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.carts[cartID]
	if !ok {
		return nil, false
	}
	// Check if same product/variant already in cart
	found := false
	for i, existing := range c.Items {
		if existing.ProductID == item.ProductID && existing.VariantID == item.VariantID {
			c.Items[i].Quantity += item.Quantity
			found = true
			break
		}
	}
	if !found {
		c.Items = append(c.Items, item)
	}
	// Recalculate total
	var total int64
	for _, it := range c.Items {
		total += it.Price.Amount * int64(it.Quantity)
	}
	c.Total.Amount = total
	c.UpdatedAt = time.Now()
	return c, true
}

// --- Orders ---

func (s *MemoryStore) CreateOrder(o *ace.Order) {
	s.mu.Lock()
	defer s.mu.Unlock()
	o.ID = s.nextID("ord_")
	now := time.Now()
	o.CreatedAt = now
	o.UpdatedAt = now
	s.orders[o.ID] = o
}

func (s *MemoryStore) GetOrder(id string) (*ace.Order, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	o, ok := s.orders[id]
	return o, ok
}

func (s *MemoryStore) ListOrders(offset, limit int) ([]ace.Order, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	all := make([]ace.Order, 0, len(s.orders))
	for _, o := range s.orders {
		all = append(all, *o)
	}
	total := len(all)
	if offset >= total {
		return []ace.Order{}, total
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return all[offset:end], total
}

func (s *MemoryStore) SetOrderStatus(id, status string) (*ace.Order, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, ok := s.orders[id]
	if !ok {
		return nil, false
	}
	o.Status = status
	o.UpdatedAt = time.Now()
	return o, true
}

func (s *MemoryStore) SetOrderPayment(orderID string, payment *ace.Payment) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, ok := s.orders[orderID]
	if !ok {
		return false
	}
	o.Payment = payment
	o.UpdatedAt = time.Now()
	return true
}

// --- Payments ---

func (s *MemoryStore) CreatePayment(p *ace.Payment) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p.ID = s.nextID("pay_")
	p.CreatedAt = time.Now()
	s.payments[p.ID] = p
}

func (s *MemoryStore) GetPayment(id string) (*ace.Payment, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.payments[id]
	return p, ok
}

func (s *MemoryStore) GetPaymentByOrderID(orderID string) (*ace.Payment, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, p := range s.payments {
		if p.OrderID == orderID {
			return p, true
		}
	}
	return nil, false
}

func (s *MemoryStore) SetPaymentStatus(id, status string) (*ace.Payment, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.payments[id]
	if !ok {
		return nil, false
	}
	p.Status = status
	return p, true
}

// --- Policies ---

func (s *MemoryStore) GetPolicies() []ace.Policy {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]ace.Policy, 0, len(s.policies))
	for _, p := range s.policies {
		result = append(result, *p)
	}
	return result
}

func (s *MemoryStore) SetPolicies(policies []ace.Policy) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.policies = make(map[string]*ace.Policy)
	for i := range policies {
		p := policies[i]
		if p.ID == "" {
			p.ID = s.nextID("pol_")
		}
		s.policies[p.Action] = &p
	}
}

func (s *MemoryStore) GetPolicyForAction(action string) (*ace.Policy, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.policies[action]
	return p, ok
}

// --- Approvals ---

func (s *MemoryStore) CreateApproval(a *ace.Approval) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a.ID = s.nextID("apr_")
	a.CreatedAt = time.Now()
	a.Status = "pending"
	s.approvals[a.ID] = a
}

func (s *MemoryStore) GetApproval(id string) (*ace.Approval, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	a, ok := s.approvals[id]
	return a, ok
}

func (s *MemoryStore) ListPendingApprovals() []ace.Approval {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []ace.Approval
	for _, a := range s.approvals {
		if a.Status == "pending" {
			result = append(result, *a)
		}
	}
	return result
}

func (s *MemoryStore) ResolveApproval(id, status, resolvedBy string) (*ace.Approval, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.approvals[id]
	if !ok {
		return nil, false
	}
	a.Status = status
	a.ResolvedBy = resolvedBy
	now := time.Now()
	a.ResolvedAt = &now
	return a, true
}

// --- Audit Log ---

func (s *MemoryStore) AppendAuditEntry(entry *ace.AuditEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry.ID = s.nextID("aud_")
	entry.Timestamp = time.Now()
	s.auditLog = append(s.auditLog, entry)
}

func (s *MemoryStore) QueryAuditLog(storeID, action, actor string, offset, limit int) ([]ace.AuditEntry, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var filtered []ace.AuditEntry
	for _, e := range s.auditLog {
		if storeID != "" && e.StoreID != storeID {
			continue
		}
		if action != "" && e.Action != action {
			continue
		}
		if actor != "" && e.Actor != actor {
			continue
		}
		filtered = append(filtered, *e)
	}
	total := len(filtered)
	if offset >= total {
		return []ace.AuditEntry{}, total
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return filtered[offset:end], total
}

// --- API Keys ---

func (s *MemoryStore) CreateAPIKey(name string, scopes []string) (ace.CreateAPIKeyResponse, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rawKey := generateRandomKey()
	keyHash := hashKey(rawKey)
	id := s.nextID("key_")
	now := time.Now()
	prefix := rawKey[:8]

	stored := &StoredAPIKey{
		APIKey: ace.APIKey{
			ID:        id,
			Name:      name,
			KeyPrefix: prefix,
			Scopes:    scopes,
			CreatedAt: now,
		},
		KeyHash: keyHash,
	}
	s.apiKeys[id] = stored

	resp := ace.CreateAPIKeyResponse{
		APIKey: stored.APIKey,
		Key:    rawKey,
	}
	return resp, rawKey
}

// CreateAPIKeyWithValue creates an API key with a specific raw key value (for demo seeding).
func (s *MemoryStore) CreateAPIKeyWithValue(name string, scopes []string, rawKey string) ace.CreateAPIKeyResponse {
	s.mu.Lock()
	defer s.mu.Unlock()
	keyHash := hashKey(rawKey)
	id := s.nextID("key_")
	now := time.Now()
	prefix := rawKey[:8]

	stored := &StoredAPIKey{
		APIKey: ace.APIKey{
			ID:        id,
			Name:      name,
			KeyPrefix: prefix,
			Scopes:    scopes,
			CreatedAt: now,
		},
		KeyHash: keyHash,
	}
	s.apiKeys[id] = stored

	return ace.CreateAPIKeyResponse{
		APIKey: stored.APIKey,
		Key:    rawKey,
	}
}

func (s *MemoryStore) ListAPIKeys() []ace.APIKey {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]ace.APIKey, 0, len(s.apiKeys))
	for _, k := range s.apiKeys {
		result = append(result, k.APIKey)
	}
	return result
}

func (s *MemoryStore) DeleteAPIKey(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.apiKeys[id]; !ok {
		return false
	}
	delete(s.apiKeys, id)
	return true
}

func (s *MemoryStore) ValidateAPIKey(rawKey string) (*ace.APIKey, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	h := hashKey(rawKey)
	for _, k := range s.apiKeys {
		if k.KeyHash == h {
			key := k.APIKey
			return &key, true
		}
	}
	return nil, false
}
