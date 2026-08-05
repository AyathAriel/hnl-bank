package chat

import "sync"

// maxHistoryMessages limita cuánta conversación se reenvía al modelo en cada
// turno, para no dejar crecer el costo/latencia indefinidamente.
const maxHistoryMessages = 20

// store mantiene el historial de cada conversación en memoria del proceso.
// Limitación conocida y documentada: no sobrevive a un reinicio del backend;
// no es un requisito funcional persistir el chat entre despliegues.
type store struct {
	mu            sync.Mutex
	conversations map[string][]Message
}

func newStore() *store {
	return &store{conversations: make(map[string][]Message)}
}

func (s *store) get(conversationID string) []Message {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Message(nil), s.conversations[conversationID]...)
}

func (s *store) set(conversationID string, messages []Message) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(messages) > maxHistoryMessages {
		messages = messages[len(messages)-maxHistoryMessages:]
	}
	s.conversations[conversationID] = messages
}
