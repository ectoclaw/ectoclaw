package irc

import (
	"github.com/ectoclaw/ectoclaw/pkg/bus"
	"github.com/ectoclaw/ectoclaw/pkg/channels"
	"github.com/ectoclaw/ectoclaw/pkg/config"
)

func init() {
	channels.RegisterFactory("irc", func(cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
		if !cfg.Channels.IRC.Enabled {
			return nil, nil
		}
		return NewIRCChannel(cfg.Channels.IRC, b)
	})
}
