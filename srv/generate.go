package main

import (
	"fmt"
	"time"

	"github.com/gorilla/websocket"
)

func generateAdventure(sessionID, prompt string) error {
	generationsMutex.RLock()
	progress, exists := activeGenerations[sessionID]
	generationsMutex.RUnlock()

	if !exists {
		return fmt.Errorf("invalid generation state for session: %s", sessionID)
	}

	// Wait for WebSocket connection
	timeout := time.After(10 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			progress.UpdateState(StateError)
			return fmt.Errorf("timeout waiting for WebSocket connection")
		case <-ticker.C:
			if progress.GetState() == StateConnected {
				goto startGeneration
			}
		}
	}

startGeneration:
	progress.UpdateState(StateGenerating)

	// Helper function for progress updates
	sendProgress := func(msg string) error {
		if progress.WSConn == nil {
			return fmt.Errorf("websocket connection lost")
		}
		return progress.WSConn.WriteMessage(websocket.TextMessage, []byte(msg))
	}

	defer func() {
		if r := recover(); r != nil {
			ErrorLogger.Printf("Panic in generation: %v", r)
			progress.UpdateState(StateError)
			sendProgress("❌ Internal server error occurred")
			progress.Done <- true
		}
	}()

	// Implement generation steps here
	steps := []struct {
		message string
		action  func() error
	}{
		{"🔗 Connected and starting generation...", nil},
		{"🎲 Generating table of contents...", func() error {
			// Your table of contents generation code
			return nil
		}},
		// Add other generation steps...
	}

	for _, step := range steps {
		if err := sendProgress(step.message); err != nil {
			progress.UpdateState(StateError)
			return err
		}
		if step.action != nil {
			if err := step.action(); err != nil {
				progress.UpdateState(StateError)
				return err
			}
		}
	}

	progress.UpdateState(StateCompleted)
	sendProgress("✅ Generation complete! You can now download your adventure.")
	progress.Done <- true
	return nil
}
