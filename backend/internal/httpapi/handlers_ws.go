package httpapi

import (
	"net/http"

	"github.com/gorilla/websocket"

	"github.com/hnl/bank-backend/internal/auth"
)

// upgrader: el chequeo de Origin que trae gorilla/websocket por defecto
// (comparar Origin contra el Host visto por el servidor) es frágil detrás de
// un proxy — se desactiva aquí a propósito porque la autorización real de
// este endpoint es el JWT (?token=...) validado explícitamente abajo, no el
// header Origin (pensado más para flujos con cookies, no con bearer tokens).
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// handleWebSocket autentica por query param (?token=...) porque la API nativa
// WebSocket del navegador no permite enviar cabeceras personalizadas al
// establecer la conexión, y registra la conexión en el hub para recibir
// notificaciones de transacciones en tiempo real.
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	if s.hub == nil {
		http.Error(w, "notifications not available", http.StatusServiceUnavailable)
		return
	}

	token := r.URL.Query().Get("token")
	claims, err := auth.ParseToken(s.cfg.JWTSecret, token)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}
	if revoked, err := s.revocation.IsRevoked(r.Context(), claims.ID); err != nil || revoked {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	userID := claims.UserID
	s.hub.Register(userID, conn)
	defer s.hub.Unregister(userID, conn)

	// El único propósito de este loop es detectar el cierre de la conexión
	// (el servidor solo envía notificaciones, no espera mensajes del cliente).
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
	}
}
