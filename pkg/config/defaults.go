package config

import (
	"os"
	"path/filepath"
)

// DefaultConfig returns the default configuration for EctoClaw.
func DefaultConfig() *Config {
	// Determine the base path for the workspace.
	// Priority: $ECTOCLAW_HOME > ~/.ectoclaw
	var homePath string
	if ectoclawHome := os.Getenv("ECTOCLAW_HOME"); ectoclawHome != "" {
		homePath = ectoclawHome
	} else {
		userHome, _ := os.UserHomeDir()
		homePath = filepath.Join(userHome, ".ectoclaw")
	}
	workspacePath := filepath.Join(homePath, "workspace")

	return &Config{
		Bridge: BridgeConfig{
			Workspace:          workspacePath,
			Provider:           "claude",
			IdleTimeoutMinutes: 5,
			MaxRetries:         3,
		},
		Channels: ChannelsConfig{
			WhatsApp: WhatsAppConfig{
				Enabled:          false,
				BridgeURL:        "ws://localhost:3001",
				UseNative:        false,
				SessionStorePath: "",
				AllowFrom:        FlexibleStringSlice{},
			},
			Telegram: TelegramConfig{
				Enabled:   false,
				Token:     "",
				AllowFrom: FlexibleStringSlice{},
				Typing:    TypingConfig{Enabled: true},
			},
			Discord: DiscordConfig{
				Enabled:     false,
				Token:       "",
				AllowFrom:   FlexibleStringSlice{},
				MentionOnly: false,
			},
			Slack: SlackConfig{
				Enabled:   false,
				BotToken:  "",
				AppToken:  "",
				AllowFrom: FlexibleStringSlice{},
			},
			Matrix: MatrixConfig{
				Enabled:      false,
				Homeserver:   "https://matrix.org",
				UserID:       "",
				AccessToken:  "",
				DeviceID:     "",
				JoinOnInvite: true,
				AllowFrom:    FlexibleStringSlice{},
				GroupTrigger: GroupTriggerConfig{
					MentionOnly: true,
				},
			},
			LINE: LINEConfig{
				Enabled:            false,
				ChannelSecret:      "",
				ChannelAccessToken: "",
				WebhookHost:        "0.0.0.0",
				WebhookPort:        18791,
				WebhookPath:        "/webhook/line",
				AllowFrom:          FlexibleStringSlice{},
				GroupTrigger:       GroupTriggerConfig{MentionOnly: true},
			},
		},
		Gateway: GatewayConfig{
			Host: "127.0.0.1",
			Port: 18790,
		},
		Skills: SkillsConfig{
			Enabled: true,
			Registries: SkillsRegistriesConfig{
				ClawHub: ClawHubRegistryConfig{
					Enabled: true,
					BaseURL: "https://clawhub.ai",
				},
			},
			MaxConcurrentSearches: 2,
			SearchCache: SearchCacheConfig{
				MaxSize:    50,
				TTLSeconds: 300,
			},
		},
		MediaCleanup: MediaCleanupConfig{
			Enabled:  true,
			MaxAge:   30,
			Interval: 5,
		},
		Heartbeat: HeartbeatConfig{
			Enabled:  true,
			Interval: 30,
		},
	}
}
