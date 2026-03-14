package gateway

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/ectoclaw/ectoclaw/cmd/ectoclaw/internal"
	"github.com/ectoclaw/ectoclaw/pkg/bridge"
	"github.com/ectoclaw/ectoclaw/pkg/bus"
	"github.com/ectoclaw/ectoclaw/pkg/channels"
	_ "github.com/ectoclaw/ectoclaw/pkg/channels/discord"
	_ "github.com/ectoclaw/ectoclaw/pkg/channels/irc"
	_ "github.com/ectoclaw/ectoclaw/pkg/channels/line"
	_ "github.com/ectoclaw/ectoclaw/pkg/channels/matrix"
	_ "github.com/ectoclaw/ectoclaw/pkg/channels/slack"
	_ "github.com/ectoclaw/ectoclaw/pkg/channels/telegram"
	_ "github.com/ectoclaw/ectoclaw/pkg/channels/whatsapp"
	_ "github.com/ectoclaw/ectoclaw/pkg/channels/whatsapp_native"
	"github.com/ectoclaw/ectoclaw/pkg/cron"
	"github.com/ectoclaw/ectoclaw/pkg/health"
	"github.com/ectoclaw/ectoclaw/pkg/heartbeat"
	"github.com/ectoclaw/ectoclaw/pkg/logger"
	"github.com/ectoclaw/ectoclaw/pkg/media"
	"github.com/ectoclaw/ectoclaw/pkg/providers"
	"github.com/ectoclaw/ectoclaw/pkg/state"
)

func gatewayCmd(debug bool) error {
	if debug {
		logger.SetLevel(logger.DEBUG)
	}

	cfg, err := internal.LoadConfig()
	if err != nil {
		return fmt.Errorf("error loading config: %w", err)
	}

	msgBus := bus.NewMessageBus()

	stateManager := state.NewManager(cfg.WorkspacePath())

	sessionsPath := filepath.Join(cfg.WorkspacePath(), "sessions.json")
	sessions := bridge.NewSessions(sessionsPath)
	if err := sessions.Load(); err != nil {
		logger.WarnCF("bridge", "Failed to load sessions", map[string]any{"error": err})
	}
	if bridge.IsBootstrap(cfg.WorkspacePath()) {
		logger.InfoCF("bridge", "Bootstrap file detected", nil)
		if err := sessions.Clear(); err != nil {
			logger.WarnCF("bridge", "Failed to clear sessions for bootstrap", map[string]any{"error": err})
		}
	}

	cronStorePath := filepath.Join(cfg.WorkspacePath(), "cron", "jobs.json")
	cronService := cron.NewCronService(cronStorePath, nil)

	heartbeatService := heartbeat.NewHeartbeatService(
		cfg.WorkspacePath(),
		cfg.Heartbeat.Interval,
		cfg.Heartbeat.Enabled,
	)
	heartbeatService.SetBus(msgBus)

	// Create media store for file lifecycle management with TTL cleanup
	mediaStore := media.NewFileMediaStoreWithCleanup(media.MediaCleanerConfig{
		Enabled:  cfg.MediaCleanup.Enabled,
		MaxAge:   time.Duration(cfg.MediaCleanup.MaxAge) * time.Minute,
		Interval: time.Duration(cfg.MediaCleanup.Interval) * time.Minute,
	})
	mediaStore.Start()

	provider, err := providers.NewProvider(cfg)
	if err != nil {
		mediaStore.Stop()
		return fmt.Errorf("error creating provider: %w", err)
	}

	channelManager, err := channels.NewManager(cfg, msgBus, mediaStore)
	if err != nil {
		mediaStore.Stop()
		return fmt.Errorf("error creating channel manager: %w", err)
	}

	bridgeLoop := bridge.NewLoop(msgBus, channelManager, stateManager, cfg, sessions, provider)
	bridgeLoop.SetMediaStore(mediaStore)

	heartbeatService.SetHandler(func(prompt, channel, chatID string) (string, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		systemPrompt, _ := bridge.AssembleSystemPrompt(cfg.WorkspacePath())
		result, err := provider.Invoke(ctx, providers.InvokeRequest{
			LogKey:       "heartbeat",
			SystemPrompt: systemPrompt,
			UserMessage:  prompt,
			WorkDir:      cfg.WorkspacePath(),
			Stateless:    true,
		})
		if err != nil {
			return "", err
		}

		text, _ := bridge.ParseOutput(result.Text)
		text = strings.TrimSpace(text)
		if text == "HEARTBEAT_OK" {
			return "", nil
		}
		return text, nil
	})

	cronService.SetOnJob(func(job *cron.CronJob) (string, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		if job.Payload.Deliver {
			_ = msgBus.PublishOutbound(ctx, bus.OutboundMessage{
				Channel: job.Payload.Channel,
				ChatID:  job.Payload.To,
				Content: job.Payload.Message,
			})
			return "delivered", nil
		}

		if job.Payload.Command != "" {
			out, err := exec.CommandContext(ctx, "sh", "-c", job.Payload.Command).Output()
			if err != nil {
				return "", fmt.Errorf("cron command failed: %w", err)
			}
			output := strings.TrimSpace(string(out))
			if job.Payload.Channel != "" && job.Payload.To != "" {
				_ = msgBus.PublishOutbound(ctx, bus.OutboundMessage{
					Channel: job.Payload.Channel,
					ChatID:  job.Payload.To,
					Content: output,
				})
			}
			return output, nil
		}

		systemPrompt, _ := bridge.AssembleSystemPrompt(cfg.WorkspacePath())
		result, err := provider.Invoke(ctx, providers.InvokeRequest{
			LogKey:       "cron",
			SystemPrompt: systemPrompt,
			UserMessage:  job.Payload.Message,
			WorkDir:      cfg.WorkspacePath(),
			Stateless:    true,
		})
		if err != nil {
			return "", err
		}

		text, _ := bridge.ParseOutput(result.Text)
		text = strings.TrimSpace(text)
		if job.Payload.Channel != "" && job.Payload.To != "" && text != "" {
			_ = msgBus.PublishOutbound(ctx, bus.OutboundMessage{
				Channel: job.Payload.Channel,
				ChatID:  job.Payload.To,
				Content: text,
			})
		}
		return text, nil
	})

	enabledChannels := channelManager.GetEnabledChannels()
	if len(enabledChannels) > 0 {
		logger.InfoCF("gateway", "Channels enabled", map[string]any{"channels": enabledChannels})
	} else {
		logger.WarnCF("gateway", "No channels enabled", nil)
	}

	logger.InfoCF("gateway", "Gateway started", map[string]any{"host": cfg.Gateway.Host, "port": cfg.Gateway.Port})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := cronService.Start(); err != nil {
		logger.ErrorCF("gateway", "Failed to start cron service", map[string]any{"error": err.Error()})
	} else {
		logger.InfoCF("gateway", "Cron service started", nil)
	}

	if err := heartbeatService.Start(); err != nil {
		logger.ErrorCF("gateway", "Failed to start heartbeat service", map[string]any{"error": err.Error()})
	} else {
		logger.InfoCF("gateway", "Heartbeat service started", nil)
	}

	// Setup shared HTTP server with health endpoints and webhook handlers
	healthServer := health.NewServer(cfg.Gateway.Host, cfg.Gateway.Port)
	addr := fmt.Sprintf("%s:%d", cfg.Gateway.Host, cfg.Gateway.Port)
	channelManager.SetupHTTPServer(addr, healthServer)

	if err := channelManager.StartAll(ctx); err != nil {
		logger.ErrorCF("gateway", "Failed to start channels", map[string]any{"error": err.Error()})
		return err
	}

	logger.InfoCF("gateway", "Health endpoints available", map[string]any{
		"health": fmt.Sprintf("http://%s:%d/health", cfg.Gateway.Host, cfg.Gateway.Port),
		"ready":  fmt.Sprintf("http://%s:%d/ready", cfg.Gateway.Host, cfg.Gateway.Port),
	})

	go bridgeLoop.Run(ctx)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	<-sigChan

	fmt.Println("\nShutting down...")
	cancel()
	msgBus.Close()

	// Use a fresh context with timeout for graceful shutdown,
	// since the original ctx is already canceled.
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()

	channelManager.StopAll(shutdownCtx)
	heartbeatService.Stop()
	cronService.Stop()
	mediaStore.Stop()
	logger.InfoCF("gateway", "Gateway stopped", nil)

	return nil
}
