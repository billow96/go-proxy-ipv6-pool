package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"net"
	"strings"
)

type OutboundSelector func() (string, error)

func newRandomOutboundSelector(cidr string) OutboundSelector {
	return func() (string, error) {
		return generateRandomIPv6(cidr)
	}
}

func newFixedOutboundSelector(ip string) OutboundSelector {
	return func() (string, error) {
		return ip, nil
	}
}

func dialWithOutbound(ctx context.Context, network, addr string, selector OutboundSelector) (net.Conn, string, error) {
	outboundIP, err := selector()
	if err != nil {
		return nil, "", err
	}

	localAddr, err := net.ResolveTCPAddr("tcp6", "["+outboundIP+"]:0")
	if err != nil {
		return nil, outboundIP, err
	}

	dialer := net.Dialer{LocalAddr: localAddr}
	conn, err := dialer.DialContext(ctx, forceTCP6Network(network), addr)
	return conn, outboundIP, err
}

func forceTCP6Network(network string) string {
	if strings.HasPrefix(network, "tcp") {
		return "tcp6"
	}
	return network
}

func generateRandomIPv6(cidr string) (string, error) {
	networkIP, ipv6Net, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", err
	}

	base := networkIP.To16()
	if base == nil || networkIP.To4() != nil {
		return "", fmt.Errorf("%s is not an ipv6 cidr", cidr)
	}

	mask := ipv6Net.Mask
	if len(mask) != net.IPv6len {
		return "", fmt.Errorf("%s is not an ipv6 cidr", cidr)
	}

	randomIP := make([]byte, net.IPv6len)
	if _, err := rand.Read(randomIP); err != nil {
		return "", err
	}

	out := make([]byte, net.IPv6len)
	for i := 0; i < net.IPv6len; i++ {
		out[i] = (base[i] & mask[i]) | (randomIP[i] & ^mask[i])
	}
	return net.IP(out).String(), nil
}

func cidrContainsIP(cidr string, ipText string) bool {
	ip := net.ParseIP(ipText)
	if ip == nil {
		return false
	}
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return false
	}
	return ipNet.Contains(ip)
}
