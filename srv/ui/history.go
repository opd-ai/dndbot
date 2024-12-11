package ui

import (
	"sync"

	"github.com/opd-ai/dndbot/srv/generator"
)

type MessageHistory struct {
	Messages []generator.WSMessage
	mu       sync.RWMutex
}

func (h *MessageHistory) AddMessage(msg generator.WSMessage) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.Messages = append(h.Messages, msg)
}

func (h *MessageHistory) GetMessages() []generator.WSMessage {
	h.mu.RLock()
	defer h.mu.RUnlock()
	messages := make([]generator.WSMessage, len(h.Messages))
	copy(messages, h.Messages)
	return messages
}
