// Package hub 提供 WebSocket 連線管理中心
package hub

import (
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// DefaultHubKey 是 Redis 頻道的預設名稱
const DefaultHubKey = "push:channel:all"

// Hub 代表 WebSocket 連線管理中心
type Hub struct {
	clients   map[string]*Client
	clientsMu sync.RWMutex
	hubKey    string
}

// Client 代表一個 WebSocket 連線客戶端
type Client struct {
	id   string
	conn *websocket.Conn
	send chan []byte
}

// New 創建新的 Hub 實例
func New() *Hub {
	return &Hub{
		clients: make(map[string]*Client),
		hubKey:  DefaultHubKey,
	}
}

// ServeConn 處理單一 WebSocket 連線
func (h *Hub) ServeConn(upgrader *websocket.Upgrader, w http.ResponseWriter, r *http.Request) error {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return err
	}

	client := &Client{
		id:   conn.RemoteAddr().String(),
		conn: conn,
		send: make(chan []byte, 256),
	}

	h.addClient(client)
	defer h.removeClient(client)

	// 啟動讀取 Go-routine
	go readPump(client)

	// 啟動寫入 Go-routine
	writePump(client)

	return nil
}

// addClient 加入新的客戶端 (加鎖版本)
func (h *Hub) addClient(c *Client) {
	h.clientsMu.Lock()
	h.clients[c.id] = c
	h.clientsMu.Unlock()
}

// removeClient 移除客戶端 (加鎖版本)
func (h *Hub) removeClient(c *Client) {
	h.clientsMu.Lock()
	delete(h.clients, c.id)
	h.clientsMu.Unlock()
}

// readPump 讀取客戶端發送的訊息 (通常用於心跳)
func readPump(client *Client) {
	defer func() {
		client.conn.Close()
		log.Println("[HUB] readPump 退出")
	}()

	for {
		_, _, err := client.conn.ReadMessage()
		if err != nil {
			log.Printf("[HUB] 讀取錯誤：%v", err)
			break
		}
	}
}

// writePump 寫入訊息給客戶端
func writePump(client *Client) {
	defer func() {
		if err := client.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")); err != nil {
			log.Printf("[HUB] 關閉連線錯誤：%v", err)
		}
		client.conn.Close()
		log.Println("[HUB] writePump 退出")
	}()

	for msg := range client.send {
		if err := client.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			log.Printf("[HUB] 寫入錯誤：%v", err)
			return
		}
	}
}

// Broadcast 廣播訊息給所有連線的客戶端
func (h *Hub) Broadcast(msg []byte) {
	h.clientsMu.RLock()
	defer h.clientsMu.RUnlock()

	for _, client := range h.clients {
		select {
		case client.send <- msg:
		default:
			log.Printf("[HUB] 客戶端 %s 訊息遺失", client.id)
		}
	}
}

// ClientCount 返回連線客戶端數量
func (h *Hub) ClientCount() int {
	h.clientsMu.RLock()
	defer h.clientsMu.RUnlock()
	return len(h.clients)
}

// Close 關閉所有客戶端連線
func (h *Hub) Close() {
	h.clientsMu.Lock()
	defer h.clientsMu.Unlock()

	for _, client := range h.clients {
		_ = client.conn.Close()
	}
	h.clients = make(map[string]*Client)
}

// ClearAllClients 清除所有客戶端並重置 map
func (h *Hub) ClearAllClients() {
	h.clientsMu.Lock()
	h.clients = make(map[string]*Client)
	h.clientsMu.Unlock()
}
