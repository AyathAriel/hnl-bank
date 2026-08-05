package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// mcpSession envuelve una sesión MCP de corta duración contra un subproceso
// mcpserver, anclado a un único usuario autenticado. Se abre y se cierra por
// cada mensaje de chat: es la forma más simple y robusta de garantizar que
// nunca queden procesos huérfanos, al costo de un pequeño overhead de arranque
// por turno (aceptable para el alcance de esta prueba técnica).
type mcpSession struct {
	client  *mcp.Client
	session *mcp.ClientSession
}

func openMCPSession(ctx context.Context, binPath, userID string) (*mcpSession, error) {
	cmd := exec.Command(binPath, "--user-id", userID)
	transport := &mcp.CommandTransport{Command: cmd}

	client := mcp.NewClient(&mcp.Implementation{Name: "hnl-bank-chat-host", Version: "1.0.0"}, nil)
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("connecting to mcp server: %w", err)
	}
	return &mcpSession{client: client, session: session}, nil
}

func (s *mcpSession) Close() {
	_ = s.session.Close()
}

// ListAnthropicTools obtiene las tools del servidor MCP y las traduce al
// formato esperado por la Anthropic Messages API.
func (s *mcpSession) ListAnthropicTools(ctx context.Context) ([]Tool, error) {
	result, err := s.session.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		return nil, fmt.Errorf("listing mcp tools: %w", err)
	}

	tools := make([]Tool, 0, len(result.Tools))
	for _, t := range result.Tools {
		tools = append(tools, Tool{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.InputSchema,
		})
	}
	return tools, nil
}

// CallTool ejecuta una tool y devuelve su resultado serializado como texto,
// listo para enviarse de vuelta al modelo como contenido de tool_result.
func (s *mcpSession) CallTool(ctx context.Context, name string, input json.RawMessage) (resultText string, isError bool, err error) {
	var args map[string]any
	if len(input) > 0 {
		if err := json.Unmarshal(input, &args); err != nil {
			return "", false, fmt.Errorf("decoding tool input: %w", err)
		}
	}

	result, err := s.session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		return "", false, fmt.Errorf("calling tool %s: %w", name, err)
	}

	if result.StructuredContent != nil {
		b, err := json.Marshal(result.StructuredContent)
		if err == nil {
			return string(b), result.IsError, nil
		}
	}

	text := ""
	for _, c := range result.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			text += tc.Text
		}
	}
	return text, result.IsError, nil
}
