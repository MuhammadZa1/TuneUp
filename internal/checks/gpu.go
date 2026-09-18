package checks

import (
	"fmt"
	"strings"

	"github.com/MuhammadZa1/tuneup/internal/report"
)

// GPUVulkanCheck detects whether Vulkan has real hardware backing or is
// silently falling back to a CPU software renderer (lavapipe). This is
// a common trap on older Intel GPUs: `vulkaninfo` succeeds either way,
// so games "launch" and then crash or run at unplayable speed with no
// obvious reason why.
type GPUVulkanCheck struct{}

func (c *GPUVulkanCheck) Name() string { return "gpu-vulkan" }

const title = "GPU / Vulkan support"

func (c *GPUVulkanCheck) Run() report.Result {
	if !commandExists("vulkaninfo") {
		return report.Result{
			Check: c.Name(), Title: title, Status: report.StatusSkipped,
			Message: "vulkaninfo isn't installed, so Vulkan backing can't be checked. Install the `vulkan-tools` package to enable this check.",
		}
	}

	out, ok := runCommandStatus("vulkaninfo", "--summary")
	if !ok || out == "" {
		return report.Result{
			Check: c.Name(), Title: title, Status: report.StatusProblem,
			Message: "vulkaninfo failed to run. No Vulkan driver appears to be working at all.",
			Detail:  "Games and apps requiring Vulkan (including most Proton/DXVK titles) will not run.",
		}
	}

	lower := strings.ToLower(out)
	softwareMarkers := []string{"llvmpipe", "lavapipe", "softpipe", "swrast"}
	isSoftware := false
	for _, m := range softwareMarkers {
		if strings.Contains(lower, m) {
			isSoftware = true
			break
		}
	}

	if isSoftware {
		return report.Result{
			Check: c.Name(), Title: title, Status: report.StatusWarning,
			Message: "Vulkan is only available via a CPU software renderer (lavapipe/llvmpipe), not real GPU hardware.",
			Detail:  "vulkaninfo succeeds so tools and games will *launch*, but will run extremely slowly or crash under load. This usually means no hardware Vulkan ICD is installed/detected for your GPU, or the GPU genuinely doesn't support Vulkan (common on older Intel HD Graphics). There's no safe automatic fix — see the README's hardware notes for your GPU generation.",
		}
	}

	device := extractDeviceName(out)
	msg := "Vulkan is backed by real GPU hardware."
	if device != "" {
		msg = fmt.Sprintf("Vulkan is hardware-backed on: %s", device)
	}
	return report.Result{
		Check: c.Name(), Title: title, Status: report.StatusOK,
		Message: msg,
	}
}

// extractDeviceName pulls a deviceName line out of `vulkaninfo --summary`
// output, if present.
func extractDeviceName(out string) string {
	for _, line := range strings.Split(out, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "deviceName") {
			parts := strings.SplitN(trimmed, "=", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return ""
}
