package models

import "time"

type UserStats struct {
	TotalLogins     int       `json:"total_logins" db:"total_logins"`
	LastLogin       time.Time `json:"last_login" db:"last_login"`
	MemberSince     time.Time `json:"member_since" db:"member_since"`
	ProfileViews    int       `json:"profile_views" db:"profile_views"`
	ApiCallsToday   int       `json:"api_calls_today" db:"api_calls_today"`
	ApiCallsMonth   int       `json:"api_calls_month" db:"api_calls_month"`
	StorageUsedMB   float64   `json:"storage_used_mb" db:"storage_used_mb"`
	StorageLimitMB  float64   `json:"storage_limit_mb" db:"storage_limit_mb"`
	ActiveSessions  int       `json:"active_sessions" db:"active_sessions"`
	TwoFactorEnabled bool     `json:"two_factor_enabled" db:"two_factor_enabled"`
}

type Activity struct {
	ID          uint      `json:"id" db:"id"`
	UserID      uint      `json:"user_id" db:"user_id"`
	Action      string    `json:"action" db:"action"`
	Resource    string    `json:"resource" db:"resource"`
	IPAddress   string    `json:"ip_address" db:"ip_address"`
	UserAgent   string    `json:"user_agent" db:"user_agent"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

type Notification struct {
	ID        uint      `json:"id" db:"id"`
	UserID    uint      `json:"user_id" db:"user_id"`
	Title     string    `json:"title" db:"title"`
	Message   string    `json:"message" db:"message"`
	Type      string    `json:"type" db:"type"` // info, warning, success, error
	Read      bool      `json:"read" db:"read"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type DashboardUpdate struct {
	Type      string                 `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

type DashboardData struct {
	User          *User           `json:"user"`
	Stats         *UserStats      `json:"stats"`
	Activities    []Activity      `json:"activities"`
	Notifications []Notification  `json:"notifications"`
}