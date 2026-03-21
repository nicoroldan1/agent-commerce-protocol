package policy

import (
	"github.com/nicroldan/ans/shared/ace"
	"github.com/nicroldan/ans/ace-server/internal/store"
)

// Result represents the outcome of a policy check.
type Result struct {
	Effect   string        // "allow", "deny", "needs_approval"
	Approval *ace.Approval // non-nil if needs_approval
}

// Engine evaluates policies for actions.
type Engine struct {
	store *store.MemoryStore
}

// NewEngine creates a new policy engine.
func NewEngine(s *store.MemoryStore) *Engine {
	return &Engine{store: s}
}

// DefaultPolicies returns the default policy set.
func DefaultPolicies() []ace.Policy {
	return []ace.Policy{
		{Action: "product.publish", Effect: "approval"},
		{Action: "order.refund", Effect: "approval"},
	}
}

// Check evaluates whether an action is allowed for the given actor.
// actorType is "human" or "agent".
func (e *Engine) Check(action, actor, actorType, resource string) Result {
	// Humans are always allowed
	if actorType == "human" {
		return Result{Effect: "allow"}
	}

	// Check if there's a specific policy for this action
	pol, found := e.store.GetPolicyForAction(action)
	if !found {
		// No explicit policy means allowed
		return Result{Effect: "allow"}
	}

	switch pol.Effect {
	case "deny":
		return Result{Effect: "deny"}
	case "approval":
		approval := &ace.Approval{
			Action:      action,
			Resource:    resource,
			RequestedBy: actor,
		}
		e.store.CreateApproval(approval)
		return Result{Effect: "needs_approval", Approval: approval}
	default:
		return Result{Effect: "allow"}
	}
}
