package config

import (
	"testing"
)

func TestMigrateChannelConfigs_DiscordMentionOnly(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Channels.Discord.MentionOnly = true
	cfg.Channels.Discord.GroupTrigger.MentionOnly = false
	cfg.migrateChannelConfigs()
	if !cfg.Channels.Discord.GroupTrigger.MentionOnly {
		t.Error("migrateChannelConfigs should propagate Discord.MentionOnly to GroupTrigger.MentionOnly")
	}
}
