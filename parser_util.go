package ssimparser

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Return if string is format of SSIM Date DDMMM e.g. 05OCT, 15MAY, 21JUN
func isDateDDMMM(date string) bool {
	months := []string{
		"JAN",
		"FEB",
		"MAR",
		"APR",
		"MAY",
		"JUN",
		"JUL",
		"AUG",
		"SEP",
		"OCT",
		"NOV",
		"DEC",
	}
	// first month
	month := date[2:]
	if !slices.Contains(months, month) {
		return false
	}
	day, err := strconv.Atoi(date[:2])
	if err != nil {
		return false
	}
	//Simple sane check - need to fix it later
	//FIXME: Make this date check better
	if day < 1 || day > 31 {
		return false
	}

	return true
}

func isSlotDataLine(line string) bool {
	actionCodes := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "L", "N", "P", "R", "V", "Y", "Z", "H", "K", "O", "P", "T", "U", "W", "X"}
	code := string(line[0])

	if !slices.Contains(actionCodes, code) {
		return false
	}
	tokens := strings.Fields(line)
	if len(tokens) != 6 && len(tokens) != 7 && len(tokens) != 8 {
		return false
	}
	return true
}

// Turnaround flight parser returns multiple slot info structs
// FIXME: redudant tokens an line - tokens are build from line

func parseTurnaroundLine(tokens []string, line string, lineNumber int) ([]*SlotItem, error) {
	//HLH4123 LH4876 01JUL26JUL 0034507 120319 HAM0700 0750FRA JJ

	departure, arrival := &SlotItem{LineNumber: lineNumber, RawDataLine: line}, &SlotItem{LineNumber: lineNumber, RawDataLine: line}

	// Shared Data fields
	sharedActionCode := ActionCode(tokens[0][0:1])
	rng := tokens[2]
	doop := tokens[3]
	cfg, aircraft := getConfAndAicraft(tokens[4])

	departure.ActionCode = sharedActionCode
	var err error
	departure.PeriodOfOperation, err = POOFromString(rng)
	if err != nil {
		return nil, fmt.Errorf("ssimparser: period of operation error: %v", err)
	}

	departure.DaysOfOperation = doop
	departure.Configuration = cfg
	departure.AircraftType = aircraft
	departure.RawDataLine = line
	departure.LineNumber = lineNumber

	arrival.ActionCode = sharedActionCode
	arrival.PeriodOfOperation, err = POOFromString(rng)
	if err != nil {
		return nil, fmt.Errorf("ssimparser: period of operation error: %v", err)
	}
	arrival.DaysOfOperation = doop
	arrival.AircraftType = aircraft
	arrival.Configuration = cfg
	arrival.RawDataLine = line
	arrival.LineNumber = lineNumber

	// Individual data fields
	departureAirport := tokens[5][:3]
	departureTime := tokens[5][3:]
	carrier, fno, err := getFlightDetail(tokens[1])
	if err != nil {
		return nil, errors.New("ssimparser: flight detail parse error")
	}
	departure.CarrierCode = carrier
	departure.FlightNumber = fno

	arrivalAirport := tokens[6][4:]
	arrivalTime := tokens[6][:4]
	carrier, fno, err = getFlightDetail(tokens[0][1:])
	if err != nil {
		return nil, errors.New("ssimparser: flight detail parse error")
	}
	arrival.CarrierCode = carrier
	arrival.FlightNumber = fno

	departure.DepartureAirport = departureAirport
	departure.DepartureTimeUTC = departureTime

	arrival.ArrivalAirport = arrivalAirport
	arrival.ArrivalTimeUTC = arrivalTime

	departure.ServiceType = ServiceType(string(tokens[7][0]))
	arrival.ServiceType = ServiceType(string(tokens[7][1]))

	bucket := make([]*SlotItem, 0)
	bucket = append(bucket, departure)
	bucket = append(bucket, arrival)

	return bucket, nil
}
func parseSingularLine(tokens []string, line string, lineNumber int) (*SlotItem, error) {
	flight := &SlotItem{LineNumber: lineNumber, RawDataLine: line}
	isDeparture := false
	if len(tokens[0]) == 1 {
		isDeparture = true
	}
	if isDeparture {
		flight.ActionCode = ActionCode(tokens[0])
		tokens = tokens[1:]
	} else {
		flight.ActionCode = ActionCode(tokens[0][0:1])
		tokens[0] = tokens[0][1:]
	}
	//->>>K<<<--LO010 24OCT24OCT 0000500 252788 ORD0730 J
	//LO010 24OCT24OCT 0000500 252788 0730ORD J
	carrier, fno, err := getFlightDetail(tokens[0])
	if err != nil {
		return nil, errors.New("ssimparser: flight detail parser error")
	}
	flight.CarrierCode = carrier
	flight.FlightNumber = fno

	//Shared fields
	flight.PeriodOfOperation, err = POOFromString(tokens[1])
	if err != nil {
		return nil, errors.New("ssimparser: period of operation error")
	}
	flight.DaysOfOperation = tokens[2]
	cfg, aircraft := getConfAndAicraft(tokens[3])
	flight.AircraftType = aircraft
	flight.Configuration = cfg

	if isDeparture {
		flight.DepartureAirport = tokens[4][:3]
		flight.DepartureTimeUTC = tokens[4][3:]
	} else {
		flight.ArrivalAirport = tokens[4][:3]
		flight.ArrivalTimeUTC = tokens[4][3:]
	}
	flight.RawDataLine = line
	flight.LineNumber = lineNumber

	//TODO: ADD aircrafttype and servicetype fields

	return flight, nil
}

func getFlightDetail(str string) (string, string, error) {
	// Test case with three last digits being flight number
	carrier := str[:len(str)-3]
	digits := "0123456789"
	if len(carrier) >= 3 && strings.ContainsAny(carrier, digits) {
		return str[:2], str[2:], nil
	}
	if len(carrier) < 3 && !strings.ContainsAny(carrier, digits) {
		return str[:2], str[2:], nil
	}
	return "", "", errors.New("ssimparser: couldn't parse flight details")
}
func getConfAndAicraft(str string) (string, string) {
	// conf-seat/aicraft code map is always six digit and in form XXXYYY
	//capacity-conf is always padded with zeros if necessary
	return str[:3], str[3:]
}

// Helper function to create PeriodOfOperation from string
// Extensive validation inside
// Example: 01JAN31JAN -> PeriodOfOperation{EffectiveDate: "01JAN", TerminationDate: "31JAN", DurationDays: 31}

func poocreator(s string) (*PeriodOfOperation, error) {
	// Check if string length is valid (10) because DDMMMDDMMM - 2+3+2+3 = 10
	if len(s) != 10 {
		return nil, errors.New(fmt.Sprintf("ssimparser: invalid period of operation string length, expected 10 characters but have %v", len(s)))
	}
	// Check format of string DDMMMDDMMM
	if !isDateDDMMM(s[:5]) || !isDateDDMMM(s[5:]) {
		return nil, errors.New(fmt.Sprintf("ssimparser: invalid period of operation format, expected DDMMMDDMMM but have %v", s))
	}
	// Check if valid date range - first DDMMM must be before or equal to second DDMMM
	fromDate, err := convertDDMMMtoDate(s[0:5])
	if err != nil {
		return nil, errors.New(fmt.Sprintf("ssimparser: invalid period of operation format, expected DDMMM but have %v", s[0:4]))
	}
	toDate, err := convertDDMMMtoDate(s[5:])
	if err != nil {
		return nil, errors.New(fmt.Sprintf("ssimparser: invalid period of operation format, expected DDMMM but have %v", s[5:]))
	}
	if fromDate.After(toDate) {
		return nil, errors.New(fmt.Sprintf("ssimparser: invalid period of operation format, expected left DDMMM before or equal right DDMMM, but have %v", s))
	}
	duration := DaysBetween(toDate, fromDate)

	//TODO: Check if any more edges necessary
	return &PeriodOfOperation{
		EffectiveDate:   s[0:5],
		TerminationDate: s[5:],
		DurationDays:    duration,
	}, nil

}

func convertDDMMMtoDate(s string) (time.Time, error) {
	// DDMMM to number conversion
	day, err := strconv.Atoi(s[:2])
	if err != nil {
		return time.Time{}, errors.New(fmt.Sprintf("ssimparser: invalid day in DDMMM, expected integer but have %v", s[:3]))
	}
	monthMap := map[string]int{
		"JAN": 1,
		"FEB": 2,
		"MAR": 3,
		"APR": 4,
		"MAY": 5,
		"JUN": 6,
		"JUL": 7,
		"AUG": 8,
		"SEP": 9,
		"OCT": 10,
		"NOV": 11,
		"DEC": 12,
	}
	month := monthMap[s[2:]]
	if month == 0 {
		return time.Time{}, errors.New(fmt.Sprintf("ssimparser: invalid month in DDMMM, expected month but have %v", s[2:5]))
	}
	layout := "02012006"
	t, err := time.Parse(layout, fmt.Sprintf("%v%v2006", day, month))
	if err != nil {
		return time.Time{}, fmt.Errorf("ssimparser: invalid DDMMM, expected date (DDMMM) but have %v", s)
	}
	return t, nil
}
func DaysBetween(date1, date2 time.Time) int {
	// Get the absolute duration between the two dates.
	duration := date1.Sub(date2)
	if duration < 0 {
		duration = -duration
	}

	// Convert the total duration to hours and divide by 24 to get days,
	// or simply divide by 24 hours. time.Hour is a time.Duration constant.
	// The result is implicitly truncated (integer division).
	return int(duration / (24 * time.Hour))
}
