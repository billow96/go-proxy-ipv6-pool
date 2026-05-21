package main

import (
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"strings"
)

type ProxyAuth struct {
	username string
	password string
	enabled  bool
}

func NewProxyAuth(cfg AuthConfig) (*ProxyAuth, error) {
	if err := validateAuthConfig(cfg); err != nil {
		return nil, err
	}
	return &ProxyAuth{
		username: cfg.Username,
		password: cfg.Password,
		enabled:  cfg.Username != "" || cfg.Password != "",
	}, nil
}

func (a *ProxyAuth) Enabled() bool {
	return a != nil && a.enabled
}

func (a *ProxyAuth) Valid(username, password string) bool {
	if !a.Enabled() {
		return true
	}
	userOK := subtle.ConstantTimeCompare([]byte(username), []byte(a.username)) == 1
	passOK := subtle.ConstantTimeCompare([]byte(password), []byte(a.password)) == 1
	return userOK && passOK
}

func (a *ProxyAuth) AllowHTTPRequest(req *http.Request) bool {
	if !a.Enabled() {
		return true
	}
	username, password, ok := parseProxyBasicAuth(req.Header.Get("Proxy-Authorization"))
	return ok && a.Valid(username, password)
}

func parseProxyBasicAuth(header string) (string, string, bool) {
	const prefix = "Basic "
	if len(header) < len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return "", "", false
	}

	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(header[len(prefix):]))
	if err != nil {
		return "", "", false
	}
	username, password, ok := strings.Cut(string(decoded), ":")
	if !ok {
		return "", "", false
	}
	return username, password, true
}

func writeProxyAuthRequired(w http.ResponseWriter) {
	w.Header().Set("Proxy-Authenticate", `Basic realm="go-proxy-ipv6-pool"`)
	http.Error(w, "Proxy Authentication Required", http.StatusProxyAuthRequired)
}
