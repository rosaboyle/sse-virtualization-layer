package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"virtualization-manager/pkg/config"
	"virtualization-manager/pkg/types"

	"github.com/go-redis/redis/v8"
)

type Client struct {
	rdb *redis.Client
	ctx context.Context
}

func NewClient(cfg config.RedisConfig) *Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	return &Client{
		rdb: rdb,
		ctx: context.Background(),
	}
}

// Connection management
func (c *Client) StoreConnection(conn *types.Connection) error {
	data, err := json.Marshal(conn)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("connections:%s", conn.ID)
	return c.rdb.Set(c.ctx, key, data, 24*time.Hour).Err()
}

func (c *Client) GetConnection(connectionID string) (*types.Connection, error) {
	key := fmt.Sprintf("connections:%s", connectionID)
	data, err := c.rdb.Get(c.ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var conn types.Connection
	err = json.Unmarshal([]byte(data), &conn)
	return &conn, err
}

func (c *Client) DeleteConnection(connectionID string) error {
	key := fmt.Sprintf("connections:%s", connectionID)
	return c.rdb.Del(c.ctx, key).Err()
}

func (c *Client) GetAllConnections() ([]*types.Connection, error) {
	keys, err := c.rdb.Keys(c.ctx, "connections:*").Result()
	if err != nil {
		return nil, err
	}

	var connections []*types.Connection
	for _, key := range keys {
		data, err := c.rdb.Get(c.ctx, key).Result()
		if err != nil {
			continue
		}

		var conn types.Connection
		if err := json.Unmarshal([]byte(data), &conn); err == nil {
			connections = append(connections, &conn)
		}
	}

	return connections, nil
}

// Function registry
func (c *Client) StoreFunction(fn *types.Function) error {
	data, err := json.Marshal(fn)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("functions:%s", fn.Name)
	return c.rdb.Set(c.ctx, key, data, 0).Err() // No expiration for functions
}

func (c *Client) GetFunction(name string) (*types.Function, error) {
	key := fmt.Sprintf("functions:%s", name)
	data, err := c.rdb.Get(c.ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var fn types.Function
	err = json.Unmarshal([]byte(data), &fn)
	return &fn, err
}

func (c *Client) GetAllFunctions() ([]*types.Function, error) {
	keys, err := c.rdb.Keys(c.ctx, "functions:*").Result()
	if err != nil {
		return nil, err
	}

	var functions []*types.Function
	for _, key := range keys {
		data, err := c.rdb.Get(c.ctx, key).Result()
		if err != nil {
			continue
		}

		var fn types.Function
		if err := json.Unmarshal([]byte(data), &fn); err == nil {
			functions = append(functions, &fn)
		}
	}

	return functions, nil
}

func (c *Client) DeleteFunction(name string) error {
	key := fmt.Sprintf("functions:%s", name)
	return c.rdb.Del(c.ctx, key).Err()
}

// Metrics and monitoring
func (c *Client) IncrementCounter(key string) error {
	return c.rdb.Incr(c.ctx, key).Err()
}

func (c *Client) SetMetric(key string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.rdb.Set(c.ctx, fmt.Sprintf("metrics:%s", key), data, time.Hour).Err()
}

func (c *Client) GetMetric(key string) (interface{}, error) {
	data, err := c.rdb.Get(c.ctx, fmt.Sprintf("metrics:%s", key)).Result()
	if err != nil {
		return nil, err
	}

	var value interface{}
	err = json.Unmarshal([]byte(data), &value)
	return value, err
}

// Health check
func (c *Client) Ping() error {
	return c.rdb.Ping(c.ctx).Err()
}

// Pub/Sub for real-time updates
func (c *Client) PublishMessage(channel string, message interface{}) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}
	return c.rdb.Publish(c.ctx, channel, data).Err()
}

func (c *Client) Subscribe(channel string) *redis.PubSub {
	return c.rdb.Subscribe(c.ctx, channel)
}
