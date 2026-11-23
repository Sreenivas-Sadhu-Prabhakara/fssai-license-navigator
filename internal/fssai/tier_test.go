package fssai

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// base is a small single-state trader, the least demanding answer set. Each
// test copies it and changes only the field under test.
func base() Input {
	return Input{
		AnnualTurnoverINR: 500_000,
		Activity:          ActivityTrader,
		CapacityUnit:      UnitKgPerDay,
		StatesOfOperation: 1,
	}
}

func withInput(f func(in *Input)) Input {
	in := base()
	f(&in)
	return in
}

func TestTierThresholds(t *testing.T) {
	tests := []struct {
		name string
		in   Input
		want LicenceTier
	}{
		{
			"petty trader well under every ceiling",
			base(),
			TierBasicRegistration,
		},
		{
			"turnover exactly at the petty ceiling stays petty",
			withInput(func(in *Input) { in.AnnualTurnoverINR = PettyFBOTurnoverCeilingINR }),
			TierBasicRegistration,
		},
		{
			"one rupee over the petty ceiling needs a state licence",
			withInput(func(in *Input) { in.AnnualTurnoverINR = PettyFBOTurnoverCeilingINR + 1 }),
			TierStateLicence,
		},
		{
			"turnover exactly at twenty crore is still a state licence",
			withInput(func(in *Input) { in.AnnualTurnoverINR = StateLicenceTurnoverCeilingINR }),
			TierStateLicence,
		},
		{
			"one rupee over twenty crore needs a central licence",
			withInput(func(in *Input) { in.AnnualTurnoverINR = StateLicenceTurnoverCeilingINR + 1 }),
			TierCentralLicence,
		},
		{
			"zero turnover and nothing else declared is a registration",
			withInput(func(in *Input) { in.AnnualTurnoverINR = 0 }),
			TierBasicRegistration,
		},
		{
			"food capacity exactly at the petty ceiling stays petty",
			withInput(func(in *Input) { in.ProductionCapacityPerDay = PettyFBOCapacityKgPerDay }),
			TierBasicRegistration,
		},
		{
			"one kilo over the petty capacity ceiling needs a state licence",
			withInput(func(in *Input) { in.ProductionCapacityPerDay = PettyFBOCapacityKgPerDay + 1 }),
			TierStateLicence,
		},
		{
			"two tonnes a day in kilos is still a state licence",
			withInput(func(in *Input) {
				in.Activity = ActivityManufacturer
				in.ProductionCapacityPerDay = 2000
				in.CapacityUnit = UnitKgPerDay
			}),
			TierStateLicence,
		},
		{
			"one kilo over two tonnes a day needs a central licence",
			withInput(func(in *Input) {
				in.Activity = ActivityManufacturer
				in.ProductionCapacityPerDay = 2001
				in.CapacityUnit = UnitKgPerDay
			}),
			TierCentralLicence,
		},
		{
			"two tonnes a day declared in tonnes is still a state licence",
			withInput(func(in *Input) {
				in.Activity = ActivityManufacturer
				in.ProductionCapacityPerDay = StateCeilingFoodMTPerDay
				in.CapacityUnit = UnitMTPerDay
			}),
			TierStateLicence,
		},
		{
			"above two tonnes a day declared in tonnes needs a central licence",
			withInput(func(in *Input) {
				in.Activity = ActivityManufacturer
				in.ProductionCapacityPerDay = StateCeilingFoodMTPerDay + 0.5
				in.CapacityUnit = UnitMTPerDay
			}),
			TierCentralLicence,
		},
		{
			"milk exactly at the petty dairy ceiling stays petty",
			withInput(func(in *Input) { in.DairyLitresPerDay = DairyBasicCeilingLitresPerDay }),
			TierBasicRegistration,
		},
		{
			"milk just over the petty dairy ceiling needs a state licence even on a tiny turnover",
			withInput(func(in *Input) { in.DairyLitresPerDay = DairyBasicCeilingLitresPerDay + 1 }),
			TierStateLicence,
		},
		{
			"milk in the middle of the state dairy band is a state licence",
			withInput(func(in *Input) { in.DairyLitresPerDay = 3_000 }),
			TierStateLicence,
		},
		{
			"milk exactly at fifty thousand litres a day is still a state licence",
			withInput(func(in *Input) { in.DairyLitresPerDay = StateCeilingDairyLitresPerDay }),
			TierStateLicence,
		},
		{
			"milk over fifty thousand litres a day needs a central licence",
			withInput(func(in *Input) { in.DairyLitresPerDay = StateCeilingDairyLitresPerDay + 1 }),
			TierCentralLicence,
		},
		{
			"milk solids exactly at the annual ceiling is still a state licence",
			withInput(func(in *Input) {
				in.DairyLitresPerDay = 20_000
				in.DairyMilkSolidsMTPerYear = StateCeilingMilkSolidsMTPerYear
			}),
			TierStateLicence,
		},
		{
			"milk solids over the annual ceiling needs a central licence",
			withInput(func(in *Input) {
				in.DairyLitresPerDay = 20_000
				in.DairyMilkSolidsMTPerYear = StateCeilingMilkSolidsMTPerYear + 1
			}),
			TierCentralLicence,
		},
		{
			"milk solids alone cannot lift a petty operator, because the petty ceiling is written in litres a day",
			withInput(func(in *Input) { in.DairyMilkSolidsMTPerYear = 40 }),
			TierBasicRegistration,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Tier(tt.in).Tier; got != tt.want {
				t.Errorf("Tier() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTierCentralTriggers(t *testing.T) {
	tests := []struct {
		name        string
		in          Input
		wantTrigger string
	}{
		{"importer", withInput(func(in *Input) { in.IsImporter = true }), "imports food articles"},
		{"exporter", withInput(func(in *Input) { in.IsExporter = true }), "exports food articles"},
		{"export oriented unit", withInput(func(in *Input) { in.Is100PctExportOriented = true }), "100% export oriented unit"},
		{"e-commerce operator", withInput(func(in *Input) { in.IsEcommerce = true }), "e-commerce food business operator"},
		{"two states", withInput(func(in *Input) { in.StatesOfOperation = 2 }), "operates in 2 states"},
		{"many states", withInput(func(in *Input) { in.StatesOfOperation = 9 }), "operates in 9 states"},
		{"airport or seaport premises", withInput(func(in *Input) { in.OperatesAtAirportOrSeaport = true }), "inside an airport or seaport"},
		{"turnover over twenty crore", withInput(func(in *Input) { in.AnnualTurnoverINR = 250_000_000 }), "above the ₹20,00,00,000 ceiling"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := Tier(tt.in)
			if d.Tier != TierCentralLicence {
				t.Fatalf("Tier() = %v, want Central Licence", d.Tier)
			}
			if len(d.CentralTriggers) != 1 {
				t.Fatalf("CentralTriggers = %v, want exactly one", d.CentralTriggers)
			}
			if !strings.Contains(d.CentralTriggers[0], tt.wantTrigger) {
				t.Errorf("trigger %q does not mention %q", d.CentralTriggers[0], tt.wantTrigger)
			}
			if !containsSubstring(d.Reasons, tt.wantTrigger) {
				t.Errorf("reasons %v do not mention %q", d.Reasons, tt.wantTrigger)
			}
		})
	}
}

func TestTierReportsEveryTriggerThatFires(t *testing.T) {
	in := withInput(func(in *Input) {
		in.AnnualTurnoverINR = 400_000_000
		in.IsImporter = true
		in.IsExporter = true
		in.StatesOfOperation = 4
		in.OperatesAtAirportOrSeaport = true
	})
	d := Tier(in)
	if d.Tier != TierCentralLicence {
		t.Fatalf("Tier() = %v, want Central Licence", d.Tier)
	}
	if len(d.CentralTriggers) != 5 {
		t.Errorf("CentralTriggers = %d, want 5: %v", len(d.CentralTriggers), d.CentralTriggers)
	}
}

func TestTierSingleStateDoesNotTriggerCentral(t *testing.T) {
	for _, states := range []int{-3, 0, 1} {
		in := withInput(func(in *Input) { in.StatesOfOperation = states })
		d := Tier(in)
		if d.Tier != TierBasicRegistration {
			t.Errorf("StatesOfOperation=%d: Tier() = %v, want Basic Registration (triggers %v)",
				states, d.Tier, d.CentralTriggers)
		}
	}
}

func TestTierFoldsActivityIntoTriggers(t *testing.T) {
	tests := []struct {
		activity Activity
		wantNote string
	}{
		{ActivityImporter, "importer trigger"},
		{ActivityExporter, "exporter trigger"},
		{ActivityEcommerce, "e-commerce trigger"},
	}
	for _, tt := range tests {
		t.Run(string(tt.activity), func(t *testing.T) {
			in := withInput(func(in *Input) { in.Activity = tt.activity })
			d := Tier(in)
			if d.Tier != TierCentralLicence {
				t.Errorf("activity %s: Tier() = %v, want Central Licence", tt.activity, d.Tier)
			}
			if !containsSubstring(d.Notes, tt.wantNote) {
				t.Errorf("notes %v do not mention %q", d.Notes, tt.wantNote)
			}
		})
	}
}

func TestTierClampsNegativeAnswers(t *testing.T) {
	in := Input{
		AnnualTurnoverINR:        -900_000,
		Activity:                 ActivityRetailer,
		ProductionCapacityPerDay: -40,
		DairyLitresPerDay:        -10,
		DairyMilkSolidsMTPerYear: -1,
		StatesOfOperation:        -2,
	}
	d := Tier(in)
	if d.Tier != TierBasicRegistration {
		t.Fatalf("Tier() = %v, want Basic Registration", d.Tier)
	}
	for _, r := range d.Reasons {
		if strings.Contains(r, "-") && strings.Contains(r, "₹-") {
			t.Errorf("a negative amount leaked into a reason: %q", r)
		}
	}
	if !containsSubstring(d.Reasons, "₹0") {
		t.Errorf("reasons %v should report the clamped turnover as ₹0", d.Reasons)
	}
}

func TestTierDefaultsMissingActivityAndUnit(t *testing.T) {
	d := Tier(Input{AnnualTurnoverINR: 100_000})
	if d.Tier != TierBasicRegistration {
		t.Errorf("Tier() = %v, want Basic Registration", d.Tier)
	}
	if !containsSubstring(d.Notes, "assumed a manufacturer") {
		t.Errorf("notes %v should say the activity was assumed", d.Notes)
	}
}

func TestTierNotesCapacityOnANonMakingActivity(t *testing.T) {
	in := withInput(func(in *Input) {
		in.Activity = ActivityRetailer
		in.ProductionCapacityPerDay = 60
	})
	d := Tier(in)
	if !containsSubstring(d.Notes, "normally declared only by manufacturing") {
		t.Errorf("notes %v should flag capacity declared by a retailer", d.Notes)
	}
}

func TestTierReasoningChainIsUsable(t *testing.T) {
	cases := map[string]Input{
		"basic":   base(),
		"state":   withInput(func(in *Input) { in.AnnualTurnoverINR = 5_000_000 }),
		"central": withInput(func(in *Input) { in.IsImporter = true }),
	}
	for name, in := range cases {
		t.Run(name, func(t *testing.T) {
			d := Tier(in)
			if len(d.Reasons) < 3 {
				t.Errorf("Reasons = %v, want at least 3 steps", d.Reasons)
			}
			for i, r := range d.Reasons {
				if strings.TrimSpace(r) == "" {
					t.Errorf("Reasons[%d] is empty", i)
				}
				if !strings.HasSuffix(r, ".") {
					t.Errorf("Reasons[%d] = %q, want a full sentence", i, r)
				}
			}
			if len(d.Notes) < 2 {
				t.Errorf("Notes = %v, want the honesty caveats", d.Notes)
			}
			if !containsSubstring(d.Notes, "FoSCoS") {
				t.Errorf("Notes = %v, want the caveat pointing at FoSCoS", d.Notes)
			}
			if !containsSubstring(d.Notes, "does not file anything") {
				t.Errorf("Notes = %v, want the caveat that nothing is filed", d.Notes)
			}
		})
	}
}

func TestTierStateReasonsNameTheFailedPettyMeasure(t *testing.T) {
	tests := []struct {
		name string
		in   Input
		want string
	}{
		{
			"turnover",
			withInput(func(in *Input) { in.AnnualTurnoverINR = 5_000_000 }),
			"Annual turnover ₹50,00,000 is above the ₹12,00,000 petty food business operator ceiling",
		},
		{
			"capacity",
			withInput(func(in *Input) {
				in.Activity = ActivityManufacturer
				in.ProductionCapacityPerDay = 500
			}),
			"Food capacity 500 kg/day is above the 100 kg/day petty food business operator ceiling",
		},
		{
			"dairy band",
			withInput(func(in *Input) { in.DairyLitresPerDay = 3_000 }),
			"Milk handled 3,000 litres/day falls in the 500 to 50,000 litres/day band",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := Tier(tt.in)
			if d.Tier != TierStateLicence {
				t.Fatalf("Tier() = %v, want State Licence", d.Tier)
			}
			if !containsSubstring(d.Reasons, tt.want) {
				t.Errorf("reasons %v do not contain %q", d.Reasons, tt.want)
			}
		})
	}
}

func TestTierFees(t *testing.T) {
	tests := []struct {
		name        string
		in          Input
		wantTier    LicenceTier
		wantPerYear int
		wantBasis   string
	}{
		{
			"basic registration",
			base(),
			TierBasicRegistration, FeeBasicRegistrationPerYearINR, "Basic Registration",
		},
		{
			"central licence is one flat rate",
			withInput(func(in *Input) { in.IsImporter = true }),
			TierCentralLicence, FeeCentralLicencePerYearINR, "one flat rate",
		},
		{
			"restaurant pays the food service rate",
			withInput(func(in *Input) {
				in.Activity = ActivityRestaurant
				in.AnnualTurnoverINR = 4_000_000
			}),
			TierStateLicence, FeeStateFoodServicePerYearINR, "food service category",
		},
		{
			"trader pays the other food business rate",
			withInput(func(in *Input) { in.AnnualTurnoverINR = 4_000_000 }),
			TierStateLicence, FeeStateOtherFBOPerYearINR, "any other food business",
		},
		{
			"storage unit pays the other food business rate",
			withInput(func(in *Input) {
				in.Activity = ActivityStorage
				in.AnnualTurnoverINR = 4_000_000
			}),
			TierStateLicence, FeeStateOtherFBOPerYearINR, "any other food business",
		},
		{
			"small manufacturer at exactly one tonne a day",
			withInput(func(in *Input) {
				in.Activity = ActivityManufacturer
				in.AnnualTurnoverINR = 4_000_000
				in.ProductionCapacityPerDay = StateFeeManufacturerBandMTPerDay
				in.CapacityUnit = UnitMTPerDay
			}),
			TierStateLicence, FeeStateManufacturerSmallPerYearINR, "at or below",
		},
		{
			"manufacturer just above one tonne a day",
			withInput(func(in *Input) {
				in.Activity = ActivityManufacturer
				in.AnnualTurnoverINR = 4_000_000
				in.ProductionCapacityPerDay = 1_001
				in.CapacityUnit = UnitKgPerDay
			}),
			TierStateLicence, FeeStateManufacturerLargePerYearINR, "above",
		},
		{
			"dairy processor at exactly ten thousand litres a day",
			withInput(func(in *Input) {
				in.Activity = ActivityProcessor
				in.AnnualTurnoverINR = 4_000_000
				in.DairyLitresPerDay = StateFeeDairyBandLitresPerDay
			}),
			TierStateLicence, FeeStateManufacturerSmallPerYearINR, "at or below",
		},
		{
			"dairy processor just above ten thousand litres a day",
			withInput(func(in *Input) {
				in.Activity = ActivityProcessor
				in.AnnualTurnoverINR = 4_000_000
				in.DairyLitresPerDay = StateFeeDairyBandLitresPerDay + 1
			}),
			TierStateLicence, FeeStateManufacturerLargePerYearINR, "above",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := Tier(tt.in)
			if d.Tier != tt.wantTier {
				t.Fatalf("Tier() = %v, want %v", d.Tier, tt.wantTier)
			}
			if d.FeePerYearINR != tt.wantPerYear {
				t.Errorf("FeePerYearINR = %d, want %d", d.FeePerYearINR, tt.wantPerYear)
			}
			if !strings.Contains(d.FeeBasis, tt.wantBasis) {
				t.Errorf("FeeBasis = %q, want it to mention %q", d.FeeBasis, tt.wantBasis)
			}
		})
	}
}

func TestTierFeeByYearsCoversEveryValidity(t *testing.T) {
	d := Tier(base())
	if got, want := len(d.FeeByYears), ValidityMaxYears-ValidityMinYears+1; got != want {
		t.Fatalf("FeeByYears has %d options, want %d", got, want)
	}
	for i, opt := range d.FeeByYears {
		wantYears := ValidityMinYears + i
		if opt.Years != wantYears {
			t.Errorf("FeeByYears[%d].Years = %d, want %d", i, opt.Years, wantYears)
		}
		if want := d.FeePerYearINR * wantYears; opt.TotalINR != want {
			t.Errorf("FeeByYears[%d].TotalINR = %d, want %d", i, opt.TotalINR, want)
		}
	}
	if d.ValidityMinYears != ValidityMinYears || d.ValidityMaxYears != ValidityMaxYears {
		t.Errorf("validity = %d..%d, want %d..%d",
			d.ValidityMinYears, d.ValidityMaxYears, ValidityMinYears, ValidityMaxYears)
	}
}

func TestTierFormAndAuthorityMatchTheTier(t *testing.T) {
	tests := []struct {
		in            Input
		wantForm      string
		wantAuthority string
	}{
		{base(), "Form A", "Registering Authority"},
		{withInput(func(in *Input) { in.AnnualTurnoverINR = 4_000_000 }), "Form B", "State or UT Licensing Authority"},
		{withInput(func(in *Input) { in.IsImporter = true }), "Form B", "Central Licensing Authority"},
	}
	for _, tt := range tests {
		d := Tier(tt.in)
		if d.Form != tt.wantForm {
			t.Errorf("%v: Form = %q, want %q", d.TierName, d.Form, tt.wantForm)
		}
		if !strings.Contains(d.Authority, tt.wantAuthority) {
			t.Errorf("%v: Authority = %q, want it to mention %q", d.TierName, d.Authority, tt.wantAuthority)
		}
		if d.TierName != d.Tier.Name() {
			t.Errorf("TierName = %q, want %q", d.TierName, d.Tier.Name())
		}
	}
}

func TestTierWithOverriddenThresholds(t *testing.T) {
	in := withInput(func(in *Input) { in.AnnualTurnoverINR = 1_500_000 })

	if got := Tier(in).Tier; got != TierStateLicence {
		t.Fatalf("with the published ceiling: Tier() = %v, want State Licence", got)
	}

	p := DefaultParams()
	p.PettyFBOTurnoverCeilingINR = 2_000_000
	raised := TierWith(p, in)
	if raised.Tier != TierBasicRegistration {
		t.Errorf("with the petty ceiling raised to ₹20,00,000: Tier() = %v, want Basic Registration", raised.Tier)
	}
	if !containsSubstring(raised.Reasons, "₹20,00,000") {
		t.Errorf("reasons %v should quote the overridden ceiling", raised.Reasons)
	}

	p = DefaultParams()
	p.FeeCentralLicencePerYearINR = 9_000
	central := TierWith(p, withInput(func(in *Input) { in.IsImporter = true }))
	if central.FeePerYearINR != 9_000 {
		t.Errorf("FeePerYearINR = %d, want the overridden 9000", central.FeePerYearINR)
	}
	if central.FeeByYears[0].TotalINR != 9_000 {
		t.Errorf("FeeByYears[0] = %d, want 9000", central.FeeByYears[0].TotalINR)
	}
}

func TestTierWithOverriddenStatesTrigger(t *testing.T) {
	p := DefaultParams()
	p.CentralMinStatesOfOperation = 3
	in := withInput(func(in *Input) { in.StatesOfOperation = 2 })
	if got := TierWith(p, in).Tier; got != TierBasicRegistration {
		t.Errorf("with a three-state trigger: Tier() = %v, want Basic Registration", got)
	}
	if got := TierWith(p, withInput(func(in *Input) { in.StatesOfOperation = 3 })).Tier; got != TierCentralLicence {
		t.Errorf("at three states: Tier() = %v, want Central Licence", got)
	}
}

func TestInputValidate(t *testing.T) {
	tests := []struct {
		name    string
		in      Input
		wantErr bool
	}{
		{"a complete answer set", base(), false},
		{"empty unit is read as kilos", withInput(func(in *Input) { in.CapacityUnit = "" }), false},
		{"tonnes per day", withInput(func(in *Input) { in.CapacityUnit = UnitMTPerDay }), false},
		{"unknown activity", withInput(func(in *Input) { in.Activity = "bakery" }), true},
		{"empty activity", withInput(func(in *Input) { in.Activity = "" }), true},
		{"unknown unit", withInput(func(in *Input) { in.CapacityUnit = "bags/day" }), true},
		{"negative turnover", withInput(func(in *Input) { in.AnnualTurnoverINR = -1 }), true},
		{"negative capacity", withInput(func(in *Input) { in.ProductionCapacityPerDay = -1 }), true},
		{"negative milk", withInput(func(in *Input) { in.DairyLitresPerDay = -1 }), true},
		{"negative milk solids", withInput(func(in *Input) { in.DairyMilkSolidsMTPerYear = -1 }), true},
		{"negative states", withInput(func(in *Input) { in.StatesOfOperation = -1 }), true},
		{"zero states is allowed and read as one", withInput(func(in *Input) { in.StatesOfOperation = 0 }), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.in.Validate()
			if tt.wantErr {
				if err == nil {
					t.Fatal("Validate() = nil, want an error")
				}
				if !errors.Is(err, ErrInvalidInput) {
					t.Errorf("Validate() = %v, want it to wrap ErrInvalidInput", err)
				}
				return
			}
			if err != nil {
				t.Errorf("Validate() = %v, want nil", err)
			}
		})
	}
}

func TestParseActivity(t *testing.T) {
	for _, a := range Activities() {
		got, err := ParseActivity(strings.ToUpper(string(a)))
		if err != nil {
			t.Errorf("ParseActivity(%q) = %v", a, err)
			continue
		}
		if got != a {
			t.Errorf("ParseActivity(%q) = %q", a, got)
		}
		if a.Label() == "" {
			t.Errorf("activity %q has no label", a)
		}
	}
	if _, err := ParseActivity("sweet shop"); !errors.Is(err, ErrUnknownActivity) {
		t.Errorf("ParseActivity(\"sweet shop\") = %v, want ErrUnknownActivity", err)
	}
	if n := len(Activities()); n != 10 {
		t.Errorf("Activities() has %d entries, want 10", n)
	}
}

func TestActivityMakes(t *testing.T) {
	makes := map[Activity]bool{ActivityManufacturer: true, ActivityProcessor: true}
	for _, a := range Activities() {
		if got := a.Makes(); got != makes[a] {
			t.Errorf("%q.Makes() = %v, want %v", a, got, makes[a])
		}
	}
}

func TestParseCapacityUnit(t *testing.T) {
	tests := []struct {
		in      string
		want    CapacityUnit
		wantErr bool
	}{
		{"kg/day", UnitKgPerDay, false},
		{"KG", UnitKgPerDay, false},
		{"", UnitKgPerDay, false},
		{" kg per day ", UnitKgPerDay, false},
		{"MT/day", UnitMTPerDay, false},
		{"mt", UnitMTPerDay, false},
		{"Tonnes", UnitMTPerDay, false},
		{"litres", "", true},
		{"quintal", "", true},
	}
	for _, tt := range tests {
		got, err := ParseCapacityUnit(tt.in)
		if tt.wantErr {
			if !errors.Is(err, ErrUnknownUnit) {
				t.Errorf("ParseCapacityUnit(%q) = %v, want ErrUnknownUnit", tt.in, err)
			}
			continue
		}
		if err != nil || got != tt.want {
			t.Errorf("ParseCapacityUnit(%q) = (%q, %v), want (%q, nil)", tt.in, got, err, tt.want)
		}
	}
}

func TestCapacityUnitConversion(t *testing.T) {
	tests := []struct {
		capacity float64
		unit     CapacityUnit
		wantMT   float64
		wantKg   float64
	}{
		{0, UnitKgPerDay, 0, 0},
		{100, UnitKgPerDay, 0.1, 100},
		{2000, UnitKgPerDay, 2, 2000},
		{2, UnitMTPerDay, 2, 2000},
		{0.5, UnitMTPerDay, 0.5, 500},
	}
	for _, tt := range tests {
		in := Input{ProductionCapacityPerDay: tt.capacity, CapacityUnit: tt.unit}
		if got := in.capacityMTPerDay(); got != tt.wantMT {
			t.Errorf("%v %s in tonnes = %v, want %v", tt.capacity, tt.unit, got, tt.wantMT)
		}
		if got := in.capacityKgPerDay(); got != tt.wantKg {
			t.Errorf("%v %s in kilos = %v, want %v", tt.capacity, tt.unit, got, tt.wantKg)
		}
	}
}

func TestParseTierAndNames(t *testing.T) {
	tests := []struct {
		in   string
		want LicenceTier
	}{
		{"basic", TierBasicRegistration},
		{"Basic Registration", TierBasicRegistration},
		{"state", TierStateLicence},
		{"State Licence", TierStateLicence},
		{"state license", TierStateLicence},
		{"central", TierCentralLicence},
		{" CENTRAL LICENCE ", TierCentralLicence},
	}
	for _, tt := range tests {
		got, err := ParseTier(tt.in)
		if err != nil || got != tt.want {
			t.Errorf("ParseTier(%q) = (%v, %v), want (%v, nil)", tt.in, got, err, tt.want)
		}
	}
	if _, err := ParseTier("provisional"); !errors.Is(err, ErrUnknownTier) {
		t.Errorf("ParseTier(\"provisional\") = %v, want ErrUnknownTier", err)
	}

	wantNames := map[LicenceTier]string{
		TierBasicRegistration: "Basic Registration",
		TierStateLicence:      "State Licence",
		TierCentralLicence:    "Central Licence",
	}
	for tier, name := range wantNames {
		if tier.Name() != name || tier.String() != name {
			t.Errorf("tier %d names itself %q/%q, want %q", tier, tier.Name(), tier.String(), name)
		}
		if tier.Slug() == "" || tier.Authority() == "" {
			t.Errorf("tier %q is missing a slug or an authority", name)
		}
	}
	if TierBasicRegistration.Form() != "Form A" {
		t.Errorf("Basic Registration uses %q, want Form A", TierBasicRegistration.Form())
	}
	for _, tier := range []LicenceTier{TierStateLicence, TierCentralLicence} {
		if tier.Form() != "Form B" {
			t.Errorf("%v uses %q, want Form B", tier, tier.Form())
		}
	}
}

func TestTierJSONRoundTrip(t *testing.T) {
	for _, tier := range []LicenceTier{TierBasicRegistration, TierStateLicence, TierCentralLicence} {
		b, err := json.Marshal(tier)
		if err != nil {
			t.Fatalf("Marshal(%v) = %v", tier, err)
		}
		if want := `"` + tier.Slug() + `"`; string(b) != want {
			t.Errorf("Marshal(%v) = %s, want %s", tier, b, want)
		}
		var back LicenceTier
		if err := json.Unmarshal(b, &back); err != nil {
			t.Fatalf("Unmarshal(%s) = %v", b, err)
		}
		if back != tier {
			t.Errorf("round trip of %v gave %v", tier, back)
		}
	}
	var bad LicenceTier
	if err := json.Unmarshal([]byte(`"provisional"`), &bad); err == nil {
		t.Error("Unmarshal of an unknown tier slug should fail")
	}
	if err := json.Unmarshal([]byte(`7`), &bad); err == nil {
		t.Error("Unmarshal of a number should fail")
	}
}

func TestParamsValidate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(p *Params)
		wantErr bool
	}{
		{"the published figures", func(*Params) {}, false},
		{"a raised petty ceiling", func(p *Params) { p.PettyFBOTurnoverCeilingINR = 2_500_000 }, false},
		{"zero petty turnover ceiling", func(p *Params) { p.PettyFBOTurnoverCeilingINR = 0 }, true},
		{"petty ceiling above the state ceiling", func(p *Params) { p.PettyFBOTurnoverCeilingINR = 300_000_000 }, true},
		{"zero petty capacity ceiling", func(p *Params) { p.PettyFBOCapacityKgPerDay = 0 }, true},
		{"petty capacity above the state ceiling", func(p *Params) { p.PettyFBOCapacityKgPerDay = 9_000 }, true},
		{"zero state food ceiling", func(p *Params) { p.StateCeilingFoodMTPerDay = 0 }, true},
		{"petty dairy above the state dairy ceiling", func(p *Params) { p.DairyBasicCeilingLitresPerDay = 80_000 }, true},
		{"zero milk solids ceiling", func(p *Params) { p.StateCeilingMilkSolidsMTPerYear = 0 }, true},
		{"a one-state central trigger", func(p *Params) { p.CentralMinStatesOfOperation = 1 }, true},
		{"negative fee", func(p *Params) { p.FeeCentralLicencePerYearINR = -1 }, true},
		{"negative penalty", func(p *Params) { p.LateRenewalPenaltyPerDayINR = -1 }, true},
		{"zero validity", func(p *Params) { p.ValidityMinYears = 0 }, true},
		{"maximum validity below the minimum", func(p *Params) { p.ValidityMaxYears = 0 }, true},
		{"warning further out than notice", func(p *Params) { p.AlertWarningDays = 120 }, true},
		{"critical further out than warning", func(p *Params) { p.AlertCriticalDays = 60 }, true},
		{"negative critical threshold", func(p *Params) { p.AlertCriticalDays = -1 }, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := DefaultParams()
			tt.mutate(&p)
			err := p.Validate()
			if tt.wantErr {
				if err == nil {
					t.Fatal("Validate() = nil, want an error")
				}
				if !errors.Is(err, ErrInvalidParams) {
					t.Errorf("Validate() = %v, want it to wrap ErrInvalidParams", err)
				}
				return
			}
			if err != nil {
				t.Errorf("Validate() = %v, want nil", err)
			}
		})
	}
}

func TestDefaultParamsMirrorTheConstants(t *testing.T) {
	p := DefaultParams()
	if p.PettyFBOTurnoverCeilingINR != PettyFBOTurnoverCeilingINR {
		t.Errorf("petty turnover ceiling = %v, want %v", p.PettyFBOTurnoverCeilingINR, PettyFBOTurnoverCeilingINR)
	}
	if p.StateLicenceTurnoverCeilingINR != StateLicenceTurnoverCeilingINR {
		t.Errorf("state turnover ceiling = %v, want %v", p.StateLicenceTurnoverCeilingINR, StateLicenceTurnoverCeilingINR)
	}
	if p.StateCeilingFoodMTPerDay != StateCeilingFoodMTPerDay {
		t.Errorf("food ceiling = %v, want %v", p.StateCeilingFoodMTPerDay, StateCeilingFoodMTPerDay)
	}
	if p.StateCeilingDairyLitresPerDay != StateCeilingDairyLitresPerDay {
		t.Errorf("dairy ceiling = %v, want %v", p.StateCeilingDairyLitresPerDay, StateCeilingDairyLitresPerDay)
	}
	if p.FeeBasicRegistrationPerYearINR != 100 || p.FeeCentralLicencePerYearINR != 7_500 {
		t.Errorf("fees = %d / %d, want 100 / 7500", p.FeeBasicRegistrationPerYearINR, p.FeeCentralLicencePerYearINR)
	}
	if p.FeeStateFoodServicePerYearINR != 2_000 || p.FeeStateManufacturerLargePerYearINR != 5_000 {
		t.Errorf("state fee band = %d..%d, want 2000..5000",
			p.FeeStateFoodServicePerYearINR, p.FeeStateManufacturerLargePerYearINR)
	}
	if err := p.Validate(); err != nil {
		t.Errorf("the published figures do not validate: %v", err)
	}
}

func containsSubstring(hay []string, needle string) bool {
	for _, s := range hay {
		if strings.Contains(s, needle) {
			return true
		}
	}
	return false
}
