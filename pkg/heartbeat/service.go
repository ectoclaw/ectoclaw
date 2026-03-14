package heartbeat

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ectoclaw/ectoclaw/pkg/bus"
	"github.com/ectoclaw/ectoclaw/pkg/constants"
	"github.com/ectoclaw/ectoclaw/pkg/logger"
	"github.com/ectoclaw/ectoclaw/pkg/state"
)

const (
	minIntervalMinutes     = 5
	defaultIntervalMinutes = 30
)

// HeartbeatHandler is the function type for handling heartbeat.
// Returns the text to send to the user, or "" for silent (nothing to report), or an error.
type HeartbeatHandler func(prompt, channel, chatID string) (string, error)

// HeartbeatService manages periodic heartbeat checks
type HeartbeatService struct {
	workspace string
	bus       *bus.MessageBus
	state     *state.Manager
	handler   HeartbeatHandler
	interval  time.Duration
	enabled   bool
	mu        sync.RWMutex
	stopChan  chan struct{}
}

// NewHeartbeatService creates a new heartbeat service
func NewHeartbeatService(workspace string, intervalMinutes int, enabled bool) *HeartbeatService {
	// Apply minimum interval
	if intervalMinutes < minIntervalMinutes && intervalMinutes != 0 {
		intervalMinutes = minIntervalMinutes
	}

	if intervalMinutes == 0 {
		intervalMinutes = defaultIntervalMinutes
	}

	return &HeartbeatService{
		workspace: workspace,
		interval:  time.Duration(intervalMinutes) * time.Minute,
		enabled:   enabled,
		state:     state.NewManager(workspace),
	}
}

// SetBus sets the message bus for delivering heartbeat results.
func (hs *HeartbeatService) SetBus(msgBus *bus.MessageBus) {
	hs.mu.Lock()
	defer hs.mu.Unlock()
	hs.bus = msgBus
}

// SetHandler sets the heartbeat handler.
func (hs *HeartbeatService) SetHandler(handler HeartbeatHandler) {
	hs.mu.Lock()
	defer hs.mu.Unlock()
	hs.handler = handler
}

// Start begins the heartbeat service
func (hs *HeartbeatService) Start() error {
	hs.mu.Lock()
	defer hs.mu.Unlock()

	if hs.stopChan != nil {
		logger.InfoC("heartbeat", "Heartbeat service already running")
		return nil
	}

	if !hs.enabled {
		logger.InfoC("heartbeat", "Heartbeat service disabled")
		return nil
	}

	hs.stopChan = make(chan struct{})
	go hs.runLoop(hs.stopChan)

	logger.InfoCF("heartbeat", "Heartbeat service started", map[string]any{
		"interval_minutes": hs.interval.Minutes(),
	})

	return nil
}

// Stop gracefully stops the heartbeat service
func (hs *HeartbeatService) Stop() {
	hs.mu.Lock()
	defer hs.mu.Unlock()

	if hs.stopChan == nil {
		return
	}

	logger.InfoC("heartbeat", "Stopping heartbeat service")
	close(hs.stopChan)
	hs.stopChan = nil
}

// IsRunning returns whether the service is running
func (hs *HeartbeatService) IsRunning() bool {
	hs.mu.RLock()
	defer hs.mu.RUnlock()
	return hs.stopChan != nil
}

// runLoop runs the heartbeat ticker
func (hs *HeartbeatService) runLoop(stopChan chan struct{}) {
	ticker := time.NewTicker(hs.interval)
	defer ticker.Stop()

	// Run first heartbeat after initial delay
	time.AfterFunc(time.Second, func() {
		hs.executeHeartbeat()
	})

	for {
		select {
		case <-stopChan:
			return
		case <-ticker.C:
			hs.executeHeartbeat()
		}
	}
}

// executeHeartbeat performs a single heartbeat check
func (hs *HeartbeatService) executeHeartbeat() {
	hs.mu.RLock()
	enabled := hs.enabled
	handler := hs.handler
	if !hs.enabled || hs.stopChan == nil {
		hs.mu.RUnlock()
		return
	}
	hs.mu.RUnlock()

	if !enabled {
		return
	}

	logger.DebugC("heartbeat", "Executing heartbeat")

	prompt := hs.buildPrompt()
	if prompt == "" {
		logger.InfoC("heartbeat", "No heartbeat prompt (HEARTBEAT.md empty or missing)")
		return
	}

	if handler == nil {
		logger.WarnC("heartbeat", "Heartbeat handler not configured")
		return
	}

	// Get last channel info for context
	channel, chatID := hs.parseLastChannel(hs.state.GetLastChannel())

	logger.DebugCF("heartbeat", "Resolved channel", map[string]any{"channel": channel, "chat_id": chatID})

	text, err := handler(prompt, channel, chatID)

	if err != nil {
		logger.WarnCF("heartbeat", "Heartbeat error", map[string]any{"error": err.Error()})
		return
	}

	if text == "" {
		logger.InfoC("heartbeat", "Heartbeat OK (noop)")
		return
	}

	hs.sendResponse(text)
	logger.InfoCF("heartbeat", "Heartbeat completed", map[string]any{"len": len(text)})
}

// buildPrompt builds the heartbeat prompt from HEARTBEAT.md
func (hs *HeartbeatService) buildPrompt() string {
	heartbeatPath := filepath.Join(hs.workspace, "HEARTBEAT.md")

	data, err := os.ReadFile(heartbeatPath)
	if err != nil {
		logger.WarnCF("heartbeat", "Error reading HEARTBEAT.md", map[string]any{"error": err.Error()})
		return ""
	}

	content := string(data)
	if len(content) == 0 {
		return ""
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	return fmt.Sprintf(`# Heartbeat Check

Current time: %s

You are a proactive AI assistant. This is a scheduled heartbeat check.
Review the following tasks and execute any necessary actions using available skills.
If there is nothing that requires attention, respond ONLY with: HEARTBEAT_OK

%s
`, now, content)
}

// sendResponse sends the heartbeat response to the last channel
func (hs *HeartbeatService) sendResponse(response string) {
	hs.mu.RLock()
	msgBus := hs.bus
	hs.mu.RUnlock()

	if msgBus == nil {
		logger.InfoC("heartbeat", "No message bus configured, heartbeat result not sent")
		return
	}

	// Get last channel from state
	platform, userID := hs.parseLastChannel(hs.state.GetLastChannel())
	if platform == "" {
		logger.InfoC("heartbeat", "No last channel recorded, heartbeat result not sent")
		return
	}

	pubCtx, pubCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer pubCancel()
	msgBus.PublishOutbound(pubCtx, bus.OutboundMessage{
		Channel: platform,
		ChatID:  userID,
		Content: response,
	})

	logger.InfoCF("heartbeat", "Heartbeat result sent", map[string]any{"platform": platform})
}

// parseLastChannel splits a "platform:chatID" string into its two parts.
// Returns empty strings if the format is invalid or the platform is internal.
func (hs *HeartbeatService) parseLastChannel(last string) (platform, chatID string) {
	parts := strings.SplitN(last, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", ""
	}
	if constants.IsInternalChannel(parts[0]) {
		return "", ""
	}
	return parts[0], parts[1]
}
