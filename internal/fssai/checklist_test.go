package fssai

import (
	"encoding/csv"
	"strings"
	"testing"
)

func TestChecklistForTierAndActivity(t *testing.T) {
	tests := []struct {
		name        string
		tier        LicenceTier
		in          Input
		wantPresent []string
		wantAbsent  []string
	}{
		{
			name: "a petty retailer carries only the basics",
			tier: TierBasicRegistration,
			in:   withInput(func(in *Input) { in.Activity = ActivityRetailer }),
			wantPresent: []string{
				"Completed Form A application for a Basic Registration",
				"Photo identity and address proof",
				"Proof of possession of the premises",
			},
			wantAbsent: []string{
				"Blueprint or layout plan",
				"Import Export Code",
				"Food safety management system",
				"No-objection certificate from the local body",
			},
		},
		{
			name: "a state-licensed manufacturer adds the plant documents",
			tier: TierStateLicence,
			in:   withInput(func(in *Input) { in.Activity = ActivityManufacturer }),
			wantPresent: []string{
				"Completed Form B application for a State Licence",
				"Blueprint or layout plan of the premises",
				"List of equipment and machinery",
				"Water test report",
				"Food safety management system",
				"Product recall plan",
				"No-objection certificate from the local body",
			},
			wantAbsent: []string{
				"Import Export Code",
				"Proof of the turnover that puts the business above",
				"Ministry of Commerce",
			},
		},
		{
			name: "a restaurant needs the water report but no plant layout",
			tier: TierStateLicence,
			in:   withInput(func(in *Input) { in.Activity = ActivityRestaurant }),
			wantPresent: []string{
				"Water test report",
				"Food safety management system",
			},
			wantAbsent: []string{
				"Blueprint or layout plan",
				"List of equipment and machinery",
			},
		},
		{
			name: "a trader is told about relabelling",
			tier: TierStateLicence,
			in:   withInput(func(in *Input) { in.Activity = ActivityTrader }),
			wantPresent: []string{
				"relabelled or repacked",
			},
			wantAbsent: []string{
				"Blueprint or layout plan",
				"List of vehicles",
			},
		},
		{
			name: "a transporter lists its vehicles",
			tier: TierStateLicence,
			in:   withInput(func(in *Input) { in.Activity = ActivityTransporter }),
			wantPresent: []string{
				"List of vehicles with registration numbers",
			},
			wantAbsent: []string{
				"Storage capacity and temperature control",
			},
		},
		{
			name: "a cold store describes its chambers",
			tier: TierStateLicence,
			in:   withInput(func(in *Input) { in.Activity = ActivityStorage }),
			wantPresent: []string{
				"Storage capacity and temperature control details",
			},
			wantAbsent: []string{
				"List of vehicles",
			},
		},
		{
			name: "an importer needs the trade code",
			tier: TierCentralLicence,
			in: withInput(func(in *Input) {
				in.Activity = ActivityImporter
				in.IsImporter = true
			}),
			wantPresent: []string{
				"Completed Form B application for a Central Licence",
				"Import Export Code certificate",
				"Proof of the turnover that puts the business above",
				"Product recall plan",
			},
			wantAbsent: []string{
				"Certificate from the Ministry of Commerce",
				"Scheme of operation for the e-commerce platform",
			},
		},
		{
			name: "an export oriented unit needs the commerce certificate",
			tier: TierCentralLicence,
			in: withInput(func(in *Input) {
				in.Activity = ActivityManufacturer
				in.Is100PctExportOriented = true
			}),
			wantPresent: []string{
				"Certificate from the Ministry of Commerce for a 100% export oriented unit",
				"Import Export Code certificate",
			},
			wantAbsent: []string{
				"Clearance for premises inside a customs area",
			},
		},
		{
			name: "an e-commerce operator explains its platform",
			tier: TierCentralLicence,
			in: withInput(func(in *Input) {
				in.Activity = ActivityEcommerce
				in.IsEcommerce = true
			}),
			wantPresent: []string{
				"Scheme of operation for the e-commerce platform",
			},
			wantAbsent: []string{
				"Import Export Code certificate",
			},
		},
		{
			name: "an airport unit needs the customs clearance",
			tier: TierCentralLicence,
			in: withInput(func(in *Input) {
				in.Activity = ActivityRetailer
				in.OperatesAtAirportOrSeaport = true
			}),
			wantPresent: []string{
				"Clearance for premises inside a customs area",
			},
			wantAbsent: []string{
				"Scheme of operation for the e-commerce platform",
			},
		},
		{
			name: "a large dairy declares where its milk comes from",
			tier: TierCentralLicence,
			in: withInput(func(in *Input) {
				in.Activity = ActivityProcessor
				in.DairyLitresPerDay = 60_000
			}),
			wantPresent: []string{
				"Source of raw material for milk, meat or meat products",
			},
			wantAbsent: []string{
				"List of vehicles",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items := ChecklistFor(tt.tier, tt.in)
			if len(items) == 0 {
				t.Fatal("ChecklistFor returned nothing")
			}
			joined := joinDocuments(items)
			for _, want := range tt.wantPresent {
				if !strings.Contains(joined, want) {
					t.Errorf("checklist is missing %q", want)
				}
			}
			for _, unwanted := range tt.wantAbsent {
				if strings.Contains(joined, unwanted) {
					t.Errorf("checklist should not contain %q", unwanted)
				}
			}
		})
	}
}

func TestChecklistStartsWithTheForm(t *testing.T) {
	tests := []struct {
		tier LicenceTier
		want string
	}{
		{TierBasicRegistration, "Completed Form A application for a Basic Registration"},
		{TierStateLicence, "Completed Form B application for a State Licence"},
		{TierCentralLicence, "Completed Form B application for a Central Licence"},
	}
	for _, tt := range tests {
		items := ChecklistFor(tt.tier, base())
		if items[0].Document != tt.want {
			t.Errorf("first item = %q, want %q", items[0].Document, tt.want)
		}
		if !strings.Contains(items[0].Note, "FoSCoS") {
			t.Errorf("first item note = %q, want it to name FoSCoS", items[0].Note)
		}
		if !strings.Contains(items[0].Note, tt.tier.Authority()) {
			t.Errorf("first item note = %q, want it to name the issuing authority", items[0].Note)
		}
	}
}

func TestChecklistItemsAreCompleteAndUnique(t *testing.T) {
	for _, tier := range []LicenceTier{TierBasicRegistration, TierStateLicence, TierCentralLicence} {
		for _, activity := range Activities() {
			in := withInput(func(in *Input) { in.Activity = activity })
			items := ChecklistFor(tier, in)
			seen := map[string]bool{}
			for i, it := range items {
				if strings.TrimSpace(it.Document) == "" {
					t.Errorf("%v/%v item %d has no document", tier, activity, i)
				}
				if strings.TrimSpace(it.AppliesTo) == "" {
					t.Errorf("%v/%v %q has no applies-to", tier, activity, it.Document)
				}
				if strings.TrimSpace(it.Note) == "" {
					t.Errorf("%v/%v %q has no note", tier, activity, it.Document)
				}
				if seen[it.Document] {
					t.Errorf("%v/%v lists %q twice", tier, activity, it.Document)
				}
				seen[it.Document] = true
			}
		}
	}
}

func TestChecklistGrowsWithTheTier(t *testing.T) {
	in := withInput(func(in *Input) { in.Activity = ActivityManufacturer })
	basic := len(ChecklistFor(TierBasicRegistration, in))
	state := len(ChecklistFor(TierStateLicence, in))
	central := len(ChecklistFor(TierCentralLicence, in))
	if !(basic < state && state < central) {
		t.Errorf("checklist lengths = %d/%d/%d, want each tier to ask for more", basic, state, central)
	}
}

func TestChecklistCSVRoundTrip(t *testing.T) {
	d := Tier(withInput(func(in *Input) {
		in.Activity = ActivityManufacturer
		in.AnnualTurnoverINR = 4_000_000
	}))
	out, err := ChecklistCSV(d)
	if err != nil {
		t.Fatalf("ChecklistCSV() = %v", err)
	}

	rows, err := csv.NewReader(strings.NewReader(out)).ReadAll()
	if err != nil {
		t.Fatalf("the exported CSV does not parse: %v", err)
	}
	if want := len(d.Checklist) + 1; len(rows) != want {
		t.Fatalf("CSV has %d rows, want %d (a header plus one row per document)", len(rows), want)
	}
	if got := strings.Join(rows[0], ","); got != strings.Join(checklistCSVHeader, ",") {
		t.Errorf("header = %q, want %q", got, strings.Join(checklistCSVHeader, ","))
	}
	for i, row := range rows[1:] {
		if len(row) != len(checklistCSVHeader) {
			t.Fatalf("row %d has %d fields, want %d", i, len(row), len(checklistCSVHeader))
		}
		if row[0] != d.TierName {
			t.Errorf("row %d tier = %q, want %q", i, row[0], d.TierName)
		}
		if row[1] != d.Form {
			t.Errorf("row %d form = %q, want %q", i, row[1], d.Form)
		}
		if row[2] != d.Checklist[i].Document {
			t.Errorf("row %d document = %q, want %q", i, row[2], d.Checklist[i].Document)
		}
		if row[3] != d.Checklist[i].AppliesTo {
			t.Errorf("row %d applies-to = %q, want %q", i, row[3], d.Checklist[i].AppliesTo)
		}
		if row[4] != d.Checklist[i].Note {
			t.Errorf("row %d note = %q, want %q", i, row[4], d.Checklist[i].Note)
		}
	}
}

func TestChecklistCSVQuotesCommas(t *testing.T) {
	d := Decision{
		TierName:  "State Licence",
		Form:      "Form B",
		Checklist: []ChecklistItem{{Document: `Deed, "signed"`, AppliesTo: "Partnerships, companies", Note: "One\ntwo"}},
	}
	out, err := ChecklistCSV(d)
	if err != nil {
		t.Fatalf("ChecklistCSV() = %v", err)
	}
	rows, err := csv.NewReader(strings.NewReader(out)).ReadAll()
	if err != nil {
		t.Fatalf("the exported CSV does not parse: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("CSV has %d rows, want 2", len(rows))
	}
	if rows[1][2] != `Deed, "signed"` || rows[1][3] != "Partnerships, companies" || rows[1][4] != "One\ntwo" {
		t.Errorf("awkward fields did not survive the round trip: %q", rows[1])
	}
}

func TestDecisionCarriesTheMatchingChecklist(t *testing.T) {
	in := withInput(func(in *Input) {
		in.Activity = ActivityManufacturer
		in.IsImporter = true
	})
	d := Tier(in)
	want := ChecklistFor(d.Tier, in)
	if len(d.Checklist) != len(want) {
		t.Fatalf("Decision.Checklist has %d items, want %d", len(d.Checklist), len(want))
	}
	for i := range want {
		if d.Checklist[i] != want[i] {
			t.Errorf("Checklist[%d] = %+v, want %+v", i, d.Checklist[i], want[i])
		}
	}
}

func joinDocuments(items []ChecklistItem) string {
	var b strings.Builder
	for _, it := range items {
		b.WriteString(it.Document)
		b.WriteString("\n")
		b.WriteString(it.AppliesTo)
		b.WriteString("\n")
		b.WriteString(it.Note)
		b.WriteString("\n")
	}
	return b.String()
}
