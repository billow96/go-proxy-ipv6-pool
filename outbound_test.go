package main

import (
	"context"
	"strings"
	"testing"
)

func TestForceTCP6Network(t *testing.T) {
	if got := forceTCP6Network("tcp"); got != "tcp6" {
		t.Fatalf("forceTCP6Network(tcp) = %q, want tcp6", got)
	}
	if got := forceTCP6Network("tcp4"); got != "tcp6" {
		t.Fatalf("forceTCP6Network(tcp4) = %q, want tcp6", got)
	}
	if got := forceTCP6Network("udp"); got != "udp" {
		t.Fatalf("forceTCP6Network(udp) = %q, want udp", got)
	}
}

func TestForceTCP6AddressRejectsIPv4(t *testing.T) {
	_, err := forceTCP6Address(context.Background(), "172.64.155.209:443")
	if err == nil {
		t.Fatal("forceTCP6Address accepted ipv4 target")
	}
	if !strings.Contains(err.Error(), "ipv4 target blocked") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestForceTCP6AddressKeepsIPv6(t *testing.T) {
	got, err := forceTCP6Address(context.Background(), "[2a06:98c1:310b::ac40:9bd1]:443")
	if err != nil {
		t.Fatalf("forceTCP6Address returned error: %v", err)
	}
	if got != "[2a06:98c1:310b::ac40:9bd1]:443" {
		t.Fatalf("forceTCP6Address = %q, want [2a06:98c1:310b::ac40:9bd1]:443", got)
	}
}
