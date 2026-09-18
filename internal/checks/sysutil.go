package checks

import (
	"os"
	"os/exec"
	"strings"
)

// commandExists reports whether a binary is available on PATH.
func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// runCommand runs name with args and returns combined stdout+stderr,
// trimmed. It never returns an error for a non-zero exit code — callers
// that care about exit status should use runCommandStatus instead.
func runCommand(name string, args ...string) string {
	out, _ := exec.Command(name, args...).CombinedOutput()
	return strings.TrimSpace(string(out))
}

// runCommandStatus runs name with args and reports both its trimmed
// output and whether it exited successfully.
func runCommandStatus(name string, args ...string) (string, bool) {
	out, err := exec.Command(name, args...).CombinedOutput()
	return strings.TrimSpace(string(out)), err == nil
}

// fileExists reports whether path exists (file or directory).
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// readFile reads path and returns its trimmed content, or "" if it
// can't be read. Diagnostics should never fail hard just because one
// optional file is missing.
func readFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}
