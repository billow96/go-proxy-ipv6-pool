package main

import (
	"context"
	"log"
	"net"
	"net/http"

	"github.com/elazarl/goproxy"
)

func newHTTPProxy(selector OutboundSelector, auth *ProxyAuth, verbose bool, name string) http.Handler {
	proxy := goproxy.NewProxyHttpServer()
	proxy.Verbose = verbose
	proxy.Tr = &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			conn, outboundIP, err := dialWithOutbound(ctx, network, addr, selector)
			if err != nil {
				log.Printf("[%s] dial %s via %s error: %v", name, addr, outboundIP, err)
				return nil, err
			}
			if verbose {
				log.Printf("[%s] %s via [%s]", name, addr, outboundIP)
			}
			return conn, nil
		},
		DisableKeepAlives: true,
	}
	proxy.ConnectDialWithReq = func(req *http.Request, network, addr string) (net.Conn, error) {
		conn, outboundIP, err := dialWithOutbound(req.Context(), network, addr, selector)
		if err != nil {
			log.Printf("[%s] connect %s via %s error: %v", name, addr, outboundIP, err)
			return nil, err
		}
		if verbose {
			log.Printf("[%s] CONNECT %s via [%s]", name, addr, outboundIP)
		}
		return conn, nil
	}

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if !auth.AllowHTTPRequest(req) {
			writeProxyAuthRequired(w)
			return
		}
		proxy.ServeHTTP(w, req)
	})
}
