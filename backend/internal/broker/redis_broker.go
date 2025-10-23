package broker

import (
	"context"
	"encoding/json"

	"github.com/Baaaki/digital-square/internal/models"
	"github.com/redis/go-redis/v9"
)

// RedisMessageBroker implements MessageBroker interface for caching
// Phase 1-2: Cache only (single node)
// Phase 3: Pub/Sub will be added for multi-node deployment
type RedisMessageBroker struct {
	client *redis.Client
	ctx    context.Context
}

func NewRedisMessageBroker(redisURL string) (*RedisMessageBroker, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(opt)

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &RedisMessageBroker{
		client: client,
		ctx:    ctx,
	}, nil
}

func (r *RedisMessageBroker) Close() error {
	return r.client.Close()
}

// CacheMessage stores message in Redis list (last 100 messages)
func (r *RedisMessageBroker) CacheMessage(msg models.Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	if err := r.client.LPush(r.ctx, "global:recent", data).Err(); err != nil {
		return err
	}

	return r.client.LTrim(r.ctx, "global:recent", 0, 99).Err()
}

// GetRecentMessages retrieves last N messages from Redis cache
func (r *RedisMessageBroker) GetRecentMessages(limit int) ([]models.Message, error) {
	results, err := r.client.LRange(r.ctx, "global:recent", 0, int64(limit-1)).Result()
	if err != nil {
		return nil, err
	}

	messages := make([]models.Message, 0, len(results))
	for _, data := range results {
		var msg models.Message
		if err := json.Unmarshal([]byte(data), &msg); err != nil {
			continue
		}
		messages = append(messages, msg)
	}

	return messages, nil
}
