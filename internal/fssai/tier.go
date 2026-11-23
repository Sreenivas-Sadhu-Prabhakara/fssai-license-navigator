package fssai

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// LicenceTier is one of the three levels of FSSAI authorisation a food
// business can hold.
type LicenceTier int

// The three tiers, in ascending order of scale.
const (
	TierBasicRegistration LicenceTier = iota
	TierStateLicence
	TierCentralLicence
)

// Name returns the display name of the tier as the regulations write it.
func (t LicenceTier) Name() string {
	switch t {
	case TierBasicRegistration:
		return "Basic Registration"
	case TierStateLicence:
		return "State Licence"
	case TierCentralLicence:
		return "Central Licence"
	}
	return "unknown"
}

func (t LicenceTier) String() string { return t.Name() }

// Slug returns a stable machine-readable key for the tier.
func (t LicenceTier) Slug() string {
	switch t {
	case TierBasicRegistration:
		return "basic"
	case TierStateLicence:
		return "state"
	case TierCentralLicence:
		return "central"
	}
	return "unknown"
}

// Form returns the application form prescribed for the tier.
func (t LicenceTier) Form() string {
	if t == TierBasicRegistration {
		return "Form A"
	}
	return "Form B"
}

// Authority returns the office that issues the tier.
func (t LicenceTier) Authority() string {
	switch t {
	case TierBasicRegistration:
		return "Registering Authority of the State or UT food safety department"
	case TierStateLicence:
		return "State or UT Licensing Authority (Food Safety Commissionerate)"
	case TierCentralLicence:
		return "Central Licensing Authority (FSSAI regional office)"
	}
	return ""
}

// ErrUnknownTier reports a tier slug that does not name one of the three
// tiers.
var ErrUnknownTier = errors.New("unknown licence tier")

// ParseTier reads a tier from its slug or display name.
func ParseTier(s string) (LicenceTier, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "basic", "basic registration", "registration":
		return TierBasicRegistration, nil
	case "state", "state licence", "state license":
		return TierStateLicence, nil
	case "central", "central licence", "central license":
		return TierCentralLicence, nil
	}
	return 0, fmt.Errorf("%w: %q", ErrUnknownTier, s)
}

// MarshalJSON writes the tier as its slug so stored data stays readable.
func (t LicenceTier) MarshalJSON() ([]byte, error) { return json.Marshal(t.Slug()) }

// UnmarshalJSON reads a tier written as a slug or display name.
func (t *LicenceTier) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	parsed, err := ParseTier(s)
	if err != nil {
		return err
	}
	*t = parsed
	return nil
}

// Activity is the kind of food business being run. It shapes the document
// checklist and the State Licence fee category.
type Activity string

// The food business activities the questionnaire offers.
const (
	ActivityManufacturer Activity = "manufacturer"
	ActivityProcessor    Activity = "processor"
	ActivityTrader       Activity = "trader"
	ActivityRetailer     Activity = "retailer"
	ActivityRestaurant   Activity = "restaurant"
	ActivityTransporter  Activity = "transporter"
	ActivityStorage      Activity = "storage"
	ActivityEcommerce    Activity = "ecommerce"
	ActivityImporter     Activity = "importer"
	ActivityExporter     Activity = "exporter"
)

// Activities lists every activity in questionnaire order.
func Activities() []Activity {
	return []Activity{
		ActivityManufacturer, ActivityProcessor, ActivityTrader, ActivityRetailer,
		ActivityRestaurant, ActivityTransporter, ActivityStorage, ActivityEcommerce,
		ActivityImporter, ActivityExporter,
	}
}

// Label returns the display name of the activity.
func (a Activity) Label() string {
	switch a {
	case ActivityManufacturer:
		return "Manufacturer"
	case ActivityProcessor:
		return "Processor"
	case ActivityTrader:
		return "Trader, wholesaler or distributor"
	case ActivityRetailer:
		return "Retailer"
	case ActivityRestaurant:
		return "Restaurant, caterer or canteen"
	case ActivityTransporter:
		return "Transporter"
	case ActivityStorage:
		return "Storage, warehouse or cold store"
	case ActivityEcommerce:
		return "E-commerce food business operator"
	case ActivityImporter:
		return "Importer"
	case ActivityExporter:
		return "Exporter"
	}
	return string(a)
}

// Makes reports whether the activity physically manufactures or processes
// food, which is what the production-capacity ceilings are written for.
func (a Activity) Makes() bool {
	return a == ActivityManufacturer || a == ActivityProcessor
}

// ErrUnknownActivity reports an activity outside the published list.
var ErrUnknownActivity = errors.New("unknown food business activity")

// ParseActivity reads an activity from its key.
func ParseActivity(s string) (Activity, error) {
	key := Activity(strings.ToLower(strings.TrimSpace(s)))
	for _, a := range Activities() {
		if a == key {
			return a, nil
		}
	}
	return "", fmt.Errorf("%w: %q", ErrUnknownActivity, s)
}

// CapacityUnit is the unit the declared production capacity is entered in.
type CapacityUnit string

// The two capacity units the questionnaire accepts.
const (
	UnitKgPerDay CapacityUnit = "kg/day"
	UnitMTPerDay CapacityUnit = "MT/day"
)

// ErrUnknownUnit reports a capacity unit that cannot be converted.
var ErrUnknownUnit = errors.New("unknown capacity unit")

// ParseCapacityUnit reads a capacity unit, tolerating the short spellings a
// form field tends to produce.
func ParseCapacityUnit(s string) (CapacityUnit, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "kg", "kg/day", "kgs", "kg per day", "kilogram", "kilograms":
		return UnitKgPerDay, nil
	case "mt", "mt/day", "mtd", "mt per day", "tonne", "tonnes", "metric tonne", "metric tonnes":
		return UnitMTPerDay, nil
	}
	return "", fmt.Errorf("%w: %q", ErrUnknownUnit, s)
}

// Input is the questionnaire answer set for one food business.
type Input struct {
	AnnualTurnoverINR float64  `json:"annualTurnoverINR"`
	Activity          Activity `json:"activity"`

	// ProductionCapacityPerDay is food made or processed per day, in
	// CapacityUnit. Traders, retailers and food service operators leave it at
	// zero.
	ProductionCapacityPerDay float64      `json:"productionCapacityPerDay"`
	CapacityUnit             CapacityUnit `json:"capacityUnit"`

	// DairyLitresPerDay is milk and milk products handled per day.
	DairyLitresPerDay float64 `json:"dairyLitresPerDay"`
	// DairyMilkSolidsMTPerYear is milk solids throughput per annum, the
	// alternative measure the dairy ceiling is written in.
	DairyMilkSolidsMTPerYear float64 `json:"dairyMilkSolidsMTPerYear"`

	StatesOfOperation int `json:"statesOfOperation"`

	IsImporter                 bool `json:"isImporter"`
	IsExporter                 bool `json:"isExporter"`
	IsEcommerce                bool `json:"isEcommerce"`
	Is100PctExportOriented     bool `json:"is100PctExportOriented"`
	OperatesAtAirportOrSeaport bool `json:"operatesAtAirportOrSeaport"`
}

// ErrInvalidInput reports a questionnaire answer that cannot be used.
var ErrInvalidInput = errors.New("invalid questionnaire answer")

// Validate reports answers a caller should fix rather than have silently
// corrected: an unknown activity or unit, or a negative quantity.
func (in Input) Validate() error {
	if _, err := ParseActivity(string(in.Activity)); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}
	if _, err := ParseCapacityUnit(string(in.CapacityUnit)); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}
	for _, f := range []struct {
		v    float64
		name string
	}{
		{in.AnnualTurnoverINR, "annual turnover"},
		{in.ProductionCapacityPerDay, "production capacity"},
		{in.DairyLitresPerDay, "milk handled per day"},
		{in.DairyMilkSolidsMTPerYear, "milk solids per year"},
	} {
		if f.v < 0 {
			return fmt.Errorf("%w: %s cannot be negative", ErrInvalidInput, f.name)
		}
	}
	if in.StatesOfOperation < 0 {
		return fmt.Errorf("%w: states of operation cannot be negative", ErrInvalidInput)
	}
	return nil
}

// normalize clamps the answers into a usable range and folds the three
// activities that are themselves Central triggers into the matching flags, so
// that picking "Importer" as the activity cannot disagree with the importer
// checkbox. It returns the notes those corrections deserve.
func (in Input) normalize() (Input, []string) {
	var notes []string
	out := in

	if out.Activity == "" {
		out.Activity = ActivityManufacturer
		notes = append(notes, "No activity was given, so the questionnaire assumed a manufacturer.")
	}
	if unit, err := ParseCapacityUnit(string(out.CapacityUnit)); err == nil {
		out.CapacityUnit = unit
	} else {
		out.CapacityUnit = UnitKgPerDay
		notes = append(notes, "The capacity unit was not recognised, so it was read as kg/day.")
	}
	for _, f := range []*float64{
		&out.AnnualTurnoverINR, &out.ProductionCapacityPerDay,
		&out.DairyLitresPerDay, &out.DairyMilkSolidsMTPerYear,
	} {
		if *f < 0 {
			*f = 0
		}
	}
	if out.StatesOfOperation < 1 {
		out.StatesOfOperation = 1
	}

	switch out.Activity {
	case ActivityImporter:
		if !out.IsImporter {
			out.IsImporter = true
			notes = append(notes, "The activity is importing, so the importer trigger was applied.")
		}
	case ActivityExporter:
		if !out.IsExporter {
			out.IsExporter = true
			notes = append(notes, "The activity is exporting, so the exporter trigger was applied.")
		}
	case ActivityEcommerce:
		if !out.IsEcommerce {
			out.IsEcommerce = true
			notes = append(notes, "The activity is e-commerce, so the e-commerce trigger was applied.")
		}
	}

	if out.DairyMilkSolidsMTPerYear > 0 && out.DairyLitresPerDay == 0 {
		notes = append(notes,
			"Milk solids a year were declared without a litres-a-day figure. Enter the litres a day as well, because the petty food business operator ceiling is written in litres a day and the solids figure only decides the step up to a Central Licence.")
	}

	if !out.Activity.Makes() && out.ProductionCapacityPerDay > 0 {
		notes = append(notes, fmt.Sprintf(
			"Production capacity is normally declared only by manufacturing and processing units; %s was read for a %s.",
			out.capacityLabel(), strings.ToLower(out.Activity.Label())))
	}
	return out, notes
}

// capacityMTPerDay converts the declared capacity to metric tonnes per day.
func (in Input) capacityMTPerDay() float64 {
	if in.CapacityUnit == UnitMTPerDay {
		return in.ProductionCapacityPerDay
	}
	return in.ProductionCapacityPerDay / KgPerMetricTonne
}

// capacityKgPerDay converts the declared capacity to kilogrammes per day.
func (in Input) capacityKgPerDay() float64 {
	if in.CapacityUnit == UnitMTPerDay {
		return in.ProductionCapacityPerDay * KgPerMetricTonne
	}
	return in.ProductionCapacityPerDay
}

func (in Input) capacityLabel() string {
	return FormatNumber(in.ProductionCapacityPerDay) + " " + string(in.CapacityUnit)
}

// FeeOption is the total indicative fee for one choice of validity.
type FeeOption struct {
	Years    int `json:"years"`
	TotalINR int `json:"totalINR"`
}

// Decision is the full answer: the tier, why, what to file and what it costs.
type Decision struct {
	Tier      LicenceTier `json:"tier"`
	TierName  string      `json:"tierName"`
	Form      string      `json:"form"`
	Authority string      `json:"authority"`

	// Reasons is the reasoning chain, in the order the rules were applied.
	Reasons []string `json:"reasons"`
	// CentralTriggers lists only the Central Licence triggers that fired.
	CentralTriggers []string `json:"centralTriggers"`

	Checklist []ChecklistItem `json:"checklist"`

	FeePerYearINR int         `json:"feePerYearINR"`
	FeeBasis      string      `json:"feeBasis"`
	FeeByYears    []FeeOption `json:"feeByYears"`

	ValidityMinYears int `json:"validityMinYears"`
	ValidityMaxYears int `json:"validityMaxYears"`

	// Notes carries the honesty caveats and any correction made to the input.
	Notes []string `json:"notes"`
}

// Tier decides the licence tier using the published indicative figures.
func Tier(in Input) Decision { return TierWith(DefaultParams(), in) }

// TierWith decides the licence tier against a caller-supplied figure set, for
// when the authority revises a threshold or a fee.
//
// The three rules, in the order they are applied:
//
//  1. Any one Central Licence trigger means a Central Licence.
//  2. Otherwise a petty food business operator — turnover, food capacity and
//     milk handling all at or below the petty ceilings — needs only a Basic
//     Registration.
//  3. Otherwise a State Licence.
func TierWith(p Params, in Input) Decision {
	norm, notes := in.normalize()

	capMT := norm.capacityMTPerDay()
	capKg := norm.capacityKgPerDay()

	triggers := centralTriggers(p, norm, capMT)

	d := Decision{
		ValidityMinYears: p.ValidityMinYears,
		ValidityMaxYears: p.ValidityMaxYears,
		CentralTriggers:  triggers,
	}

	if len(triggers) > 0 {
		d.Tier = TierCentralLicence
		d.Reasons = append(d.Reasons,
			"Step 1 — the Central Licence triggers are checked first, because any single one of them decides the tier.")
		d.Reasons = append(d.Reasons, triggers...)
		d.Reasons = append(d.Reasons,
			"Each trigger above is on its own enough, so the turnover and capacity tests for the two lower tiers are not reached.")
	} else {
		d.Reasons = append(d.Reasons,
			"Step 1 — none of the Central Licence triggers apply.")
		d.Reasons = append(d.Reasons, centralCleared(p, norm, capMT)...)

		turnoverPetty := norm.AnnualTurnoverINR <= p.PettyFBOTurnoverCeilingINR
		capacityPetty := capKg <= p.PettyFBOCapacityKgPerDay
		dairyPetty := norm.DairyLitresPerDay <= p.DairyBasicCeilingLitresPerDay

		d.Reasons = append(d.Reasons,
			"Step 2 — the petty food business operator test, which all three measures must pass.")

		if turnoverPetty {
			d.Reasons = append(d.Reasons, fmt.Sprintf(
				"Annual turnover %s is at or below the %s petty food business operator ceiling.",
				FormatINR(norm.AnnualTurnoverINR), FormatINR(p.PettyFBOTurnoverCeilingINR)))
		} else {
			d.Reasons = append(d.Reasons, fmt.Sprintf(
				"Annual turnover %s is above the %s petty food business operator ceiling, so Basic Registration does not apply.",
				FormatINR(norm.AnnualTurnoverINR), FormatINR(p.PettyFBOTurnoverCeilingINR)))
		}

		if capacityPetty {
			d.Reasons = append(d.Reasons, fmt.Sprintf(
				"Food capacity %s kg/day is at or below the %s kg/day petty food business operator ceiling.",
				FormatNumber(capKg), FormatNumber(p.PettyFBOCapacityKgPerDay)))
		} else {
			d.Reasons = append(d.Reasons, fmt.Sprintf(
				"Food capacity %s kg/day is above the %s kg/day petty food business operator ceiling, so Basic Registration does not apply.",
				FormatNumber(capKg), FormatNumber(p.PettyFBOCapacityKgPerDay)))
		}

		if dairyPetty {
			d.Reasons = append(d.Reasons, fmt.Sprintf(
				"Milk handled %s litres/day is at or below the %s litres/day petty food business operator ceiling.",
				FormatNumber(norm.DairyLitresPerDay), FormatNumber(p.DairyBasicCeilingLitresPerDay)))
		} else {
			d.Reasons = append(d.Reasons, fmt.Sprintf(
				"Milk handled %s litres/day falls in the %s to %s litres/day band, which is a State Licence.",
				FormatNumber(norm.DairyLitresPerDay),
				FormatNumber(p.DairyBasicCeilingLitresPerDay),
				FormatNumber(p.StateCeilingDairyLitresPerDay)))
		}

		if turnoverPetty && capacityPetty && dairyPetty {
			d.Tier = TierBasicRegistration
			d.Reasons = append(d.Reasons,
				"All three petty measures pass, so this is a petty food business operator and a Basic Registration is enough.")
		} else {
			d.Tier = TierStateLicence
			d.Reasons = append(d.Reasons,
				"At least one petty measure is exceeded while every Central trigger is clear, so the tier between them applies: a State Licence.")
		}
	}

	d.TierName = d.Tier.Name()
	d.Form = d.Tier.Form()
	d.Authority = d.Tier.Authority()
	d.FeePerYearINR, d.FeeBasis = feePerYear(p, d.Tier, norm, capMT)
	d.FeeByYears = feeByYears(p, d.FeePerYearINR)
	d.Checklist = ChecklistFor(d.Tier, norm)
	d.Notes = append(notes, tierNotes(d.Tier, norm)...)
	return d
}

// centralTriggers collects every Central Licence trigger that fires. All of
// them are collected rather than the first, because the explanation is the
// point of the tool.
func centralTriggers(p Params, in Input, capMT float64) []string {
	var t []string
	if in.AnnualTurnoverINR > p.StateLicenceTurnoverCeilingINR {
		t = append(t, fmt.Sprintf(
			"Annual turnover %s is above the %s ceiling for a State Licence.",
			FormatINR(in.AnnualTurnoverINR), FormatINR(p.StateLicenceTurnoverCeilingINR)))
	}
	if in.IsImporter {
		t = append(t, "The business imports food articles, and every importer is centrally licensed.")
	}
	if in.IsExporter {
		t = append(t, "The business exports food articles, and exporters are centrally licensed.")
	}
	if in.Is100PctExportOriented {
		t = append(t, "A 100% export oriented unit is centrally licensed whatever its turnover.")
	}
	if in.IsEcommerce {
		t = append(t, "An e-commerce food business operator is centrally licensed.")
	}
	if in.StatesOfOperation >= p.CentralMinStatesOfOperation {
		t = append(t, fmt.Sprintf(
			"The business operates in %d states, and from %d states upward the head office needs a Central Licence.",
			in.StatesOfOperation, p.CentralMinStatesOfOperation))
	}
	if in.OperatesAtAirportOrSeaport {
		t = append(t, "The premises are inside an airport or seaport, which is centrally licensed.")
	}
	if capMT > p.StateCeilingFoodMTPerDay {
		t = append(t, fmt.Sprintf(
			"Food capacity %s MT/day is above the %s MT/day ceiling a State Licence covers.",
			FormatNumber(capMT), FormatNumber(p.StateCeilingFoodMTPerDay)))
	}
	if in.DairyLitresPerDay > p.StateCeilingDairyLitresPerDay {
		t = append(t, fmt.Sprintf(
			"Milk handled %s litres/day is above the %s litres/day ceiling a State Licence covers.",
			FormatNumber(in.DairyLitresPerDay), FormatNumber(p.StateCeilingDairyLitresPerDay)))
	}
	if in.DairyMilkSolidsMTPerYear > p.StateCeilingMilkSolidsMTPerYear {
		t = append(t, fmt.Sprintf(
			"Milk solids %s MT a year is above the %s MT a year ceiling a State Licence covers.",
			FormatNumber(in.DairyMilkSolidsMTPerYear), FormatNumber(p.StateCeilingMilkSolidsMTPerYear)))
	}
	return t
}

// centralCleared explains, trigger by trigger, why no Central Licence is
// needed. It is the other half of the reasoning chain.
func centralCleared(p Params, in Input, capMT float64) []string {
	c := []string{
		fmt.Sprintf("Annual turnover %s is at or below the %s Central Licence threshold.",
			FormatINR(in.AnnualTurnoverINR), FormatINR(p.StateLicenceTurnoverCeilingINR)),
		"Not an importer, not an exporter and not a 100% export oriented unit.",
		"Not an e-commerce food business operator.",
		fmt.Sprintf("Operating in %s, below the %d-state trigger.",
			plural(in.StatesOfOperation, "state"), p.CentralMinStatesOfOperation),
		"The premises are not inside an airport or seaport.",
	}
	if in.ProductionCapacityPerDay > 0 {
		c = append(c, fmt.Sprintf("Food capacity %s MT/day is within the %s MT/day State ceiling.",
			FormatNumber(capMT), FormatNumber(p.StateCeilingFoodMTPerDay)))
	} else {
		c = append(c, "No food manufacturing or processing capacity was declared.")
	}
	if in.DairyLitresPerDay > 0 || in.DairyMilkSolidsMTPerYear > 0 {
		c = append(c, fmt.Sprintf("Milk handled %s litres/day is within the %s litres/day State ceiling.",
			FormatNumber(in.DairyLitresPerDay), FormatNumber(p.StateCeilingDairyLitresPerDay)))
	} else {
		c = append(c, "No milk or milk products were declared.")
	}
	return c
}

// feePerYear returns the indicative government fee for one year of validity
// and the sentence that explains which fee category was used.
func feePerYear(p Params, t LicenceTier, in Input, capMT float64) (int, string) {
	switch t {
	case TierBasicRegistration:
		return p.FeeBasicRegistrationPerYearINR, fmt.Sprintf(
			"Basic Registration, %s a year.", FormatINR(float64(p.FeeBasicRegistrationPerYearINR)))
	case TierCentralLicence:
		return p.FeeCentralLicencePerYearINR, fmt.Sprintf(
			"Central Licence, %s a year at one flat rate.", FormatINR(float64(p.FeeCentralLicencePerYearINR)))
	}

	switch {
	case in.Activity.Makes():
		if capMT > p.StateFeeManufacturerBandMTPerDay || in.DairyLitresPerDay > p.StateFeeDairyBandLitresPerDay {
			return p.FeeStateManufacturerLargePerYearINR, fmt.Sprintf(
				"State Licence, %s a year: a manufacturing or processing unit above %s MT/day of food or %s litres/day of milk.",
				FormatINR(float64(p.FeeStateManufacturerLargePerYearINR)),
				FormatNumber(p.StateFeeManufacturerBandMTPerDay),
				FormatNumber(p.StateFeeDairyBandLitresPerDay))
		}
		return p.FeeStateManufacturerSmallPerYearINR, fmt.Sprintf(
			"State Licence, %s a year: a manufacturing or processing unit at or below %s MT/day of food and %s litres/day of milk.",
			FormatINR(float64(p.FeeStateManufacturerSmallPerYearINR)),
			FormatNumber(p.StateFeeManufacturerBandMTPerDay),
			FormatNumber(p.StateFeeDairyBandLitresPerDay))
	case in.Activity == ActivityRestaurant:
		return p.FeeStateFoodServicePerYearINR, fmt.Sprintf(
			"State Licence, %s a year: the food service category.",
			FormatINR(float64(p.FeeStateFoodServicePerYearINR)))
	default:
		return p.FeeStateOtherFBOPerYearINR, fmt.Sprintf(
			"State Licence, %s a year: the category for any other food business.",
			FormatINR(float64(p.FeeStateOtherFBOPerYearINR)))
	}
}

// feeByYears prices each validity the applicant may choose.
func feeByYears(p Params, perYear int) []FeeOption {
	var out []FeeOption
	for y := p.ValidityMinYears; y <= p.ValidityMaxYears; y++ {
		out = append(out, FeeOption{Years: y, TotalINR: perYear * y})
	}
	return out
}

// tierNotes returns the honesty caveats that belong with every answer, plus
// the ones specific to this tier and activity.
func tierNotes(t LicenceTier, in Input) []string {
	notes := []string{
		"Fees, turnover thresholds and capacity ceilings are indicative and are revised from time to time. Confirm the current figures and your eligibility on the FoSCoS portal before you apply.",
		"This tool works out which tier to apply for and what to carry. It does not file anything on your behalf.",
	}
	switch t {
	case TierBasicRegistration:
		notes = append(notes, "A Basic Registration stops covering you the moment turnover or capacity crosses a petty ceiling; at that point you must move up to a State Licence.")
	case TierStateLicence:
		notes = append(notes, "A State Licence is issued by the state you operate in. A second state, an import or an export turns the requirement into a Central Licence.")
	case TierCentralLicence:
		notes = append(notes, "A Central Licence for the head office does not replace the State Licence each manufacturing or storage premises in a state needs in its own right.")
	}
	if in.Activity == ActivityTrader || in.Activity == ActivityRetailer {
		notes = append(notes, "If you relabel or repack goods made by someone else, you are treated as a manufacturer for licensing and need that manufacturer's no-objection certificate.")
	}
	if in.DairyLitresPerDay > 0 || in.DairyMilkSolidsMTPerYear > 0 {
		notes = append(notes, "Milk and milk products have their own capacity bands, which is why a small dairy can need a State Licence on a turnover that would otherwise be petty.")
	}
	return notes
}
