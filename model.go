package ssimparser

import "fmt"

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
	Items []SlotItem
	// Optional addtional information GI - General Information, SI - Supplementary Information
	GeneralInfo string
	SpecialInfo string
}

// Slot item represents a single slot transaction - request, reply or offer
type SlotItem struct {
	// ===
	ActionCode ActionCode

	// Slot Identfication Data
	CarrierCode  string
	FlightNumber string

	// ScheduleData
	PeriodOfOperation string //Period FIXME: change to Period
	DaysOfOperation   string //[]int FIXME: change to []int
	AircraftType      string // IATA 3-letter CODE
	Configuration     string // Capacity/Seats

	ServiceType []ServiceType

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
		s.PeriodOfOperation.EffectiveDate,
		s.PeriodOfOperation.TerminationDate,
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
	ActionConfirmSlot   ActionCode = "K" // New/Revise Slot Confirmation
	ActionOfferSlot     ActionCode = "O" // Offer
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

type Period struct {
	EffectiveDate   string //
	TerminationDate string
}
