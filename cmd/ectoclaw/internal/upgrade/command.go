package upgrade

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ectoclaw/ectoclaw/cmd/ectoclaw/internal"
)

func NewUpgradeCommand() *cobra.Command {
	var nightly, check bool

	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade ectoclaw to the latest release",
		RunE: func(_ *cobra.Command, _ []string) error {
			return run(nightly, check)
		},
	}

	cmd.Flags().BoolVar(&nightly, "nightly", false, "upgrade to the latest nightly build")
	cmd.Flags().BoolVar(&check, "check", false, "check for updates without downloading")

	return cmd
}

func run(nightly, check bool) error {
	currentVersion := internal.GetVersion()

	release, err := fetchRelease(nightly)
	if err != nil {
		return fmt.Errorf("fetch release info: %w", err)
	}

	latestTag := release.TagName

	if check {
		fmt.Printf("Current version: %s\n", currentVersion)
		fmt.Printf("Latest release:  %s\n", latestTag)
		return nil
	}

	// Skip version check for dev builds or nightly mode
	if !nightly && currentVersion != "dev" {
		if currentVersion == latestTag {
			fmt.Printf("Already up to date (%s).\n", currentVersion)
			return nil
		}
	}

	fmt.Printf("Current version: %s\n", currentVersion)
	fmt.Printf("Latest release:  %s\n", latestTag)

	name, err := assetName()
	if err != nil {
		return err
	}

	asset, err := findAsset(release, name)
	if err != nil {
		return err
	}

	binaryPath, err := currentBinaryPath()
	if err != nil {
		return err
	}

	if err := downloadAndReplace(asset.BrowserDownloadURL, binaryPath); err != nil {
		return err
	}

	fmt.Printf("Installed %s → %s\n", latestTag, binaryPath)
	return nil
}
