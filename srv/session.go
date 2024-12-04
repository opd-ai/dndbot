// session.go
package main

import (
	"sync"
	"time"
)

type SessionManager struct {
	sessions map[string]*GenerationProgress
	mu       sync.RWMutex
}

var GlobalSessionManager = &SessionManager{
	sessions: make(map[string]*GenerationProgress),
}

func (sm *SessionManager) CreateSession(sessionID string) *GenerationProgress {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	progress := &GenerationProgress{
		SessionID: sessionID,
		StartTime: time.Now(),
		State:     StateInitialized,
		Done:      make(chan bool),
		IsActive:  true,
	}

	sm.sessions[sessionID] = progress
	return progress
}

func (sm *SessionManager) GetSession(sessionID string) (*GenerationProgress, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	progress, exists := sm.sessions[sessionID]
	return progress, exists && progress.IsStillActive()
}

func (sm *SessionManager) CleanupSession(sessionID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if progress, exists := sm.sessions[sessionID]; exists {
		progress.SetActive(false)
		if progress.WSConn != nil {
			progress.WSConn.Close()
		}
		close(progress.Done)
		delete(sm.sessions, sessionID)
	}
}
