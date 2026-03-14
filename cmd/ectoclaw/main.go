// EctoClaw - Personal AI assistant powered by Claude Code
// Inspired by and based on PicoClaw: https://github.com/sipeed/picoclaw
// License: MIT
//
// Copyright (c) 2026 Octomatic

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
	"github.com/ectoclaw/ectoclaw/cmd/ectoclaw/internal/skills"
	"github.com/ectoclaw/ectoclaw/cmd/ectoclaw/internal/status"
	"github.com/ectoclaw/ectoclaw/cmd/ectoclaw/internal/version"
)

func NewEctoclawCommand() *cobra.Command {
	short := fmt.Sprintf("%s ectoclaw - Personal AI assistant powered by Claude Code v%s\n\n", internal.Logo, internal.GetVersion())

	cmd := &cobra.Command{
		Use:     "ectoclaw",
		Short:   short,
		Example: "ectoclaw version",
	}

	cmd.AddCommand(
		onboard.NewOnboardCommand(),
		agent.NewAgentCommand(),
		gateway.NewGatewayCommand(),
		status.NewStatusCommand(),
		cron.NewCronCommand(),
		skills.NewSkillsCommand(),
		version.NewVersionCommand(),
	)

	return cmd
}

const (
	colorLeft  = "\033[1;38;2;55;90;150m"
	colorRight = "\033[1;38;2;213;70;70m"
	banner     = "\r\n" +
		colorLeft + "███████╗ ██████╗████████╗ ██████╗ " + colorRight + " ██████╗██╗      █████╗ ██╗    ██╗\n" +
		colorLeft + "██╔════╝██╔════╝╚══██╔══╝██╔═══██╗" + colorRight + "██╔════╝██║     ██╔══██╗██║    ██║\n" +
		colorLeft + "█████╗  ██║        ██║   ██║   ██║" + colorRight + "██║     ██║     ███████║██║ █╗ ██║\n" +
		colorLeft + "██╔══╝  ██║        ██║   ██║   ██║" + colorRight + "██║     ██║     ██╔══██║██║███╗██║\n" +
		colorLeft + "███████╗╚██████╗   ██║   ╚██████╔╝" + colorRight + "╚██████╗███████╗██║  ██║╚███╔███╔╝\n" +
		colorLeft + "╚══════╝ ╚═════╝   ╚═╝    ╚═════╝ " + colorRight + " ╚═════╝╚══════╝╚═╝  ╚═╝ ╚══╝╚══╝\n" +
		"\033[0m\r\n"
)

func main() {
	fmt.Printf("%s", banner)
	cmd := NewEctoclawCommand()
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
