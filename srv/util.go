// main.go
package main

import (
	"time"

	"github.com/opd-ai/dndbot/srv/util"
)

func cleanupOldSessions() {
	for {
		time.Sleep(15 * time.Minute)
		threshold := time.Now().Add(-1 * time.Hour)

		GlobalSessionManager.mu.Lock()
		for id, progress := range GlobalSessionManager.sessions {
			if progress.StartTime.Before(threshold) {
				GlobalSessionManager.CleanupSession(id)
				util.InfoLogger.Printf("Cleaned up stale session: %s", id)
			}
		}
		GlobalSessionManager.mu.Unlock()
	}
}
