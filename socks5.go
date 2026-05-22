package main

import (
	"context"
	"fmt"
	"log"
	"net"

	socks5 "github.com/armon/go-socks5"
	xcontext "golang.org/x/net/context"
)

func newSocks5Proxy(selector OutboundSelector, auth *ProxyAuth, name string) (*socks5.Server, error) {
	conf := &socks5.Config{
		Resolver: ipv6Resolver{},
		Dial: func(ctx context.Context, network, addr string) (net.Conn, error) {
			conn, outboundIP, err := dialWithOutbound(ctx, network, addr, selector)
			if err != nil {
				log.Printf("[%s] dial %s via %s error: %v", name, addr, outboundIP, err)
				return nil, err
			}
			log.Printf("[%s] %s via [%s]", name, addr, outboundIP)
			return conn, nil
		},
	}

	if auth.Enabled() {
		conf.Credentials = socks5.StaticCredentials{
			auth.username: auth.password,
		}
	}

	return socks5.New(conf)
}

type ipv6Resolver struct{}

func (r ipv6Resolver) Resolve(ctx xcontext.Context, name string) (xcontext.Context, net.IP, error) {
	ips, err := net.DefaultResolver.LookupIP(ctx, "ip6", name)
	if err != nil {
		return ctx, nil, err
	}

	for _, ip := range ips {
		if ip.To4() == nil && ip.To16() != nil {
			return ctx, ip, nil
		}
	}
	return ctx, nil, fmt.Errorf("no ipv6 address found for %s", name)
}
