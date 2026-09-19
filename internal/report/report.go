// Package report defines the shared data types that every check produces.
// Keeping this separate from the checks themselves means the output
// format (plain text or JSON) can change without touching check logic.
package report

import "encoding/json"

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

// MarshalJSON serializes a Status as its string form ("OK", "WARNING",
// ...) rather than its underlying int, so JSON output is self-describing.
func (s Status) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
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

	// FixApplied and FixError are set by the caller after attempting a
	// fix (e.g. in --json --yes mode), so JSON output can report the
	// outcome. Both are ignored by the interactive text renderer.
	FixApplied bool
	FixError   string
}

// jsonResult is the JSON-serializable view of a Result. Fix.Apply is a
// function and can't be marshaled, so only its description and
// availability are exposed.
type jsonResult struct {
	Check          string `json:"check"`
	Title          string `json:"title"`
	Status         Status `json:"status"`
	Message        string `json:"message"`
	Detail         string `json:"detail,omitempty"`
	FixAvailable   bool   `json:"fix_available"`
	FixDescription string `json:"fix_description,omitempty"`
	FixApplied     bool   `json:"fix_applied,omitempty"`
	FixError       string `json:"fix_error,omitempty"`
}

// MarshalJSON implements json.Marshaler for Result.
func (r Result) MarshalJSON() ([]byte, error) {
	jr := jsonResult{
		Check:      r.Check,
		Title:      r.Title,
		Status:     r.Status,
		Message:    r.Message,
		Detail:     r.Detail,
		FixApplied: r.FixApplied,
		FixError:   r.FixError,
	}
	if r.Fix != nil {
		jr.FixAvailable = true
		jr.FixDescription = r.Fix.Description
	}
	return json.Marshal(jr)
}
