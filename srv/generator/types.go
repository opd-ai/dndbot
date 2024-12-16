package generator

import (
	"log"
	"sync"
	"time"
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
	RWMutex   sync.RWMutex
	SessionID string
	State     GenerationState
	Output    string
	Error     error
	Done      chan bool
	StartTime time.Time
	IsActive  bool
}

// Add these methods to GenerationProgress
func (gp *GenerationProgress) Close() {
	gp.Lock()
	defer gp.Unlock()
}

// srv/generator/types.go
func (p *GenerationProgress) SendUpdate(message string) error {
	p.Lock()
	defer p.Unlock()

	msg := Message{
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

	return nil
}

// srv/generator/types.go
func (p *GenerationProgress) UpdateOutput(output string) {
	p.Lock()
	p.Output = output
	p.Unlock()

	log.Printf("[Session %s] Updating output: %s", p.SessionID, output)
	p.SendUpdate("Updating adventure content...")
}

func (p *GenerationProgress) UpdateState(state GenerationState) {
	p.Lock()
	oldState := p.State
	p.State = state
	p.Unlock()

	log.Printf("[Session %s] State transition: %s -> %s", p.SessionID, oldState, state)

	message := ""
	switch state {
	case StateGenerating:
		message = "🎲 Generating your adventure..."
	case StateCompleted:
		message = "✨ Adventure generation completed!"
	case StateError:
		message = "❌ Error generating adventure"
	}

	if message != "" {
		p.SendUpdate(message)
	}
}

func (p *GenerationProgress) SetActive(active bool) {
	p.Lock()
	p.IsActive = active
	p.Unlock()
}

func (gp *GenerationProgress) Lock() {
	gp.RWMutex.Lock()
}

func (gp *GenerationProgress) Unlock() {
	gp.RWMutex.Unlock()
}

func (gp *GenerationProgress) GetState() GenerationState {
	gp.Lock()
	defer gp.Unlock()
	return gp.State
}

func (gp *GenerationProgress) IsStillActive() bool {
	gp.RWMutex.RLock()
	defer gp.RWMutex.RUnlock()
	return gp.IsActive
}

func (gp *GenerationProgress) IsDone() bool {
	gp.Lock()
	defer gp.Unlock()
	return (StateCompleted == gp.GetState())
}

type Message struct {
	Type      string    `json:"type"`
	Status    string    `json:"status"`
	Message   string    `json:"message"`
	Output    string    `json:"output"`
	Timestamp time.Time `json:"timestamp"`
}

// srv/generator/types.go
func NewMessage(msgType, status, message, output string) Message {
	return Message{
		Type:      msgType,
		Status:    status,
		Message:   message,
		Output:    output,
		Timestamp: time.Now(),
	}
}

// srv/generator/types.go
var (
	messageEmitter func(sessionID string, msg Message) error
)

func emitMessage(sessionID string, msg Message) error {
	if messageEmitter != nil {
		return messageEmitter(sessionID, msg)
	}
	return nil
}

func SetMessageEmitter(emitter func(sessionID string, msg Message) error) {
	messageEmitter = emitter
}
