package routes

import (
	"net/http"

	"github.com/big-pickle/internal/auth"
	"github.com/big-pickle/internal/handlers"
)

func RegisterDashboardRoutes(mux *http.ServeMux, dashboardHandler *handlers.DashboardHandler, authMiddleware *auth.Middleware) {
	// Dashboard page
	mux.Handle("/dashboard", authMiddleware.RequireAuth(
		http.HandlerFunc(dashboardHandler.ServeDashboard),
	))

	// WebSocket endpoint for real-time updates
	mux.HandleFunc("/ws/dashboard", dashboardHandler.HandleWebSocket)

	// API endpoints
	mux.Handle("/api/dashboard/stats", authMiddleware.RequireAuth(
		http.HandlerFunc(dashboardHandler.GetStatsAPI),
	))

	// Start background real-time data broadcaster
	dashboardHandler.StartBackgroundUpdates(5 * time.Second)
}