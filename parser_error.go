package ssimparser

import "fmt"

type SCRErrorLevel int

const (
	Minor SCRErrorLevel = iota
	Major
	Critical
)

func (l SCRErrorLevel) String() string {
	switch l {
	case Minor:
		return "Minor"
	case Major:
		return "Major"
	case Critical:
		return "Critical"
	default:
		return fmt.Sprintf("Level(%d)", l)
	}
}

type ParserError struct {
	Message    string // error message
	LineNumber int    // line number of error occurence
	RawLine    string // raw data line that caused the error
	Err        error  // underlying error
	Severity   SCRErrorLevel
}

func (e ParserError) Error() string {
	return fmt.Sprintf("ssimparser: %s at line %d: %s", e.Message, e.LineNumber, e.RawLine)
}

func NewParserError(message string, lineNumber int, rawLine string, err error, severity SCRErrorLevel) *ParserError {
	return &ParserError{Message: message, LineNumber: lineNumber, RawLine: rawLine, Err: err, Severity: severity}
}

type ParsingValidator struct {
	Container []*ParserError
}

func NewParsingValidator() *ParsingValidator {
	return &ParsingValidator{
		Container: make([]*ParserError, 0),
	}
}

// AddError adds a validation error to the validator
func (pv *ParsingValidator) AddError(err *ParserError) {
	pv.Container = append(pv.Container, err)
}

// ValidateSCR validates an already parsed SCRMessage and adds issues to the validator
func (pv *ParsingValidator) ValidateSCR(message *SCRMessage) {
	// Add validation logic here
	// Example: check if required fields are present
	if message.Identifier != "SCR" {
		pv.AddError(NewParserError("missing SCR identifier", 0, "", nil, Critical))
	}
	if message.Season == "" {
		pv.AddError(NewParserError("missing season", 0, "", nil, Critical))
	}
	if message.AirportCode == "" {
		pv.AddError(NewParserError("missing airport code", 0, "", nil, Critical))
	}
	// Add more validation rules as needed
}

func (pv *ParsingValidator) AssesErrors() (int, int, int) {
	minor, major, critical := 0, 0, 0
	for _, el := range pv.Container {
		switch el.Severity {
		case 0:
			minor++
		case 1:
			major++
		case 2:
			critical++
		}
	}
	return minor, major, critical
}
func (pv *ParsingValidator) Report() string {
	minor, major, critical := pv.AssesErrors()
	if critical > 0 {
		return fmt.Sprintf("There is %v minor, %v major and %v CRITICAL errors, therefore it is impossible to create SCR", minor, major, critical)
	}
	if critical == 0 && major > 1 {
		return fmt.Sprintf("There is %v minor and %v major errors, therefore fixes need to be introduced to create SCR", minor, major)
	}
	return fmt.Sprintf("There is %v minor errors - SCR will be created but consider fixing those issues", minor)
}
