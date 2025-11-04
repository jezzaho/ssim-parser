package ssimparser

import (
	"fmt"
	"strings"
)

// SCR
//      S23
//      01MAY
//      ICN
//      NAB123 AB124 26MAR28OCT 1234567 189738 PVG0110 0210PVG JJ
//      N AB456 26MAR28OCT 0204060 189738 0030KIX J
//      NAB457 26MAR28OCT 0204060 189738 KIX0500 J
//      SI ALL TIMES IN UTC
//      SI IF UNAVBL PLS OFFR NEXT LATER AVBL
//      GI BRGDS COMPANY/SENDER NAME

type SCRMessage struct {
	Identifier  string // "SCR"
	Season      string // S23 - S for Summer W for Winter
	MessageDate string // DDMMM format
	AirportCode string // IATA 3-letter code e.g KRK
	// Administrative lines
	AdministrativeLines []string
	//Payload core: slice of the individual slot requests/replies
	Items []*SlotItem
	// Optional addtional information GI - General Information, SI - Supplementary Information
	GeneralInfo string
	SpecialInfo string
}

func (msg SCRMessage) PrettyPrint() string {
	var sb strings.Builder

	sb.WriteString("=========== SCR MESSAGE ===========\n")
	sb.WriteString(fmt.Sprintf("Identifier:   %s\n", msg.Identifier))
	sb.WriteString(fmt.Sprintf("Season:       %s\n", msg.Season))
	sb.WriteString(fmt.Sprintf("Message Date: %s\n", msg.MessageDate))
	sb.WriteString(fmt.Sprintf("Airport Code: %s\n", msg.AirportCode))
	sb.WriteString("-----------------------------------\n")

	if len(msg.AdministrativeLines) > 0 {
		sb.WriteString("Administrative Lines:\n")
		for _, line := range msg.AdministrativeLines {
			sb.WriteString(fmt.Sprintf("  - %s\n", line))
		}
		sb.WriteString("-----------------------------------\n")
	}

	if len(msg.Items) > 0 {
		sb.WriteString("Slot Items:\n")
		for i, item := range msg.Items {
			sb.WriteString(item.prettyPrint(i + 1))
		}
		sb.WriteString("-----------------------------------\n")
	}

	if msg.GeneralInfo != "" {
		sb.WriteString("General Information (GI):\n")
		sb.WriteString(fmt.Sprintf("  %s\n", msg.GeneralInfo))
		sb.WriteString("-----------------------------------\n")
	}

	if msg.SpecialInfo != "" {
		sb.WriteString("Special Information (SI):\n")
		sb.WriteString(fmt.Sprintf("  %s\n", msg.SpecialInfo))
		sb.WriteString("-----------------------------------\n")
	}

	return sb.String()
}

func (s SlotItem) prettyPrint(index int) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("  #%d\n", index))
	sb.WriteString(fmt.Sprintf("    Action Code:        %s\n", s.ActionCode))
	sb.WriteString(fmt.Sprintf("    Carrier / Flight:   %s %s\n", s.CarrierCode, s.FlightNumber))
	sb.WriteString(fmt.Sprintf("%v\n", s.PeriodOfOperation.prettyPrint()))
	sb.WriteString(fmt.Sprintf("    Days of Operation:  %s\n", s.DaysOfOperation))
	if s.AircraftType != "" {
		sb.WriteString(fmt.Sprintf("    Aircraft Type:      %s\n", s.AircraftType))
	}
	if s.Configuration != "" {
		sb.WriteString(fmt.Sprintf("    Configuration:      %s\n", s.Configuration))
	}
	if s.ServiceType != "" {
		sb.WriteString(fmt.Sprintf("    Service Type:       %s\n", s.ServiceType))
	}

	if s.DepartureAirport != "" {
		sb.WriteString(fmt.Sprintf("    Departure:          %s at %s UTC\n",
			s.DepartureAirport, s.DepartureTimeUTC))
	}
	if s.ArrivalAirport != "" {
		sb.WriteString(fmt.Sprintf("    Arrival:            %s at %s UTC (ΔDay %+d)\n",
			s.ArrivalAirport, s.ArrivalTimeUTC, s.DayChangeIndicator))
	}

	if s.SlotKey != "" {
		sb.WriteString(fmt.Sprintf("    Slot Key:           %s\n", s.SlotKey))
	}

	sb.WriteString(fmt.Sprintf("    Source Line #%d:    %s\n", s.LineNumber, s.RawDataLine))
	sb.WriteString("  -----------------------------------\n")
	return sb.String()
}

// Slot item represents a single slot transaction - request, reply or offer
type SlotItem struct {
	// ===
	ActionCode ActionCode

	// Slot Identfication Data
	CarrierCode  string
	FlightNumber string

	// ScheduleData
	PeriodOfOperation *PeriodOfOperation //Period FIXME: change to Period
	DaysOfOperation   string             //[]int FIXME: change to []int
	AircraftType      string             // IATA 3-letter CODE
	Configuration     string             // Capacity/Seats

	ServiceType ServiceType

	// Arrival data - if exists
	ArrivalAirport     string
	ArrivalTimeUTC     string
	DayChangeIndicator int

	//Departure data - if exists
	DepartureAirport string
	DepartureTimeUTC string

	//Internal Metadata
	RawDataLine string
	LineNumber  int

	// SLOT KEY
	SlotKey string
}

func (s SlotItem) GetSlotKey(clearanceAirport string) string {
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		s.CarrierCode,
		s.FlightNumber,
		//FIXME: fix this later
		"a", "b",
		//s.PeriodOfOperation.EffectiveDate,
		//s.PeriodOfOperation.TerminationDate,
		clearanceAirport,
	)
}

type ActionCode string

const (
	// Airline Action Codes
	ActionNewRequest         ActionCode = "A" // Acceptance of an offer - no further improvement desired
	ActionNewEntrant         ActionCode = "B" // New Entrant
	ActionChangeSlot         ActionCode = "C" // Slot to be changed
	ActionDeleteSlot         ActionCode = "D" // Delete slot
	ActionEliminateSlot      ActionCode = "E" // Eliminate slot
	ActionHistoricUse        ActionCode = "F" // Historic Slot use
	ActionRevisedCont        ActionCode = "I" // Revised slot
	ActionRevisedNoOffer     ActionCode = "L" // Revised slot
	ActionNewSlot            ActionCode = "N" // New slot
	ActionAcceptanceMaintain ActionCode = "P" // Acceptance of an offer - maintain as outstanding request
	ActionNewRevised         ActionCode = "R" // Revised slot - acceptable
	ActionNewEntrantRound    ActionCode = "V" // New Entrant with year round status
	ActionNewSlotCont        ActionCode = "Y" // New Slot - continuation from previous adjacent season
	ActionDeclineOffer       ActionCode = "Z" // Decline Offer
	//Coordinator Action Codes
	ActionHoldingSlot   ActionCode = "H" // Holding Slot
	ActionConfirmation  ActionCode = "K" // New/Revise Slot Confirmation
	ActionOffer         ActionCode = "O" // Offer
	ActionPendingSlot   ActionCode = "P" // Request Pending
	ActionConditionSlot ActionCode = "T" // Allocated Slots subject to Conditions
	ActionUnableSlot    ActionCode = "U" // Unable to Confirm, Slot not allocated
	ActionUnableInfo    ActionCode = "W" // Unable to Reconcile Flight Information
	ActionDeleteandAck  ActionCode = "X" // Slot Delete Confirmation

)

type ServiceType string

const (
	ServiceTypePassenger    ServiceType = "J" // Passenger normal service
	ServiceTypeCargo        ServiceType = "F" // Scheduled Cargo
	ServiceTypeAdditional   ServiceType = "G" // Additional passenger
	ServiceTypeCharterPax   ServiceType = "C" // Charter passenger
	ServiceTypeCharterCargo ServiceType = "H" // Charter cargo
	ServiceTypePositioning  ServiceType = "P" // Position/ferry
	ServiceTypeTechTest     ServiceType = "T" // Technical Test
	ServiceTypeTraining     ServiceType = "K" // Training
	ServiceTypeTechStop     ServiceType = "X" // Technical stop
)

type PeriodOfOperation struct {
	EffectiveDate   string //
	TerminationDate string
	DurationDays    int
}

func POOFromString(s string) (*PeriodOfOperation, error) {
	return poocreator(s)
}

func (poo PeriodOfOperation) prettyPrint() string {
	return fmt.Sprintf("Period of Operation: %s to %s (%d days)", poo.EffectiveDate, poo.TerminationDate, poo.DurationDays)
}
