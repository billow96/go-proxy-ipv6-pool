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
		Resolver: ipv6Resolver{name: name},
		Rewriter: ipv6Rewriter{name: name},
		Dial: func(ctx context.Context, network, addr string) (net.Conn, error) {
			conn, outboundIP, finalNetwork, finalAddr, err := dialWithOutbound(ctx, network, addr, selector)
			if err != nil {
				log.Printf("[%s] dial network=%s addr=%s final_network=%s final_addr=%s via=%s error=%v", name, network, addr, finalNetwork, finalAddr, outboundIP, err)
				return nil, err
			}
			log.Printf("[%s] dial network=%s addr=%s final_network=%s final_addr=%s via=%s", name, network, addr, finalNetwork, finalAddr, outboundIP)
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

type ipv6Resolver struct {
	name string
}

func (r ipv6Resolver) Resolve(ctx xcontext.Context, name string) (xcontext.Context, net.IP, error) {
	ip, err := lookupIPv6(ctx, name)
	if err != nil {
		log.Printf("[%s] resolve domain=%s error=%v", r.name, name, err)
		return ctx, nil, err
	}

	log.Printf("[%s] resolve domain=%s ipv6=%s", r.name, name, ip)
	return ctx, ip, nil
}

type ipv6Rewriter struct {
	name string
}

func (r ipv6Rewriter) Rewrite(ctx xcontext.Context, request *socks5.Request) (xcontext.Context, *socks5.AddrSpec) {
	dest := request.DestAddr
	if dest == nil || dest.FQDN == "" {
		if dest != nil && dest.IP != nil && dest.IP.To4() != nil {
			log.Printf("[%s] reject ipv4 target=%s", r.name, dest.Address())
		}
		return ctx, dest
	}

	ip, err := lookupIPv6(ctx, dest.FQDN)
	if err != nil {
		log.Printf("[%s] rewrite domain=%s error=%v", r.name, dest.FQDN, err)
		return ctx, dest
	}

	rewritten := &socks5.AddrSpec{
		FQDN: dest.FQDN,
		IP:   ip,
		Port: dest.Port,
	}
	log.Printf("[%s] rewrite domain=%s ipv6=%s port=%d", r.name, dest.FQDN, ip, dest.Port)
	return ctx, rewritten
}

func lookupIPv6(ctx xcontext.Context, name string) (net.IP, error) {
	ips, err := net.DefaultResolver.LookupIP(ctx, "ip6", name)
	if err != nil {
		return nil, err
	}

	for _, ip := range ips {
		if ip.To4() == nil && ip.To16() != nil {
			return ip, nil
		}
	}
	return nil, fmt.Errorf("no ipv6 address found for %s", name)
}
