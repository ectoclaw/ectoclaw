package discord

import (
	"github.com/ectoclaw/ectoclaw/pkg/bus"
	"github.com/ectoclaw/ectoclaw/pkg/channels"
	"github.com/ectoclaw/ectoclaw/pkg/config"
)

func init() {
	channels.RegisterFactory("discord", func(cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
		return NewDiscordChannel(cfg.Channels.Discord, b)
	})
}
