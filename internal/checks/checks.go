// Package checks contains one file per diagnostic area (audio, gpu,
// kernel, swap, ...). Each check is a small, self-contained struct that
// implements the Check interface, so adding a new one never requires
// touching the others.
package checks

import "github.com/MuhammadZa1/tuneup/internal/report"

// Check is implemented by every diagnostic in this package.
type Check interface {
	// Name is the short machine-readable id, e.g. "audio-crackle".
	Name() string
	// Run performs the diagnostic and returns a Result. It must never
	// modify the system — only Fix.Apply (opt-in, user-confirmed) may.
	Run() report.Result
}

// All returns every registered check, in the order they should run.
func All() []Check {
	return []Check{
		&AudioCrackleCheck{},
		&GPUVulkanCheck{},
		&KernelDKMSCheck{},
		&SwapZramCheck{},
		&BatteryWearCheck{},
		&DiskSchedulerCheck{},
		&ThermalThrottleCheck{},
		&WifiPowerSaveCheck{},
	}
}
