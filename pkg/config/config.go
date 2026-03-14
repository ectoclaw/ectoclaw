package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/caarlos0/env/v11"

	"github.com/ectoclaw/ectoclaw/pkg/fileutil"
)

// FlexibleStringSlice is a []string that also accepts JSON numbers,
// so allow_from can contain both "123" and 123.
type FlexibleStringSlice []string

func (f *FlexibleStringSlice) UnmarshalJSON(data []byte) error {
	var ss []string
	if err := json.Unmarshal(data, &ss); err == nil {
		*f = ss
		return nil
	}

	var raw []any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	result := make([]string, 0, len(raw))
	for _, v := range raw {
		switch val := v.(type) {
		case string:
			result = append(result, val)
		case float64:
			result = append(result, fmt.Sprintf("%.0f", val))
		default:
			result = append(result, fmt.Sprintf("%v", val))
		}
	}
	*f = result
	return nil
}

// BridgeConfig controls how EctoClaw invokes the coding-agent CLI subprocess.
type BridgeConfig struct {
	// Workspace is the path used for sessions, skills, cron jobs, heartbeat, etc.
	// Defaults to ~/.ectoclaw/workspace or $ECTOCLAW_HOME/workspace.
	Workspace string `json:"workspace,omitempty" env:"ECTOCLAW_WORKSPACE"`
	// Provider selects the coding-agent CLI backend: "claude" (default) or "codex".
	Provider string `json:"provider,omitempty" env:"ECTOCLAW_BRIDGE_PROVIDER"`
	// Model is passed as --model to the CLI. If empty, the CLI uses its own default.
	Model string `json:"model,omitempty" env:"ECTOCLAW_BRIDGE_MODEL"`
}

type Config struct {
	Channels     ChannelsConfig     `json:"channels"`
Gateway      GatewayConfig      `json:"gateway"`
	Skills       SkillsConfig       `json:"skills,omitempty"`
	MediaCleanup MediaCleanupConfig `json:"media_cleanup,omitempty"`
	Heartbeat    HeartbeatConfig    `json:"heartbeat"`
	Logs         LogsConfig         `json:"logs,omitempty"`
	Bridge       BridgeConfig       `json:"bridge,omitempty"`
}

type ChannelsConfig struct {
	WhatsApp WhatsAppConfig `json:"whatsapp"`
	Telegram TelegramConfig `json:"telegram"`
	Discord  DiscordConfig  `json:"discord"`
	Slack    SlackConfig    `json:"slack"`
	Matrix   MatrixConfig   `json:"matrix"`
	LINE     LINEConfig     `json:"line"`
	IRC      IRCConfig      `json:"irc"`
}

// GroupTriggerConfig controls when the bot responds in group chats.
type GroupTriggerConfig struct {
	MentionOnly bool     `json:"mention_only,omitempty"`
	Prefixes    []string `json:"prefixes,omitempty"`
}

// TypingConfig controls typing indicator behavior.
type TypingConfig struct {
	Enabled bool `json:"enabled,omitempty"`
}

type WhatsAppConfig struct {
	Enabled            bool                `json:"enabled"              env:"ECTOCLAW_CHANNELS_WHATSAPP_ENABLED"`
	BridgeURL          string              `json:"bridge_url"           env:"ECTOCLAW_CHANNELS_WHATSAPP_BRIDGE_URL"`
	UseNative          bool                `json:"use_native"           env:"ECTOCLAW_CHANNELS_WHATSAPP_USE_NATIVE"`
	SessionStorePath   string              `json:"session_store_path"   env:"ECTOCLAW_CHANNELS_WHATSAPP_SESSION_STORE_PATH"`
	AllowFrom          FlexibleStringSlice `json:"allow_from"           env:"ECTOCLAW_CHANNELS_WHATSAPP_ALLOW_FROM"`
	ReasoningChannelID string              `json:"reasoning_channel_id" env:"ECTOCLAW_CHANNELS_WHATSAPP_REASONING_CHANNEL_ID"`
}

type TelegramConfig struct {
	Enabled            bool                `json:"enabled"                 env:"ECTOCLAW_CHANNELS_TELEGRAM_ENABLED"`
	Token              string              `json:"token"                   env:"ECTOCLAW_CHANNELS_TELEGRAM_TOKEN"`
	BaseURL            string              `json:"base_url"                env:"ECTOCLAW_CHANNELS_TELEGRAM_BASE_URL"`
	Proxy              string              `json:"proxy"                   env:"ECTOCLAW_CHANNELS_TELEGRAM_PROXY"`
	AllowFrom          FlexibleStringSlice `json:"allow_from"              env:"ECTOCLAW_CHANNELS_TELEGRAM_ALLOW_FROM"`
	GroupTrigger       GroupTriggerConfig  `json:"group_trigger,omitempty"`
	Typing             TypingConfig        `json:"typing,omitempty"`
	ReasoningChannelID string              `json:"reasoning_channel_id"    env:"ECTOCLAW_CHANNELS_TELEGRAM_REASONING_CHANNEL_ID"`
}

type DiscordConfig struct {
	Enabled            bool                `json:"enabled"                 env:"ECTOCLAW_CHANNELS_DISCORD_ENABLED"`
	Token              string              `json:"token"                   env:"ECTOCLAW_CHANNELS_DISCORD_TOKEN"`
	Proxy              string              `json:"proxy"                   env:"ECTOCLAW_CHANNELS_DISCORD_PROXY"`
	AllowFrom          FlexibleStringSlice `json:"allow_from"              env:"ECTOCLAW_CHANNELS_DISCORD_ALLOW_FROM"`
	MentionOnly        bool                `json:"mention_only"            env:"ECTOCLAW_CHANNELS_DISCORD_MENTION_ONLY"`
	GroupTrigger       GroupTriggerConfig  `json:"group_trigger,omitempty"`
	Typing             TypingConfig        `json:"typing,omitempty"`
	ReasoningChannelID string              `json:"reasoning_channel_id"    env:"ECTOCLAW_CHANNELS_DISCORD_REASONING_CHANNEL_ID"`
}

type SlackConfig struct {
	Enabled            bool                `json:"enabled"                 env:"ECTOCLAW_CHANNELS_SLACK_ENABLED"`
	BotToken           string              `json:"bot_token"               env:"ECTOCLAW_CHANNELS_SLACK_BOT_TOKEN"`
	AppToken           string              `json:"app_token"               env:"ECTOCLAW_CHANNELS_SLACK_APP_TOKEN"`
	AllowFrom          FlexibleStringSlice `json:"allow_from"              env:"ECTOCLAW_CHANNELS_SLACK_ALLOW_FROM"`
	GroupTrigger       GroupTriggerConfig  `json:"group_trigger,omitempty"`
	Typing             TypingConfig        `json:"typing,omitempty"`
	ReasoningChannelID string              `json:"reasoning_channel_id"    env:"ECTOCLAW_CHANNELS_SLACK_REASONING_CHANNEL_ID"`
}

type MatrixConfig struct {
	Enabled            bool                `json:"enabled"             env:"ECTOCLAW_CHANNELS_MATRIX_ENABLED"`
	Homeserver         string              `json:"homeserver"          env:"ECTOCLAW_CHANNELS_MATRIX_HOMESERVER"`
	UserID             string              `json:"user_id"             env:"ECTOCLAW_CHANNELS_MATRIX_USER_ID"`
	AccessToken        string              `json:"access_token"        env:"ECTOCLAW_CHANNELS_MATRIX_ACCESS_TOKEN"`
	DeviceID           string              `json:"device_id,omitempty" env:"ECTOCLAW_CHANNELS_MATRIX_DEVICE_ID"`
	JoinOnInvite       bool                `json:"join_on_invite"      env:"ECTOCLAW_CHANNELS_MATRIX_JOIN_ON_INVITE"`
	AllowFrom          FlexibleStringSlice `json:"allow_from"          env:"ECTOCLAW_CHANNELS_MATRIX_ALLOW_FROM"`
	GroupTrigger       GroupTriggerConfig  `json:"group_trigger,omitempty"`
	ReasoningChannelID string              `json:"reasoning_channel_id" env:"ECTOCLAW_CHANNELS_MATRIX_REASONING_CHANNEL_ID"`
}

type LINEConfig struct {
	Enabled            bool                `json:"enabled"              env:"ECTOCLAW_CHANNELS_LINE_ENABLED"`
	ChannelSecret      string              `json:"channel_secret"       env:"ECTOCLAW_CHANNELS_LINE_CHANNEL_SECRET"`
	ChannelAccessToken string              `json:"channel_access_token" env:"ECTOCLAW_CHANNELS_LINE_CHANNEL_ACCESS_TOKEN"`
	WebhookHost        string              `json:"webhook_host"         env:"ECTOCLAW_CHANNELS_LINE_WEBHOOK_HOST"`
	WebhookPort        int                 `json:"webhook_port"         env:"ECTOCLAW_CHANNELS_LINE_WEBHOOK_PORT"`
	WebhookPath        string              `json:"webhook_path"         env:"ECTOCLAW_CHANNELS_LINE_WEBHOOK_PATH"`
	AllowFrom          FlexibleStringSlice `json:"allow_from"           env:"ECTOCLAW_CHANNELS_LINE_ALLOW_FROM"`
	GroupTrigger       GroupTriggerConfig  `json:"group_trigger,omitempty"`
	Typing             TypingConfig        `json:"typing,omitempty"`
	ReasoningChannelID string              `json:"reasoning_channel_id" env:"ECTOCLAW_CHANNELS_LINE_REASONING_CHANNEL_ID"`
}

type IRCConfig struct {
	Enabled            bool                `json:"enabled"                env:"ECTOCLAW_CHANNELS_IRC_ENABLED"`
	Server             string              `json:"server"                 env:"ECTOCLAW_CHANNELS_IRC_SERVER"`
	TLS                bool                `json:"tls"                    env:"ECTOCLAW_CHANNELS_IRC_TLS"`
	Nick               string              `json:"nick"                   env:"ECTOCLAW_CHANNELS_IRC_NICK"`
	User               string              `json:"user,omitempty"         env:"ECTOCLAW_CHANNELS_IRC_USER"`
	RealName           string              `json:"real_name,omitempty"    env:"ECTOCLAW_CHANNELS_IRC_REAL_NAME"`
	Password           string              `json:"password"               env:"ECTOCLAW_CHANNELS_IRC_PASSWORD"`
	NickServPassword   string              `json:"nickserv_password"      env:"ECTOCLAW_CHANNELS_IRC_NICKSERV_PASSWORD"`
	SASLUser           string              `json:"sasl_user"              env:"ECTOCLAW_CHANNELS_IRC_SASL_USER"`
	SASLPassword       string              `json:"sasl_password"          env:"ECTOCLAW_CHANNELS_IRC_SASL_PASSWORD"`
	Channels           FlexibleStringSlice `json:"channels"               env:"ECTOCLAW_CHANNELS_IRC_CHANNELS"`
	RequestCaps        FlexibleStringSlice `json:"request_caps,omitempty" env:"ECTOCLAW_CHANNELS_IRC_REQUEST_CAPS"`
	AllowFrom          FlexibleStringSlice `json:"allow_from"             env:"ECTOCLAW_CHANNELS_IRC_ALLOW_FROM"`
	GroupTrigger       GroupTriggerConfig  `json:"group_trigger,omitempty"`
	Typing             TypingConfig        `json:"typing,omitempty"`
	ReasoningChannelID string              `json:"reasoning_channel_id"   env:"ECTOCLAW_CHANNELS_IRC_REASONING_CHANNEL_ID"`
}

type HeartbeatConfig struct {
	Enabled  bool `json:"enabled"  env:"ECTOCLAW_HEARTBEAT_ENABLED"`
	Interval int  `json:"interval" env:"ECTOCLAW_HEARTBEAT_INTERVAL"` // minutes, min 5
}

type LogsConfig struct {
	Agent bool `json:"agent" env:"ECTOCLAW_LOGS_AGENT"` // write per-session invoke logs to workspace/logs/
}


type GatewayConfig struct {
	Host string `json:"host" env:"ECTOCLAW_GATEWAY_HOST"`
	Port int    `json:"port" env:"ECTOCLAW_GATEWAY_PORT"`
}

type SkillsConfig struct {
	Enabled               bool                   `json:"enabled"                env:"ECTOCLAW_SKILLS_ENABLED"`
	Registries            SkillsRegistriesConfig `json:"registries"`
	MaxConcurrentSearches int                    `json:"max_concurrent_searches" env:"ECTOCLAW_SKILLS_MAX_CONCURRENT_SEARCHES"`
	SearchCache           SearchCacheConfig      `json:"search_cache"`
}

type MediaCleanupConfig struct {
	Enabled  bool `json:"enabled"          env:"ECTOCLAW_MEDIA_CLEANUP_ENABLED"`
	MaxAge   int  `json:"max_age_minutes"  env:"ECTOCLAW_MEDIA_CLEANUP_MAX_AGE"`
	Interval int  `json:"interval_minutes" env:"ECTOCLAW_MEDIA_CLEANUP_INTERVAL"`
}

type SearchCacheConfig struct {
	MaxSize    int `json:"max_size"    env:"ECTOCLAW_SKILLS_SEARCH_CACHE_MAX_SIZE"`
	TTLSeconds int `json:"ttl_seconds" env:"ECTOCLAW_SKILLS_SEARCH_CACHE_TTL_SECONDS"`
}

type SkillsRegistriesConfig struct {
	ClawHub ClawHubRegistryConfig `json:"clawhub"`
}

type ClawHubRegistryConfig struct {
	Enabled         bool   `json:"enabled"           env:"ECTOCLAW_SKILLS_REGISTRIES_CLAWHUB_ENABLED"`
	BaseURL         string `json:"base_url"          env:"ECTOCLAW_SKILLS_REGISTRIES_CLAWHUB_BASE_URL"`
	AuthToken       string `json:"auth_token"        env:"ECTOCLAW_SKILLS_REGISTRIES_CLAWHUB_AUTH_TOKEN"`
	SearchPath      string `json:"search_path"       env:"ECTOCLAW_SKILLS_REGISTRIES_CLAWHUB_SEARCH_PATH"`
	SkillsPath      string `json:"skills_path"       env:"ECTOCLAW_SKILLS_REGISTRIES_CLAWHUB_SKILLS_PATH"`
	DownloadPath    string `json:"download_path"     env:"ECTOCLAW_SKILLS_REGISTRIES_CLAWHUB_DOWNLOAD_PATH"`
	Timeout         int    `json:"timeout"           env:"ECTOCLAW_SKILLS_REGISTRIES_CLAWHUB_TIMEOUT"`
	MaxZipSize      int    `json:"max_zip_size"      env:"ECTOCLAW_SKILLS_REGISTRIES_CLAWHUB_MAX_ZIP_SIZE"`
	MaxResponseSize int    `json:"max_response_size" env:"ECTOCLAW_SKILLS_REGISTRIES_CLAWHUB_MAX_RESPONSE_SIZE"`
}

func LoadConfig(path string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	cfg.migrateChannelConfigs()

	return cfg, nil
}

func SaveConfig(path string, cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return fileutil.WriteFileAtomic(path, data, 0o600)
}

func (c *Config) WorkspacePath() string {
	return expandHome(c.Bridge.Workspace)
}

func expandHome(path string) string {
	if path == "" {
		return path
	}
	if path[0] == '~' {
		home, _ := os.UserHomeDir()
		if len(path) > 1 && path[1] == '/' {
			return home + path[1:]
		}
		return home
	}
	return path
}

