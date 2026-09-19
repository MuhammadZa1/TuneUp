package checks

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/MuhammadZa1/tuneup/internal/report"
)

const thermalZoneDir = "/sys/class/thermal"
const cpuSysDir = "/sys/devices/system/cpu"

// highTempWarningC flags CPU temps above this as worth mentioning even
// without a confirmed throttle event. Most laptop CPUs start throttling
// somewhere in the 90-100°C range, but sustained time above this
// already means the cooling solution is struggling.
const highTempWarningC = 85

// ThermalThrottleCheck looks for evidence of thermal throttling: the
// kernel's own per-core throttle-event counters (Intel's
// thermal_throttle sysfs interface) when available, and current
// thermal zone temperatures as a secondary signal.
type ThermalThrottleCheck struct{}

func (c *ThermalThrottleCheck) Name() string { return "thermal-throttle" }

func (c *ThermalThrottleCheck) Run() report.Result {
	title := "Thermal throttling"

	throttleEvents, sawThrottleCounters := totalThrottleEvents()
	maxTempC, zoneType, sawTemp := hottestThermalZone()

	if !sawThrottleCounters && !sawTemp {
		return report.Result{
			Check: c.Name(), Title: title, Status: report.StatusSkipped,
			Message: "No thermal zone or CPU throttle-event data available on this system.",
		}
	}

	var details []string
	if sawTemp {
		details = append(details, "Hottest zone ("+zoneType+"): "+strconv.Itoa(maxTempC)+"°C")
	}
	if sawThrottleCounters {
		details = append(details, "Throttle events recorded since boot: "+strconv.Itoa(throttleEvents))
	}
	detail := strings.Join(details, "\n")

	if sawThrottleCounters && throttleEvents > 0 {
		return report.Result{
			Check: c.Name(), Title: title, Status: report.StatusWarning,
			Message: "The CPU has throttled itself due to heat at least once since boot.",
			Detail:  detail + "\n\nNo automatic fix is offered: throttling is the CPU protecting itself. The real fix is improving cooling (clean fans/vents, reapply thermal paste, check for blocked airflow) or reducing sustained load.",
		}
	}

	if sawTemp && maxTempC >= highTempWarningC {
		return report.Result{
			Check: c.Name(), Title: title, Status: report.StatusWarning,
			Message: "CPU temperature is high enough that throttling may occur under sustained load, even though no throttle event has been recorded yet.",
			Detail:  detail,
		}
	}

	return report.Result{
		Check: c.Name(), Title: title, Status: report.StatusOK,
		Message: "No thermal throttling detected.",
		Detail:  detail,
	}
}

// totalThrottleEvents sums Intel's core_throttle_count across all CPU
// cores, when the kernel exposes it. This counter only increments on
// an actual thermal throttle event, so unlike a frequency snapshot it
// can't be confused with a CPU that's simply idle.
func totalThrottleEvents() (total int, found bool) {
	entries, err := os.ReadDir(cpuSysDir)
	if err != nil {
		return 0, false
	}

	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), "cpu") {
			continue
		}
		if _, err := strconv.Atoi(strings.TrimPrefix(e.Name(), "cpu")); err != nil {
			continue
		}
		countPath := filepath.Join(cpuSysDir, e.Name(), "thermal_throttle", "core_throttle_count")
		if !fileExists(countPath) {
			continue
		}
		found = true
		total += readIntFile(countPath)
	}
	return total, found
}

// hottestThermalZone scans /sys/class/thermal/thermal_zone* and
// returns the highest temperature found, in whole degrees C, along
// with that zone's type (e.g. "x86_pkg_temp", "acpitz").
func hottestThermalZone() (tempC int, zoneType string, found bool) {
	entries, err := os.ReadDir(thermalZoneDir)
	if err != nil {
		return 0, "", false
	}

	highest := -1
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), "thermal_zone") {
			continue
		}
		zonePath := filepath.Join(thermalZoneDir, e.Name())
		milliC := readIntFile(filepath.Join(zonePath, "temp"))
		if milliC <= 0 {
			continue
		}
		degC := milliC / 1000
		if degC > highest {
			highest = degC
			zoneType = readFile(filepath.Join(zonePath, "type"))
		}
	}

	if highest < 0 {
		return 0, "", false
	}
	return highest, zoneType, true
}
