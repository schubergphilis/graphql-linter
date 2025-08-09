package models

type DescriptionError struct {
	FilePath    string
	LineNum     int
	Message     string
	LineContent string
}
