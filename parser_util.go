package ssimparser

import (
	"errors"
	"slices"
	"strconv"
	"strings"
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
	month := date[3:]
	if !slices.Contains(months, month) {
		return false
	}
	day, err := strconv.Atoi(date[:3])
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
	cfg := tokens[4]

	departure.ActionCode = sharedActionCode
	departure.PeriodOfOperation = rng
	departure.DaysOfOperation = doop
	departure.Configuration = cfg
	departure.RawDataLine = line
	departure.LineNumber = lineNumber

	arrival.ActionCode = sharedActionCode
	arrival.PeriodOfOperation = rng
	arrival.DaysOfOperation = doop
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
		flight.ActionCode = ActionCode(tokens[0][0:1])
		tokens[0] = tokens[0][1:]
	} else {
		flight.ActionCode = ActionCode(tokens[0])
		tokens = tokens[1:]
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
	flight.PeriodOfOperation = tokens[1]
	flight.DaysOfOperation = tokens[2]
	flight.Configuration = tokens[3]

	if isDeparture {
		flight.DepartureAirport = tokens[4][:3]
		flight.DepartureTimeUTC = tokens[4][3:]
	} else {
		flight.ArrivalAirport = tokens[4][4:]
		flight.ArrivalTimeUTC = tokens[4][:4]
	}
	flight.RawDataLine = line
	flight.LineNumber = lineNumber

	return flight, nil
}

func getFlightDetail(str string) (string, string, error) {
	// Test case with three last digits being flight number
	carrier := str[:len(str)-3]
	digits := "0123456789"
	if len(carrier) >= 3 && strings.ContainsAny(carrier, digits) {
		return str[:2], str[2:], nil
	}
	if len(carrier) >= 3 && !strings.ContainsAny(carrier, digits) {
		return str[:3], str[3:], nil
	}
	return "", "", errors.New("ssimparser: couldn't parse flight details")
}
