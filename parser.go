package ssimparser

import (
	"bufio"
	"io"
	"strings"
)

// SCR
// W25
// 15OCT
// KRK
// REYT/15OCT25/
// X FR7840 01JAN01JAN 0004000 18973H 1105GOT J
// K FR7840 01JAN01JAN 0004000 18973H 1155GOT J
// X FR5610 01JAN01JAN 0004000 18973H 1255FMM J
// K FR5610 01JAN01JAN 0004000 18973H 1315FMM J

// First four lines are always in the same order for valid SCR
// SCR \n W25 \n 15OCT \n KRK
// Then any number of administrative lines

type SCRParser interface {
	Parse(r io.Reader) (*SCRMessage, error)
}

type ScrParser struct {
	MIN_SSIM_LINE_LENGTH int
	validator            *ParsingValidator // Optional validator for collecting issues
}

// Non-Argument Initializer
func NewScrParser() *ScrParser {
	return &ScrParser{MIN_SSIM_LINE_LENGTH: 37}
}

// Argument Initializer
func NewScrParserWithMinLength(minLength int) *ScrParser {
	return &ScrParser{MIN_SSIM_LINE_LENGTH: minLength}
}

// NewScrParserWithValidator creates a parser with an embedded validator.
// This allows the parser to collect validation issues during parsing.
// The validator can be used to track minor, major, and critical issues.
//
// Usage:
//
//	parser := NewScrParserWithValidator()
//	message, err := parser.Parse(reader)
//	if err != nil { /* handle critical parsing error */ }
//	validator := parser.GetValidator()
//	validator.ValidateSCR(message) // post-parse validation
//	report := validator.Report() // get validation report
func NewScrParserWithValidator() *ScrParser {
	return &ScrParser{
		MIN_SSIM_LINE_LENGTH: 37,
		validator:            NewParsingValidator(),
	}
}

// NewScrParserWithValidatorAndMinLength creates a parser with validator and custom min length
func NewScrParserWithValidatorAndMinLength(minLength int) *ScrParser {
	return &ScrParser{
		MIN_SSIM_LINE_LENGTH: minLength,
		validator:            NewParsingValidator(),
	}
}

// SetValidator attaches a validator to the parser.
// This allows you to use a shared validator across multiple parsers,
// or attach a validator to an existing parser.
func (scr *ScrParser) SetValidator(v *ParsingValidator) {
	scr.validator = v
}

// GetValidator returns the parser's validator (creates one if none exists).
// Use this to access the validator after parsing to check for validation issues.
func (scr *ScrParser) GetValidator() *ParsingValidator {
	if scr.validator == nil {
		scr.validator = NewParsingValidator()
	}
	return scr.validator
}

// HasValidator returns true if the parser has a validator attached.
// Useful for checking if validation is enabled before accessing the validator.
func (scr *ScrParser) HasValidator() bool {
	return scr.validator != nil
}

// addValidationIssue adds a validation issue to the parser's validator if one exists
func (scr *ScrParser) addValidationIssue(message string, lineNumber int, rawLine string, err error, severity SCRErrorLevel) {
	if scr.validator != nil {
		scr.validator.AddError(NewParserError(message, lineNumber, rawLine, err, severity))
	}
}

func (scr *ScrParser) Parse(r io.Reader) (*SCRMessage, *ParserError) {
	message := &SCRMessage{
		AdministrativeLines: make([]string, 0),
		Items:               make([]*SlotItem, 0),
	}
	headerComplete := false
	lineNumber := 0
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()
		line = strings.TrimSpace(line)
		// if empty just skip
		if len(line) == 0 {
			continue
		}
		if !headerComplete {
			if scr.isSlotDataLine(line) || strings.HasPrefix(line, "GI") || strings.HasPrefix(line, "SI") {
				headerComplete = true
			} else {
				if err := scr.parseHeader(line, message, lineNumber); err != nil {
					return nil, err
				}
				continue
			}
		}
		if headerComplete {
			switch {
			case scr.isSlotDataLine(line):
				items, err := scr.parseData(line, lineNumber, message.AirportCode)
				if err != nil {
					return nil, err
				}
				for _, item := range items {
					item.GetSlotKey(message.AirportCode)
					message.Items = append(message.Items, item)
				}

			case strings.HasPrefix(line, "GI"):
				message.GeneralInfo += line[2:]
			case strings.HasPrefix(line, "SI"):
				message.SpecialInfo += line[2:]
			default:
			}
		}
	}
	return message, nil
}

func (scr *ScrParser) parseHeader(line string, message *SCRMessage, lineNumber int) *ParserError {
	tokens := strings.Fields(line)

	if line == "SCR" {
		message.Identifier = "SCR"
		return nil
	}
	//Process single-field lines
	if len(tokens) == 1 {
		if message.Season == "" && len(tokens[0]) == 3 && (tokens[0][0] == 'S' || tokens[0][0] == 'W') {
			message.Season = tokens[0]
			return nil
		}
		if message.MessageDate == "" && len(tokens[0]) == 5 && isDateDDMMM(tokens[0]) {
			message.MessageDate = tokens[0]
			return nil
		}
		if message.AirportCode == "" && len(tokens[0]) == 3 {
			message.AirportCode = tokens[0]
			return nil
		}
	}
	// Process rest as administrative
	message.AdministrativeLines = append(message.AdministrativeLines, line)

	return nil
}
func (scr *ScrParser) parseData(line string, lineNumber int, messageAirportCode string) ([]*SlotItem, *ParserError) {
	tokens := strings.Fields(line)
	bucket := make([]*SlotItem, 0, 2)

	// Three cases: Turnaround - 2 SlotItems
	// Arrival or departure - 1 SlotItem

	// separate functions to deal with it
	if len(tokens) == 8 {
		turnarounds, err := parseTurnaroundLine(tokens, line, lineNumber)
		if err != nil {
			return nil, NewParserError("ssimparser: parsing turnaround parser error", lineNumber, line, err, Critical)
		}
		bucket = append(bucket, turnarounds...)
	} else {
		slot, err := parseSingularLine(tokens, line, lineNumber)
		if err != nil {
			return nil, NewParserError("ssimparser: single slot parser error", lineNumber, line, err, Critical)
		}
		bucket = append(bucket, slot)
	}
	return bucket, nil
}

// ssimparser/parser.go or ssimparser/parser_util.go

// isSlotDataLine checks if a line starts with a recognized Action Code,
// indicating it is a Schedule Information Data Line (SlotItem data).
func (p *ScrParser) isSlotDataLine(line string) bool {
	if len(line) >= p.MIN_SSIM_LINE_LENGTH {
		return true
	}
	return false
	//TODO: Add better check later
}
