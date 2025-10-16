package ssimparser

import (
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
	if len(tokens) != 6 && len(tokens) != 7 && len(token) != 8 {
		return false
	}
	return true
}
