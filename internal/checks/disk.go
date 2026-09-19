package checks

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/MuhammadZa1/tuneup/internal/report"
)

const blockDevDir = "/sys/block"

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

	var hddFindings []string
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
			hddFindings = append(hddFindings, name+" is using \""+scheduler+"\"")
		}
	}

	if !sawAnyDisk {
		return report.Result{
			Check: c.Name(), Title: title, Status: report.StatusSkipped,
			Message: "No spinning hard drives detected (SSD/NVMe-only or couldn't determine media type).",
		}
	}

	if len(hddFindings) > 0 {
		return report.Result{
			Check: c.Name(), Title: title, Status: report.StatusWarning,
			Message: "One or more HDDs aren't using a scheduler well-suited to spinning disks.",
			Detail: strings.Join(hddFindings, "\n") +
				"\n\nbfq or mq-deadline generally perform better than \"none\" on HDDs, since they reorder and batch requests to reduce seek time. No automatic fix is offered since the best choice depends on your workload — see the README for how to change it (it's a one-line write to /sys/block/<dev>/queue/scheduler, or a udev rule to make it persistent).",
		}
	}

	return report.Result{
		Check: c.Name(), Title: title, Status: report.StatusOK,
		Message: "All detected HDDs are using a scheduler well-suited to spinning disks.",
	}
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
