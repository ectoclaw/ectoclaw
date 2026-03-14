package matrix

import (
	"github.com/ectoclaw/ectoclaw/pkg/bus"
	"github.com/ectoclaw/ectoclaw/pkg/channels"
	"github.com/ectoclaw/ectoclaw/pkg/config"
)

func init() {
	channels.RegisterFactory("matrix", func(cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
		return NewMatrixChannel(cfg.Channels.Matrix, b)
	})
}
