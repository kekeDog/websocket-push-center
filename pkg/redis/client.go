// Package redis 提供 Redis 客戶端和 Mock 客戶端
package redis

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// ClientInterface 定義 Redis 客戶端的統一介面
type ClientInterface interface {
	Close() error
	Publish(channel string, message interface{}) error
}

// Client 代表真實 Redis 客戶端
type Client struct {
	client *redis.Client
}

// MockClient 代表測試用的 Mock 客戶端
type MockClient struct {
	log *log.Logger
}

// New 創建新的 Redis 客戶端 (只提供 addr 即可，db 選填)
func New(addr string, db ...string) (ClientInterface, error) {
	// 建立預設選項
	opts := &redis.Options{
		Addr:           addr,
		DialTimeout:    5 * time.Second,
		MaxIdleConns:   10,
		MaxActiveConns: 100,
		PoolTimeout:    4 * time.Second,
	}

	// 如果有提供 db 參數，設定資料庫
	if len(db) > 0 && db[0] != "" {
		opts.DB = 0
	}

	// 建立客戶端
	client := redis.NewClient(opts)

	// 測試連接
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if _, err := client.Ping(ctx).Result(); err != nil {
		return nil, fmt.Errorf("無法連接 Redis: %v", err)
	}

	return &Client{client: client}, nil
}

// NewMock 創建 Mock 客戶端 (測試用，不需要 Redis 連線)
func NewMock() *MockClient {
	log := log.Default()
	return &MockClient{log: log}
}

// Close 關閉 Redis 連接
func (c *Client) Close() error {
	if c.client == nil {
		return nil
	}
	if err := c.client.Close(); err != nil {
		return err
	}
	log.Println("[REDIS] 已關閉連接")
	return nil
}

func (m *MockClient) Close() error {
	return nil
}

// Publish 發佈訊息到 Redis 頻道
func (c *Client) Publish(channel string, message interface{}) error {
	if c.client == nil {
		log.Println("[REDIS] Mock 模式：模擬發佈")
		return nil
	}
	return c.client.Publish(context.Background(), channel, message).Err()
}

// Mock 客戶端的方法 (無实际操作)
func (m *MockClient) Publish(channel string, message interface{}) error {
	m.log.Printf("[MOCK] 模擬發佈：%s = %v", channel, message)
	return nil
}
