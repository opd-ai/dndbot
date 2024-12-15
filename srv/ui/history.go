// Package ui provides the web user interface handlers for the DND bot generator
package ui

import (
	"sync"

	"github.com/opd-ai/dndbot/srv/generator"
)

// MessageHistory maintains a thread-safe list of WebSocket messages for a generation session.
// It provides concurrent-safe operations for adding and retrieving messages.
type MessageHistory struct {
	Messages []generator.WSMessage
	mu       sync.RWMutex
}

// AddMessage appends a new WebSocket message to the history in a thread-safe manner.
//
// Parameters:
//   - msg: generator.WSMessage to add to the history
//
// The method uses mutex locking to ensure thread-safe append operations
// when multiple goroutines are modifying the message history.
func (h *MessageHistory) AddMessage(msg generator.WSMessage) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.Messages = append(h.Messages, msg)
}

// GetMessages returns a copy of all messages in the history in a thread-safe manner.
//
// Returns:
//   - []generator.WSMessage: A new slice containing copies of all messages
//
// The method creates a deep copy of the messages slice to prevent
// external modifications to the internal state. Uses read lock for
// concurrent access optimization.
func (h *MessageHistory) GetMessages() []generator.WSMessage {
	h.mu.RLock()
	defer h.mu.RUnlock()
	messages := make([]generator.WSMessage, len(h.Messages))
	copy(messages, h.Messages)
	return messages
}
