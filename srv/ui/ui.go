// srv/ui/ui.go
package ui

import (
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/patrickmn/go-cache"

	"github.com/opd-ai/dndbot/srv/generator"
)

type GeneratorUI struct {
	router      chi.Router
	sessions    map[string]*generator.GenerationProgress
	sessionsM   sync.RWMutex
	msgHistory  map[string]*MessageHistory
	cache       *cache.Cache
	historyFile string
}

// srv/ui/ui.go
func NewGeneratorUI() *GeneratorUI {
	ui := &GeneratorUI{
		router:      chi.NewRouter(),
		sessions:    make(map[string]*generator.GenerationProgress),
		msgHistory:  make(map[string]*MessageHistory),
		cache:       cache.New(24*time.Hour, 1*time.Hour),
		historyFile: "session_history.json",
	}

	// Set up message emitter
	generator.SetMessageEmitter(func(sessionID string, msg generator.WSMessage) error {
		ui.AddMessage(sessionID, msg)
		return nil
	})

	ui.loadHistory()
	ui.setupRoutes()
	ui.startCleanup()
	return ui
}

func (ui *GeneratorUI) startCleanup() {
	go func() {
		cleanupTicker := time.NewTicker(10 * time.Minute)
		saveTicker := time.NewTicker(5 * time.Minute)
		defer cleanupTicker.Stop()
		defer saveTicker.Stop()

		for {
			select {
			case <-cleanupTicker.C:
				ui.cleanupOldSessions()
			case <-saveTicker.C:
				ui.saveHistory()
			}
		}
	}()
}

func (ui *GeneratorUI) cleanupOldSessions() {
	ui.sessionsM.Lock()
	defer ui.sessionsM.Unlock()

	changed := false
	for sessionID, history := range ui.msgHistory {
		history.mu.RLock()
		if len(history.Messages) > 0 {
			lastMsg := history.Messages[len(history.Messages)-1]
			if time.Since(lastMsg.Timestamp) > 1*time.Hour {
				delete(ui.msgHistory, sessionID)
				changed = true
			}
		}
		history.mu.RUnlock()
	}

	// Save history if any sessions were cleaned up
	if changed {
		ui.saveHistory()
	}
}

func (ui *GeneratorUI) loadHistory() {
	// Load from persistent storage first
	file, err := os.OpenFile(ui.historyFile, os.O_RDONLY|os.O_CREATE, 0o644)
	if err != nil {
		log.Printf("Error opening history file: %v", err)
		return
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	var history map[string]*MessageHistory
	if err := decoder.Decode(&history); err != nil && err != io.EOF {
		log.Printf("Error decoding history: %v", err)
	} else if err == nil {
		ui.msgHistory = history
	}

	// Then check cache for any newer data
	data, ok := ui.cache.Get("message_history")
	if ok {
		if history, ok := data.(map[string]*MessageHistory); ok {
			ui.msgHistory = history
		}
	}
}

func (ui *GeneratorUI) saveHistory() {
	ui.sessionsM.Lock()
	defer ui.sessionsM.Unlock()

	// Save to cache
	ui.cache.Set("message_history", ui.msgHistory, cache.DefaultExpiration)

	// Save to file
	file, err := os.OpenFile(ui.historyFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		log.Printf("Error opening history file for writing: %v", err)
		return
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	if err := encoder.Encode(ui.msgHistory); err != nil {
		log.Printf("Error encoding history: %v", err)
	}
}

func (ui *GeneratorUI) AddMessage(sessionID string, msg generator.WSMessage) {
	ui.sessionsM.Lock()
	history, exists := ui.msgHistory[sessionID]
	if !exists {
		history = &MessageHistory{
			Messages: make([]generator.WSMessage, 0),
		}
		ui.msgHistory[sessionID] = history
	}
	ui.sessionsM.Unlock()

	history.AddMessage(msg)
	ui.saveHistory()
}

func (ui *GeneratorUI) cleanupSession(sessionID string, progress *generator.GenerationProgress) {
	progress.SetActive(false)
	progress.Lock()
	if progress.WSConn != nil {
		progress.WSConn.Close()
		progress.WSConn = nil
	}
	progress.Unlock()

	ui.sessionsM.Lock()
	delete(ui.sessions, sessionID)
	ui.sessionsM.Unlock()

	// Cache the progress for later retrieval
	ui.cache.Set(sessionID, progress, 24*time.Hour)

	// Save history after cleanup
	ui.saveHistory()

	close(progress.Done)
}

func (ui *GeneratorUI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ui.router.ServeHTTP(w, r)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Requested-With, HX-Request, HX-Current-URL")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Expose-Headers", "X-Session-Id")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (ui *GeneratorUI) setupRoutes() {
	// r := chi.NewRouter()

	// Apply middleware
	ui.router.Use(middleware.Logger)
	ui.router.Use(middleware.Recoverer)
	ui.router.Use(corsMiddleware)

	// Session management middleware
	ui.router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Ensure session cookie exists
			cookie, err := r.Cookie("session_id")
			if err != nil || cookie.Value == "" {
				sessionID := uuid.New().String()
				http.SetCookie(w, &http.Cookie{
					Name:     "session_id",
					Value:    sessionID,
					Path:     "/",
					MaxAge:   86400, // 24 hours
					HttpOnly: true,
					SameSite: http.SameSiteLaxMode,
				})
			}
			next.ServeHTTP(w, r)
		})
	})
	//pw, err := paywall.ConstructPaywall()
	//if err != nil {
	//	log.Fatal(err)
	//}

	// Routes
	ui.router.Get("/", ui.handleHome)
	// ui.router.Post("/generate", pw.MiddlewareFuncFunc(rateLimit(ui.handleGenerate)))
	// ui.router.Post("/generate", rateLimit(ui.handleGenerate))
	ui.router.Post("/generate", ui.handleGenerate)
	ui.router.Get("/api/messages/{sessionID}", ui.handleGetMessages)
	ui.router.Get("/ws/{sessionID}", ui.handleWebSocket)
	ui.router.Get("/check-session", ui.handleCheckSession)

	fileServer := http.FileServer(http.Dir("static"))
	ui.router.Handle("/static/*", http.StripPrefix("/static/", fileServer))
	ui.router.Get("/favicon.ico", handleFavicon)
	outputServer := http.FileServer(http.Dir("outputs"))
	ui.router.Handle("/outputs/*", http.StripPrefix("/outputs/", outputServer))
}

type logRequest []time.Time

func (l *logRequest) update() error {
	return nil
}

func newLogRequest() logRequest {
	var lr []time.Time
	lr = append(lr, time.Now())
	return lr
}

func (l logRequest) limit() bool {
	count := 0
	for i := range l {
		reversei := len(l) - 1 - i
		lastTime := l[reversei]
		fourHoursAgo := time.Now().Add(time.Duration(-4) * time.Hour)
		if lastTime.After(fourHoursAgo) {
			count++
		}
		if count >= 2 {
			return true
		}
	}
	return false
}

var loggedRequests map[net.Addr]logRequest

func init() {
	loggedRequests = make(map[net.Addr]logRequest)
}

func rateLimit(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		remoteIP := r.Header.Get("REMOTE_ADDR")
		if limit, err := exceededTheLimit(remoteIP); err != nil || limit {
			w.WriteHeader(429)
			// it then returns, not passing the request down the chain
		} else {
			h.ServeHTTP(w, r)
		}
	}
}

func exceededTheLimit(remoteIP string) (bool, error) {
	ipAddr, err := net.ResolveIPAddr("ip", remoteIP)
	if err != nil {
		return true, err
	}
	logs, ok := loggedRequests[ipAddr]
	if !ok {

		loggedRequests[ipAddr] = newLogRequest()
		return false, nil
	}
	return logs.limit(), nil
}
