package models

type DescriptionError struct {
	FilePath    string
	LineNum     int
	Message     string
	LineContent string
	// Value is the name of the offending element, matched against a
	// suppression's value.
	Value string
}
