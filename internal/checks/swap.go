package checks

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/MuhammadZa1/tuneup/internal/report"
)

// lowRAMThresholdKB flags machines with 8GB or less as worth checking
// swap on. This isn't a hard rule (plenty of 8GB machines are fine
// without swap) — it's just the cutoff below which we bother to look.
const lowRAMThresholdKB = 8 * 1024 * 1024

// SwapZramCheck reports whether any swap (disk swap or zram) is
// configured on machines with a modest amount of RAM.
type SwapZramCheck struct{}

func (c *SwapZramCheck) Name() string { return "swap-zram" }

func (c *SwapZramCheck) Run() report.Result {
	title := "Swap / zram configuration"

	totalKB, ok := totalMemoryKB()
	if !ok {
		return report.Result{
			Check: c.Name(), Title: title, Status: report.StatusInfo,
			Message: "Couldn't read total memory from /proc/meminfo.",
		}
	}

	hasSwap, swapDetail := swapStatus()
	totalGB := float64(totalKB) / 1024 / 1024

	if hasSwap {
		return report.Result{
			Check: c.Name(), Title: title, Status: report.StatusOK,
			Message: fmt.Sprintf("Swap/zram is configured (%.1f GB RAM detected).", totalGB),
			Detail:  swapDetail,
		}
	}

	if totalKB > lowRAMThresholdKB {
		return report.Result{
			Check: c.Name(), Title: title, Status: report.StatusInfo,
			Message: fmt.Sprintf("No swap is configured, but you have %.1f GB of RAM — likely not an issue for typical use.", totalGB),
		}
	}

	return report.Result{
		Check: c.Name(), Title: title, Status: report.StatusWarning,
		Message: fmt.Sprintf("No swap or zram is configured, and this machine only has %.1f GB of RAM. Memory-heavy tasks (browser tabs, builds, games) can trigger the OOM killer or freeze the system instead of degrading gracefully.", totalGB),
		Detail:  "No automatic fix is offered here since swap file size and zram vs. disk swap are genuine preferences. See the README for a copy-pasteable zram-generator setup for your distro.",
	}
}

// totalMemoryKB reads MemTotal from /proc/meminfo.
func totalMemoryKB() (int, bool) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, false
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				if kb, err := strconv.Atoi(fields[1]); err == nil {
					return kb, true
				}
			}
		}
	}
	return 0, false
}

// swapStatus reports whether any swap (disk or zram) is active, and a
// one-line detail describing it.
func swapStatus() (bool, string) {
	data := readFile("/proc/swaps")
	lines := strings.Split(data, "\n")
	// First line is always the header ("Filename Type Size Used Priority").
	if len(lines) <= 1 {
		return false, ""
	}

	var active []string
	for _, line := range lines[1:] {
		line = strings.TrimSpace(line)
		if line != "" {
			active = append(active, line)
		}
	}
	if len(active) == 0 {
		return false, ""
	}
	return true, strings.Join(active, "\n")
}
