package redisclient

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
)

// New constructs a Redis client using the provided configuration.
func New(cfg config.RedisConfig) *redis.Client {
	opts, err := redis.ParseURL(cfg.URL)
	if err != nil {
		// Fall back to manual parsing; ParseURL fails for unix sockets, so allow direct options.
		opts = &redis.Options{
			Addr: cfg.URL,
		}
	}

	if cfg.DB != 0 {
		opts.DB = cfg.DB
	}
	if cfg.PoolSize > 0 {
		opts.PoolSize = cfg.PoolSize
	}

	client := redis.NewClient(opts)
	client.AddHook(&disableMaintNotifications{})
	return client
}

// Ping verifies connectivity to Redis with a short timeout.
func Ping(ctx context.Context, client *redis.Client) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if err := client.Ping(timeoutCtx).Err(); err != nil {
		return fmt.Errorf("ping redis: %w", err)
	}
	return nil
}

type disableMaintNotifications struct{}

func (h *disableMaintNotifications) DialHook(next redis.DialHook) redis.DialHook { return next }

func (h *disableMaintNotifications) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		if strings.EqualFold(cmd.FullName(), "client") && len(cmd.Args()) > 1 {
			if name, ok := cmd.Args()[1].(string); ok && strings.EqualFold(name, "maint_notifications") {
				return nil
			}
		}
		return next(ctx, cmd)
	}
}

func (h *disableMaintNotifications) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		filtered := cmds[:0]
		for _, cmd := range cmds {
			if strings.EqualFold(cmd.FullName(), "client") && len(cmd.Args()) > 1 {
				if name, ok := cmd.Args()[1].(string); ok && strings.EqualFold(name, "maint_notifications") {
					continue
				}
			}
			filtered = append(filtered, cmd)
		}
		return next(ctx, filtered)
	}
}
