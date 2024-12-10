package generator

import (
	"fmt"
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

// In your GenerationProgress struct
func (p *GenerationProgress) SendUpdate(message string) error {
	p.Lock()
	defer p.Unlock()

	if p.WSConn == nil {
		return fmt.Errorf("no WebSocket connection")
	}

	msg := WSMessage{
		Type:    "update",
		Status:  string(p.State),
		Message: message,
		Output:  p.Output,
	}

	log.Printf("Sending update: %+v", msg)
	return p.WSConn.WriteJSON(msg)
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
	Type    string `json:"type"`
	Status  string `json:"status"`
	Message string `json:"message"`
	Output  string `json:"output"`
}
