package fssai

import (
	"encoding/csv"
	"fmt"
	"io"
	"strings"
)

// ChecklistItem is one document the applicant has to carry, with the class of
// applicant it is asked of and a note on what satisfies it.
type ChecklistItem struct {
	Document  string `json:"document"`
	AppliesTo string `json:"appliesTo"`
	Note      string `json:"note"`
}

// checklistRule is one row of the document table. when decides whether the row
// belongs on this applicant's checklist.
type checklistRule struct {
	document  string
	appliesTo string
	note      string
	when      func(t LicenceTier, in Input) bool
}

// Predicates used by the document table.
func anyLicence(t LicenceTier, _ Input) bool { return t != TierBasicRegistration }
func centralOnly(t LicenceTier, _ Input) bool {
	return t == TierCentralLicence
}
func licenceAnd(f func(in Input) bool) func(LicenceTier, Input) bool {
	return func(t LicenceTier, in Input) bool { return t != TierBasicRegistration && f(in) }
}
func centralAnd(f func(in Input) bool) func(LicenceTier, Input) bool {
	return func(t LicenceTier, in Input) bool { return t == TierCentralLicence && f(in) }
}
func everyone(LicenceTier, Input) bool { return true }

func makesFood(in Input) bool  { return in.Activity.Makes() }
func servesFood(in Input) bool { return in.Activity == ActivityRestaurant }
func resells(in Input) bool {
	return in.Activity == ActivityTrader || in.Activity == ActivityRetailer
}
func handlesDairy(in Input) bool {
	return in.DairyLitresPerDay > 0 || in.DairyMilkSolidsMTPerYear > 0
}
func crossesBorder(in Input) bool {
	return in.IsImporter || in.IsExporter || in.Is100PctExportOriented
}

// checklistTable is the single place the document requirements live. Rows are
// emitted in this order, so the checklist always reads the same way.
var checklistTable = []checklistRule{
	{
		document:  "Passport-size photograph of the applicant",
		appliesTo: "All applicants",
		note:      "A recent colour photograph, uploaded with the application.",
		when:      everyone,
	},
	{
		document:  "Photo identity and address proof of the proprietor, partners or directors",
		appliesTo: "All applicants",
		note:      "Aadhaar, PAN, voter identity card, passport or driving licence.",
		when:      everyone,
	},
	{
		document:  "Proof of possession of the premises",
		appliesTo: "All applicants",
		note:      "Rent agreement, electricity bill, property tax receipt, sale deed, or a no-objection certificate from the owner.",
		when:      everyone,
	},
	{
		document:  "Declaration of the food products to be handled",
		appliesTo: "All applicants",
		note:      "Picked from the food category list on FoSCoS; it has to match what you actually make or sell.",
		when:      everyone,
	},
	{
		document:  "Declaration of the annual turnover of the food business",
		appliesTo: "All applicants",
		note:      "A self-declaration or the last filed return. This is the figure that fixes the tier.",
		when:      everyone,
	},
	{
		document:  "Constitution of the business",
		appliesTo: "Partnerships, companies, societies and trusts",
		note:      "Partnership deed, certificate of incorporation with the memorandum and articles, or the society or trust registration certificate.",
		when:      everyone,
	},
	{
		document:  "List of the directors, partners or the proprietor with address and contact details",
		appliesTo: "All licence applicants",
		note:      "On the applicant's letterhead, signed.",
		when:      anyLicence,
	},
	{
		document:  "Food safety management system plan or certificate",
		appliesTo: "All licence applicants",
		note:      "Sets out the hazards, the controls at each step, and the trained food safety supervisor.",
		when:      anyLicence,
	},
	{
		document:  "Nomination of the person responsible, in Form IX",
		appliesTo: "Companies and bodies corporate",
		note:      "Names the director or manager answerable for compliance.",
		when:      anyLicence,
	},
	{
		document:  "No-objection certificate from the local body or municipality",
		appliesTo: "Where the state or municipality asks for one",
		note:      "This one varies state by state; check what your state's licensing authority asks for.",
		when:      anyLicence,
	},
	{
		document:  "Blueprint or layout plan of the premises",
		appliesTo: "Manufacturing and processing units",
		note:      "Dimensions in metres, with the area allocated operation by operation.",
		when:      licenceAnd(makesFood),
	},
	{
		document:  "List of equipment and machinery with the number of units and installed capacity",
		appliesTo: "Manufacturing and processing units",
		note:      "The installed capacity has to agree with the capacity declared in the application.",
		when:      licenceAnd(makesFood),
	},
	{
		document:  "Water test report from a NABL-accredited or state-approved laboratory",
		appliesTo: "Units where water is an ingredient or touches the food",
		note:      "Chemical and bacteriological analysis, usually expected to be recent.",
		when:      licenceAnd(func(in Input) bool { return makesFood(in) || servesFood(in) }),
	},
	{
		document:  "Analysis report of the product, where a standard prescribes one",
		appliesTo: "Manufacturing and processing units",
		note:      "From a NABL-accredited or FSSAI-notified laboratory.",
		when:      licenceAnd(makesFood),
	},
	{
		document:  "Product recall plan with contact details",
		appliesTo: "Manufacturers, processors and importers",
		note:      "Required where the regulations prescribe one for your products.",
		when:      licenceAnd(func(in Input) bool { return makesFood(in) || in.IsImporter }),
	},
	{
		document:  "No-objection certificate and licence copy of the manufacturer whose goods are relabelled or repacked",
		appliesTo: "Relabellers and repackers",
		note:      "Relabelling or repacking somebody else's goods is licensed as manufacturing.",
		when:      licenceAnd(resells),
	},
	{
		document:  "List of vehicles with registration numbers",
		appliesTo: "Transporters",
		note:      "Every vehicle used to carry food has to be listed, with the temperature control it provides.",
		when:      licenceAnd(func(in Input) bool { return in.Activity == ActivityTransporter }),
	},
	{
		document:  "Storage capacity and temperature control details",
		appliesTo: "Storage, warehouse and cold-store units",
		note:      "Chamber-wise capacity and the temperature band each chamber holds.",
		when:      licenceAnd(func(in Input) bool { return in.Activity == ActivityStorage }),
	},
	{
		document:  "Proof of the turnover that puts the business above the State Licence ceiling",
		appliesTo: "All Central Licence applicants",
		note:      "Audited accounts, a GST return, or a self-declaration of turnover.",
		when:      centralOnly,
	},
	{
		document:  "Import Export Code certificate issued by the Directorate General of Foreign Trade",
		appliesTo: "Importers and exporters",
		note:      "The name and address on the IEC must match the application.",
		when:      centralAnd(crossesBorder),
	},
	{
		document:  "Certificate from the Ministry of Commerce for a 100% export oriented unit",
		appliesTo: "100% export oriented units",
		note:      "Establishes the export-oriented status claimed in the application.",
		when:      centralAnd(func(in Input) bool { return in.Is100PctExportOriented }),
	},
	{
		document:  "Source of raw material for milk, meat or meat products",
		appliesTo: "Dairy, meat and meat processing units",
		note:      "The declared supplier or own farm, with the collection route for milk.",
		when:      centralAnd(func(in Input) bool { return handlesDairy(in) || makesFood(in) }),
	},
	{
		document:  "Pesticide residue report for water used as an ingredient",
		appliesTo: "Packaged drinking water, mineral water and carbonated water units",
		note:      "From a NABL-accredited laboratory.",
		when:      centralAnd(makesFood),
	},
	{
		document:  "Certificate from the Ministry of Tourism",
		appliesTo: "Hotels claiming a star classification",
		note:      "Only where a star classification is claimed for the establishment.",
		when:      centralAnd(servesFood),
	},
	{
		document:  "Scheme of operation for the e-commerce platform",
		appliesTo: "E-commerce food business operators",
		note:      "Lists the sellers onboarded and how each seller's own licence is verified.",
		when:      centralAnd(func(in Input) bool { return in.IsEcommerce }),
	},
	{
		document:  "Clearance for premises inside a customs area",
		appliesTo: "Units inside an airport or seaport",
		note:      "The port health officer or the airport authority has to clear the premises.",
		when:      centralAnd(func(in Input) bool { return in.OperatesAtAirportOrSeaport }),
	},
}

// ChecklistFor returns the documents this applicant has to carry for the given
// tier, starting with the application form itself.
func ChecklistFor(t LicenceTier, in Input) []ChecklistItem {
	norm, _ := in.normalize()

	what := "licence"
	if t == TierBasicRegistration {
		what = "registration"
	}
	out := []ChecklistItem{{
		Document:  fmt.Sprintf("Completed %s application for a %s", t.Form(), t.Name()),
		AppliesTo: "All applicants",
		Note: fmt.Sprintf("Filed online on FoSCoS and submitted to the %s, which issues the %s.",
			t.Authority(), what),
	}}
	for _, r := range checklistTable {
		if r.when(t, norm) {
			out = append(out, ChecklistItem{Document: r.document, AppliesTo: r.appliesTo, Note: r.note})
		}
	}
	return out
}

// checklistCSVHeader names the columns of the exported checklist.
var checklistCSVHeader = []string{"Tier", "Form", "Document", "Applies to", "Note"}

// WriteChecklistCSV writes the decision's checklist as CSV, one document per
// row, with the tier repeated on every row so the file explains itself when it
// is opened in a spreadsheet.
func WriteChecklistCSV(w io.Writer, d Decision) error {
	cw := csv.NewWriter(w)
	if err := cw.Write(checklistCSVHeader); err != nil {
		return err
	}
	for _, it := range d.Checklist {
		if err := cw.Write([]string{d.TierName, d.Form, it.Document, it.AppliesTo, it.Note}); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// ChecklistCSV renders the decision's checklist as a CSV string.
func ChecklistCSV(d Decision) (string, error) {
	var b strings.Builder
	if err := WriteChecklistCSV(&b, d); err != nil {
		return "", err
	}
	return b.String(), nil
}
