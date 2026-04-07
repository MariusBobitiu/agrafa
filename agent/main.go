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
	"time"

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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signalCh)

	runResultCh := make(chan error, 1)
	go func() {
		runResultCh <- agentRunner.Start(ctx)
	}()

	var runErr error
	var shutdownReason string
	var shutdownPayload map[string]any

	select {
	case sig := <-signalCh:
		shutdownReason, shutdownPayload = shutdownMetadataForSignal(sig)
		cancel()
		runErr = <-runResultCh
	case runErr = <-runResultCh:
	}

	if shutdownReason != "" {
		sendBestEffortShutdown(agentRunner, shutdownReason, shutdownPayload)
	}

	if runErr != nil && !errors.Is(runErr, context.Canceled) {
		if shutdownReason == "" {
			sendBestEffortShutdown(agentRunner, "error_occurred", map[string]any{
				"error": runErr.Error(),
			})
		}
		log.Fatalf("run agent: %v", runErr)
	}

	log.Printf("agent stopped")
}

func agentTokenFingerprint(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:4])
}

func sendBestEffortShutdown(agentRunner *runner.Runner, reason string, payload map[string]any) {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := agentRunner.NotifyShutdown(shutdownCtx, reason, payload); err != nil {
		log.Printf("agent shutdown signal failed\n  reason: %s\n  error: %v", reason, err)
		return
	}

	log.Printf("agent shutdown signal sent\n  reason: %s", reason)
}

func shutdownMetadataForSignal(sig os.Signal) (string, map[string]any) {
	switch sig {
	case os.Interrupt:
		return "user_closed", map[string]any{"signal": "SIGINT"}
	case syscall.SIGTERM:
		return "terminated", map[string]any{"signal": "SIGTERM"}
	default:
		return "signal_received", map[string]any{"signal": sig.String()}
	}
}
