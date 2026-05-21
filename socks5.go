package main

import (
	"context"
	"log"
	"net"

	socks5 "github.com/armon/go-socks5"
)

func newSocks5Proxy(selector OutboundSelector, auth *ProxyAuth, name string) (*socks5.Server, error) {
	conf := &socks5.Config{
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
