package broker

import (
	"context"
	"encoding/json"

	"github.com/redis/go-redis/v9"
)

//RedisMessageBroker implements MessageBroker interface using pub/sub
type RedisMessageBroker struct{
	client *redis.Client
	pubsub *redis.PubSub
	ctx context.Context
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
		ctx: ctx,
	}, nil
}

func (r *RedisMessageBroker) Publish(msg Message) error{
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return r.client.Publish(r.ctx, "chat:global", data).Err()
}

func (r *RedisMessageBroker) Subscribe() (<-chan Message, error) {
	r.pubsub = r.client.Subscribe(r.ctx, "chat:global")

	msgChan := make(chan Message, 100)

	go func() {
		defer close(msgChan)

		for redisMsg := range r.pubsub.Channel() {
			var msg Message

			if err := json.Unmarshal([]byte(redisMsg.Payload), &msg); err != nil {
				continue
			}

			msgChan <- msg
		}
	}()

	return msgChan, nil
}

func (r *RedisMessageBroker) Close() error {
	if r.pubsub != nil {
		r.pubsub.Close()
	}
	return r.client.Close()
}