// session.go
package handlers

import (
	"sync"
	"time"

	generator "github.com/opd-ai/dndbot/srv/generator"
	"github.com/opd-ai/dndbot/srv/util"
)

type SessionManager struct {
	sessions map[string]*generator.GenerationProgress
	mu       sync.RWMutex
}

var GlobalSessionManager = &SessionManager{
	sessions: make(map[string]*generator.GenerationProgress),
}

func (sm *SessionManager) Lock() {
	sm.mu.Lock()
}

func (sm *SessionManager) Unlock() {
	sm.mu.Unlock()
}

func (sm *SessionManager) CreateSession(sessionID string) *generator.GenerationProgress {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	progress := &generator.GenerationProgress{
		SessionID: sessionID,
		Done:      make(chan bool),
		StartTime: time.Now(),
		State:     generator.StateInitialized,
		IsActive:  true,
	}
	sm.sessions[sessionID] = progress
	return progress
}

func (sm *SessionManager) GetSession(sessionID string) (*generator.GenerationProgress, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	progress, exists := sm.sessions[sessionID]
	if !exists || !progress.IsStillActive() {
		return nil, false
	}
	return progress, true
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

func (sm *SessionManager) CleanupOldSessions() {
	for {
		time.Sleep(15 * time.Minute)
		threshold := time.Now().Add(-1 * time.Hour)

		sm.Lock()
		for id, progress := range sm.sessions {
			if progress.StartTime.Before(threshold) {
				sm.CleanupSession(id)
				util.InfoLogger.Printf("Cleaned up stale session: %s", id)
			}
		}
		sm.Unlock()
	}
}
