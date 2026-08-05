// Package ws implementa un hub simple de WebSockets para notificar en tiempo
// real al usuario dueño de una sesión cuando ocurre una transacción sobre
// sus cuentas (bonus: "WebSockets: Notificaciones en tiempo real").
package ws

import (
	"encoding/json"
	"sync"

	"github.com/gorilla/websocket"
)

// Hub mantiene las conexiones WebSocket activas agrupadas por usuario.
type Hub struct {
	mu    sync.Mutex
	conns map[string]map[*websocket.Conn]bool
}

func NewHub() *Hub {
	return &Hub{conns: make(map[string]map[*websocket.Conn]bool)}
}

func (h *Hub) Register(userID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.conns[userID] == nil {
		h.conns[userID] = make(map[*websocket.Conn]bool)
	}
	h.conns[userID][conn] = true
}

func (h *Hub) Unregister(userID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.conns[userID], conn)
	if len(h.conns[userID]) == 0 {
		delete(h.conns, userID)
	}
}

// Notify envía un evento JSON a todas las conexiones activas del usuario
// (puede tener el dashboard abierto en varias pestañas/dispositivos).
func (h *Hub) Notify(userID string, event any) {
	payload, err := json.Marshal(event)
	if err != nil {
		return
	}

	h.mu.Lock()
	conns := make([]*websocket.Conn, 0, len(h.conns[userID]))
	for c := range h.conns[userID] {
		conns = append(conns, c)
	}
	h.mu.Unlock()

	for _, c := range conns {
		_ = c.WriteMessage(websocket.TextMessage, payload)
	}
}
