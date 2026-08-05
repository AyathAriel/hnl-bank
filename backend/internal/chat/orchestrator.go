package chat

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

const systemPrompt = `Eres el asistente bancario de HNL Bank. Ayudas al usuario autenticado a consultar
saldos, ver su historial y realizar depósitos, retiros y transferencias mediante las herramientas
disponibles. Responde siempre en español, de forma breve y clara.

Reglas importantes:
- Las operaciones de depósito, retiro y transferencia son CRÍTICAS. Antes de ejecutarlas, describe
  la operación exacta (monto, cuenta origen/destino) y pide confirmación explícita al usuario.
- Nunca llames a una tool de depósito/retiro/transferencia con confirmed=true a menos que el
  usuario ya haya confirmado esa operación específica en un mensaje anterior de esta conversación.
- Si una tool responde con status "needs_confirmation", no la reintentes: pregunta al usuario y
  espera su respuesta en el siguiente turno.
- Si una tool responde con status "error", explica el problema al usuario en lenguaje natural.
- No inventes saldos, cuentas ni transacciones: usa siempre las tools para obtener datos reales.`

const maxToolIterations = 6

type Service struct {
	anthropic   *AnthropicClient
	store       *store
	mcpBinPath  string
}

func NewService(anthropicClient *AnthropicClient, mcpBinPath string) *Service {
	return &Service{
		anthropic:  anthropicClient,
		store:      newStore(),
		mcpBinPath: mcpBinPath,
	}
}

// HandleMessage procesa un mensaje del usuario dentro de una conversación,
// ejecutando el loop de tool-use contra el servidor MCP cuando el modelo lo
// requiera, hasta producir una respuesta final en lenguaje natural.
func (s *Service) HandleMessage(ctx context.Context, userID, conversationID, message string) (string, string, error) {
	if conversationID == "" {
		conversationID = uuid.NewString()
	}

	session, err := openMCPSession(ctx, s.mcpBinPath, userID)
	if err != nil {
		return "", conversationID, fmt.Errorf("starting mcp session: %w", err)
	}
	defer session.Close()

	tools, err := session.ListAnthropicTools(ctx)
	if err != nil {
		return "", conversationID, fmt.Errorf("listing tools: %w", err)
	}

	messages := s.store.get(conversationID)
	messages = append(messages, Message{Role: "user", Content: []ContentBlock{{Type: "text", Text: message}}})

	var finalText string

	for i := 0; i < maxToolIterations; i++ {
		resp, err := s.anthropic.createMessage(ctx, systemPrompt, messages, tools)
		if err != nil {
			return "", conversationID, err
		}

		messages = append(messages, Message{Role: "assistant", Content: resp.Content})

		if resp.StopReason != "tool_use" {
			finalText = extractText(resp.Content)
			break
		}

		toolResults := make([]ContentBlock, 0)
		for _, block := range resp.Content {
			if block.Type != "tool_use" {
				continue
			}
			resultText, isError, callErr := session.CallTool(ctx, block.Name, block.Input)
			if callErr != nil {
				resultText = fmt.Sprintf(`{"status":"error","message":%q}`, callErr.Error())
				isError = true
			}
			toolResults = append(toolResults, ContentBlock{
				Type:      "tool_result",
				ToolUseID: block.ID,
				Content:   resultText,
				IsError:   isError,
			})
		}

		messages = append(messages, Message{Role: "user", Content: toolResults})

		if i == maxToolIterations-1 {
			finalText = "Lo siento, no pude completar la solicitud tras varios intentos. ¿Puedes reformularla?"
		}
	}

	s.store.set(conversationID, messages)

	if finalText == "" {
		finalText = "No tengo una respuesta en este momento, ¿puedes intentar de nuevo?"
	}
	return finalText, conversationID, nil
}

func extractText(blocks []ContentBlock) string {
	text := ""
	for _, b := range blocks {
		if b.Type == "text" {
			text += b.Text
		}
	}
	return text
}
