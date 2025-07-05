package storage

import (
	"errors"
	"sync"
	"time"

	"redirect_helper/internal/models"
)

type MemoryStorage struct {
	data map[string]*models.ForwardingEntry
	mu   sync.RWMutex
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		data: make(map[string]*models.ForwardingEntry),
	}
}

func (s *MemoryStorage) CreateForwarding(name, token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.data[name]; exists {
		return errors.New("forwarding name already exists")
	}

	s.data[name] = &models.ForwardingEntry{
		Name:      name,
		Token:     token,
		Target:    "",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return nil
}

func (s *MemoryStorage) SetTarget(name, token, target string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, exists := s.data[name]
	if !exists {
		return errors.New("forwarding name not found")
	}

	if entry.Token != token {
		return errors.New("invalid token")
	}

	entry.Target = target
	entry.UpdatedAt = time.Now()

	return nil
}

func (s *MemoryStorage) GetTarget(name string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, exists := s.data[name]
	if !exists {
		return "", errors.New("forwarding name not found")
	}

	if entry.Target == "" {
		return "", errors.New("target not set")
	}

	return entry.Target, nil
}

func (s *MemoryStorage) GetForwarding(name string) (*models.ForwardingEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, exists := s.data[name]
	if !exists {
		return nil, errors.New("forwarding name not found")
	}

	return entry, nil
}

func (s *MemoryStorage) ListForwardings() ([]*models.ForwardingEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*models.ForwardingEntry, 0, len(s.data))
	for _, entry := range s.data {
		result = append(result, entry)
	}

	return result, nil
}