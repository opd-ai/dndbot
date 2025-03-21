// Package ui provides the web user interface and HTTP handlers for the DND bot generator
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
	"github.com/go-chi/httprate"
	"github.com/google/uuid"
	"github.com/patrickmn/go-cache"

	"github.com/opd-ai/dndbot/srv/generator"
	"github.com/opd-ai/paywall"

	secure "github.com/srikrsna/security-headers"
)

// GeneratorUI manages the web interface for the DND adventure generator.
// It handles session management, message history, and HTTP routing.
type GeneratorUI struct {
	router      chi.Router
	sessions    map[string]*generator.GenerationProgress
	sessionsM   sync.RWMutex
	msgHistory  map[string]*MessageHistory
	cache       *cache.Cache
	historyFile string
	zoltar      *paywall.Paywall
	usePaywall  bool
}

// NewGeneratorUI creates and initializes a new GeneratorUI instance.
//
// Returns:
//   - *GeneratorUI: Configured UI handler with initialized routes and session management
//
// Sets up message handling, loads history, initializes cleanup routines,
// and configures HTTP routes.
func NewGeneratorUI(usePaywall bool) *GeneratorUI {
	ui := &GeneratorUI{
		router:      chi.NewRouter(),
		sessions:    make(map[string]*generator.GenerationProgress),
		msgHistory:  make(map[string]*MessageHistory),
		cache:       cache.New(24*time.Hour, 1*time.Hour),
		historyFile: "session_history.json",
		usePaywall:  usePaywall,
	}

	// Set up message emitter
	generator.SetMessageEmitter(func(sessionID string, msg generator.Message) error {
		ui.AddMessage(sessionID, msg)
		return nil
	})

	ui.loadHistory()
	ui.setupRoutes()
	ui.startCleanup()
	return ui
}

// startCleanup initiates background goroutines for periodic maintenance tasks.
// Runs two concurrent tasks:
// - Session cleanup every 100 minutes
// - History saving every 50 minutes
func (ui *GeneratorUI) startCleanup() {
	go func() {
		cleanupTicker := time.NewTicker(100 * time.Minute)
		saveTicker := time.NewTicker(50 * time.Minute)
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

// cleanupOldSessions removes sessions that have been inactive for more than 1 hour.
// Saves history if any sessions are removed.
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

// loadHistory restores message history from persistent storage and cache.
// Attempts to load from file first, then updates with any newer cache data.
// Creates history file if it doesn't exist.
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

// saveHistory persists the current message history to both cache and file storage.
// Thread-safe operation that maintains message history in two locations:
// - In-memory cache for quick access
// - JSON file for persistence across restarts
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

// AddMessage adds a new message to a session's history and persists the update.
//
// Parameters:
//   - sessionID: string identifier for the session
//   - msg: generator.Message to add to history
//
// Creates new history if session doesn't exist.
func (ui *GeneratorUI) AddMessage(sessionID string, msg generator.Message) {
	ui.sessionsM.Lock()
	history, exists := ui.msgHistory[sessionID]
	if !exists {
		history = &MessageHistory{
			Messages: make([]generator.Message, 0),
		}
		ui.msgHistory[sessionID] = history
	}
	ui.sessionsM.Unlock()

	history.AddMessage(msg)
	ui.saveHistory()
}

// cleanupSession handles the graceful shutdown of a generation session.
//
// Parameters:
//   - sessionID: string identifier for the session to cleanup
//   - progress: *generator.GenerationProgress associated with the session
//
// Closes WebSocket connections, removes from active sessions,
// caches progress, and saves history.
func (ui *GeneratorUI) cleanupSession(sessionID string, progress *generator.GenerationProgress) {
	progress.SetActive(false)

	ui.sessionsM.Lock()
	delete(ui.sessions, sessionID)
	ui.sessionsM.Unlock()

	// Cache the progress for later retrieval
	ui.cache.Set(sessionID, progress, 24*time.Hour)

	// Save history after cleanup
	ui.saveHistory()

	close(progress.Done)
}

// ServeHTTP implements the http.Handler interface for the GeneratorUI.
//
// Parameters:
//   - w: http.ResponseWriter to write the response
//   - r: *http.Request containing the request details
func (ui *GeneratorUI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ui.router.ServeHTTP(w, r)
}

// corsMiddleware provides Cross-Origin Resource Sharing support.
//
// Parameters:
//   - next: http.Handler to wrap with CORS support
//
// Returns:
//   - http.Handler: Middleware that handles CORS headers and preflight requests
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Requested-With, HX-Request, HX-Current-URL")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Expose-Headers", "X-Session-Id")
		w.Header().Set("X-Content-Type-Options", "nosniff")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func hstsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		next.ServeHTTP(w, r)
	})
}

// setupRoutes configures all HTTP routes and middleware for the UI.
// Sets up:
// - Standard middleware (logging, recovery, CORS)
// - Session management
// - Static file serving
// - API endpoints
func (ui *GeneratorUI) setupRoutes() {
	csp := &secure.CSP{
		Value: `default-src 'self'; script-src 'self' {{nonce}} 'sha256-6O8L648x8Xhmzia0qI/zQdbTlpVaGehLozFzz/i2dIE=' 'sha256-EekPIHyJRz0hoIvCnNHdTpzW+jhqMbyTyTu2nXJO7b0=' 'sha256-UI0Byo+Bdsa/9HfswzVRrNuMaEt5s26sucqVpf8iPxw=' 'sha256-61RzUnEfEIq/j80eD9kxIi6+FHZECO1+ZjF5aAkRgcI=' 'sha256-2j5Uk7r3oOJ3KJIcT7QE7NrJ0DmojlDl0qjjoQbPsz8=' https://cdnjs.cloudflare.com https://cdn.jsdelivr.net; style-src 'self' 'unsafe-inline' https://cdnjs.cloudflare.com https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com; img-src 'self' data:;`,
	}
	log.Println("csp", csp)
	// r := chi.NewRouter()

	// Apply middleware
	ui.router.Use(middleware.Logger)
	ui.router.Use(middleware.Recoverer)
	ui.router.Use(corsMiddleware)
	ui.router.Use(httprate.Limit(
		40,          // requests
		time.Minute, // per duration
		httprate.WithKeyFuncs(httprate.KeyByIP, httprate.KeyByEndpoint),
	))
	ui.router.Use(hstsMiddleware)
	// ui.router.Use(csp.Middleware())

	// Session management middleware
	ui.router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Ensure session cookie exists
			sessionID := r.Header.Get("X-Session-Id")
			if sessionID == "" {
				log.Println("no client side sessionID:", sessionID)
				// Try cookie as fallback
				if cookie, err := r.Cookie("session_id"); err == nil && cookie.Value != "null" {
					log.Println("cookie found", cookie.Value, err)
					sessionID = cookie.Value
				} else {
					sessionID = uuid.New().String()
					http.SetCookie(w, &http.Cookie{
						Name:     "session_id",
						Value:    sessionID,
						Path:     "/",
						MaxAge:   864000,
						HttpOnly: false,
						SameSite: http.SameSiteLaxMode,
					})
				}
			}
			w.Header().Set("X-Session-Id", sessionID)
			next.ServeHTTP(w, r)
		})
	})
	var err error

	ui.zoltar, err = paywall.NewPaywall(paywall.Config{
		PriceInBTC:       0.0001, // 0.0001 BTC
		PriceInXMR:       0.0000, // 0.0001 XMR
		TestNet:          false,
		Store:            paywall.NewFileStore("./paywallet"), // Required for payment tracking
		PaymentTimeout:   time.Hour * 24,
		MinConfirmations: 1,
		XMRUser:          "user", // XMR disabled for now
		XMRPassword:      "password",
		XMRRPC:           "http://localhost:18081/",
	})
	if err != nil {
		log.Fatal(err)
	}

	// Routes
	ui.router.Get("/", ui.handleHome)
	if ui.usePaywall {
		ui.router.Post("/generate", ui.zoltar.MiddlewareFuncFunc(rateLimit(ui.handleGenerate)))
	} else {
		ui.router.Post("/generate", rateLimit(ui.handleGenerate))
	}
	ui.router.Get("/api/messages/{sessionID}", ui.handleGetMessages)
	ui.router.Get("/check-session", ui.handleCheckSession)

	fileServer := http.FileServer(http.Dir("static"))
	ui.router.Handle("/static/*", http.StripPrefix("/static/", fileServer))
	ui.router.Get("/favicon.ico", handleFavicon)
	outputServer := http.FileServer(http.Dir("outputs"))
	ui.router.Handle("/outputs/*", http.StripPrefix("/outputs/", outputServer))
	archiveServer := http.FileServer(http.Dir("archive"))
	ui.router.Handle("/archive/*", http.StripPrefix("/archive/", archiveServer))
}

// logRequest represents a time-ordered list of request timestamps
// used for rate limiting.
type logRequest []time.Time

// update records a new request timestamp.
//
// Returns:
//   - error: any error encountered during update
func (l logRequest) update() error {
	l = append(l, time.Now())
	return nil
}

// newLogRequest creates a new request log with the current timestamp.
//
// Returns:
//   - logRequest: initialized with current time
func newLogRequest() logRequest {
	var lr []time.Time
	lr = append(lr, time.Now())
	return lr
}

// limit checks if the request count exceeds 3 requests in the last 4 hours.
//
// Returns:
//   - bool: true if limit exceeded, false otherwise
func (l logRequest) limit() bool {
	count := 0
	for i := range l {
		reversei := len(l) - 1 - i
		lastTime := l[reversei]
		fourHoursAgo := time.Now().Add(time.Duration(-4) * time.Hour)
		if lastTime.After(fourHoursAgo) {
			count++
		}
		if count >= 3 {
			return true
		}
	}
	l.update()
	return false
}

var loggedRequests map[net.Addr]logRequest

func init() {
	loggedRequests = make(map[net.Addr]logRequest)
}

// rateLimit wraps an http.HandlerFunc with request rate limiting.
//
// Parameters:
//   - h: http.HandlerFunc to protect with rate limiting
//
// Returns:
//   - http.HandlerFunc: Handler that enforces rate limits
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

// exceededTheLimit checks if a given IP has exceeded the rate limit.
//
// Parameters:
//   - remoteIP: string IP address to check
//
// Returns:
//   - bool: true if limit exceeded
//   - error: any error in IP resolution
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
