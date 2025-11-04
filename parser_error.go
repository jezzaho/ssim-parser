package ssimparser 

import "fmt"

type ParserError struct {
	Message string // error message 
	LineNumber int // line number of error occurence 
	RawLine string // raw data line that caused the error 
	Err error // underlying error 
}

func (e ParserError) Error() string {
	return fmt.Sprintf("ssimparser: %s at line %d: %s", e.Message, e.LineNumber, e.RawLine)
}

func NewParserError(message string, lineNumber int, rawLine string, err error) *ParserError {
	return &ParserError{Message: message, LineNumber: lineNumber, RawLine: rawLine, Err: err}
}
