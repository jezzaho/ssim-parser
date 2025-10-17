package main

import (
	"fmt"
	"log"
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
`

	scr := ssimparser.NewScrParser()
	message, err := scr.Parse(strings.NewReader(testScrMessage))
	if err != nil {
		log.Fatal("ssimparser: Fatal parse error")
	}
	fmt.Println(message.PrettyPrint())
}
