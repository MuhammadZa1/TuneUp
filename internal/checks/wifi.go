package checks

import (
	"fmt"
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
			Detail:  "Disabling power save trades a small amount of battery life for reliability. If this fix doesn't improve things for you, it's easy to re-enable from your network settings.",
			Fix: &report.Fix{
				Description: "Turn off Wi-Fi power saving on " + strings.Join(onInterfaces, ", ") + " now, and keep it off after reconnects/reboots",
				Apply:       func() error { return applyWifiPowerSaveFix(onInterfaces) },
			},
		}
	}

	return report.Result{
		Check: c.Name(), Title: title, Status: report.StatusOK,
		Message: "Wi-Fi power saving is off on all detected interfaces.",
	}
}

// applyWifiPowerSaveFix turns off power save immediately via iw, and
// additionally persists the setting through NetworkManager when it's
// managing the connection — a plain `iw` change alone resets on the
// next reconnect or reboot.
func applyWifiPowerSaveFix(interfaces []string) error {
	for _, iface := range interfaces {
		if out, ok := runCommandStatus("pkexec", "iw", "dev", iface, "set", "power_save", "off"); !ok {
			return fmt.Errorf("disabling power save on %s: %s", iface, out)
		}
	}

	if commandExists("nmcli") {
		for _, conn := range activeConnectionsFor(interfaces) {
			// wifi.powersave: 1 = ignore/default, 2 = disable, 3 = enable.
			runCommand("pkexec", "nmcli", "connection", "modify", conn, "wifi.powersave", "2")
		}
		// Re-apply so the persisted setting takes effect on the
		// current connection immediately, not just on the next one.
		for _, iface := range interfaces {
			runCommand("pkexec", "nmcli", "device", "reapply", iface)
		}
	}

	return nil
}

// activeConnectionsFor returns the NetworkManager connection names
// currently active on the given interfaces.
func activeConnectionsFor(interfaces []string) []string {
	out, ok := runCommandStatus("nmcli", "-t", "-f", "DEVICE,NAME", "connection", "show", "--active")
	if !ok {
		return nil
	}
	want := make(map[string]bool)
	for _, iface := range interfaces {
		want[iface] = true
	}

	var names []string
	for _, line := range strings.Split(out, "\n") {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 && want[parts[0]] {
			names = append(names, parts[1])
		}
	}
	return names
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
