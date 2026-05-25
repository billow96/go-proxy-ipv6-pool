package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
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

	app, err := NewApp(cfg)
	if err != nil {
		log.Fatal(err)
	}
	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
	app.logStartup()
	app.Wait()
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
	cfg.StateFilePath = cfg.StateFile
	cfg.ConfigSource = "command-line"
	return cfg, nil
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
