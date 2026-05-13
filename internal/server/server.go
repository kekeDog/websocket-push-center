// Package server 提供 WebSocket 伺服器實作
package server

import (
	"log"
	"net/http"

	"github.com/g0382/websocket-push-center/internal/hub"
	"github.com/gorilla/websocket"
)

// Server 代表 WebSocket 伺服器
type Server struct {
	hub      *hub.Hub
	upgrader *websocket.Upgrader
}

// NewServer 創建新的伺服器
func NewServer(h *hub.Hub, upgrader *websocket.Upgrader) *Server {
	return &Server{
		hub:      h,
		upgrader: upgrader,
	}
}

// ServeHTTP 實作 http.Handler 介面
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/ws":
		s.handleWebSocket(w, r)
	case "/health":
		s.handleHealth(w, r)
	case "/metrics":
		s.handleMetrics(w, r)
	default:
		http.NotFound(w, r)
	}
}

// handleWebSocket 處理 WebSocket 連線
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	if err := s.hub.ServeConn(s.upgrader, w, r); err != nil {
		log.Printf("[ERROR] WebSocket 錯誤：%v", err)
	}
}

// handleHealth 健康檢查端點
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

// handleMetrics 匯出目前連線數
func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"connected_clients":` + string(rune(s.hub.ClientCount())) + `}`))
}
