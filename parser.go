package ssimparser

import (
	"bufio"
	"errors"
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
}

func NewScrParser() *ScrParser {
	return &ScrParser{MIN_SSIM_LINE_LENGTH: 37}
}

func (scr *ScrParser) Parse(r io.Reader) (*SCRMessage, error) {
	message := &SCRMessage{
		AdministrativeLines: make([]string, 0),
		Items:               make([]SlotItem, 0),
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

func (scr *ScrParser) parseHeader(line string, message *SCRMessage, lineNumber int) error {
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
func (scr *ScrParser) parseData(line string, lineNumber int, messageAirportCode string) ([]*SlotItem, error) {
	tokens := strings.Fields(line)
	bucket := make([]*SlotItem, 0, 2)

	// Three cases: Turnaround - 2 SlotItems
	// Arrival or departure - 1 SlotItem

	// separate functions to deal with it
	if len(tokens) == 8 {
		turnarounds, err := parseTurnaroundLine(tokens, line, lineNumber)
		if err != nil {
			return nil, errors.New("ssimparser: parsing turnaround parser error")
		}
		bucket = append(bucket, turnarounds...)
	} else {
		slot, err := parseSingularLine(tokens, line, lineNumber)
		if err != nil {
			return nil, errors.New("ssimparser: single slot parser error")
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
