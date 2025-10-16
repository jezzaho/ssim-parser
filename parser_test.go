package ssimparser

import (
	"fmt"
	"log"
	"strings"
	"testing"
)

func TestSingle(t *testing.T) {
	const testScrMessage = `SCR
S25
15OCT
KRK
NLO3924 20OCT20OCT 1000000 082E75 1625WAW J
`

	scr := NewScrParser()
	message, err := scr.Parse(strings.NewReader(testScrMessage))
	if err != nil {
		log.Fatal("ssimparser: Fatal parse error")
	}
	fmt.Printf("%v\n", message)
}
