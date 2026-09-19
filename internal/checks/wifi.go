package checks

import (
	"strings"

	"github.com/MuhammadZa1/tuneup/internal/report"
)

// WifiPowerSaveCheck looks for wireless interfaces with power
// management enabled, which on some chipsets/drivers (notably several
// Broadcom and older Intel parts) causes intermittent dropped
// connections, laggy pings, or slow reconnects — especially at close
// range to the router, which makes it a confusing symptom to diagnose.
type WifiPowerSaveCheck struct{}

func (c *WifiPowerSaveCheck) Name() string { return "wifi-power-save" }

func (c *WifiPowerSaveCheck) Run() report.Result {
	title := "Wi-Fi power saving"

	if !commandExists("iw") {
		return report.Result{
			Check: c.Name(), Title: title, Status: report.StatusSkipped,
			Message: "The `iw` tool isn't installed, so Wi-Fi power-save state can't be checked. Install the `iw` package to enable this check.",
		}
	}

	interfaces := wirelessInterfaces()
	if len(interfaces) == 0 {
		return report.Result{
			Check: c.Name(), Title: title, Status: report.StatusSkipped,
			Message: "No wireless interfaces found.",
		}
	}

	var onInterfaces []string
	for _, iface := range interfaces {
		out, ok := runCommandStatus("iw", "dev", iface, "get", "power_save")
		if !ok {
			continue
		}
		if strings.Contains(out, "Power save: on") {
			onInterfaces = append(onInterfaces, iface)
		}
	}

	if len(onInterfaces) > 0 {
		return report.Result{
			Check: c.Name(), Title: title, Status: report.StatusWarning,
			Message: "Power saving is enabled on one or more Wi-Fi interfaces (" + strings.Join(onInterfaces, ", ") + "). On some chipsets this causes intermittent drops, laggy pings, or slow reconnects.",
			Detail:  "No automatic fix is offered since disabling power save trades some battery life for reliability, and the right choice is a real preference. To test the tradeoff yourself: `sudo iw dev " + onInterfaces[0] + " set power_save off`, then check if connection stability improves. This resets on reboot; see the README for how to make it persistent if it helps.",
		}
	}

	return report.Result{
		Check: c.Name(), Title: title, Status: report.StatusOK,
		Message: "Wi-Fi power saving is off on all detected interfaces.",
	}
}

// wirelessInterfaces returns interface names that `iw dev` reports.
func wirelessInterfaces() []string {
	out, ok := runCommandStatus("iw", "dev")
	if !ok {
		return nil
	}
	var names []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Interface ") {
			names = append(names, strings.TrimPrefix(line, "Interface "))
		}
	}
	return names
}
