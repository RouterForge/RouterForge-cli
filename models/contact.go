package models

import (
	"time"
	"github.com/google/uuid"
)

type Contact struct {
	ID        string    `json:"id"`
	FirstName string    `json:"first_name"`