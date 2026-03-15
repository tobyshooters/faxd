package source

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func Install() error {
	bin, err := os.Executable()
	if err != nil {
		return err
	}

	switch runtime.GOOS {
	case "darwin":
		return installLaunchd(bin)
	case "linux":
		return installSystemd(bin)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

func Uninstall() error {
	switch runtime.GOOS {
	case "darwin":
		return uninstallLaunchd()
	case "linux":
		return uninstallSystemd()
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// --- launchd (macOS) ---

func launchdPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "LaunchAgents", "com.faxd.plist")
}

func installLaunchd(bin string) error {
	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>com.faxd</string>
  <key>ProgramArguments</key>
  <array>
    <string>%s</string>
    <string>run</string>
  </array>
  <key>RunAtLoad</key>
  <true/>
  <key>KeepAlive</key>
  <true/>
  <key>StandardOutPath</key>
  <string>/tmp/faxd.log</string>
  <key>StandardErrorPath</key>
  <string>/tmp/faxd.log</string>
</dict>
</plist>`, bin)

	if err := os.WriteFile(launchdPath(), []byte(plist), 0644); err != nil {
		return err
	}

	return exec.Command("launchctl", "load", launchdPath()).Run()
}

func uninstallLaunchd() error {
	exec.Command("launchctl", "unload", launchdPath()).Run()
	return os.Remove(launchdPath())
}

// --- systemd (Linux) ---

func systemdDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "systemd", "user")
}

func systemdPath() string {
	return filepath.Join(systemdDir(), "faxd.service")
}

func installSystemd(bin string) error {
	unit := fmt.Sprintf(`[Unit]
Description=faxd - local fax daemon

[Service]
ExecStart=%s run
Restart=on-failure

[Install]
WantedBy=default.target
`, bin)

	os.MkdirAll(systemdDir(), 0755)
	if err := os.WriteFile(systemdPath(), []byte(unit), 0644); err != nil {
		return err
	}

	exec.Command("systemctl", "--user", "daemon-reload").Run()
	return exec.Command("systemctl", "--user", "enable", "--now", "faxd").Run()
}

func uninstallSystemd() error {
	exec.Command("systemctl", "--user", "disable", "--now", "faxd").Run()
	os.Remove(systemdPath())
	return exec.Command("systemctl", "--user", "daemon-reload").Run()
}
