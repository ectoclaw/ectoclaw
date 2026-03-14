package slack

import (
	"github.com/ectoclaw/ectoclaw/pkg/bus"
	"github.com/ectoclaw/ectoclaw/pkg/channels"
	"github.com/ectoclaw/ectoclaw/pkg/config"
)

func init() {
	channels.RegisterFactory("slack", func(cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
		return NewSlackChannel(cfg.Channels.Slack, b)
	})
}
