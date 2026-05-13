// Package main 是 WebSocket 推播伺服器的入口點
package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/g0382/websocket-push-center/internal/hub"
	"github.com/g0382/websocket-push-center/internal/server"
	"github.com/g0382/websocket-push-center/pkg/redis"
	"github.com/gorilla/websocket"
)

func main() {
	// 解析命令行參數
	addr := flag.String("addr", ":8080", "HTTP 伺服器監聽地址 (例如 :8080)")
	redisAddr := flag.String("redis-addr", "redis:6379", "Redis 連線地址 (例如 localhost:6379 或 192.168.1.100:6379)")
	flag.Parse()

	log.SetFlags(0)
	log.SetOutput(os.Stdout)

	log.Println("[INFO] 🚀 啟動高併發 WebSocket 即時推播伺服器")

	// 創建 Hub
	h := hub.New()

	// 創建 Redis 客戶端
	// 預設使用 Mock 模式（適合開發測試，不需要 Redis）
	// 若提供有效的 Redis 地址，則嘗試連線
	var redisClient redis.ClientInterface

	if *redisAddr != "" {
		client, err := redis.New(*redisAddr)
		if err == nil {
			// 成功連線到 Redis
			redisClient = client
			log.Println("[INFO] 已連線到 Redis")
		} else {
			// Redis 連線失敗，繼續使用 Mock 模式
			log.Printf("[WARN] 無法連線到 Redis (Mock 模式): %v", err)
		}
	}

	// 若尚未初始化，使用 Mock 客戶端
	if redisClient == nil {
		redisClient = redis.NewMock()
	} else {
		defer func() {
			if err := redisClient.Close(); err != nil {
				log.Printf("[WARN] 關閉 Redis 時發生錯誤：%v", err)
			}
		}()
	}

	// 創建 WebSocket Upgrader (處理 CORS)
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // 開發環境允許所有來源
		},
	}

	// 創建 Server 實例
	svr := server.NewServer(h, &upgrader)

	// 啟動背景 go-routine 監聽訊號
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 啟動 HTTP 伺服器
	server := &http.Server{
		Addr:         *addr,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 60 * time.Second,
		Handler:      svr,
	}

	go func() {
		log.Printf("[INFO] 伺服器啟動於 %s", *addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[ERROR] 伺服器錯誤：%v", err)
		}
	}()

	// 等待使用者輸入或訊號
	log.Println("[INFO] 💡 按 Ctrl+C 以關閉伺服器")
	<-sigChan

	// 優雅關機
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Println("[SERVER] 開始優雅關機...")
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("[WARN] Shutdown 錯誤：%v", err)
	}

	// 關閉所有 WebSocket 連線
	h.Close()

	log.Println("[INFO] 👋 伺服器已停止")
}
