package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type State struct {
	FixedPorts map[string]string `json:"fixed_ports"`
}

func LoadState(path string) (*State, error) {
	state := &State{FixedPorts: make(map[string]string)}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return state, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read state file %s: %w", path, err)
	}
	if len(data) == 0 {
		return state, nil
	}
	if err := json.Unmarshal(data, state); err != nil {
		return nil, fmt.Errorf("parse state file %s: %w", path, err)
	}
	if state.FixedPorts == nil {
		state.FixedPorts = make(map[string]string)
	}
	return state, nil
}

func (s *State) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create state directory: %w", err)
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0600); err != nil {
		return fmt.Errorf("write state file %s: %w", path, err)
	}
	return nil
}

func (s *State) EnsureFixedPortIPs(ports []int, cidr string) error {
	used := make(map[string]bool)
	for _, ip := range s.FixedPorts {
		if ip != "" && cidrContainsIP(cidr, ip) {
			used[ip] = true
		}
	}

	for _, port := range ports {
		key := fmt.Sprint(port)
		if ip := s.FixedPorts[key]; ip != "" && cidrContainsIP(cidr, ip) {
			continue
		}

		ip, err := generateUniqueIPv6(cidr, used)
		if err != nil {
			return fmt.Errorf("assign fixed ipv6 for port %d: %w", port, err)
		}
		s.FixedPorts[key] = ip
		used[ip] = true
	}
	return nil
}

func generateUniqueIPv6(cidr string, used map[string]bool) (string, error) {
	var lastErr error
	for i := 0; i < 1000; i++ {
		ip, err := generateRandomIPv6(cidr)
		if err != nil {
			return "", err
		}
		if !used[ip] {
			return ip, nil
		}
		lastErr = fmt.Errorf("generated duplicate ip %s", ip)
	}
	if lastErr != nil {
		return "", lastErr
	}
	return "", fmt.Errorf("unable to generate unique ipv6")
}
