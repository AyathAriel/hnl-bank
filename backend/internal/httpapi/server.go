package httpapi

import (
	"context"

	"github.com/hnl/bank-backend/internal/auth"
	"github.com/hnl/bank-backend/internal/banking"
	"github.com/hnl/bank-backend/internal/config"
	"github.com/hnl/bank-backend/internal/ws"
)

// ChatService abstrae el orquestador de chat con IA (internal/chat), para que
// este paquete no dependa directamente de la integración con Anthropic/MCP.
type ChatService interface {
	HandleMessage(ctx context.Context, userID, conversationID, message string) (reply string, resolvedConversationID string, err error)
}

type Server struct {
	cfg        *config.Config
	banking    *banking.Service
	revocation *auth.RevocationStore
	chat       ChatService
	hub        *ws.Hub
}

func NewServer(cfg *config.Config, bankingService *banking.Service, revocation *auth.RevocationStore, chatService ChatService, hub *ws.Hub) *Server {
	return &Server{
		cfg:        cfg,
		banking:    bankingService,
		revocation: revocation,
		chat:       chatService,
		hub:        hub,
	}
}
