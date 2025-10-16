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
			case scr.isSlotDataLine(line, lineNumber):
				item, err := scr.parseData(line, lineNumber)
				if err != nil {
					return nil, err
				}
				item.SlotKey = item.GetSlotKey(message.AirportCode)

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

	if lineNumber == 1 && line == "SCR" {
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

	// Check line structure based on token count
	// The number of tokens is an imperfect but necessary way to differentiate structure

	// --- SCENARIO 1: TURNAROUND (HLH4123 LH4876 ...) ---
	if len(tokens) == 8 || len(tokens) == 9 {
		// 8 tokens if AC is consolidated (HLH4123)
		// 9 tokens if AC is separate (H LH4123)

		// This is the only scenario that produces two items.
		// Delegate parsing to a dedicated turnover function.
		return scr.parseTurnaround(tokens, line, lineNumber, messageAirportCode)

	} else if len(tokens) == 6 || len(tokens) == 7 {
		// --- SCENARIO 2: SINGLE SEGMENT (Departure or Arrival) ---
		// 6 tokens if AC is consolidated (KLO3923)
		// 7 tokens if AC is separate (K LO3923)

		// This is the only scenario that produces one item.
		// Delegate parsing to a dedicated single segment function.
		item, err := scr.parseSingleSegment(tokens, line, lineNumber, messageAirportCode)
		if err != nil {
			return nil, err
		}
		bucket = append(bucket, item)
		return bucket, nil

	} else {
		// --- SCENARIO 3: ERROR ---
		return nil, scr.newParsingError(lineNumber, "TokenCount", line, "Unexpected number of tokens. Expected 6, 7, 8, or 9.")
	}
}

// parseTurnaround handles lines that produce two SlotItems (inbound and outbound).
func (scr *ScrParser) parseTurnaround(tokens []string, line string, lineNumber int, messageAirportCode string) ([]*SlotItem, error) {

	// --- Phase 1: Normalize Tokens and Identify AC/Flights ---
	var sharedActionCode ActionCode
	var flightDesignator1, flightDesignator2 string

	// Handle consolidated vs. space-separated Action Code
	if len(tokens) == 9 { // H LH4123 LH4876...
		sharedActionCode = ActionCode(tokens[0])
		flightDesignator1 = tokens[1]
		flightDesignator2 = tokens[2]
		tokens = tokens[3:] // Normalize: tokens now start at Dates Block (01JUL26JUL)
	} else { // HLH4123 LH4876...
		sharedActionCode = ActionCode(tokens[0][0:1])
		flightDesignator1 = tokens[0][1:]
		flightDesignator2 = tokens[1]
		tokens = tokens[2:] // Normalize: tokens now start at Dates Block (01JUL26JUL)
	}

	// --- Phase 2: Create Base Items ---

	// Extract Carrier/Flight from Designators
	carrier, primaryFlightNum, err := scr.dissectfno(flightDesignator1)
	if err != nil {
		return nil, scr.newParsingError(lineNumber, "FlightDesignator1", flightDesignator1, "Failed to dissect flight number.")
	}

	_, secondaryFlightNum, err := scr.dissectfno(flightDesignator2)
	if err != nil {
		return nil, scr.newParsingError(lineNumber, "FlightDesignator2", flightDesignator2, "Failed to dissect flight number.")
	}

	// Initialize two separate structs
	outboundItem := scr.createBaseSlotItem(line, lineNumber, sharedActionCode, carrier, primaryFlightNum)
	inboundItem := scr.createBaseSlotItem(line, lineNumber, sharedActionCode, carrier, secondaryFlightNum)

	// --- Phase 3: Parse and Assign Data Blocks ---

	// 0: Dates, 1: DaysOp, 2: Config, 3: DepTime/Airport, 4: ArrTime/Airport, 5: ServiceType

	// Common Fields
	rawDates, rawDays, rawConf := tokens[0], tokens[1], tokens[2]
	depStr, arrStr := tokens[3], tokens[4]

	// Service Type (Often consolidated in the last token for turnover, e.g., 'JJ')
	rawServiceType := tokens[5]

	// Assign Common Fields
	// Use a helper function to set common fields and validate/parse dates/days (e.g., setCommonData(items, ...))

	// Time/Airport
	// You need logic here to determine which token is the arrival and which is the departure
	// and correctly set ArrivalAirport/Time vs DepartureAirport/Time on the correct item.

	return []*SlotItem{outboundItem, inboundItem}, nil
}

// ssimparser/parser.go

// parseSingleSegment handles lines that produce a single SlotItem (Arrival Only or Departure Only).
func (scr *ScrParser) parseSingleSegment(tokens []string, line string, lineNumber int, messageAirportCode string) (*SlotItem, error) {

	// --- Phase 1: Normalize Tokens and Identify AC/Flight ---

	var actionCode ActionCode
	var flightDesignator string

	// Handle consolidated vs. space-separated Action Code
	if len(tokens) == 7 { // K LO3923 20OCT20OCT... (Space-separated AC)
		actionCode = ActionCode(tokens[0])
		flightDesignator = tokens[1]
		tokens = tokens[2:] // Normalize: tokens now start at Dates Block (20OCT20OCT)
	} else { // KLO3923 20OCT20OCT... (Consolidated AC)
		actionCode = ActionCode(tokens[0][0:1])
		flightDesignator = tokens[0][1:]
		tokens = tokens[1:] // Normalize: tokens now start at Dates Block (20OCT20OCT)
	}

	// --- Phase 2: Create Base Item ---

	// Extract Carrier/Flight
	carrier, flightNum, err := scr.dissectfno(flightDesignator)
	if err != nil {
		return nil, scr.newParsingError(lineNumber, "FlightDesignator", flightDesignator, "Failed to dissect flight number.")
	}

	item := scr.createBaseSlotItem(line, lineNumber, actionCode, carrier, flightNum)

	// --- Phase 3: Parse and Assign Data Blocks (Tokens are now normalized to 5 elements) ---
	// [0] DatesBlock, [1] DaysOp, [2] Config, [3] Time/Airport, [4] ServiceType

	if len(tokens) != 5 {
		return nil, scr.newParsingError(lineNumber, "OperationalData", line, "After normalization, expected 5 operational tokens.")
	}

	// 1. Common Fields
	rawDates, rawDays, rawConf := tokens[0], tokens[1], tokens[2]

	// Set common fields (requires implementing helper functions)
	// if err := scr.setPeriodOfOperation(rawDates, item); err != nil { return nil, err }
	// if err := scr.setDaysOfOperation(rawDays, item); err != nil { return nil, err }
	// item.Configuration = rawConf

	// 2. Service Type (Final Token)
	rawServiceType := tokens[4]
	item.ServiceType = ServiceType(rawServiceType)
	// Add validation for ServiceType here

	// 3. Time/Airport (The single, defining token)
	rawTimeAirport := tokens[3]

	// Determine if the line is Arrival Only (Time-Airport) or Departure Only (Airport-Time)
	if err := scr.parseSingleSegmentTimeAirport(rawTimeAirport, item); err != nil {
		return nil, scr.newParsingError(lineNumber, "TimeAirportData", rawTimeAirport, err.Error())
	}

	// --- Phase 4: Finalize ---
	item.SlotKey = item.GetSlotKey(messageAirportCode)

	return item, nil
}
