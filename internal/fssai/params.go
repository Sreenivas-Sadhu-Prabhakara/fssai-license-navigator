// Package fssai decides which tier of food business licence an Indian food
// business operator needs under the Food Safety and Standards Act, 2006 and
// the Food Safety and Standards (Licensing and Registration of Food
// Businesses) Regulations, 2011, and tracks the renewal date of a licence.
//
// Every statutory threshold, ceiling and fee lives in this file as a named
// constant and is mirrored into [Params] so a caller can override it when the
// authority revises the figures. All figures are INDICATIVE: confirm the
// current ones on the FoSCoS portal before you apply.
package fssai

import (
	"errors"
	"fmt"
)

// Turnover thresholds, in rupees per financial year.
const (
	// PettyFBOTurnoverCeilingINR is the annual turnover ceiling for a petty
	// food business operator, who needs only a Basic Registration.
	// Twelve lakh rupees.
	PettyFBOTurnoverCeilingINR = 1_200_000.0

	// StateLicenceTurnoverCeilingINR is the annual turnover above which a
	// Central Licence is required instead of a State Licence.
	// Twenty crore rupees.
	StateLicenceTurnoverCeilingINR = 200_000_000.0
)

// Production-capacity ceilings. Capacity is read per day and is meant for
// manufacturing and processing units; traders, retailers and food service
// operators normally declare zero.
const (
	// PettyFBOCapacityKgPerDay is the food production capacity ceiling for a
	// petty food business operator. Above it a State Licence is needed even
	// when turnover is small.
	PettyFBOCapacityKgPerDay = 100.0

	// StateCeilingFoodMTPerDay is the food manufacturing or processing
	// capacity above which a Central Licence is required. Two metric tonnes
	// per day.
	StateCeilingFoodMTPerDay = 2.0
)

// Milk and milk products ceilings.
const (
	// DairyBasicCeilingLitresPerDay is the milk handling ceiling for a petty
	// food business operator. Five hundred litres per day.
	DairyBasicCeilingLitresPerDay = 500.0

	// StateCeilingDairyLitresPerDay is the milk handling capacity above which
	// a Central Licence is required. Fifty thousand litres per day.
	StateCeilingDairyLitresPerDay = 50_000.0

	// StateCeilingMilkSolidsMTPerYear is the milk solids throughput above
	// which a Central Licence is required. Two thousand five hundred metric
	// tonnes per annum.
	StateCeilingMilkSolidsMTPerYear = 2_500.0
)

// CentralMinStatesOfOperation is the number of states of operation at which
// the head office needs a Central Licence. Operating in two or more states
// triggers it.
const CentralMinStatesOfOperation = 2

// Indicative government fees, in rupees per year of validity.
const (
	// FeeBasicRegistrationPerYearINR is the fee for a Basic Registration.
	FeeBasicRegistrationPerYearINR = 100

	// FeeStateFoodServicePerYearINR is the State Licence fee for food service
	// operators: restaurants, caterers, canteens and clubs.
	FeeStateFoodServicePerYearINR = 2_000

	// FeeStateOtherFBOPerYearINR is the State Licence fee for any other food
	// business operator: traders, retailers, distributors, transporters and
	// storage units.
	FeeStateOtherFBOPerYearINR = 2_000

	// FeeStateManufacturerSmallPerYearINR is the State Licence fee for a
	// manufacturer or processor at or below the small-unit capacity band.
	FeeStateManufacturerSmallPerYearINR = 3_000

	// FeeStateManufacturerLargePerYearINR is the State Licence fee for a
	// manufacturer or processor above the small-unit capacity band.
	FeeStateManufacturerLargePerYearINR = 5_000

	// FeeCentralLicencePerYearINR is the fee for a Central Licence.
	FeeCentralLicencePerYearINR = 7_500
)

// Capacity bands that separate the two State Licence manufacturing fees.
const (
	// StateFeeManufacturerBandMTPerDay is the food capacity at or below which
	// the small-unit manufacturing fee applies. One metric tonne per day.
	StateFeeManufacturerBandMTPerDay = 1.0

	// StateFeeDairyBandLitresPerDay is the milk capacity at or below which the
	// small-unit manufacturing fee applies. Ten thousand litres per day.
	StateFeeDairyBandLitresPerDay = 10_000.0
)

// Validity a licence or registration can be applied for, in years.
const (
	ValidityMinYears = 1
	ValidityMaxYears = 5
)

// Renewal alert thresholds, in days remaining before expiry.
const (
	AlertNoticeDays   = 90
	AlertWarningDays  = 30
	AlertCriticalDays = 7
)

// LateRenewalPenaltyPerDayINR is the indicative late fee charged for each day
// a renewal is filed after the licence has expired.
const LateRenewalPenaltyPerDayINR = 100

// LicenceNumberDigits is the length of an FSSAI registration or licence
// number.
const LicenceNumberDigits = 14

// KgPerMetricTonne converts kilogrammes to metric tonnes.
const KgPerMetricTonne = 1000.0

// Params holds every figure the tier decision depends on. [DefaultParams]
// fills it from the named constants above; a caller may change any field and
// pass it to [TierWith] when the authority revises a figure.
type Params struct {
	PettyFBOTurnoverCeilingINR     float64 `json:"pettyFBOTurnoverCeilingINR"`
	StateLicenceTurnoverCeilingINR float64 `json:"stateLicenceTurnoverCeilingINR"`

	PettyFBOCapacityKgPerDay float64 `json:"pettyFBOCapacityKgPerDay"`
	StateCeilingFoodMTPerDay float64 `json:"stateCeilingFoodMTPerDay"`

	DairyBasicCeilingLitresPerDay   float64 `json:"dairyBasicCeilingLitresPerDay"`
	StateCeilingDairyLitresPerDay   float64 `json:"stateCeilingDairyLitresPerDay"`
	StateCeilingMilkSolidsMTPerYear float64 `json:"stateCeilingMilkSolidsMTPerYear"`

	CentralMinStatesOfOperation int `json:"centralMinStatesOfOperation"`

	FeeBasicRegistrationPerYearINR      int `json:"feeBasicRegistrationPerYearINR"`
	FeeStateFoodServicePerYearINR       int `json:"feeStateFoodServicePerYearINR"`
	FeeStateOtherFBOPerYearINR          int `json:"feeStateOtherFBOPerYearINR"`
	FeeStateManufacturerSmallPerYearINR int `json:"feeStateManufacturerSmallPerYearINR"`
	FeeStateManufacturerLargePerYearINR int `json:"feeStateManufacturerLargePerYearINR"`
	FeeCentralLicencePerYearINR         int `json:"feeCentralLicencePerYearINR"`

	StateFeeManufacturerBandMTPerDay float64 `json:"stateFeeManufacturerBandMTPerDay"`
	StateFeeDairyBandLitresPerDay    float64 `json:"stateFeeDairyBandLitresPerDay"`

	ValidityMinYears int `json:"validityMinYears"`
	ValidityMaxYears int `json:"validityMaxYears"`

	AlertNoticeDays   int `json:"alertNoticeDays"`
	AlertWarningDays  int `json:"alertWarningDays"`
	AlertCriticalDays int `json:"alertCriticalDays"`

	LateRenewalPenaltyPerDayINR int `json:"lateRenewalPenaltyPerDayINR"`
}

// DefaultParams returns the indicative figures published for the FSSAI
// licensing tiers. Confirm them on the FoSCoS portal before applying.
func DefaultParams() Params {
	return Params{
		PettyFBOTurnoverCeilingINR:     PettyFBOTurnoverCeilingINR,
		StateLicenceTurnoverCeilingINR: StateLicenceTurnoverCeilingINR,

		PettyFBOCapacityKgPerDay: PettyFBOCapacityKgPerDay,
		StateCeilingFoodMTPerDay: StateCeilingFoodMTPerDay,

		DairyBasicCeilingLitresPerDay:   DairyBasicCeilingLitresPerDay,
		StateCeilingDairyLitresPerDay:   StateCeilingDairyLitresPerDay,
		StateCeilingMilkSolidsMTPerYear: StateCeilingMilkSolidsMTPerYear,

		CentralMinStatesOfOperation: CentralMinStatesOfOperation,

		FeeBasicRegistrationPerYearINR:      FeeBasicRegistrationPerYearINR,
		FeeStateFoodServicePerYearINR:       FeeStateFoodServicePerYearINR,
		FeeStateOtherFBOPerYearINR:          FeeStateOtherFBOPerYearINR,
		FeeStateManufacturerSmallPerYearINR: FeeStateManufacturerSmallPerYearINR,
		FeeStateManufacturerLargePerYearINR: FeeStateManufacturerLargePerYearINR,
		FeeCentralLicencePerYearINR:         FeeCentralLicencePerYearINR,

		StateFeeManufacturerBandMTPerDay: StateFeeManufacturerBandMTPerDay,
		StateFeeDairyBandLitresPerDay:    StateFeeDairyBandLitresPerDay,

		ValidityMinYears: ValidityMinYears,
		ValidityMaxYears: ValidityMaxYears,

		AlertNoticeDays:   AlertNoticeDays,
		AlertWarningDays:  AlertWarningDays,
		AlertCriticalDays: AlertCriticalDays,

		LateRenewalPenaltyPerDayINR: LateRenewalPenaltyPerDayINR,
	}
}

// ErrInvalidParams reports an overridden figure set that cannot be used.
var ErrInvalidParams = errors.New("invalid thresholds")

// Validate rejects an override set that is internally inconsistent, so a typo
// in a revised figure cannot silently produce a wrong tier.
func (p Params) Validate() error {
	type check struct {
		ok  bool
		msg string
	}
	checks := []check{
		{p.PettyFBOTurnoverCeilingINR > 0, "petty food business operator turnover ceiling must be above zero"},
		{p.StateLicenceTurnoverCeilingINR > p.PettyFBOTurnoverCeilingINR, "State Licence turnover ceiling must be above the petty food business operator ceiling"},
		{p.PettyFBOCapacityKgPerDay > 0, "petty food business operator capacity ceiling must be above zero"},
		{p.StateCeilingFoodMTPerDay > 0, "State food capacity ceiling must be above zero"},
		{p.StateCeilingFoodMTPerDay*KgPerMetricTonne > p.PettyFBOCapacityKgPerDay, "State food capacity ceiling must be above the petty food business operator capacity ceiling"},
		{p.DairyBasicCeilingLitresPerDay > 0, "petty dairy ceiling must be above zero"},
		{p.StateCeilingDairyLitresPerDay > p.DairyBasicCeilingLitresPerDay, "State dairy ceiling must be above the petty dairy ceiling"},
		{p.StateCeilingMilkSolidsMTPerYear > 0, "milk solids ceiling must be above zero"},
		{p.CentralMinStatesOfOperation >= 2, "the multi-state trigger must be two states or more"},
		{p.StateFeeManufacturerBandMTPerDay > 0, "manufacturing fee band must be above zero"},
		{p.StateFeeDairyBandLitresPerDay > 0, "dairy fee band must be above zero"},
		{p.ValidityMinYears >= 1, "minimum validity must be at least one year"},
		{p.ValidityMaxYears >= p.ValidityMinYears, "maximum validity must not be below the minimum"},
		{p.AlertNoticeDays > p.AlertWarningDays, "the notice threshold must be further out than the warning threshold"},
		{p.AlertWarningDays > p.AlertCriticalDays, "the warning threshold must be further out than the critical threshold"},
		{p.AlertCriticalDays >= 0, "the critical threshold cannot be negative"},
	}
	for _, c := range checks {
		if !c.ok {
			return fmt.Errorf("%w: %s", ErrInvalidParams, c.msg)
		}
	}
	for _, f := range []struct {
		v    int
		name string
	}{
		{p.FeeBasicRegistrationPerYearINR, "Basic Registration fee"},
		{p.FeeStateFoodServicePerYearINR, "State Licence food service fee"},
		{p.FeeStateOtherFBOPerYearINR, "State Licence other food business fee"},
		{p.FeeStateManufacturerSmallPerYearINR, "State Licence small manufacturing fee"},
		{p.FeeStateManufacturerLargePerYearINR, "State Licence large manufacturing fee"},
		{p.FeeCentralLicencePerYearINR, "Central Licence fee"},
		{p.LateRenewalPenaltyPerDayINR, "late renewal penalty"},
	} {
		if f.v < 0 {
			return fmt.Errorf("%w: %s cannot be negative", ErrInvalidParams, f.name)
		}
	}
	return nil
}
