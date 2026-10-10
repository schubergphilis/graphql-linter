package models

// Severity tells whether a finding fails the run. The zero value is an error.
type Severity int

const (
	SeverityError Severity = iota
	// SeverityWarning findings are reported but do not fail the run.
	SeverityWarning
)

type DescriptionError struct {
	FilePath    string
	LineNum     int
	Message     string
	LineContent string
	// Value is the name of the offending element, matched against a
	// suppression's value.
	Value    string
	Severity Severity
}
