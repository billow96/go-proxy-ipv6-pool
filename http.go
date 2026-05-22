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
			conn, outboundIP, finalNetwork, finalAddr, err := dialWithOutbound(ctx, network, addr, selector)
			if err != nil {
				log.Printf("[%s] dial network=%s addr=%s final_network=%s final_addr=%s via=%s error=%v", name, network, addr, finalNetwork, finalAddr, outboundIP, err)
				return nil, err
			}
			if verbose {
				log.Printf("[%s] dial network=%s addr=%s final_network=%s final_addr=%s via=%s", name, network, addr, finalNetwork, finalAddr, outboundIP)
			}
			return conn, nil
		},
		DisableKeepAlives: true,
	}
	proxy.ConnectDialWithReq = func(req *http.Request, network, addr string) (net.Conn, error) {
		conn, outboundIP, finalNetwork, finalAddr, err := dialWithOutbound(req.Context(), network, addr, selector)
		if err != nil {
			log.Printf("[%s] connect network=%s addr=%s final_network=%s final_addr=%s via=%s error=%v", name, network, addr, finalNetwork, finalAddr, outboundIP, err)
			return nil, err
		}
		if verbose {
			log.Printf("[%s] CONNECT network=%s addr=%s final_network=%s final_addr=%s via=%s", name, network, addr, finalNetwork, finalAddr, outboundIP)
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
