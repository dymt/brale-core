package initexit

import (
	"fmt"
	"strings"
	"sync"
)

type Registry struct {
	mu       sync.RWMutex
	policies map[string]Policy
}

var defaultRegistry = NewDefaultRegistry()

func NewRegistry() *Registry {
	return &Registry{policies: map[string]Policy{}}
}

func NewDefaultRegistry() *Registry {
	reg := NewRegistry()
	for _, policy := range []Policy{
		atrStructureV1Policy{},
		fixedRRV1Policy{},
		structureTPV1Policy{},
	} {
		_ = reg.Register(policy)
	}
	return reg
}

func (r *Registry) Register(policy Policy) error {
	if policy == nil {
		return fmt.Errorf("initexit: nil policy")
	}
	name := normalizePolicyName(policy.Name())
	if name == "" {
		return fmt.Errorf("initexit: empty policy name")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.policies[name]; exists {
		return fmt.Errorf("initexit: duplicate policy: %s", name)
	}
	r.policies[name] = policy
	return nil
}

func Register(policy Policy) error {
	return defaultRegistry.Register(policy)
}

func (r *Registry) Get(name string) (Policy, bool) {
	key := normalizePolicyName(name)
	if key == "" {
		return nil, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.policies[key]
	return p, ok
}

func Get(name string) (Policy, bool) {
	return defaultRegistry.Get(name)
}

func (r *Registry) MustGet(name string) (Policy, error) {
	p, ok := r.Get(name)
	if ok {
		return p, nil
	}
	return nil, fmt.Errorf("initial exit policy not found: %s", strings.TrimSpace(name))
}

func MustGet(name string) (Policy, error) {
	return defaultRegistry.MustGet(name)
}

func normalizePolicyName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
