package line

import (
	"github.com/ectoclaw/ectoclaw/pkg/bus"
	"github.com/ectoclaw/ectoclaw/pkg/channels"
	"github.com/ectoclaw/ectoclaw/pkg/config"
)

func init() {
	channels.RegisterFactory("line", func(cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
		return NewLINEChannel(cfg.Channels.LINE, b)
	})
}
