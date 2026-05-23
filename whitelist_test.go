package main

import (
	"net"
	"testing"
)

func TestParseWhitelistSupportsIPAndCIDR(t *testing.T) {
	whitelist, err := ParseWhitelist([]string{
		"127.0.0.1",
		"192.168.1.0/24",
		"::1",
		"2001:db8::/32",
	})
	if err != nil {
		t.Fatalf("ParseWhitelist returned error: %v", err)
	}

	tests := []struct {
		ip   string
		want bool
	}{
		{"127.0.0.1", true},
		{"192.168.1.20", true},
		{"192.168.2.20", false},
		{"::1", true},
		{"2001:db8::1", true},
		{"2001:db9::1", false},
	}

	for _, tt := range tests {
		if got := whitelist.Contains(net.ParseIP(tt.ip)); got != tt.want {
			t.Fatalf("Contains(%s) = %v, want %v", tt.ip, got, tt.want)
		}
	}
}

func TestProxyAuthAllowsWhitelistedClientWithoutPassword(t *testing.T) {
	auth, err := NewProxyAuth(AuthConfig{Username: "user", Password: "pass"}, []string{"127.0.0.1"})
	if err != nil {
		t.Fatalf("NewProxyAuth returned error: %v", err)
	}

	if !auth.ClientWhitelisted("127.0.0.1:12345") {
		t.Fatal("expected 127.0.0.1 to be whitelisted")
	}
	if auth.ClientWhitelisted("127.0.0.2:12345") {
		t.Fatal("did not expect 127.0.0.2 to be whitelisted")
	}
}
