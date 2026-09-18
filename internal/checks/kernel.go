package checks

import (
	"fmt"
	"strings"

	"github.com/MuhammadZa1/tuneup/internal/report"
)

// KernelDKMSCheck looks for dkms modules that failed to build against
// the currently running kernel, or a newer installed kernel whose dkms
// modules never built at all. Either case can mean a critical driver
// (Wi-Fi, GPU) silently vanishes on the next boot.
type KernelDKMSCheck struct{}

func (c *KernelDKMSCheck) Name() string { return "kernel-dkms" }

func (c *KernelDKMSCheck) Run() report.Result {
	title := "Kernel / dkms module status"

	if !commandExists("dkms") {
		return report.Result{
			Check: c.Name(), Title: title, Status: report.StatusSkipped,
			Message: "dkms isn't installed, so there's nothing this check can look at.",
		}
	}

	out, ok := runCommandStatus("dkms", "status")
	if !ok {
		return report.Result{
			Check: c.Name(), Title: title, Status: report.StatusInfo,
			Message: "Couldn't read dkms status.",
		}
	}

	if strings.TrimSpace(out) == "" {
		return report.Result{
			Check: c.Name(), Title: title, Status: report.StatusOK,
			Message: "dkms is installed but has no modules registered.",
		}
	}

	running := runCommand("uname", "-r")
	var broken []string
	var staleForRunning = true

	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lower := strings.ToLower(line)
		if strings.Contains(lower, "installed") && strings.Contains(line, running) {
			staleForRunning = false
		}
		if strings.Contains(lower, "not installed") || strings.Contains(lower, "error") || strings.Contains(lower, "fail") {
			broken = append(broken, line)
		}
	}

	if len(broken) > 0 {
		return report.Result{
			Check: c.Name(), Title: title, Status: report.StatusProblem,
			Message: "One or more dkms modules failed to build. A driver (often Wi-Fi or GPU) may be missing after your next reboot.",
			Detail:  strings.Join(broken, "\n") + "\n\nNo automatic fix is offered — a bad rebuild attempt can make things worse. Run `sudo dkms build <module>/<version> -k " + running + "` to see the real build error, or hold back the newer kernel until it's resolved.",
		}
	}

	if staleForRunning && running != "" {
		return report.Result{
			Check: c.Name(), Title: title, Status: report.StatusInfo,
			Message: fmt.Sprintf("No dkms module shows as installed for the currently running kernel (%s). This may be fine if none of your dkms modules target this kernel.", running),
		}
	}

	return report.Result{
		Check: c.Name(), Title: title, Status: report.StatusOK,
		Message: "All registered dkms modules are built for the running kernel.",
	}
}
