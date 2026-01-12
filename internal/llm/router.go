package llm

import (
	"fmt"
	"sync"
)

// Router manages LLM providers and routing
type Router struct {
	providers       map[string]Provider
	defaultProvider string
	mu              sync.RWMutex
}

// NewRouter creates a new LLM router
func NewRouter(defaultProvider string) *Router {
	return &Router{
		providers:       make(map[string]Provider),
		defaultProvider: defaultProvider,
	}
}

// RegisterProvider registers an LLM provider
func (r *Router) RegisterProvider(provider Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[provider.Name()] = provider
}

// ListProviders returns list of configured provider names
func (r *Router) ListProviders() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var providers []string
	for name, p := range r.providers {
		if p.IsConfigured() {
			providers = append(providers, name)
		}
	}
	return providers
}

// GetProvider returns a provider by name
func (r *Router) GetProvider(name string) (Provider, error) {
	if name == "" {
		name = r.defaultProvider
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.providers[name]
	if !ok {
		return nil, fmt.Errorf("provider not found: %s", name)
	}

	if !p.IsConfigured() {
		return nil, fmt.Errorf("provider not configured: %s", name)
	}

	return p, nil
}

// DefaultProvider returns the default provider name
func (r *Router) DefaultProvider() string {
	return r.defaultProvider
}

// ProviderInfo contains information about an LLM provider
type ProviderInfo struct {
	Name       string   `json:"name"`
	Models     []string `json:"models"`
	Default    bool     `json:"default"`
	Configured bool     `json:"configured"`
}

// GetProvidersInfo returns information about all providers
func (r *Router) GetProvidersInfo() []ProviderInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var infos []ProviderInfo
	for name, p := range r.providers {
		infos = append(infos, ProviderInfo{
			Name:       name,
			Models:     p.AvailableModels(),
			Default:    name == r.defaultProvider,
			Configured: p.IsConfigured(),
		})
	}
	return infos
}
