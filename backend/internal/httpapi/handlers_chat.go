package httpapi

import (
	"log"
	"net/http"

	"github.com/hnl/bank-backend/internal/auth"
)

func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	if s.chat == nil {
		writeError(w, http.StatusServiceUnavailable, "El chat no está configurado (falta ANTHROPIC_API_KEY).")
		return
	}

	userID := auth.UserIDFromContext(r.Context())
	var req ChatRequest
	if err := decodeAndValidate(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Escribe un mensaje válido.")
		return
	}

	reply, conversationID, err := s.chat.HandleMessage(r.Context(), userID, req.ConversationID, req.Message)
	if err != nil {
		log.Printf("chat error for user %s: %v", userID, err)
		writeError(w, http.StatusBadGateway, "No se pudo contactar al asistente. Intenta de nuevo.")
		return
	}

	writeJSON(w, http.StatusOK, ChatResponse{ConversationID: conversationID, Reply: reply})
}
