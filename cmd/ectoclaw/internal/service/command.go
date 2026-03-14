package service

import (
	"github.com/spf13/cobra"
)

func NewServiceCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "service",
		Short: "Manage ectoclaw background service",
	}

	cmd.AddCommand(
		newInstallCommand(),
		newUninstallCommand(),
		newStartCommand(),
		newStopCommand(),
		newStatusCommand(),
	)

	return cmd
}

func newInstallCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "Install ectoclaw as a system service",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return install()
		},
	}
}

func newUninstallCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall the ectoclaw system service",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return uninstall()
		},
	}
}

func newStartCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start the ectoclaw service",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return start()
		},
	}
}

func newStopCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop the ectoclaw service",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return stop()
		},
	}
}

func newStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show ectoclaw service status",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return status()
		},
	}
}
