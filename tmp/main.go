package main

import (
	"fmt"
	"os"
	"strings"

	ssimparser "github.com/jezzaho/ssim-parser"
)

func main() {
	const testScrMessage = `SCR
S20
22APR
GVA
NBA998 BA997 19OCT19OCT 1000000 168320 MAN1125 1215MAN CC
NBA996 BA995 20OCT20OCT 0200000 259763 LGW1525 1615MAN CC
NBA990 BA991 21OCT21OCT 0030000 168320 EDI0800 0850GLA CP
GI BRGDS
SI HAPPYEASTERACKACK
`

	// Option 1: Create parser with embedded validator
	scr := ssimparser.NewScrParserWithValidator()
	message, parserError := scr.Parse(strings.NewReader(testScrMessage))

	// Handle critical parsing errors (these stop parsing)
	if parserError != nil && parserError.Err != nil {
		fmt.Println(parserError.Error())
		os.Exit(1)
	}

	// Validate the parsed message (checks for issues)
	if scr.HasValidator() {
		validator := scr.GetValidator()
		validator.ValidateSCR(message)

		// Check validation results
		minor, major, critical := validator.AssesErrors()
		if minor > 0 || major > 0 || critical > 0 {
			fmt.Println("Validation Report:")
			fmt.Println(validator.Report())
			fmt.Println()
		}

		// You can decide whether to proceed based on validation results
		if critical > 0 {
			fmt.Println("Critical errors found - cannot proceed")
			os.Exit(1)
		}
	}

	fmt.Println(message.PrettyPrint())
}
