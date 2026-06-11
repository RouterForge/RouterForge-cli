package store

import (
	"database/sql"
	"time"

	"github.com/big-pickle/internal/models"
)

type ActivityStore struct {
	db *sql.DB
}

func NewActivityStore(db *sql.DB) *ActivityStore {
	return &ActivityStore{db: db}
}

func (s *ActivityStore) GetUserActivity(userID uint, limit int) ([]models.Activity, error) {
	rows, err := s.db.Query(`
		SELECT id, user_id, action, resource, ip_address, user_agent, created_at
		FROM activities
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var activities []models.Activity
	for rows.Next() {
		var a models.Activity
		if err := rows.Scan(&a.ID, &a.UserID, &a.Action, &a.Resource, &a.IPAddress, &a.UserAgent, &a.CreatedAt); err != nil {
			return nil, err
		}
		activities = append(activities, a)
	}

	if activities == nil {
		activities = []models.Activity{}
	}
	return activities, nil
}

func (s *ActivityStore) CreateActivity(userID uint, action, resource, ip, userAgent string) error {
	_, err := s.db.Exec(`
		INSERT INTO activities (user_id, action, resource, ip_address, user_agent, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, userID, action, resource, ip, userAgent, time.Now())
	return err
}

func (s *ActivityStore) GetUserNotifications(userID uint, limit int) ([]models.Notification, error) {
	rows, err := s.db.Query(`
		SELECT id, user_id, title, message, type, read, created_at
		FROM notifications
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []models.Notification
	for rows.Next() {
		var n models.Notification
		if err := rows.Scan(&n.ID, &n.UserID, &n.Title, &n.Message, &n.Type, &n.Read, &n.CreatedAt); err != nil {
			return nil, err
		}
		notifications = append(notifications, n)
	}

	if notifications == nil {
		notifications = []models.Notification{}
	}
	return notifications, nil
}

func (s *ActivityStore) CreateNotification(userID uint, title, message, nType string) error {
	_, err := s.db.Exec(`
		INSERT INTO notifications (user_id, title, message, type, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, userID, title, message, nType, time.Now())
	return err
}

func (s *ActivityStore) MarkAllRead(userID uint) error {
	_, err := s.db.Exec(`
		UPDATE notifications SET read = TRUE
		WHERE user_id = $1 AND read = FALSE
	`, userID)
	return err
}