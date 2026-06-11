package handlers

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/big-pickle/internal/auth"
	"github.com/big-pickle/internal/models"
)

type DashboardHandler struct {
	hub       *Hub
	userStore UserStore
	tmpl      *template.Template
}

type Hub struct {
	clients    map[*websocket.Conn]bool
	broadcast  chan []byte
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	mu         sync.RWMutex
}

type UserStore interface {
	GetUserByID(id uint) (*models.User, error)
	GetUserStats(id uint) (*models.UserStats, error)
	GetUserActivity(id uint) ([]models.Activity, error)
	GetUserNotifications(id uint) ([]models.Notification, error)
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Tighten in production
	},
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				client.Close()
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				err := client.WriteMessage(websocket.TextMessage, message)
				if err != nil {
					client.Close()
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

func NewDashboardHandler(userStore UserStore) *DashboardHandler {
	tmpl, err := template.ParseGlob("templates/dashboard/*.html")
	if err != nil {
		log.Printf("Warning: could not parse dashboard templates: %v", err)
		tmpl = template.New("dashboard")
	}

	return &DashboardHandler{
		hub:       NewHub(),
		userStore: userStore,
		tmpl:      tmpl,
	}
}

func (h *DashboardHandler) StartBackgroundUpdates(interval time.Duration) {
	go h.hub.Run()

	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			h.hub.mu.RLock()
			for client := range h.hub.clients {
				// In production, fetch per-user data from DB
				update := models.DashboardUpdate{
					Type:      "stats",
					Timestamp: time.Now(),
					Data: map[string]interface{}{
						"online_users":  len(h.hub.clients),
						"last_updated": time.Now().Format(time.RFC3339),
					},
				}
				msg, _ := json.Marshal(update)
				h.hub.broadcast <- msg
			}
			h.hub.mu.RUnlock()
		}
	}()
}

func (h *DashboardHandler) ServeDashboard(w http.ResponseWriter, r *http.Request) {
	session := auth.SessionFromContext(r.Context())
	if session == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	user, err := h.userStore.GetUserByID(session.UserID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	stats, _ := h.userStore.GetUserStats(session.UserID)
	activities, _ := h.userStore.GetUserActivity(session.UserID)
	notifications, _ := h.userStore.GetUserNotifications(session.UserID)

	data := map[string]interface{}{
		"User":          user,
		"Stats":         stats,
		"Activities":    activities,
		"Notifications": notifications,
		"WSURL":         "ws://" + r.Host + "/ws/dashboard",
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(w, "dashboard.html", data); err != nil {
		log.Printf("Dashboard template error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *DashboardHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	h.hub.register <- conn

	// Send initial connection confirmation
	welcome := models.DashboardUpdate{
		Type:      "connected",
		Timestamp: time.Now(),
		Data:      map[string]interface{}{"message": "Connected to dashboard"},
	}
	msg, _ := json.Marshal(welcome)
	conn.WriteMessage(websocket.TextMessage, msg)

	// Read loop to detect disconnects
	go func() {
		defer func() {
			h.hub.unregister <- conn
		}()
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	}()
}

func (h *DashboardHandler) GetStatsAPI(w http.ResponseWriter, r *http.Request) {
	session := auth.SessionFromContext(r.Context())
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	stats, err := h.userStore.GetUserStats(session.UserID)
	if err != nil {
		http.Error(w, "Error fetching stats", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}