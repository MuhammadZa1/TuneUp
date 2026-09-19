// Command tuneup diagnoses common problems on aging or low-spec Linux
// hardware and offers to fix the ones it can fix safely.
//
// It never changes anything on its own: every check is read-only, and
// a fix is only ever applied after the user explicitly confirms it
// (or passes --yes).
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/MuhammadZa1/tuneup/internal/checks"
	"github.com/MuhammadZa1/tuneup/internal/report"
)

// version is set at build time for tagged releases via
// -ldflags "-X main.version=...". It defaults to "dev" for local/untagged
// builds (e.g. `go build` or `go install ...@latest` without a version).
var version = "dev"

// ANSI colors, disabled automatically when stdout isn't a terminal.
const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[90m"
)

// jsonOutput is the top-level shape written to stdout in --json mode.
type jsonOutput struct {
	Results       []report.Result `json:"results"`
	ProblemsFound int             `json:"problems_found"`
	FixesApplied  int             `json:"fixes_applied"`
}

func main() {
	autoYes := flag.Bool("yes", false, "apply every offered fix without prompting")
	noColor := flag.Bool("no-color", false, "disable colored output")
	jsonOut := flag.Bool("json", false, "print results as JSON instead of interactive text (no prompts; combine with --yes to also apply fixes)")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println("tuneup " + version)
		return
	}

	if *jsonOut {
		runJSON(*autoYes)
		return
	}

	runInteractive(*autoYes, *noColor)
}

// runInteractive is the original human-facing flow: colored text,
// printed as each check runs, with an interactive y/N prompt before
// applying any fix (unless --yes was passed).
func runInteractive(autoYes, noColor bool) {
	useColor := !noColor && isTerminal()

	fmt.Println(paint(useColor, colorCyan, "tuneup") + " — diagnostics for aging Linux hardware")
	fmt.Println(paint(useColor, colorGray, "Read-only checks first. Nothing is changed without your confirmation."))
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)
	problemsFound := 0
	fixesApplied := 0

	for _, chk := range checks.All() {
		result := chk.Run()
		printResult(result, useColor)

		if result.Status == report.StatusWarning || result.Status == report.StatusProblem {
			problemsFound++
		}

		if result.Fix == nil {
			continue
		}

		apply := autoYes
		if !apply {
			apply = promptYesNo(reader, "  Apply this fix?")
		}
		if apply {
			if err := result.Fix.Apply(); err != nil {
				fmt.Println("  " + paint(useColor, colorRed, "✗ Fix failed: "+err.Error()))
			} else {
				fmt.Println("  " + paint(useColor, colorGreen, "✓ Fix applied."))
				fixesApplied++
			}
		}
		fmt.Println()
	}

	fmt.Println(strings.Repeat("-", 40))
	fmt.Printf("%d check(s) flagged something, %d fix(es) applied.\n", problemsFound, fixesApplied)
}

// runJSON runs every check with no interactive prompts and writes a
// single JSON object to stdout, suitable for scripting/CI. Fixes are
// only applied when autoYes is true — otherwise every fix is reported
// as available but left untouched, since a non-interactive context has
// no safe way to ask for confirmation.
func runJSON(autoYes bool) {
	out := jsonOutput{Results: []report.Result{}}

	for _, chk := range checks.All() {
		result := chk.Run()

		if result.Status == report.StatusWarning || result.Status == report.StatusProblem {
			out.ProblemsFound++
		}

		if result.Fix != nil && autoYes {
			if err := result.Fix.Apply(); err != nil {
				result.FixError = err.Error()
			} else {
				result.FixApplied = true
				out.FixesApplied++
			}
		}

		out.Results = append(out.Results, result)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		fmt.Fprintln(os.Stderr, "tuneup: failed to encode JSON output:", err)
		os.Exit(1)
	}
}

func printResult(r report.Result, useColor bool) {
	color := colorGray
	switch r.Status {
	case report.StatusOK:
		color = colorGreen
	case report.StatusWarning:
		color = colorYellow
	case report.StatusProblem:
		color = colorRed
	case report.StatusInfo:
		color = colorCyan
	}

	fmt.Printf("%s %s\n", paint(useColor, color, r.Status.Symbol()), boldish(useColor, r.Title))
	fmt.Printf("  %s\n", r.Message)
	if r.Detail != "" {
		for _, line := range strings.Split(r.Detail, "\n") {
			fmt.Println("  " + paint(useColor, colorGray, line))
		}
	}
	if r.Fix != nil {
		fmt.Println("  " + paint(useColor, colorCyan, "Fix available: ") + r.Fix.Description)
	}
	fmt.Println()
}

func promptYesNo(reader *bufio.Reader, prompt string) bool {
	fmt.Print(prompt + " [y/N] ")
	line, _ := reader.ReadString('\n')
	line = strings.ToLower(strings.TrimSpace(line))
	return line == "y" || line == "yes"
}

func paint(useColor bool, color, text string) string {
	if !useColor {
		return text
	}
	return color + text + colorReset
}

func boldish(useColor bool, text string) string {
	if !useColor {
		return text
	}
	return "\033[1m" + text + colorReset
}

func isTerminal() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}
