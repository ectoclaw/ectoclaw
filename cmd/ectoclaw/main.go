package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/ectoclaw/ectoclaw/cmd/ectoclaw/internal"
	"github.com/ectoclaw/ectoclaw/cmd/ectoclaw/internal/agent"
	"github.com/ectoclaw/ectoclaw/cmd/ectoclaw/internal/cron"
	"github.com/ectoclaw/ectoclaw/cmd/ectoclaw/internal/gateway"
	"github.com/ectoclaw/ectoclaw/cmd/ectoclaw/internal/onboard"
	"github.com/ectoclaw/ectoclaw/cmd/ectoclaw/internal/service"
	"github.com/ectoclaw/ectoclaw/cmd/ectoclaw/internal/skills"
	"github.com/ectoclaw/ectoclaw/cmd/ectoclaw/internal/status"
	"github.com/ectoclaw/ectoclaw/cmd/ectoclaw/internal/upgrade"
	"github.com/ectoclaw/ectoclaw/cmd/ectoclaw/internal/version"
)

func NewEctoclawCommand() *cobra.Command {
	short := fmt.Sprintf("%s ectoclaw - Personal AI assistant powered by your coding agent v%s\n\n", internal.Logo, internal.GetVersion())

	cmd := &cobra.Command{
		Use:     "ectoclaw",
		Short:   short,
		Example: "ectoclaw version",
	}

	cmd.AddCommand(
		onboard.NewOnboardCommand(),
		agent.NewAgentCommand(),
		gateway.NewGatewayCommand(),
		service.NewServiceCommand(),
		status.NewStatusCommand(),
		cron.NewCronCommand(),
		skills.NewSkillsCommand(),
		upgrade.NewUpgradeCommand(),
		version.NewVersionCommand(),
	)

	return cmd
}


func main() {
	cmd := NewEctoclawCommand()
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
