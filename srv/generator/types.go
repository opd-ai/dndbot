package generator

import (
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type GenerationState string

const (
	StateInitialized GenerationState = "initialized"
	StateConnected   GenerationState = "connected"
	StateGenerating  GenerationState = "generating"
	StateCompleted   GenerationState = "completed"
	StateError       GenerationState = "error"
)

type GenerationProgress struct {
	mu        sync.RWMutex
	SessionID string
	State     GenerationState
	Output    string
	Error     error
	WSConn    *websocket.Conn
	Done      chan bool
	StartTime time.Time
	IsActive  bool
}

// Add these methods to GenerationProgress
func (gp *GenerationProgress) Close() {
	gp.Lock()
	defer gp.Unlock()
	if gp.WSConn != nil {
		gp.WSConn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		gp.WSConn.Close()
		gp.WSConn = nil
	}
}

// srv/generator/types.go
func (p *GenerationProgress) SendUpdate(message string) error {
	p.Lock()
	defer p.Unlock()

	msg := WSMessage{
		Type:      "update",
		Status:    string(p.State),
		Message:   message,
		Output:    p.Output,
		Timestamp: time.Now(),
	}

	// Always emit the message to history first
	if messageEmitter != nil {
		if err := messageEmitter(p.SessionID, msg); err != nil {
			log.Printf("[Session %s] Failed to emit message to history: %v", p.SessionID, err)
		}
	}

	// Try to send via WebSocket if available
	if p.WSConn != nil {
		if err := p.WSConn.WriteJSON(msg); err != nil {
			log.Printf("[Session %s] Failed to send WebSocket message: %v", p.SessionID, err)
			return err
		}
		log.Printf("[Session %s] Message sent via WebSocket: %s", p.SessionID, message)
	} else {
		log.Printf("[Session %s] Message queued (no WebSocket): %s", p.SessionID, message)
	}

	return nil
}

func (p *GenerationProgress) UpdateState(state GenerationState) {
	p.Lock()
	oldState := p.State
	p.State = state
	log.Printf("State transition: %s -> %s", oldState, state)
	p.Unlock()

	// Send state update via WebSocket
	message := ""
	switch state {
	case StateGenerating:
		message = "🎲 Generating your adventure..."
	case StateCompleted:
		message = "✨ Adventure generation completed!"
	case StateError:
		message = "❌ Error generating adventure"
	}

	p.SendUpdate(message)
}

func (p *GenerationProgress) UpdateOutput(output string) {
	p.Lock()
	p.Output = output
	p.Unlock()
	p.SendUpdate("Updating adventure content...")
}

func (p *GenerationProgress) SetActive(active bool) {
	p.Lock()
	p.IsActive = active
	p.Unlock()
}

func (gp *GenerationProgress) Lock() {
	gp.mu.Lock()
}

func (gp *GenerationProgress) Unlock() {
	gp.mu.Unlock()
}

func (gp *GenerationProgress) GetState() GenerationState {
	gp.mu.Lock()
	defer gp.mu.Unlock()
	return gp.State
}

func (gp *GenerationProgress) IsStillActive() bool {
	gp.mu.RLock()
	defer gp.mu.RUnlock()
	return gp.IsActive
}

func (gp *GenerationProgress) IsDone() bool {
	gp.mu.Lock()
	defer gp.mu.Unlock()
	return (StateCompleted == gp.GetState())
}

type WSMessage struct {
	Type      string    `json:"type"`
	Status    string    `json:"status"`
	Message   string    `json:"message"`
	Output    string    `json:"output"`
	Timestamp time.Time `json:"timestamp"`
}

// srv/generator/types.go
func NewWSMessage(msgType, status, message, output string) WSMessage {
	return WSMessage{
		Type:      msgType,
		Status:    status,
		Message:   message,
		Output:    output,
		Timestamp: time.Now(),
	}
}

// srv/generator/types.go
var (
	messageEmitter func(sessionID string, msg WSMessage) error
)

func wsEmitMessage(sessionID string, msg WSMessage) error {
	if messageEmitter != nil {
		return messageEmitter(sessionID, msg)
	}
	return nil
}

func SetMessageEmitter(emitter func(sessionID string, msg WSMessage) error) {
	messageEmitter = emitter
}
