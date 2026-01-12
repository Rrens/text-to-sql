package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Rrens/text-to-sql/internal/domain"
	"github.com/google/uuid"
)

const (
	schemaCachePrefix = "schema:"
	schemaCacheTTL    = 5 * time.Minute
)

// SchemaCache handles schema caching in Redis
type SchemaCache struct {
	client *Client
}

// NewSchemaCache creates a new schema cache
func NewSchemaCache(client *Client) *SchemaCache {
	return &SchemaCache{client: client}
}

// Get retrieves cached schema for a connection
func (c *SchemaCache) Get(ctx context.Context, connectionID uuid.UUID) (*domain.SchemaInfo, error) {
	key := fmt.Sprintf("%s%s", schemaCachePrefix, connectionID.String())

	data, err := c.client.rdb.Get(ctx, key).Bytes()
	if err != nil {
		return nil, nil // Cache miss
	}

	var schema domain.SchemaInfo
	if err := json.Unmarshal(data, &schema); err != nil {
		return nil, fmt.Errorf("failed to unmarshal schema: %w", err)
	}

	return &schema, nil
}

// Set caches schema for a connection
func (c *SchemaCache) Set(ctx context.Context, connectionID uuid.UUID, schema *domain.SchemaInfo) error {
	key := fmt.Sprintf("%s%s", schemaCachePrefix, connectionID.String())

	data, err := json.Marshal(schema)
	if err != nil {
		return fmt.Errorf("failed to marshal schema: %w", err)
	}

	return c.client.rdb.Set(ctx, key, data, schemaCacheTTL).Err()
}

// Invalidate removes cached schema for a connection
func (c *SchemaCache) Invalidate(ctx context.Context, connectionID uuid.UUID) error {
	key := fmt.Sprintf("%s%s", schemaCachePrefix, connectionID.String())
	return c.client.rdb.Del(ctx, key).Err()
}

// FlushAll removes all cached schemas
func (c *SchemaCache) FlushAll(ctx context.Context) (int64, error) {
	pattern := schemaCachePrefix + "*"
	var cursor uint64
	var deleted int64

	for {
		keys, nextCursor, err := c.client.rdb.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return deleted, fmt.Errorf("failed to scan keys: %w", err)
		}

		if len(keys) > 0 {
			count, err := c.client.rdb.Del(ctx, keys...).Result()
			if err != nil {
				return deleted, fmt.Errorf("failed to delete keys: %w", err)
			}
			deleted += count
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return deleted, nil
}
