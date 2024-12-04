// types.go
package main

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type GenerationState int

const (
	StateInitialized GenerationState = iota
	StateConnected
	StateGenerating
	StateCompleted
	StateError
)

type GenerationProgress struct {
	SessionID string
	WSConn    *websocket.Conn
	Done      chan bool
	StartTime time.Time
	State     GenerationState
	Error     error
	mu        sync.RWMutex
	IsActive  bool
}

func (gp *GenerationProgress) UpdateState(state GenerationState) {
	gp.mu.Lock()
	defer gp.mu.Unlock()
	gp.State = state
}

func (gp *GenerationProgress) GetState() GenerationState {
	gp.mu.Lock()
	defer gp.mu.Unlock()
	return gp.State
}

// Add these methods
func (gp *GenerationProgress) SetActive(active bool) {
	gp.mu.Lock()
	defer gp.mu.Unlock()
	gp.IsActive = active
}

func (gp *GenerationProgress) IsStillActive() bool {
	gp.mu.RLock()
	defer gp.mu.RUnlock()
	return gp.IsActive
}
