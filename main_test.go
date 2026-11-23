package main

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sreenivas-sadhu-prabhakara/fssai-license-navigator/internal/fssai"
)

func tempStore(t *testing.T) *store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "data.json")
	st, err := openStore(path)
	if err != nil {
		t.Fatalf("openStore() = %v", err)
	}
	return st
}

func TestOpenStoreOnAMissingFile(t *testing.T) {
	st := tempStore(t)
	if len(st.licences()) != 0 {
		t.Errorf("a fresh store has %d licences, want none", len(st.licences()))
	}
	if st.overridden() {
		t.Error("a fresh store should use the published figures")
	}
	if _, err := os.Stat(st.path); !errors.Is(err, os.ErrNotExist) {
		t.Error("opening a store should not create the data file until something is saved")
	}
}

func TestStoreRoundTrip(t *testing.T) {
	st := tempStore(t)

	saved, err := st.addLicence(fssai.Licence{
		BusinessName:  "  Ananda Dairy  ",
		LicenceNumber: "1234 5678 9012 34",
		Tier:          fssai.TierStateLicence,
		ExpiryDate:    "2027-03-31",
		Notes:         " plant ",
	})
	if err != nil {
		t.Fatalf("addLicence() = %v", err)
	}
	if saved.ID == "" {
		t.Error("a saved licence should be given an identifier")
	}
	if saved.BusinessName != "Ananda Dairy" {
		t.Errorf("BusinessName = %q, want it trimmed", saved.BusinessName)
	}
	if saved.LicenceNumber != "12345678901234" {
		t.Errorf("LicenceNumber = %q, want the spaces stripped", saved.LicenceNumber)
	}
	if saved.Notes != "plant" {
		t.Errorf("Notes = %q, want it trimmed", saved.Notes)
	}

	p := fssai.DefaultParams()
	p.FeeCentralLicencePerYearINR = 8_500
	if err := st.setParams(p); err != nil {
		t.Fatalf("setParams() = %v", err)
	}

	again, err := openStore(st.path)
	if err != nil {
		t.Fatalf("reopening the store = %v", err)
	}
	ls := again.licences()
	if len(ls) != 1 || ls[0].ID != saved.ID || ls[0].BusinessName != "Ananda Dairy" {
		t.Fatalf("reloaded licences = %+v, want the one that was saved", ls)
	}
	if ls[0].Tier != fssai.TierStateLicence {
		t.Errorf("reloaded tier = %v, want State Licence", ls[0].Tier)
	}
	if !again.overridden() || again.params().FeeCentralLicencePerYearINR != 8_500 {
		t.Errorf("reloaded fee = %d, want the overridden 8500", again.params().FeeCentralLicencePerYearINR)
	}

	raw, err := os.ReadFile(st.path)
	if err != nil {
		t.Fatalf("reading the data file = %v", err)
	}
	if !strings.Contains(string(raw), `"tier": "state"`) {
		t.Errorf("the data file should store the tier readably:\n%s", raw)
	}
	if !strings.Contains(string(raw), `"version": 1`) {
		t.Errorf("the data file should carry a schema version:\n%s", raw)
	}
}

func TestStoreWritesAtomicallyAndLeavesNoLitter(t *testing.T) {
	st := tempStore(t)
	for _, name := range []string{"One", "Two", "Three"} {
		if _, err := st.addLicence(fssai.Licence{
			BusinessName: name, Tier: fssai.TierBasicRegistration, ExpiryDate: "2027-01-01",
		}); err != nil {
			t.Fatalf("addLicence(%s) = %v", name, err)
		}
	}
	entries, err := os.ReadDir(filepath.Dir(st.path))
	if err != nil {
		t.Fatalf("ReadDir() = %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != filepath.Base(st.path) {
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("the data directory holds %v, want only the data file", names)
	}
	info, err := os.Stat(st.path)
	if err != nil {
		t.Fatalf("Stat() = %v", err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Errorf("data file mode = %04o, want 0600", mode)
	}
}

func TestStoreReplacesALicenceWithTheSameID(t *testing.T) {
	st := tempStore(t)
	first, err := st.addLicence(fssai.Licence{
		BusinessName: "Calm Cafe", Tier: fssai.TierStateLicence, ExpiryDate: "2027-01-01",
	})
	if err != nil {
		t.Fatalf("addLicence() = %v", err)
	}
	first.ExpiryDate = "2029-01-01"
	if _, err := st.addLicence(first); err != nil {
		t.Fatalf("re-saving = %v", err)
	}
	ls := st.licences()
	if len(ls) != 1 {
		t.Fatalf("store holds %d licences, want 1", len(ls))
	}
	if ls[0].ExpiryDate != "2029-01-01" {
		t.Errorf("ExpiryDate = %q, want the updated date", ls[0].ExpiryDate)
	}
}

func TestStoreRejectsAnUnusableLicence(t *testing.T) {
	st := tempStore(t)
	bad := []fssai.Licence{
		{BusinessName: "", Tier: fssai.TierStateLicence, ExpiryDate: "2027-01-01"},
		{BusinessName: "No Date", Tier: fssai.TierStateLicence, ExpiryDate: ""},
		{BusinessName: "Bad Date", Tier: fssai.TierStateLicence, ExpiryDate: "2027-02-29"},
		{BusinessName: "Bad Number", Tier: fssai.TierStateLicence, ExpiryDate: "2027-01-01", LicenceNumber: "123"},
	}
	for _, l := range bad {
		if _, err := st.addLicence(l); !errors.Is(err, fssai.ErrInvalidLicence) {
			t.Errorf("addLicence(%+v) = %v, want ErrInvalidLicence", l, err)
		}
	}
	if len(st.licences()) != 0 {
		t.Error("nothing should have been stored")
	}
}

func TestStoreDeleteLicence(t *testing.T) {
	st := tempStore(t)
	saved, err := st.addLicence(fssai.Licence{
		BusinessName: "Calm Cafe", Tier: fssai.TierStateLicence, ExpiryDate: "2027-01-01",
	})
	if err != nil {
		t.Fatalf("addLicence() = %v", err)
	}
	if err := st.deleteLicence("nope"); !errors.Is(err, errNoSuchLicence) {
		t.Errorf("deleteLicence(\"nope\") = %v, want errNoSuchLicence", err)
	}
	if err := st.deleteLicence(saved.ID); err != nil {
		t.Fatalf("deleteLicence() = %v", err)
	}
	if len(st.licences()) != 0 {
		t.Error("the licence should be gone")
	}
}

func TestStoreRejectsAndDropsImpossibleParams(t *testing.T) {
	st := tempStore(t)
	bad := fssai.DefaultParams()
	bad.PettyFBOTurnoverCeilingINR = 0
	if err := st.setParams(bad); !errors.Is(err, fssai.ErrInvalidParams) {
		t.Fatalf("setParams(bad) = %v, want ErrInvalidParams", err)
	}

	good := fssai.DefaultParams()
	good.AlertCriticalDays = 10
	if err := st.setParams(good); err != nil {
		t.Fatalf("setParams(good) = %v", err)
	}
	if err := st.resetParams(); err != nil {
		t.Fatalf("resetParams() = %v", err)
	}
	if st.overridden() {
		t.Error("after a reset the published figures should be back")
	}

	// A data file whose stored thresholds no longer validate must fall back to
	// the published figures rather than decide a tier with them.
	path := filepath.Join(t.TempDir(), "corrupt.json")
	body := `{"version":1,"params":{"pettyFBOTurnoverCeilingINR":0},"licences":[]}`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("WriteFile() = %v", err)
	}
	loaded, err := openStore(path)
	if err != nil {
		t.Fatalf("openStore() = %v", err)
	}
	if loaded.overridden() {
		t.Error("an invalid stored threshold set should be ignored")
	}
	if loaded.params().PettyFBOTurnoverCeilingINR != fssai.PettyFBOTurnoverCeilingINR {
		t.Error("the published petty ceiling should be in force")
	}
}

func TestOpenStoreOnAnUnreadableFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "broken.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatalf("WriteFile() = %v", err)
	}
	if _, err := openStore(path); err == nil {
		t.Error("openStore should refuse a file it cannot parse rather than discard it")
	}

	empty := filepath.Join(t.TempDir(), "empty.json")
	if err := os.WriteFile(empty, []byte("   \n"), 0o600); err != nil {
		t.Fatalf("WriteFile() = %v", err)
	}
	st, err := openStore(empty)
	if err != nil {
		t.Fatalf("openStore(empty) = %v", err)
	}
	if len(st.licences()) != 0 {
		t.Error("an empty file should load as an empty store")
	}
}

// ------------------------------------------------------------------- the API

func do(t *testing.T, h http.Handler, method, target string, body string) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, target, nil)
	} else {
		req = httptest.NewRequest(method, target, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestServerServesTheEmbeddedUI(t *testing.T) {
	h := newServer(tempStore(t))
	rec := do(t, h, http.MethodGet, "/", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET / = %d, want 200", rec.Code)
	}
	page := rec.Body.String()
	for _, want := range []string{
		"<h1>FSSAI licence tier finder and renewal tracker for food businesses</h1>",
		"Content-Security-Policy",
		"assets/app.js",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("the served page is missing %q", want)
		}
	}
	for _, unwanted := range []string{"<style", "style=\"", "onclick="} {
		if strings.Contains(page, unwanted) {
			t.Errorf("the served page contains %q, which the strict policy forbids", unwanted)
		}
	}
	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("responses should be sent with nosniff")
	}
}

func TestServerTierEndpoint(t *testing.T) {
	h := newServer(tempStore(t))
	tests := []struct {
		name  string
		query string
		want  string
	}{
		{"petty trader", "turnover=500000&activity=trader", "basic"},
		{"mid-sized restaurant", "turnover=5000000&activity=restaurant", "state"},
		{"importer", "turnover=500000&activity=trader&importer=1", "central"},
		{"two states", "turnover=500000&activity=trader&states=2", "central"},
		{"large plant in tonnes", "turnover=5000000&activity=manufacturer&capacity=3&unit=MT/day", "central"},
		{"small plant in kilos", "turnover=5000000&activity=manufacturer&capacity=900&unit=kg/day", "state"},
		{"airport kiosk", "turnover=400000&activity=retailer&port=on", "central"},
		{"no answers at all", "", "basic"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := do(t, h, http.MethodGet, "/api/tier?"+tt.query, "")
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
			}
			var d fssai.Decision
			if err := json.Unmarshal(rec.Body.Bytes(), &d); err != nil {
				t.Fatalf("decoding the decision = %v", err)
			}
			if d.Tier.Slug() != tt.want {
				t.Errorf("tier = %q, want %q", d.Tier.Slug(), tt.want)
			}
			if len(d.Reasons) == 0 {
				t.Error("the decision should carry its reasoning")
			}
			if len(d.Checklist) == 0 {
				t.Error("the decision should carry a checklist")
			}
			if d.FeePerYearINR <= 0 {
				t.Errorf("FeePerYearINR = %d, want a positive fee", d.FeePerYearINR)
			}
		})
	}
}

func TestServerTierRejectsNonsense(t *testing.T) {
	h := newServer(tempStore(t))
	for _, query := range []string{
		"activity=bakery",
		"unit=bags",
		"turnover=lots",
		"states=many",
		"capacity=-5",
	} {
		rec := do(t, h, http.MethodGet, "/api/tier?"+query, "")
		if rec.Code != http.StatusBadRequest {
			t.Errorf("GET /api/tier?%s = %d, want 400", query, rec.Code)
			continue
		}
		var body map[string]string
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body["error"] == "" {
			t.Errorf("GET /api/tier?%s should explain the problem, got %s", query, rec.Body.String())
		}
	}
}

func TestServerChecklistCSV(t *testing.T) {
	h := newServer(tempStore(t))
	rec := do(t, h, http.MethodGet,
		"/api/checklist.csv?turnover=5000000&activity=manufacturer&capacity=500", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/csv") {
		t.Errorf("Content-Type = %q, want text/csv", ct)
	}
	if cd := rec.Header().Get("Content-Disposition"); !strings.Contains(cd, "fssai-state-checklist.csv") {
		t.Errorf("Content-Disposition = %q, want the state checklist filename", cd)
	}
	rows, err := csv.NewReader(strings.NewReader(rec.Body.String())).ReadAll()
	if err != nil {
		t.Fatalf("the exported CSV does not parse: %v", err)
	}
	if len(rows) < 5 {
		t.Fatalf("the export has %d rows, want a header and the documents", len(rows))
	}
	if rows[0][0] != "Tier" || rows[0][2] != "Document" {
		t.Errorf("header = %v, want it to start with the tier and name the document column", rows[0])
	}
	if rows[1][0] != "State Licence" {
		t.Errorf("first row tier = %q, want State Licence", rows[1][0])
	}
}

func TestServerLicenceLifecycle(t *testing.T) {
	st := tempStore(t)
	h := newServer(st)

	rec := do(t, h, http.MethodPost, "/api/licences?asOf=2026-11-24",
		`{"businessName":"Ananda Dairy","licenceNumber":"12345678901234","tier":"state","expiryDate":"2026-12-01","notes":"plant"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var state struct {
		AsOf      string            `json:"asOf"`
		Licences  []fssai.Licence   `json:"licences"`
		Watchlist []fssai.WatchItem `json:"watchlist"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &state); err != nil {
		t.Fatalf("decoding = %v", err)
	}
	if len(state.Licences) != 1 || len(state.Watchlist) != 1 {
		t.Fatalf("after the POST: %d licences, %d watched", len(state.Licences), len(state.Watchlist))
	}
	if state.AsOf != "2026-11-24" {
		t.Errorf("asOf = %q, want the date that was asked for", state.AsOf)
	}
	r := state.Watchlist[0].Renewal
	if r.DaysRemaining != 7 || r.Level != fssai.AlertCritical {
		t.Errorf("renewal = %d days at %v, want 7 days at Critical", r.DaysRemaining, r.Level)
	}

	id := state.Licences[0].ID

	rec = do(t, h, http.MethodGet, "/api/licences?asOf=2026-11-24", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Ananda Dairy") {
		t.Fatalf("GET = %d: %s", rec.Code, rec.Body.String())
	}

	rec = do(t, h, http.MethodDelete, "/api/licences/"+id, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("DELETE = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if len(st.licences()) != 0 {
		t.Error("the licence should be gone from the store")
	}

	rec = do(t, h, http.MethodDelete, "/api/licences/"+id, "")
	if rec.Code != http.StatusNotFound {
		t.Errorf("deleting twice = %d, want 404", rec.Code)
	}
}

func TestServerRejectsABadLicencePost(t *testing.T) {
	h := newServer(tempStore(t))
	tests := []string{
		`{"businessName":"","tier":"state","expiryDate":"2027-01-01"}`,
		`{"businessName":"X","tier":"state","expiryDate":"2027-02-29"}`,
		`{"businessName":"X","tier":"provisional","expiryDate":"2027-01-01"}`,
		`{"businessName":"X","tier":"state","expiryDate":"2027-01-01","surprise":true}`,
		`not json`,
	}
	for _, body := range tests {
		rec := do(t, h, http.MethodPost, "/api/licences", body)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("POST %s = %d, want 400", body, rec.Code)
		}
	}
}

func TestServerBadAsOf(t *testing.T) {
	h := newServer(tempStore(t))
	rec := do(t, h, http.MethodGet, "/api/licences?asOf=31-12-2026", "")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("a malformed asOf = %d, want 400", rec.Code)
	}
}

func TestServerParamsOverrideChangesTheAnswer(t *testing.T) {
	h := newServer(tempStore(t))
	const query = "/api/tier?turnover=1500000&activity=trader"

	tierOf := func() string {
		rec := do(t, h, http.MethodGet, query, "")
		var d fssai.Decision
		if err := json.Unmarshal(rec.Body.Bytes(), &d); err != nil {
			t.Fatalf("decoding = %v", err)
		}
		return d.Tier.Slug()
	}

	if got := tierOf(); got != "state" {
		t.Fatalf("with the published figures: tier = %q, want state", got)
	}

	raised := fssai.DefaultParams()
	raised.PettyFBOTurnoverCeilingINR = 2_000_000
	body, err := json.Marshal(raised)
	if err != nil {
		t.Fatalf("Marshal() = %v", err)
	}
	rec := do(t, h, http.MethodPut, "/api/params", string(body))
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT /api/params = %d: %s", rec.Code, rec.Body.String())
	}
	if got := tierOf(); got != "basic" {
		t.Errorf("with the ceiling raised: tier = %q, want basic", got)
	}

	rec = do(t, h, http.MethodDelete, "/api/params", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("DELETE /api/params = %d", rec.Code)
	}
	if got := tierOf(); got != "state" {
		t.Errorf("after the reset: tier = %q, want state", got)
	}
}

func TestServerRejectsImpossibleParams(t *testing.T) {
	h := newServer(tempStore(t))
	rec := do(t, h, http.MethodPut, "/api/params", `{"pettyFBOTurnoverCeilingINR":0}`)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("PUT with a zero ceiling = %d, want 400: %s", rec.Code, rec.Body.String())
	}
}

func TestServerStateEndpoint(t *testing.T) {
	h := newServer(tempStore(t))
	rec := do(t, h, http.MethodGet, "/api/state?asOf=2026-11-24", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/state = %d", rec.Code)
	}
	var state struct {
		AsOf       string `json:"asOf"`
		DataFile   string `json:"dataFile"`
		Overridden bool   `json:"overridden"`
		Params     struct {
			PettyCeiling float64 `json:"pettyFBOTurnoverCeilingINR"`
		} `json:"params"`
		Activities []struct {
			Key   string `json:"key"`
			Label string `json:"label"`
		} `json:"activities"`
		Units     []map[string]string `json:"units"`
		Tiers     []map[string]string `json:"tiers"`
		Watchlist []fssai.WatchItem   `json:"watchlist"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &state); err != nil {
		t.Fatalf("decoding = %v", err)
	}
	if state.AsOf != "2026-11-24" {
		t.Errorf("asOf = %q", state.AsOf)
	}
	if len(state.Activities) != len(fssai.Activities()) {
		t.Errorf("activities = %d, want %d", len(state.Activities), len(fssai.Activities()))
	}
	for _, a := range state.Activities {
		if a.Key == "" || a.Label == "" {
			t.Errorf("activity option %+v is incomplete", a)
		}
	}
	if len(state.Units) != 2 || len(state.Tiers) != 3 {
		t.Errorf("units = %d, tiers = %d, want 2 and 3", len(state.Units), len(state.Tiers))
	}
	if state.Params.PettyCeiling != fssai.PettyFBOTurnoverCeilingINR {
		t.Errorf("params petty ceiling = %v, want %v", state.Params.PettyCeiling, fssai.PettyFBOTurnoverCeilingINR)
	}
	if state.Overridden {
		t.Error("a fresh store is not overridden")
	}
	if state.DataFile == "" {
		t.Error("the user should be told where their data lives")
	}
	if state.Watchlist == nil {
		t.Error("the watchlist should serialise as an empty list, not null")
	}
}

func TestServerRefusesACrossOriginWrite(t *testing.T) {
	h := newServer(tempStore(t))
	req := httptest.NewRequest(http.MethodPost, "/api/licences",
		strings.NewReader(`{"businessName":"X","tier":"state","expiryDate":"2027-01-01"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://evil.example")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("a cross-origin POST = %d, want 403", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/licences",
		strings.NewReader(`{"businessName":"X","tier":"state","expiryDate":"2027-01-01"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://"+req.Host)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("a same-origin POST = %d, want 200: %s", rec.Code, rec.Body.String())
	}
}

func TestInputFromQuery(t *testing.T) {
	in, err := inputFromQuery(map[string][]string{
		"turnover": {" 2500000 "},
		"activity": {"processor"},
		"capacity": {"1.5"},
		"unit":     {"MT/day"},
		"dairy":    {"400"},
		"solids":   {"12"},
		"states":   {"3"},
		"importer": {"true"},
		"exporter": {"0"},
		"port":     {"yes"},
	})
	if err != nil {
		t.Fatalf("inputFromQuery() = %v", err)
	}
	if in.AnnualTurnoverINR != 2_500_000 {
		t.Errorf("turnover = %v", in.AnnualTurnoverINR)
	}
	if in.Activity != fssai.ActivityProcessor {
		t.Errorf("activity = %v", in.Activity)
	}
	if in.ProductionCapacityPerDay != 1.5 || in.CapacityUnit != fssai.UnitMTPerDay {
		t.Errorf("capacity = %v %v", in.ProductionCapacityPerDay, in.CapacityUnit)
	}
	if in.DairyLitresPerDay != 400 || in.DairyMilkSolidsMTPerYear != 12 {
		t.Errorf("dairy = %v / %v", in.DairyLitresPerDay, in.DairyMilkSolidsMTPerYear)
	}
	if in.StatesOfOperation != 3 {
		t.Errorf("states = %v", in.StatesOfOperation)
	}
	if !in.IsImporter || in.IsExporter || !in.OperatesAtAirportOrSeaport || in.IsEcommerce {
		t.Errorf("flags = %+v", in)
	}

	empty, err := inputFromQuery(map[string][]string{})
	if err != nil {
		t.Fatalf("inputFromQuery(empty) = %v", err)
	}
	if empty.StatesOfOperation != 1 {
		t.Errorf("states default = %d, want 1", empty.StatesOfOperation)
	}
	if empty.Activity != fssai.ActivityManufacturer {
		t.Errorf("activity default = %q", empty.Activity)
	}
	if empty.CapacityUnit != fssai.UnitKgPerDay {
		t.Errorf("unit default = %q", empty.CapacityUnit)
	}
}

func TestAsOfFromDefaultsToToday(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/state", nil)
	got, err := asOfFrom(req)
	if err != nil {
		t.Fatalf("asOfFrom() = %v", err)
	}
	if got.Format(fssai.DateLayout) != time.Now().Format(fssai.DateLayout) {
		t.Errorf("asOfFrom() = %v, want today", got)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/state?asOf=2029-02-28", nil)
	got, err = asOfFrom(req)
	if err != nil {
		t.Fatalf("asOfFrom() = %v", err)
	}
	if got.Format(fssai.DateLayout) != "2029-02-28" {
		t.Errorf("asOfFrom() = %v, want 2029-02-28", got)
	}
}
