package llm

import (
	"fmt"
	"sync"
)

// Router manages LLM providers and routing
type Router struct {
	providers       map[string]Provider
	factories       map[string]ProviderFactory
	defaultProvider string
	mu              sync.RWMutex
}

// NewRouter creates a new LLM router
func NewRouter(defaultProvider string) *Router {
	return &Router{
		providers:       make(map[string]Provider),
		factories:       make(map[string]ProviderFactory),
		defaultProvider: defaultProvider,
	}
}

// RegisterProvider registers an LLM provider
func (r *Router) RegisterProvider(provider Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[provider.Name()] = provider
}

// RegisterFactory registers a provider factory
func (r *Router) RegisterFactory(name string, factory ProviderFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.factories[name] = factory
}

// GetProviderWithConfig returns a provider instance, potentially creating it from factory if config is provided
func (r *Router) GetProviderWithConfig(name string, config map[string]any) (Provider, error) {
	if name == "" {
		name = r.defaultProvider
	}

	r.mu.RLock()
	factory, hasFactory := r.factories[name]
	provider, hasProvider := r.providers[name]
	r.mu.RUnlock()

	// If config is provided and we have a factory, create new instance
	if len(config) > 0 && hasFactory {
		return factory(config)
	}

	// Fallback to registered instance
	if !hasProvider {
		return nil, fmt.Errorf("provider not found: %s", name)
	}

	if !provider.IsConfigured() {
		return nil, fmt.Errorf("provider not configured: %s", name)
	}

	return provider, nil
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
