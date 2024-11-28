package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"

	. "github.com/opd-ai/dndbot/src"
)

type Server struct {
	router       *mux.Router
	templates    *template.Template
	claude       *ClaudeClient
	moneroPayURL string
	orders       sync.Map // map[string]*Order
	upgrader     websocket.Upgrader
}

type Order struct {
	ID          string
	Status      string  // "pending", "paid", "processing", "complete", "error"
	PaymentID   string  // MoneroPay payment ID
	Address     string  // XMR address
	Amount      float64 // 10.00 USD in XMR
	CreatedAt   time.Time
	CompletedAt time.Time
	PromptText  string
	OutputPath  string
	LogChan     chan string
}

// MoneroPay API types
type CreatePaymentRequest struct {
	Amount      float64 `json:"amount"`
	Currency    string  `json:"currency"`
	CallbackURL string  `json:"callback_url"`
}

type CreatePaymentResponse struct {
	PaymentID string  `json:"payment_id"`
	Address   string  `json:"address"`
	Amount    float64 `json:"amount"`
}

type PaymentCallback struct {
	PaymentID string `json:"payment_id"`
	Status    string `json:"status"`
}

func NewServer(claudeClient *ClaudeClient, moneroPayURL string) *Server {
	s := &Server{
		router:       mux.NewRouter(),
		claude:       claudeClient,
		moneroPayURL: moneroPayURL,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	}

	// Load HTML templates
	s.templates = template.Must(template.ParseGlob("templates/*.html"))

	s.routes()
	return s
}

func (s *Server) routes() {
	s.router.PathPrefix("/static/").Handler(http.StripPrefix("/static/",
		http.FileServer(http.Dir("static"))))

	s.router.HandleFunc("/", s.handleIndex()).Methods("GET")
	s.router.HandleFunc("/order/new", s.handleNewOrder()).Methods("POST")
	s.router.HandleFunc("/order/{id}", s.handleViewOrder()).Methods("GET")
	s.router.HandleFunc("/order/{id}/submit", s.handleSubmitPrompt()).Methods("POST")
	s.router.HandleFunc("/order/{id}/download", s.handleDownload()).Methods("GET")
	s.router.HandleFunc("/ws/{id}", s.handleWebSocket())
	s.router.HandleFunc("/callback", s.handleMoneroPayCallback()).Methods("POST")
}

func (s *Server) handleNewOrder() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Create payment request to MoneroPay
		callbackURL := fmt.Sprintf("https://%s/callback", r.Host)
		paymentReq := CreatePaymentRequest{
			Amount:      10.00,
			Currency:    "USD",
			CallbackURL: callbackURL,
		}

		// Send request to MoneroPay
		resp, err := http.Post(s.moneroPayURL+"/create_payment",
			"application/json",
			encodeJSON(paymentReq))
		if err != nil {
			http.Error(w, "Payment creation failed", http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		var paymentResp CreatePaymentResponse
		if err := json.NewDecoder(resp.Body).Decode(&paymentResp); err != nil {
			http.Error(w, "Invalid payment response", http.StatusInternalServerError)
			return
		}

		// Create order
		order := &Order{
			ID:        paymentResp.PaymentID,
			Status:    "pending",
			PaymentID: paymentResp.PaymentID,
			Address:   paymentResp.Address,
			Amount:    paymentResp.Amount,
			CreatedAt: time.Now(),
			LogChan:   make(chan string, 100),
		}

		s.orders.Store(order.ID, order)

		http.Redirect(w, r, "/order/"+order.ID, http.StatusSeeOther)
	}
}

func (s *Server) handleMoneroPayCallback() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var callback PaymentCallback
		if err := json.NewDecoder(r.Body).Decode(&callback); err != nil {
			http.Error(w, "Invalid callback data", http.StatusBadRequest)
			return
		}

		order, ok := s.orders.Load(callback.PaymentID)
		if !ok {
			http.Error(w, "Order not found", http.StatusNotFound)
			return
		}

		o := order.(*Order)
		if callback.Status == "confirmed" {
			o.Status = "paid"
			o.LogChan <- "Payment confirmed! You can now enter your prompt."
		}

		w.WriteHeader(http.StatusOK)
	}
}

func (s *Server) handleWebSocket() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		orderID := vars["id"]

		conn, err := s.upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		order, ok := s.orders.Load(orderID)
		if !ok {
			conn.WriteMessage(websocket.TextMessage, []byte("Order not found"))
			return
		}

		// Stream logs to WebSocket
		for msg := range order.(*Order).LogChan {
			if err := conn.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
				break
			}
		}
	}
}

func (s *Server) handleSubmitPrompt() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		orderID := vars["id"]

		order, ok := s.orders.Load(orderID)
		if !ok {
			http.Error(w, "Order not found", http.StatusNotFound)
			return
		}

		o := order.(*Order)
		if o.Status != "paid" {
			http.Error(w, "Payment required", http.StatusPaymentRequired)
			return
		}

		o.PromptText = r.FormValue("prompt")
		o.Status = "processing"

		// Start adventure generation
		go s.generateAdventure(o)

		http.Redirect(w, r, "/order/"+orderID, http.StatusSeeOther)
	}
}

func main() {
	claude := NewClaudeClient(os.Getenv("ANTHROPIC_API_KEY"))
	server := NewServer(claude, os.Getenv("MONERO_PAY_URL"))

	log.Printf("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", server.router))
}
