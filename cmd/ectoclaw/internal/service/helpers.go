package service

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"
)

// serviceName is the system service identifier.
const serviceName = "ectoclaw"

// serviceUser is the dedicated system user the service runs as.
const serviceUser = "ectoclaw"

// ─── Linux / systemd ─────────────────────────────────────────────────────────

const systemdUnit = `[Unit]
Description=EctoClaw personal AI assistant
After=network.target

[Service]
Type=simple
User={{ .User }}
Group={{ .User }}
ExecStart={{ .ExecPath }} gateway
Restart=on-failure
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
`

func systemdUnitPath() string {
	return fmt.Sprintf("/etc/systemd/system/%s.service", serviceName)
}

func installSystemd() error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable path: %w", err)
	}
	execPath, err = filepath.Abs(execPath)
	if err != nil {
		return fmt.Errorf("resolve absolute path: %w", err)
	}

	// Create dedicated system user if it doesn't exist.
	if err := ensureSystemUser(); err != nil {
		return err
	}

	tmpl, err := template.New("unit").Parse(systemdUnit)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(systemdUnitPath(), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("write unit file (try sudo): %w", err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, map[string]string{
		"ExecPath": execPath,
		"User":     serviceUser,
	}); err != nil {
		return err
	}

	if err := systemctl("daemon-reload"); err != nil {
		return err
	}
	if err := systemctl("enable", serviceName); err != nil {
		return err
	}

	fmt.Printf("Service installed (running as user %q).\n", serviceUser)
	fmt.Printf("Run `ectoclaw service start` to start it.\n")
	return nil
}

// ensureSystemUser creates the ectoclaw system user and home directory if they don't exist.
func ensureSystemUser() error {
	// Check if user already exists.
	out, _ := exec.Command("id", serviceUser).CombinedOutput()
	if len(out) > 0 && !strings.Contains(string(out), "no such user") {
		return nil
	}

	homeDir := "/home/" + serviceUser
	args := []string{
		"-r",
		"-s", "/sbin/nologin",
		"-m",
		"-d", homeDir,
		serviceUser,
	}
	if out, err := exec.Command("useradd", args...).CombinedOutput(); err != nil {
		return fmt.Errorf("create user %q: %w\n%s", serviceUser, err, out)
	}

	// Create .ectoclaw config dir owned by the service user.
	cfgDir := filepath.Join(homeDir, ".ectoclaw")
	if err := os.MkdirAll(cfgDir, 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	if out, err := exec.Command("chown", "-R", serviceUser+":"+serviceUser, cfgDir).CombinedOutput(); err != nil {
		return fmt.Errorf("chown config dir: %w\n%s", err, out)
	}

	fmt.Printf("Created system user %q with home %s\n", serviceUser, homeDir)
	return nil
}

func uninstallSystemd() error {
	_ = systemctl("stop", serviceName)
	_ = systemctl("disable", serviceName)

	if err := os.Remove(systemdUnitPath()); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove unit file: %w", err)
	}

	_ = systemctl("daemon-reload")
	fmt.Printf("Service uninstalled.\n")
	return nil
}

func startSystemd() error  { return systemctl("start", serviceName) }
func stopSystemd() error   { return systemctl("stop", serviceName) }
func statusSystemd() error { return systemctlInteractive("status", serviceName) }

func systemctl(args ...string) error {
	out, err := exec.Command("systemctl", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("systemctl %s: %w\n%s", strings.Join(args, " "), err, out)
	}
	return nil
}

func systemctlInteractive(args ...string) error {
	cmd := exec.Command("systemctl", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// ─── macOS / launchd ─────────────────────────────────────────────────────────

const launchdPlist = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>{{ .Label }}</string>
  <key>ProgramArguments</key>
  <array>
    <string>{{ .ExecPath }}</string>
    <string>gateway</string>
  </array>
  <key>RunAtLoad</key>
  <true/>
  <key>KeepAlive</key>
  <true/>
  <key>StandardOutPath</key>
  <string>{{ .LogDir }}/ectoclaw.log</string>
  <key>StandardErrorPath</key>
  <string>{{ .LogDir }}/ectoclaw.err</string>
</dict>
</plist>
`

func launchdLabel() string { return "io.ectoclaw." + serviceName }

func launchdPlistPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "LaunchAgents", launchdLabel()+".plist"), nil
}

func installLaunchd() error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable path: %w", err)
	}
	execPath, err = filepath.Abs(execPath)
	if err != nil {
		return fmt.Errorf("resolve absolute path: %w", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	logDir := filepath.Join(home, "Library", "Logs")

	plistPath, err := launchdPlistPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(plistPath), 0o755); err != nil {
		return err
	}

	tmpl, err := template.New("plist").Parse(launchdPlist)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(plistPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("write plist: %w", err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, map[string]string{
		"Label":    launchdLabel(),
		"ExecPath": execPath,
		"LogDir":   logDir,
	}); err != nil {
		return err
	}

	fmt.Printf("Service installed at %s\nRun `ectoclaw service start` to start it.\n", plistPath)
	return nil
}

func uninstallLaunchd() error {
	plistPath, err := launchdPlistPath()
	if err != nil {
		return err
	}

	_ = launchctl("unload", plistPath)

	if err := os.Remove(plistPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove plist: %w", err)
	}

	fmt.Printf("Service uninstalled.\n")
	return nil
}

func startLaunchd() error {
	plistPath, err := launchdPlistPath()
	if err != nil {
		return err
	}
	return launchctl("load", plistPath)
}

func stopLaunchd() error {
	plistPath, err := launchdPlistPath()
	if err != nil {
		return err
	}
	return launchctl("unload", plistPath)
}

func statusLaunchd() error {
	out, err := exec.Command("launchctl", "list", launchdLabel()).Output()
	if err != nil {
		fmt.Printf("Service is not running (not loaded).\n")
		return nil
	}
	fmt.Printf("%s\n", out)
	return nil
}

func launchctl(args ...string) error {
	out, err := exec.Command("launchctl", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("launchctl %s: %w\n%s", strings.Join(args, " "), err, out)
	}
	return nil
}

// ─── Dispatch ─────────────────────────────────────────────────────────────────

func install() error {
	switch runtime.GOOS {
	case "linux":
		return installSystemd()
	case "darwin":
		return installLaunchd()
	default:
		return fmt.Errorf("service management is not supported on %s", runtime.GOOS)
	}
}

func uninstall() error {
	switch runtime.GOOS {
	case "linux":
		return uninstallSystemd()
	case "darwin":
		return uninstallLaunchd()
	default:
		return fmt.Errorf("service management is not supported on %s", runtime.GOOS)
	}
}

func start() error {
	switch runtime.GOOS {
	case "linux":
		return startSystemd()
	case "darwin":
		return startLaunchd()
	default:
		return fmt.Errorf("service management is not supported on %s", runtime.GOOS)
	}
}

func stop() error {
	switch runtime.GOOS {
	case "linux":
		return stopSystemd()
	case "darwin":
		return stopLaunchd()
	default:
		return fmt.Errorf("service management is not supported on %s", runtime.GOOS)
	}
}

func status() error {
	switch runtime.GOOS {
	case "linux":
		return statusSystemd()
	case "darwin":
		return statusLaunchd()
	default:
		return fmt.Errorf("service management is not supported on %s", runtime.GOOS)
	}
}
