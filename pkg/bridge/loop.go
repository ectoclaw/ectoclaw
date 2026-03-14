package bridge

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ectoclaw/ectoclaw/pkg/bus"
	"github.com/ectoclaw/ectoclaw/pkg/channels"
	"github.com/ectoclaw/ectoclaw/pkg/config"
	"github.com/ectoclaw/ectoclaw/pkg/logger"
	"github.com/ectoclaw/ectoclaw/pkg/media"
	"github.com/ectoclaw/ectoclaw/pkg/providers"
	"github.com/ectoclaw/ectoclaw/pkg/state"
)

// Loop is the main event loop that bridges inbound messages to the coding-agent CLI.
type Loop struct {
	bus            *bus.MessageBus
	sessions       *Sessions
	channelManager *channels.Manager
	stateManager   *state.Manager
	cfg            *config.Config
	provider       providers.Provider
	mediaStore     media.MediaStore
	inFlight       sync.Map // SessionKey → struct{}{}: dedup in-flight requests
}

// NewLoop creates a new bridge Loop.
func NewLoop(
	msgBus *bus.MessageBus,
	cm *channels.Manager,
	sm *state.Manager,
	cfg *config.Config,
	sessions *Sessions,
	provider providers.Provider,
) *Loop {
	return &Loop{
		bus:            msgBus,
		sessions:       sessions,
		channelManager: cm,
		stateManager:   sm,
		cfg:            cfg,
		provider:       provider,
	}
}

// SetMediaStore sets the media store used for file/image delivery.
func (l *Loop) SetMediaStore(s media.MediaStore) {
	l.mediaStore = s
}

// Run consumes inbound messages and dispatches a goroutine per message.
// Returns nil when the context is cancelled or the bus is closed.
func (l *Loop) Run(ctx context.Context) error {
	for {
		msg, ok := l.bus.ConsumeInbound(ctx)
		if !ok {
			return nil
		}
		go l.handleMessage(ctx, msg)
	}
}

func (l *Loop) handleMessage(ctx context.Context, msg bus.InboundMessage) {
	// Dedup in-flight requests per session.
	if _, loaded := l.inFlight.LoadOrStore(msg.SessionKey, struct{}{}); loaded {
		logger.WarnCF("bridge", "Dropping duplicate in-flight message", map[string]any{
			"session_key": msg.SessionKey,
		})
		return
	}
	defer l.inFlight.Delete(msg.SessionKey)

	// Start typing indicator if the channel supports it.
	stop := func() {}
	if ch, ok := l.channelManager.GetChannel(msg.Channel); ok {
		if tc, ok := ch.(channels.TypingCapable); ok {
			if stopFn, err := tc.StartTyping(ctx, msg.ChatID); err == nil {
				stop = stopFn
			}
		}
	}
	defer func() { stop() }()

	// Always assemble system prompt — required on every call since --resume does not persist it.
	prompt, err := AssembleSystemPrompt(l.cfg.WorkspacePath())
	if err != nil {
		logger.WarnCF("bridge", "Failed to assemble system prompt", map[string]any{
			"error": err.Error(),
		})
	}

	// Look up existing session.
	existingID, _ := l.sessions.Get(msg.SessionKey)

	// Append resolved media paths to the user message so the AI can access the files.
	userMessage := msg.Content
	if len(msg.Media) > 0 && l.mediaStore != nil {
		filesDir := filepath.Join(l.cfg.WorkspacePath(), "history",
			time.Now().UTC().Format("2006-01-02"), "files")
		for _, ref := range msg.Media {
			tempPath, meta, err := l.mediaStore.ResolveWithMeta(ref)
			if err != nil {
				continue
			}
			destPath := persistInboundFile(tempPath, meta.Filename, filesDir)
			userMessage += "\n[file: " + destPath + "]"
		}
	}

	// Invoke the provider.
	logger.DebugCF("bridge", "Invoking provider", map[string]any{
		"provider":   l.provider.Name(),
		"work_dir":   l.cfg.WorkspacePath(),
		"session_id": existingID,
	})
	result, err := l.provider.Invoke(ctx, providers.InvokeRequest{
		LogKey:       sanitizeSessionKey(msg.SessionKey),
		SessionKey:   msg.SessionKey,
		SessionID:    existingID,
		SystemPrompt: prompt,
		UserMessage:  userMessage,
		WorkDir:      l.cfg.WorkspacePath(),
		Model:        l.cfg.Bridge.Model,
	})

	logger.DebugCF("bridge", "Invoke result", map[string]any{
		"session_id": result.SessionID,
		"tokens_in":  result.TokensIn,
		"tokens_out": result.TokensOut,
		"text_len":   len(result.Text),
	})

	if err != nil {
		stop()
		stop = func() {}
		var provMsg *providers.ErrProviderMessage
		if errors.As(err, &provMsg) {
			// Human-readable provider error — forward directly to the user, no internal logging needed.
			_ = l.bus.PublishOutbound(ctx, bus.OutboundMessage{
				Channel: msg.Channel,
				ChatID:  msg.ChatID,
				Content: provMsg.Message,
			})
		} else {
			logger.ErrorCF("bridge", "Failed to invoke provider", map[string]any{
				"session_key": msg.SessionKey,
				"error":       err.Error(),
			})
			_ = l.bus.PublishOutbound(ctx, bus.OutboundMessage{
				Channel: msg.Channel,
				ChatID:  msg.ChatID,
				Content: fmt.Sprintf("Error: %v", err),
			})
		}
		return
	}

	// Stop typing before publishing response.
	stop()
	stop = func() {}

	// Parse output and strip file/image markers.
	cleanText, filePaths := ParseOutput(result.Text)

	// Build media parts from any file paths the AI produced.
	var parts []bus.MediaPart
	if len(filePaths) > 0 && l.mediaStore != nil {
		for _, fp := range filePaths {
			ref, storeErr := l.mediaStore.Store(fp, media.MediaMeta{
				Filename: filepath.Base(fp),
				Source:   "bridge",
			}, msg.SessionKey)
			if storeErr != nil {
				logger.WarnCF("bridge", "Failed to store media file", map[string]any{
					"path":  fp,
					"error": storeErr.Error(),
				})
				continue
			}
			parts = append(parts, bus.MediaPart{
				Type:     inferMediaType(fp),
				Ref:      ref,
				Filename: filepath.Base(fp),
			})
		}
	}

	// Publish a single message — channel decides how to present text alongside media.
	if cleanText != "" || len(parts) > 0 {
		if pubErr := l.bus.PublishOutbound(ctx, bus.OutboundMessage{
			Channel: msg.Channel,
			ChatID:  msg.ChatID,
			Content: cleanText,
			Parts:   parts,
		}); pubErr != nil {
			logger.WarnCF("bridge", "Failed to publish outbound message", map[string]any{
				"error": pubErr.Error(),
			})
		}
	}

	// Append to conversation history (best-effort).
	if cleanText != "" {
		if appendErr := AppendHistory(l.cfg.WorkspacePath(), userMessage, cleanText); appendErr != nil {
			logger.WarnCF("bridge", "Failed to append history", map[string]any{"error": appendErr.Error()})
		}
	}

	// Persist session mapping.
	if result.SessionID != "" {
		l.sessions.Set(msg.SessionKey, result.SessionID)
		if saveErr := l.sessions.Save(); saveErr != nil {
			logger.WarnCF("bridge", "Failed to save sessions", map[string]any{
				"error": saveErr.Error(),
			})
		}
	}

	// Update last-active state.
	if err := l.stateManager.SetLastChannel(msg.Channel + ":" + msg.ChatID); err != nil {
		logger.WarnCF("bridge", "Failed to set last channel", map[string]any{"error": err.Error()})
	}
}

// inferMediaType returns the bus MediaPart type string based on file extension.
// persistInboundFile copies a temp file into the daily history files directory.
// Returns the destination path, or the original path on any error.
func persistInboundFile(src, filename, destDir string) string {
	if err := os.MkdirAll(destDir, 0o700); err != nil {
		return src
	}

	if filename == "" {
		filename = filepath.Base(src)
	}

	// Avoid collisions: prefix with timestamp millis.
	dest := filepath.Join(destDir, fmt.Sprintf("%d_%s", time.Now().UnixMilli(), filename))

	in, err := os.Open(src)
	if err != nil {
		return src
	}
	defer in.Close()

	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o600)
	if err != nil {
		return src
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		os.Remove(dest)
		return src
	}

	return dest
}

// sanitizeSessionKey converts a bridge session key like "telegram:300185392"
// to a safe filename component like "telegram_300185392".
func sanitizeSessionKey(key string) string {
	return strings.NewReplacer(":", "_", "/", "_", " ", "_").Replace(key)
}

func inferMediaType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		return "image"
	case ".mp4", ".mov", ".avi":
		return "video"
	case ".mp3", ".wav", ".ogg", ".m4a":
		return "audio"
	default:
		return "file"
	}
}
