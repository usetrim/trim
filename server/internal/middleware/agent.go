package middleware

import (
	"context"
	"fmt"
	"strings"

	"github.com/redis/go-redis/v9"
)

const agentCatalogCacheKey = "agent:catalog:ids"

// agentIDAllowed reports whether id is in public.agent_identity_catalog.
// Fail closed: empty id, DB miss, or error → false.
func (m *AuthQuota) agentIDAllowed(ctx context.Context, id string) (bool, error) {
	id = strings.ToLower(strings.TrimSpace(id))
	if id == "" {
		return false, nil
	}
	if m.Redis != nil {
		ok, err := m.Redis.SIsMember(ctx, agentCatalogCacheKey, id).Result()
		if err == nil && ok {
			return true, nil
		}
		// Warm cache on miss / empty set.
		if err == nil || err == redis.Nil {
			if err := m.warmAgentCatalogCache(ctx); err == nil {
				ok, err = m.Redis.SIsMember(ctx, agentCatalogCacheKey, id).Result()
				if err == nil {
					return ok, nil
				}
			}
		}
	}
	if m.DB == nil {
		return false, fmt.Errorf("database unavailable")
	}
	var exists bool
	err := m.DB.QueryRow(ctx, `
		select exists(select 1 from public.agent_identity_catalog where id = $1)
	`, id).Scan(&exists)
	return exists, err
}

func (m *AuthQuota) warmAgentCatalogCache(ctx context.Context) error {
	if m.Redis == nil || m.DB == nil {
		return fmt.Errorf("unavailable")
	}
	rows, err := m.DB.Query(ctx, `select id from public.agent_identity_catalog`)
	if err != nil {
		return err
	}
	defer rows.Close()
	ids := make([]interface{}, 0, 8)
	for rows.Next() {
		var id string
		if rows.Scan(&id) != nil || id == "" {
			continue
		}
		ids = append(ids, id)
	}
	_ = m.Redis.Del(ctx, agentCatalogCacheKey).Err()
	if len(ids) == 0 {
		return nil
	}
	if err := m.Redis.SAdd(ctx, agentCatalogCacheKey, ids...).Err(); err != nil {
		return err
	}
	// Soft TTL so catalog edits eventually show without restart.
	ttl := m.redisTTL(m.Config.RedisIPRiskTTLSec)
	if ttl > 0 {
		_ = m.Redis.Expire(ctx, agentCatalogCacheKey, ttl).Err()
	}
	return nil
}
