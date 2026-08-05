// Package chat implementa el host/cliente MCP que conecta el chat del
// dashboard con un modelo de Anthropic: por cada mensaje, arranca una sesión
// MCP contra cmd/mcpserver (herramientas bancarias del usuario autenticado),
// las ofrece al modelo como tools, y ejecuta el loop de tool-use hasta obtener
// una respuesta final en lenguaje natural.
package chat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const anthropicAPIURL = "https://api.anthropic.com/v1/messages"
const anthropicVersion = "2023-06-01"

// ContentBlock representa un bloque de contenido de un mensaje de la Messages API
// (texto, uso de herramienta, resultado de herramienta o razonamiento extendido).
//
// Los bloques "thinking" deben reenviarse a la API EXACTAMENTE como se
// recibieron (incluida su firma criptográfica) cuando se reconstruye el
// historial para la siguiente llamada del loop de tool-use; reconstruirlos
// campo por campo es frágil (Anthropic puede añadir campos nuevos, o omitir
// el texto de "thinking" dejando solo la firma). Por eso, todo bloque que
// llega DESDE la API se guarda también como JSON crudo (`raw`) y se reenvía
// verbatim. Los bloques que construimos nosotros mismos (tool_result) no
// tienen `raw` y se serializan normalmente a partir de sus campos tipados.
type ContentBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Content   string          `json:"content,omitempty"`
	IsError   bool            `json:"is_error,omitempty"`

	raw json.RawMessage
}

func (c *ContentBlock) UnmarshalJSON(data []byte) error {
	type alias ContentBlock
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*c = ContentBlock(a)
	c.raw = append(json.RawMessage(nil), data...)
	return nil
}

func (c ContentBlock) MarshalJSON() ([]byte, error) {
	if len(c.raw) > 0 {
		return c.raw, nil
	}
	type alias ContentBlock
	return json.Marshal(alias(c))
}

type Message struct {
	Role    string         `json:"role"`
	Content []ContentBlock `json:"content"`
}

type Tool struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	InputSchema any    `json:"input_schema"`
}

type messagesRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	System    string    `json:"system,omitempty"`
	Messages  []Message `json:"messages"`
	Tools     []Tool    `json:"tools,omitempty"`
}

type messagesResponse struct {
	ID         string         `json:"id"`
	Role       string         `json:"role"`
	Content    []ContentBlock `json:"content"`
	StopReason string         `json:"stop_reason"`
	Error      *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

type AnthropicClient struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

func NewAnthropicClient(apiKey, model string) *AnthropicClient {
	return &AnthropicClient{
		apiKey:     apiKey,
		model:      model,
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

func (c *AnthropicClient) createMessage(ctx context.Context, system string, messages []Message, tools []Tool) (*messagesResponse, error) {
	reqBody := messagesRequest{
		Model:     c.model,
		MaxTokens: 1024,
		System:    system,
		Messages:  messages,
		Tools:     tools,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, anthropicAPIURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", anthropicVersion)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("calling anthropic api: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	var parsed messagesResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("parsing response (status %d): %w", resp.StatusCode, err)
	}

	if resp.StatusCode != http.StatusOK {
		if parsed.Error != nil {
			return nil, fmt.Errorf("anthropic api error: %s", parsed.Error.Message)
		}
		return nil, fmt.Errorf("anthropic api returned status %d", resp.StatusCode)
	}

	return &parsed, nil
}
