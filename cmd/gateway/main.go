package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"gateway/internal/config"
	"gateway/internal/control"
	"gateway/internal/rdp"
	"gateway/internal/tunnel"
	"gateway/pkg/logger"
)

func main() {
	configPath := flag.String("config", "/etc/zasca/gateway.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	logger.Init(cfg.Logging.Level, cfg.Logging.Format)
	logger.Info("ZASCA Gateway starting")

	pool := tunnel.NewPool()
	heartbeat := tunnel.NewHeartbeat(cfg.Tunnel.HeartbeatSec, cfg.Tunnel.TimeoutSec, pool)
	notifier := control.NewNotifier()

	router := rdp.NewRouter(pool, notifier)
	handler := control.NewHandler(pool, router)

	controlServer := control.NewServer(cfg.Control.SocketPath, handler, notifier)
	if err := controlServer.Start(); err != nil {
		logger.Error("failed to start control socket", "err", err)
		os.Exit(1)
	}

	tunnelServer := tunnel.NewServer(cfg.Tunnel.Port, pool, heartbeat, notifier)
	go func() {
		if err := tunnelServer.Start(); err != nil {
			logger.Error("failed to start tunnel server", "err", err)
			os.Exit(1)
		}
	}()

	rdpProxy := rdp.NewProxy(cfg.RDP, router)
	go func() {
		if err := rdpProxy.Start(); err != nil {
			logger.Error("failed to start RDP proxy", "err", err)
		}
	}()

	logger.Info("ZASCA Gateway started",
		"tunnel_port", cfg.Tunnel.Port,
		"rdp_port", cfg.RDP.Port,
		"control_socket", cfg.Control.SocketPath,
	)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	logger.Info("ZASCA Gateway shutting down")
	controlServer.Close()
	rdpProxy.Close()
}
