package checks

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/MuhammadZa1/tuneup/internal/report"
)

// pipewireOverridePath is where we write the quantum override. PipeWire
// reads every *.conf file under pipewire.conf.d, so a dedicated file is
// safer than editing the shipped config.
const pipewireOverridePath = "/etc/pipewire/pipewire.conf.d/99-tuneup-quantum.conf"

const pipewireOverrideContent = `# Written by tuneup to reduce audio crackling on some hardware.
# Raises the minimum scheduling quantum so the audio graph has more
# breathing room on CPUs that can't keep up with PipeWire's default.
context.properties = {
    default.clock.quantum = 1024
    default.clock.min-quantum = 1024
}
`

// AudioCrackleCheck looks for a common cause of audio crackling on
// older/slower CPUs: PipeWire's default scheduling quantum being too
// small for the hardware to service in time.
type AudioCrackleCheck struct{}

func (c *AudioCrackleCheck) Name() string { return "audio-crackle" }

func (c *AudioCrackleCheck) Run() report.Result {
	title := "Audio crackling (PipeWire)"

	if !commandExists("pipewire") {
		return report.Result{
			Check: c.Name(), Title: title, Status: report.StatusSkipped,
			Message: "PipeWire isn't installed on this system, skipping.",
		}
	}

	if fileExists(pipewireOverridePath) {
		return report.Result{
			Check: c.Name(), Title: title, Status: report.StatusOK,
			Message: "A quantum override is already in place.",
			Detail:  pipewireOverridePath,
		}
	}

	// Check whether any existing config already sets a quantum, in
	// which case we shouldn't suggest adding a second, conflicting one.
	if existing := findExistingQuantumConfig(); existing != "" {
		return report.Result{
			Check: c.Name(), Title: title, Status: report.StatusInfo,
			Message: "A custom clock.quantum setting already exists elsewhere.",
			Detail:  fmt.Sprintf("Found in %s — leaving it alone.", existing),
		}
	}

	return report.Result{
		Check: c.Name(), Title: title, Status: report.StatusWarning,
		Message: "No custom scheduling quantum is set. On slower CPUs this is a common cause of audio crackling/dropouts.",
		Detail:  "Raising default.clock.quantum / min-quantum to 1024 gives PipeWire more time per audio buffer.",
		Fix: &report.Fix{
			Description: fmt.Sprintf("Write a quantum override to %s (requires sudo) and restart PipeWire", pipewireOverridePath),
			Apply:       applyQuantumFix,
		},
	}
}

// findExistingQuantumConfig scans the usual PipeWire config locations
// for a pre-existing clock.quantum setting so we don't suggest a
// conflicting override.
func findExistingQuantumConfig() string {
	dirs := []string{
		"/etc/pipewire/pipewire.conf.d",
		filepath.Join(os.Getenv("HOME"), ".config/pipewire/pipewire.conf.d"),
	}
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".conf") {
				continue
			}
			path := filepath.Join(dir, e.Name())
			if strings.Contains(readFile(path), "clock.quantum") {
				return path
			}
		}
	}
	return ""
}

func applyQuantumFix() error {
	dir := filepath.Dir(pipewireOverridePath)
	if !fileExists(dir) {
		if out, ok := runCommandStatus("sudo", "mkdir", "-p", dir); !ok {
			return fmt.Errorf("creating %s: %s", dir, out)
		}
	}

	tmp, err := os.CreateTemp("", "tuneup-pw-*.conf")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(pipewireOverrideContent); err != nil {
		tmp.Close()
		return err
	}
	tmp.Close()

	if out, ok := runCommandStatus("sudo", "cp", tmp.Name(), pipewireOverridePath); !ok {
		return fmt.Errorf("writing %s: %s", pipewireOverridePath, out)
	}

	// Restart the user-level PipeWire services so the change takes
	// effect immediately. These are harmless if PipeWire isn't
	// running as a user service (e.g. system-wide setups).
	runCommand("systemctl", "--user", "restart", "pipewire", "pipewire-pulse", "wireplumber")

	return nil
}
