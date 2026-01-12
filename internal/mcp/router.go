package mcp

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
)

// Router manages database adapters and connection pooling
type Router struct {
	factories map[string]AdapterFactory
	pool      map[string]Adapter
	mu        sync.RWMutex
}

// NewRouter creates a new adapter router
func NewRouter() *Router {
	return &Router{
		factories: make(map[string]AdapterFactory),
		pool:      make(map[string]Adapter),
	}
}

// RegisterAdapter registers an adapter factory for a database type
func (r *Router) RegisterAdapter(dbType string, factory AdapterFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.factories[dbType] = factory
}

// SupportedDatabases returns list of supported database types
func (r *Router) SupportedDatabases() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	types := make([]string, 0, len(r.factories))
	for dbType := range r.factories {
		types = append(types, dbType)
	}
	return types
}

// GetAdapter returns an adapter for the given connection, creating if needed
func (r *Router) GetAdapter(ctx context.Context, connectionID uuid.UUID, dbType string, config ConnectionConfig) (Adapter, error) {
	connKey := connectionID.String()

	// Check for existing healthy connection
	r.mu.RLock()
	if adapter, ok := r.pool[connKey]; ok {
		r.mu.RUnlock()
		if err := adapter.HealthCheck(ctx); err == nil {
			return adapter, nil
		}
		// Connection unhealthy, will recreate
		r.mu.Lock()
		adapter.Close()
		delete(r.pool, connKey)
		r.mu.Unlock()
	} else {
		r.mu.RUnlock()
	}

	// Create new connection
	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check after acquiring write lock
	if adapter, ok := r.pool[connKey]; ok {
		if err := adapter.HealthCheck(ctx); err == nil {
			return adapter, nil
		}
		adapter.Close()
		delete(r.pool, connKey)
	}

	factory, ok := r.factories[dbType]
	if !ok {
		return nil, fmt.Errorf("unsupported database type: %s", dbType)
	}

	adapter := factory()
	if err := adapter.Connect(ctx, config); err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	r.pool[connKey] = adapter
	return adapter, nil
}

// CloseConnection closes a specific connection
func (r *Router) CloseConnection(connectionID uuid.UUID) error {
	connKey := connectionID.String()

	r.mu.Lock()
	defer r.mu.Unlock()

	if adapter, ok := r.pool[connKey]; ok {
		err := adapter.Close()
		delete(r.pool, connKey)
		return err
	}

	return nil
}

// CloseAll closes all connections
func (r *Router) CloseAll() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for connKey, adapter := range r.pool {
		adapter.Close()
		delete(r.pool, connKey)
	}
}

// PoolSize returns the current number of pooled connections
func (r *Router) PoolSize() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.pool)
}
