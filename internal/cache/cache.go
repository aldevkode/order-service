package cache

import (
    "sync"
    "github.com/aldevkode/order-service/internal/model"
)

type Store struct {
    mu sync.RWMutex
    m  map[string]model.Order
}

func New() *Store { return &Store{m: make(map[string]model.Order)} }

func (s *Store) Get(id string) (model.Order, bool) {
    s.mu.RLock(); defer s.mu.RUnlock()
    v, ok := s.m[id]
    return v, ok
}

func (s *Store) Set(o model.Order) {
    s.mu.Lock(); defer s.mu.Unlock()
    s.m[o.OrderUID] = o
}

func (s *Store) BulkSet(list []model.Order) {
    s.mu.Lock(); defer s.mu.Unlock()
    for _, o := range list { s.m[o.OrderUID] = o }
}