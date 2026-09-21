package checks

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/MuhammadZa1/tuneup/internal/report"
)

const blockDevDir = "/sys/block"
const diskSchedulerUdevRule = "/etc/udev/rules.d/60-tuneup-hdd-scheduler.rules"

// preferredHDDSchedulers are I/O schedulers well-suited to spinning
// disks, in order of preference. "none"/"noop" is fine for SSDs/NVMe
// but hurts HDDs, where reordering and merging requests matters a lot.
var preferredHDDSchedulers = []string{"bfq", "mq-deadline"}

// DiskSchedulerCheck looks for spinning hard drives using an I/O
// scheduler that isn't a good fit for their access pattern.
type DiskSchedulerCheck struct{}

func (c *DiskSchedulerCheck) Name() string { return "disk-scheduler" }

func (c *DiskSchedulerCheck) Run() report.Result {
	title := "Disk I/O scheduler"

	devices, err := os.ReadDir(blockDevDir)
	if err != nil {
		return report.Result{
			Check: c.Name(), Title: title, Status: report.StatusSkipped,
			Message: "Couldn't read block device info from " + blockDevDir + ".",
		}
	}

	var affected []string
	sawAnyDisk := false

	for _, dev := range devices {
		name := dev.Name()
		// Skip partitions, loop devices, and virtual devices — only
		// look at whole physical disks.
		if strings.HasPrefix(name, "loop") || strings.HasPrefix(name, "ram") || strings.HasPrefix(name, "dm-") {
			continue
		}
		devPath := filepath.Join(blockDevDir, name)

		rotational := readFile(filepath.Join(devPath, "queue", "rotational"))
		if rotational != "1" {
			continue // SSD/NVMe (rotational == "0"), or unknown — skip
		}
		sawAnyDisk = true

		scheduler := currentScheduler(devPath)
		if scheduler == "" {
			continue
		}

		if !isPreferredScheduler(scheduler) {
			affected = append(affected, name)
		}
	}

	if !sawAnyDisk {
		return report.Result{
			Check: c.Name(), Title: title, Status: report.StatusSkipped,
			Message: "No spinning hard drives detected (SSD/NVMe-only or couldn't determine media type).",
		}
	}

	if len(affected) > 0 {
		findings := make([]string, len(affected))
		for i, name := range affected {
			findings[i] = name + " is using \"" + currentScheduler(filepath.Join(blockDevDir, name)) + "\""
		}
		return report.Result{
			Check: c.Name(), Title: title, Status: report.StatusWarning,
			Message: "One or more HDDs aren't using a scheduler well-suited to spinning disks.",
			Detail: strings.Join(findings, "\n") +
				"\n\nbfq generally performs better than \"none\" on HDDs, since it reorders and batches requests to reduce seek time.",
			Fix: &report.Fix{
				Description: "Switch affected HDDs to bfq now, and add a udev rule so it stays set after reboots",
				Apply:       func() error { return applyDiskSchedulerFix(affected) },
			},
		}
	}

	return report.Result{
		Check: c.Name(), Title: title, Status: report.StatusOK,
		Message: "All detected HDDs are using a scheduler well-suited to spinning disks.",
	}
}

// applyDiskSchedulerFix sets bfq immediately on each affected disk and
// writes a udev rule so newly-attached or re-enumerated HDDs get it
// automatically in the future, surviving reboots.
func applyDiskSchedulerFix(devices []string) error {
	for _, name := range devices {
		schedulerPath := filepath.Join(blockDevDir, name, "queue", "scheduler")
		if out, ok := runCommandStatus("pkexec", "sh", "-c", fmt.Sprintf("echo bfq > %s", schedulerPath)); !ok {
			return fmt.Errorf("setting scheduler for %s: %s", name, out)
		}
	}

	rule := `# Written by tuneup: use bfq for spinning HDDs.
ACTION=="add|change", KERNEL=="sd[a-z]", ATTR{queue/rotational}=="1", ATTR{queue/scheduler}="bfq"
`
	tmp, err := os.CreateTemp("", "tuneup-udev-*.rules")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(rule); err != nil {
		tmp.Close()
		return err
	}
	tmp.Close()

	if out, ok := runCommandStatus("pkexec", "cp", tmp.Name(), diskSchedulerUdevRule); !ok {
		return fmt.Errorf("writing udev rule: %s", out)
	}
	runCommand("pkexec", "udevadm", "control", "--reload-rules")

	return nil
}

// currentScheduler reads /sys/block/<dev>/queue/scheduler, which looks
// like "mq-deadline [bfq] none" with the active one in brackets.
func currentScheduler(devPath string) string {
	raw := readFile(filepath.Join(devPath, "queue", "scheduler"))
	for _, field := range strings.Fields(raw) {
		if strings.HasPrefix(field, "[") && strings.HasSuffix(field, "]") {
			return strings.Trim(field, "[]")
		}
	}
	return ""
}

func isPreferredScheduler(scheduler string) bool {
	for _, s := range preferredHDDSchedulers {
		if scheduler == s {
			return true
		}
	}
	return false
}
