package checks

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/MuhammadZa1/tuneup/internal/report"
)

// batteryWarnThresholdPct flags batteries that have degraded below this
// percentage of their original design capacity.
const batteryWarnThresholdPct = 70.0

// powerSupplyDir is where the kernel exposes battery info.
const powerSupplyDir = "/sys/class/power_supply"

// BatteryWearCheck reports how much capacity a laptop battery has lost
// compared to its original design capacity, using whichever sysfs
// attributes the kernel/driver actually expose (they vary: some report
// energy in µWh, others report charge in µAh).
type BatteryWearCheck struct{}

func (c *BatteryWearCheck) Name() string { return "battery-wear" }

func (c *BatteryWearCheck) Run() report.Result {
	title := "Battery wear"

	bat := findBattery()
	if bat == "" {
		return report.Result{
			Check: c.Name(), Title: title, Status: report.StatusSkipped,
			Message: "No battery detected — likely a desktop or VM.",
		}
	}

	full, design, ok := batteryCapacities(bat)
	if !ok || design == 0 {
		return report.Result{
			Check: c.Name(), Title: title, Status: report.StatusInfo,
			Message: "A battery was found, but this driver doesn't expose design capacity, so wear can't be calculated.",
			Detail:  bat,
		}
	}

	wearPct := (full / design) * 100
	cycles := readIntFile(filepath.Join(bat, "cycle_count"))

	detail := ""
	if cycles > 0 {
		detail = "Cycle count: " + strconv.Itoa(cycles)
	}

	if wearPct < batteryWarnThresholdPct {
		return report.Result{
			Check: c.Name(), Title: title, Status: report.StatusWarning,
			Message: fmt.Sprintf("Battery holds only %.1f%% of its original design capacity.", wearPct),
			Detail:  detail,
		}
	}

	return report.Result{
		Check: c.Name(), Title: title, Status: report.StatusOK,
		Message: fmt.Sprintf("Battery holds %.1f%% of its original design capacity.", wearPct),
		Detail:  detail,
	}
}

// findBattery returns the sysfs path of the first entry under
// powerSupplyDir whose type is "Battery", or "" if none is found.
func findBattery() string {
	entries, err := os.ReadDir(powerSupplyDir)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		path := filepath.Join(powerSupplyDir, e.Name())
		if readFile(filepath.Join(path, "type")) == "Battery" {
			return path
		}
	}
	return ""
}

// batteryCapacities reads the full and design capacity for a battery,
// preferring energy_* (µWh) and falling back to charge_* (µAh) —
// different drivers expose different attributes.
func batteryCapacities(bat string) (full, design float64, ok bool) {
	if f := readIntFile(filepath.Join(bat, "energy_full")); f > 0 {
		if d := readIntFile(filepath.Join(bat, "energy_full_design")); d > 0 {
			return float64(f), float64(d), true
		}
	}
	if f := readIntFile(filepath.Join(bat, "charge_full")); f > 0 {
		if d := readIntFile(filepath.Join(bat, "charge_full_design")); d > 0 {
			return float64(f), float64(d), true
		}
	}
	return 0, 0, false
}

func readIntFile(path string) int {
	v, err := strconv.Atoi(strings.TrimSpace(readFile(path)))
	if err != nil {
		return 0
	}
	return v
}
