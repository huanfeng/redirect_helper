package models

import (
	"time"
)

type ForwardingEntry struct {
	Name      string    `json:"name"`
	Token     string    `json:"token"`
	Target    string    `json:"target"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type DomainEntry struct {
	Domain    string    `json:"domain"`
	Token     string    `json:"token"`
	Target    string    `json:"target"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Response struct {
	State   string `json:"state"`
	Message string `json:"message,omitempty"`
}