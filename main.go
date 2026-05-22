package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
)

func main() {
	var configPath string
	var legacyCIDR string
	var legacyPort int
	var showVersion bool

	flag.StringVar(&configPath, "config", "config.yaml", "config file path")
	flag.StringVar(&legacyCIDR, "cidr", "", "ipv6 cidr, kept for compatibility when config file is absent")
	flag.IntVar(&legacyPort, "port", 52122, "dynamic http port, kept for compatibility when config file is absent")
	flag.BoolVar(&showVersion, "version", false, "print version and exit")
	flag.Parse()

	if showVersion {
		fmt.Println(versionString())
		return
	}

	explicitConfig := false
	explicitCIDR := false
	explicitPort := false
	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "config":
			explicitConfig = true
		case "cidr":
			explicitCIDR = true
		case "port":
			explicitPort = true
		}
	})

	cfg, err := loadRuntimeConfig(configPath, explicitConfig, legacyCIDR, legacyPort, explicitCIDR, explicitPort)
	if err != nil {
		log.Fatal(err)
	}

	if err := cfg.Validate(); err != nil {
		log.Fatal(err)
	}

	fixedPorts := cfg.Fixed.AllPorts()
	state := &State{FixedPorts: make(map[string]string)}
	if len(fixedPorts) > 0 {
		state, err = LoadState(cfg.StateFile)
		if err != nil {
			log.Fatal(err)
		}
		if err := state.EnsureFixedPortIPs(fixedPorts, cfg.CIDR); err != nil {
			log.Fatal(err)
		}
		if err := state.Save(cfg.StateFile); err != nil {
			log.Fatal(err)
		}
	}

	auth, err := NewProxyAuth(cfg.Auth)
	if err != nil {
		log.Fatal(err)
	}

	var wg sync.WaitGroup
	startHTTPServer(&wg, cfg.Dynamic.HTTPPort, newRandomOutboundSelector(cfg.CIDR), auth, cfg.Verbose, "dynamic-http")
	startSocks5Server(&wg, cfg.Dynamic.Socks5Port, newRandomOutboundSelector(cfg.CIDR), auth, "dynamic-socks5")

	for _, port := range cfg.Fixed.HTTPPorts {
		ip := state.FixedPorts[fmt.Sprint(port)]
		startHTTPServer(&wg, port, newFixedOutboundSelector(ip), auth, cfg.Verbose, fmt.Sprintf("fixed-http-%d", port))
	}
	for _, port := range cfg.Fixed.Socks5Ports {
		ip := state.FixedPorts[fmt.Sprint(port)]
		startSocks5Server(&wg, port, newFixedOutboundSelector(ip), auth, fmt.Sprintf("fixed-socks5-%d", port))
	}

	log.Println("server running ...")
	log.Printf("build: %s", versionString())
	log.Printf("config: %s", cfg.ConfigSource)
	log.Printf("state: %s", cfg.StateFile)
	log.Printf("ipv6 cidr: [%s]", cfg.CIDR)
	log.Printf("auth enabled: %v", auth.Enabled())
	log.Printf("dynamic http: 0.0.0.0:%d", cfg.Dynamic.HTTPPort)
	log.Printf("dynamic socks5: 0.0.0.0:%d", cfg.Dynamic.Socks5Port)
	for _, port := range cfg.Fixed.HTTPPorts {
		log.Printf("fixed http: 0.0.0.0:%d -> %s", port, state.FixedPorts[fmt.Sprint(port)])
	}
	for _, port := range cfg.Fixed.Socks5Ports {
		log.Printf("fixed socks5: 0.0.0.0:%d -> %s", port, state.FixedPorts[fmt.Sprint(port)])
	}

	wg.Wait()
}

func loadRuntimeConfig(configPath string, explicitConfig bool, legacyCIDR string, legacyPort int, explicitCIDR bool, explicitPort bool) (*Config, error) {
	if _, err := os.Stat(configPath); err == nil {
		cfg, err := LoadConfig(configPath)
		if err != nil {
			return nil, err
		}
		if explicitCIDR {
			cfg.CIDR = legacyCIDR
		}
		if explicitPort {
			cfg.Dynamic.HTTPPort = legacyPort
			cfg.Dynamic.Socks5Port = legacyPort + 1
		}
		cfg.ConfigSource = configPath
		return cfg, nil
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("stat config file %s: %w", configPath, err)
	}

	if explicitConfig {
		return nil, fmt.Errorf("config file %s does not exist", configPath)
	}
	if legacyCIDR == "" {
		return nil, fmt.Errorf("config file %s does not exist and --cidr is empty", configPath)
	}

	cfg := DefaultConfig()
	cfg.CIDR = legacyCIDR
	cfg.Dynamic.HTTPPort = legacyPort
	cfg.Dynamic.Socks5Port = legacyPort + 1
	cfg.ConfigSource = "command-line"
	return cfg, nil
}

func startHTTPServer(wg *sync.WaitGroup, port int, selector OutboundSelector, auth *ProxyAuth, verbose bool, name string) {
	server := &http.Server{
		Addr:    fmt.Sprintf("0.0.0.0:%d", port),
		Handler: newHTTPProxy(selector, auth, verbose, name),
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("%s server err: %v", name, err)
		}
	}()
}

func startSocks5Server(wg *sync.WaitGroup, port int, selector OutboundSelector, auth *ProxyAuth, name string) {
	server, err := newSocks5Proxy(selector, auth, name)
	if err != nil {
		log.Fatalf("%s init err: %v", name, err)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := server.ListenAndServe("tcp", fmt.Sprintf("0.0.0.0:%d", port)); err != nil {
			log.Fatalf("%s server err: %v", name, err)
		}
	}()
}

func validateIPv6CIDR(cidr string) error {
	ip, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return err
	}
	if ip.To4() != nil || ip.To16() == nil {
		return fmt.Errorf("%s is not an ipv6 cidr", cidr)
	}
	ones, bits := ipNet.Mask.Size()
	if bits != 128 || ones < 0 {
		return fmt.Errorf("%s is not an ipv6 cidr", cidr)
	}
	return nil
}
