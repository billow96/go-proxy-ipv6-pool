package main

import (
	"fmt"
	"net"
	"strings"
)

type IPWhitelist struct {
	nets []*net.IPNet
}

func ParseWhitelist(entries []string) (*IPWhitelist, error) {
	whitelist := &IPWhitelist{}
	for _, entry := range entries {
		value := strings.TrimSpace(entry)
		if value == "" {
			continue
		}

		if strings.Contains(value, "/") {
			_, ipNet, err := net.ParseCIDR(value)
			if err != nil {
				return nil, fmt.Errorf("invalid whitelist cidr %q: %w", value, err)
			}
			whitelist.nets = append(whitelist.nets, ipNet)
			continue
		}

		ip := net.ParseIP(value)
		if ip == nil {
			return nil, fmt.Errorf("invalid whitelist ip %q", value)
		}
		whitelist.nets = append(whitelist.nets, singleIPNet(ip))
	}
	return whitelist, nil
}

func singleIPNet(ip net.IP) *net.IPNet {
	if ipv4 := ip.To4(); ipv4 != nil {
		return &net.IPNet{IP: ipv4, Mask: net.CIDRMask(32, 32)}
	}
	return &net.IPNet{IP: ip.To16(), Mask: net.CIDRMask(128, 128)}
}

func (w *IPWhitelist) Contains(ip net.IP) bool {
	if w == nil || ip == nil {
		return false
	}
	for _, ipNet := range w.nets {
		if ipNet.Contains(ip) {
			return true
		}
	}
	return false
}

func (w *IPWhitelist) Empty() bool {
	return w == nil || len(w.nets) == 0
}

func clientIPFromRemoteAddr(remoteAddr string) net.IP {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	return net.ParseIP(strings.Trim(host, "[]"))
}
