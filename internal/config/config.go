package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Tunnel   TunnelConfig   `yaml:"tunnel"`
	RDP      RDPConfig      `yaml:"rdp"`
	Control  ControlConfig  `yaml:"control"`
	Logging  LoggingConfig  `yaml:"logging"`
}

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type TunnelConfig struct {
	Port           int `yaml:"port"`
	HeartbeatSec   int `yaml:"heartbeat_sec"`
	TimeoutSec     int `yaml:"timeout_sec"`
	MaxConnections int `yaml:"max_connections"`
}

type RDPConfig struct {
	Port       int    `yaml:"port"`
	TLSCert    string `yaml:"tls_cert"`
	TLSKey     string `yaml:"tls_key"`
	RDPDomain  string `yaml:"rdp_domain"`
}

type ControlConfig struct {
	SocketPath string `yaml:"socket_path"`
}

type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

func Default() *Config {
	return &Config{
		Server: ServerConfig{
			Host: "0.0.0.0",
			Port: 9000,
		},
		Tunnel: TunnelConfig{
			Port:           9000,
			HeartbeatSec:   30,
			TimeoutSec:     90,
			MaxConnections: 1000,
		},
		RDP: RDPConfig{
			Port:      443,
			TLSCert:   "/etc/zasca/tls/cert.pem",
			TLSKey:    "/etc/zasca/tls/key.pem",
			RDPDomain: "zasca.com",
		},
		Control: ControlConfig{
			SocketPath: "/run/zasca/control.sock",
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
		},
	}
}

func Load(path string) (*Config, error) {
	cfg := Default()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
