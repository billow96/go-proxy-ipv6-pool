package main

import (
	"fmt"
	"sync"
)

type FixedIPStore struct {
	mu  sync.RWMutex
	ips map[int]string
}

func NewFixedIPStore(state *State) *FixedIPStore {
	store := &FixedIPStore{ips: make(map[int]string)}
	if state == nil {
		return store
	}
	for portText, ip := range state.FixedPorts {
		var port int
		if _, err := fmt.Sscanf(portText, "%d", &port); err == nil {
			store.ips[port] = ip
		}
	}
	return store
}

func (s *FixedIPStore) Get(port int) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ip, ok := s.ips[port]
	return ip, ok
}

func (s *FixedIPStore) Set(port int, ip string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ips[port] = ip
}

func (s *FixedIPStore) Selector(port int) OutboundSelector {
	return func() (string, error) {
		ip, ok := s.Get(port)
		if !ok || ip == "" {
			return "", fmt.Errorf("fixed ip for port %d is not available", port)
		}
		return ip, nil
	}
}
