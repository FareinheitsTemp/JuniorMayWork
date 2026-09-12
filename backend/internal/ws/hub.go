// Пакет ws: WebSocket-хаб для розсилки подій у реальному часі.
package ws

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"

	"github.com/FareinheitsTemp/JuniorMayWork/backend/internal/model"
)

type Hub struct {
	mu      sync.Mutex
	clients map[*websocket.Conn]struct{}
	origins []string
	log     *slog.Logger
}

func NewHub(log *slog.Logger, allowedOrigins []string) *Hub {
	return &Hub{
		clients: make(map[*websocket.Conn]struct{}),
		origins: allowedOrigins,
		log:     log,
	}
}

// Notify реалізує scraper.Notifier: подія летить усім підписникам /ws.
func (h *Hub) Notify(event model.Event) {
	blob, err := json.Marshal(event)
	if err != nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for conn := range h.clients {
		wctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := conn.Write(wctx, websocket.MessageText, blob)
		cancel()
		if err != nil {
			delete(h.clients, conn)
			_ = conn.CloseNow()
		}
	}
}

// Handler — точка підключення /ws.
func (h *Hub) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{OriginPatterns: h.origins})
		if err != nil {
			h.log.Warn("websocket accept", "err", err)
			return
		}
		h.mu.Lock()
		h.clients[conn] = struct{}{}
		h.mu.Unlock()
		go h.readLoop(conn)
	}
}

// readLoop тримає з'єднання живим, поки клієнт не відключиться.
func (h *Hub) readLoop(conn *websocket.Conn) {
	for {
		if _, _, err := conn.Read(context.Background()); err != nil {
			h.mu.Lock()
			delete(h.clients, conn)
			h.mu.Unlock()
			_ = conn.CloseNow()
			return
		}
	}
}
