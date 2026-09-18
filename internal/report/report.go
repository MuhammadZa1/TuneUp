// Package report defines the shared data types that every check produces.
// Keeping this separate from the checks themselves means the output
// format (plain text today, maybe JSON later) can change without
// touching check logic.
package report

// Status represents the outcome of a single check.
type Status int

const (
	StatusOK Status = iota
	StatusInfo
	StatusWarning
	StatusProblem
	StatusSkipped
)

func (s Status) String() string {
	switch s {
	case StatusOK:
		return "OK"
	case StatusInfo:
		return "INFO"
	case StatusWarning:
		return "WARNING"
	case StatusProblem:
		return "PROBLEM"
	case StatusSkipped:
		return "SKIPPED"
	default:
		return "UNKNOWN"
	}
}

// Symbol returns a short glyph used in terminal output.
func (s Status) Symbol() string {
	switch s {
	case StatusOK:
		return "✓"
	case StatusInfo:
		return "i"
	case StatusWarning:
		return "!"
	case StatusProblem:
		return "✗"
	case StatusSkipped:
		return "-"
	default:
		return "?"
	}
}

// Fix describes an optional remediation a check can offer.
// Apply is only invoked when the user explicitly confirms.
type Fix struct {
	Description string
	// Apply performs the fix and returns an error if it fails.
	// It must be idempotent and safe to run more than once.
	Apply func() error
}

// Result is what every check returns.
type Result struct {
	Check   string // short machine name, e.g. "audio-crackle"
	Title   string // human-readable name, e.g. "Audio crackling (PipeWire)"
	Status  Status
	Message string // one or two sentence explanation
	Detail  string // optional extra detail (paths checked, values found, etc.)
	Fix     *Fix   // nil if there's nothing to offer
}
