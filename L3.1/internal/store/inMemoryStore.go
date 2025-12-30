package store

import (
	"fmt"
	"sync"

	"L3.1/internal/types"
)

// InMemoryStore — простое потокобезопасное хранилище уведомлений,
// используемое для разработки и тестирования.
type InMemoryStore struct {
	data map[string]types.Notification
	mu   sync.RWMutex
}

// NewInMemoryStore создаёт новое in-memory хранилище уведомлений.
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		data: make(map[string]types.Notification),
	}
}

// Save сохраняет новое уведомление по указанному ID.
func (s *InMemoryStore) Save(id string, ntf types.Notification) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[id] = ntf
	return nil
}

// Get возвращает уведомление по ID, если оно существует.
func (s *InMemoryStore) Get(id string) (*types.Notification, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ntf, ok := s.data[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}

	n := ntf // копия, чтобы избежать гонок
	return &n, nil
}

// Cancel помечает уведомление как отменённое.
func (s *InMemoryStore) Cancel(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ntf, ok := s.data[id]
	if !ok {
		return fmt.Errorf("not found")
	}

	ntf.Canceled = true
	s.data[id] = ntf
	return nil
}

// Update обновляет существующее уведомление по ID.
func (s *InMemoryStore) Update(id string, ntf types.Notification) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.data[id]; !ok {
		return fmt.Errorf("not found")
	}

	s.data[id] = ntf
	return nil
}
