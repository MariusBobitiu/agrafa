package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/MariusBobitiu/agrafa-agent/src/client"
	"github.com/MariusBobitiu/agrafa-agent/src/collectors"
	"github.com/MariusBobitiu/agrafa-agent/src/config"
	"github.com/MariusBobitiu/agrafa-agent/src/health"
	"github.com/MariusBobitiu/agrafa-agent/src/runner"
)

func main() {
	log.SetFlags(log.LstdFlags | log.LUTC)

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	apiClient := client.NewAPIClient(cfg.APIBaseURL, cfg.AgentToken, cfg.APITimeout, cfg.APIRetryCount)
	metricsCollector := collectors.NewSystemMetricsCollector(cfg.DiskPath)
	healthChecker := health.NewHTTPChecker(cfg.HTTPTimeout)

	agentRunner, err := runner.New(cfg, apiClient, metricsCollector, healthChecker)
	if err != nil {
		log.Fatalf("create runner: %v", err)
	}

	log.Printf(
		"agent startup\n  node_id: %d\n  source: %s\n  backend: %s\n  auth: token:%s\n  api_timeout: %s\n  api_retry_count: %d\n  heartbeat_interval: %s\n  metrics_interval: %s\n  health_interval: %s\n  config_refresh_interval: %s\n  health_http_timeout: %s\n  fallback_health_checks: %d\n  disk_path: %s",
		cfg.NodeID,
		cfg.Source,
		cfg.APIBaseURL,
		agentTokenFingerprint(cfg.AgentToken),
		cfg.APITimeout,
		cfg.APIRetryCount,
		cfg.HeartbeatInterval,
		cfg.MetricsInterval,
		cfg.HealthInterval,
		cfg.ConfigRefreshInterval,
		cfg.HTTPTimeout,
		len(cfg.HealthChecks),
		cfg.DiskPath,
	)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := agentRunner.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
		log.Fatalf("run agent: %v", err)
	}

	log.Printf("agent stopped")
}

func agentTokenFingerprint(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:4])
}
