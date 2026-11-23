package fssai

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func day(s string) time.Time {
	t, err := time.Parse(DateLayout, s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestDaysBetween(t *testing.T) {
	tests := []struct {
		name string
		from string
		to   string
		want int
	}{
		{"same day", "2026-11-24", "2026-11-24", 0},
		{"next day", "2026-11-24", "2026-11-25", 1},
		{"yesterday", "2026-11-24", "2026-11-23", -1},
		{"a full week", "2026-11-24", "2026-12-01", 7},
		{"across the month end", "2026-11-24", "2026-12-24", 30},
		{"across the year end", "2026-12-24", "2027-01-24", 31},
		{"thirty-one day month", "2026-01-31", "2026-03-01", 29},
		{"a common year february", "2027-02-28", "2027-03-01", 1},
		{"a leap year february", "2028-02-28", "2028-03-01", 2},
		{"onto the leap day", "2028-02-28", "2028-02-29", 1},
		{"off the leap day", "2028-02-29", "2028-03-01", 1},
		{"a common year", "2027-01-01", "2028-01-01", 365},
		{"a leap year", "2028-01-01", "2029-01-01", 366},
		{"five years of validity", "2026-11-24", "2031-11-24", 1826},
		{"a century year that is not a leap year", "2100-02-28", "2100-03-01", 1},
		{"a four-hundred year leap year", "2000-02-28", "2000-03-01", 2},
		{"before the epoch", "1969-12-01", "1970-01-01", 31},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DaysBetween(day(tt.from), day(tt.to)); got != tt.want {
				t.Errorf("DaysBetween(%s, %s) = %d, want %d", tt.from, tt.to, got, tt.want)
			}
		})
	}
}

func TestDaysBetweenIgnoresTheClock(t *testing.T) {
	kolkata, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		t.Skipf("no timezone database available: %v", err)
	}
	lateNight := time.Date(2026, 11, 24, 23, 55, 0, 0, kolkata)
	earlyMorning := time.Date(2026, 11, 25, 0, 5, 0, 0, kolkata)
	if got := DaysBetween(lateNight, earlyMorning); got != 1 {
		t.Errorf("DaysBetween across midnight = %d, want 1", got)
	}
	if got := DaysBetween(earlyMorning, lateNight); got != -1 {
		t.Errorf("DaysBetween backwards across midnight = %d, want -1", got)
	}
	noon := time.Date(2026, 11, 24, 12, 0, 0, 0, kolkata)
	if got := DaysBetween(lateNight, noon); got != 0 {
		t.Errorf("DaysBetween within one day = %d, want 0", got)
	}
}

func TestRenewalStatusBoundaries(t *testing.T) {
	asOf := day("2026-11-24")
	tests := []struct {
		name        string
		expiry      string
		wantDays    int
		wantLevel   AlertLevel
		wantOverdue bool
		wantPenalty int
	}{
		{"well in date", "2027-06-30", 218, AlertOK, false, 0},
		{"one day past the notice threshold", "2027-02-23", 91, AlertOK, false, 0},
		{"exactly on the notice threshold", "2027-02-22", 90, AlertNotice, false, 0},
		{"one day past the warning threshold", "2026-12-25", 31, AlertNotice, false, 0},
		{"exactly on the warning threshold", "2026-12-24", 30, AlertWarning, false, 0},
		{"one day past the critical threshold", "2026-12-02", 8, AlertWarning, false, 0},
		{"exactly on the critical threshold", "2026-12-01", 7, AlertCritical, false, 0},
		{"tomorrow", "2026-11-25", 1, AlertCritical, false, 0},
		{"expires today, still valid", "2026-11-24", 0, AlertCritical, false, 0},
		{"expired yesterday", "2026-11-23", -1, AlertOverdue, true, 100},
		{"expired a week ago", "2026-11-17", -7, AlertOverdue, true, 700},
		{"expired forty-five days ago", "2026-10-10", -45, AlertOverdue, true, 4500},
		{"expired a year ago", "2025-11-24", -365, AlertOverdue, true, 36500},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := RenewalStatus(day(tt.expiry), asOf)
			if r.DaysRemaining != tt.wantDays {
				t.Errorf("DaysRemaining = %d, want %d", r.DaysRemaining, tt.wantDays)
			}
			if r.Level != tt.wantLevel {
				t.Errorf("Level = %v, want %v", r.Level, tt.wantLevel)
			}
			if r.LevelName != tt.wantLevel.Name() {
				t.Errorf("LevelName = %q, want %q", r.LevelName, tt.wantLevel.Name())
			}
			if r.Overdue != tt.wantOverdue {
				t.Errorf("Overdue = %v, want %v", r.Overdue, tt.wantOverdue)
			}
			if r.PenaltyINR != tt.wantPenalty {
				t.Errorf("PenaltyINR = %d, want %d", r.PenaltyINR, tt.wantPenalty)
			}
			if r.ExpiryDate != tt.expiry {
				t.Errorf("ExpiryDate = %q, want %q", r.ExpiryDate, tt.expiry)
			}
			if r.AsOf != "2026-11-24" {
				t.Errorf("AsOf = %q, want 2026-11-24", r.AsOf)
			}
			if strings.TrimSpace(r.Message) == "" {
				t.Error("Message is empty")
			}
		})
	}
}

func TestRenewalStatusMessageWording(t *testing.T) {
	asOf := day("2026-11-24")
	tests := []struct {
		expiry string
		want   string
	}{
		{"2026-11-25", "1 day left"},
		{"2026-11-26", "2 days left"},
		{"2026-11-24", "Expires today"},
		{"2026-11-23", "Expired 1 day ago"},
		{"2026-11-22", "Expired 2 days ago"},
		{"2026-10-10", "₹4,500 so far"},
	}
	for _, tt := range tests {
		t.Run(tt.expiry, func(t *testing.T) {
			got := RenewalStatus(day(tt.expiry), asOf).Message
			if !strings.Contains(got, tt.want) {
				t.Errorf("message %q does not contain %q", got, tt.want)
			}
		})
	}
}

func TestRenewalStatusAcrossALeapDay(t *testing.T) {
	// The licence expires on the leap day itself.
	r := RenewalStatus(day("2028-02-29"), day("2028-02-01"))
	if r.DaysRemaining != 28 {
		t.Errorf("DaysRemaining = %d, want 28", r.DaysRemaining)
	}
	// The same span in a common year is one day shorter.
	common := RenewalStatus(day("2027-03-01"), day("2027-02-01"))
	if common.DaysRemaining != 28 {
		t.Errorf("DaysRemaining in a common year = %d, want 28", common.DaysRemaining)
	}
	leap := RenewalStatus(day("2028-03-01"), day("2028-02-01"))
	if leap.DaysRemaining != 29 {
		t.Errorf("DaysRemaining in a leap year = %d, want 29", leap.DaysRemaining)
	}
}

func TestRenewalStatusWithOverriddenThresholds(t *testing.T) {
	asOf := day("2026-11-24")
	expiry := day("2026-12-04") // ten days out

	if got := RenewalStatus(expiry, asOf).Level; got != AlertWarning {
		t.Fatalf("with the published thresholds: Level = %v, want Warning", got)
	}

	p := DefaultParams()
	p.AlertCriticalDays = 14
	if got := RenewalStatusWith(p, expiry, asOf).Level; got != AlertCritical {
		t.Errorf("with a fourteen-day critical threshold: Level = %v, want Critical", got)
	}

	p = DefaultParams()
	p.LateRenewalPenaltyPerDayINR = 250
	overdue := RenewalStatusWith(p, day("2026-11-14"), asOf)
	if overdue.PenaltyINR != 2_500 {
		t.Errorf("PenaltyINR = %d, want 2500 at the overridden rate", overdue.PenaltyINR)
	}
}

func TestParseDate(t *testing.T) {
	tests := []struct {
		in      string
		wantErr bool
	}{
		{"2026-11-24", false},
		{" 2026-11-24 ", false},
		{"2028-02-29", false},
		{"2000-02-29", false},
		{"2027-02-29", true},
		{"2100-02-29", true},
		{"2026-13-01", true},
		{"2026-00-10", true},
		{"2026-04-31", true},
		{"2026-11-32", true},
		{"24-11-2026", true},
		{"24/11/2026", true},
		{"2026-11", true},
		{"", true},
		{"tomorrow", true},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := ParseDate(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ParseDate(%q) = %v, want an error", tt.in, got)
				}
				if !errors.Is(err, ErrInvalidDate) {
					t.Errorf("ParseDate(%q) = %v, want it to wrap ErrInvalidDate", tt.in, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseDate(%q) = %v", tt.in, err)
			}
			if got.Format(DateLayout) != strings.TrimSpace(tt.in) {
				t.Errorf("ParseDate(%q) came back as %q", tt.in, got.Format(DateLayout))
			}
		})
	}
}

func TestAlertLevelNamesAndJSON(t *testing.T) {
	tests := []struct {
		level AlertLevel
		name  string
		slug  string
	}{
		{AlertUnknown, "Unknown", "unknown"},
		{AlertOK, "In date", "ok"},
		{AlertNotice, "Notice", "notice"},
		{AlertWarning, "Warning", "warning"},
		{AlertCritical, "Critical", "critical"},
		{AlertOverdue, "Overdue", "overdue"},
	}
	for _, tt := range tests {
		if tt.level.Name() != tt.name || tt.level.String() != tt.name {
			t.Errorf("level %d names itself %q, want %q", tt.level, tt.level.Name(), tt.name)
		}
		if tt.level.Slug() != tt.slug {
			t.Errorf("level %q slug = %q, want %q", tt.name, tt.level.Slug(), tt.slug)
		}
		b, err := json.Marshal(tt.level)
		if err != nil {
			t.Fatalf("Marshal(%v) = %v", tt.level, err)
		}
		if want := `"` + tt.slug + `"`; string(b) != want {
			t.Errorf("Marshal(%v) = %s, want %s", tt.level, b, want)
		}
		var back AlertLevel
		if err := json.Unmarshal(b, &back); err != nil {
			t.Fatalf("Unmarshal(%s) = %v", b, err)
		}
		if back != tt.level {
			t.Errorf("round trip of %v gave %v", tt.level, back)
		}
		if got, err := ParseAlertLevel(strings.ToUpper(tt.slug)); err != nil || got != tt.level {
			t.Errorf("ParseAlertLevel(%q) = (%v, %v), want (%v, nil)", tt.slug, got, err, tt.level)
		}
	}
	if AlertOverdue <= AlertOK {
		t.Error("the alert levels should rank by urgency")
	}
	if _, err := ParseAlertLevel("lapsed"); !errors.Is(err, ErrUnknownAlertLevel) {
		t.Errorf("ParseAlertLevel(\"lapsed\") = %v, want ErrUnknownAlertLevel", err)
	}
	var bad AlertLevel
	if err := json.Unmarshal([]byte(`3`), &bad); err == nil {
		t.Error("Unmarshal of a number should fail")
	}
}

func TestLicenceValidate(t *testing.T) {
	good := Licence{
		ID:            "l1",
		BusinessName:  "Ananda Dairy",
		LicenceNumber: "12345678901234",
		Tier:          TierStateLicence,
		ExpiryDate:    "2027-03-31",
	}
	tests := []struct {
		name    string
		mutate  func(l *Licence)
		wantErr bool
	}{
		{"a complete licence", func(*Licence) {}, false},
		{"no licence number is allowed", func(l *Licence) { l.LicenceNumber = "" }, false},
		{"spaces in the number are stripped", func(l *Licence) { l.LicenceNumber = "1234 5678 9012 34" }, false},
		{"hyphens in the number are stripped", func(l *Licence) { l.LicenceNumber = "12345678-901234" }, false},
		{"no business name", func(l *Licence) { l.BusinessName = "   " }, true},
		{"no expiry date", func(l *Licence) { l.ExpiryDate = "" }, true},
		{"an impossible expiry date", func(l *Licence) { l.ExpiryDate = "2027-02-29" }, true},
		{"a short licence number", func(l *Licence) { l.LicenceNumber = "1234" }, true},
		{"a long licence number", func(l *Licence) { l.LicenceNumber = "123456789012345" }, true},
		{"letters in the licence number", func(l *Licence) { l.LicenceNumber = "1234567890123A" }, true},
		{"a tier out of range", func(l *Licence) { l.Tier = LicenceTier(9) }, true},
		{"a negative tier", func(l *Licence) { l.Tier = LicenceTier(-1) }, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := good
			tt.mutate(&l)
			err := l.Validate()
			if tt.wantErr {
				if err == nil {
					t.Fatal("Validate() = nil, want an error")
				}
				if !errors.Is(err, ErrInvalidLicence) {
					t.Errorf("Validate() = %v, want it to wrap ErrInvalidLicence", err)
				}
				return
			}
			if err != nil {
				t.Errorf("Validate() = %v, want nil", err)
			}
		})
	}
}

func TestNormalizeLicenceNumber(t *testing.T) {
	tests := []struct{ in, want string }{
		{"", ""},
		{"12345678901234", "12345678901234"},
		{"1234 5678 9012 34", "12345678901234"},
		{"12345678-901234", "12345678901234"},
		{"\t123 456", "123456"},
	}
	for _, tt := range tests {
		if got := NormalizeLicenceNumber(tt.in); got != tt.want {
			t.Errorf("NormalizeLicenceNumber(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestWatchlistSortsByUrgency(t *testing.T) {
	asOf := day("2026-11-24")
	licences := []Licence{
		{ID: "a", BusinessName: "Calm Cafe", Tier: TierStateLicence, ExpiryDate: "2027-08-01"},
		{ID: "b", BusinessName: "Overdue Oils", Tier: TierStateLicence, ExpiryDate: "2026-10-01"},
		{ID: "c", BusinessName: "Broken Date", Tier: TierStateLicence, ExpiryDate: "not-a-date"},
		{ID: "d", BusinessName: "Due Tomorrow", Tier: TierBasicRegistration, ExpiryDate: "2026-11-25"},
		{ID: "e", BusinessName: "Long Overdue", Tier: TierCentralLicence, ExpiryDate: "2025-01-01"},
		{ID: "f", BusinessName: "Due In A Month", Tier: TierStateLicence, ExpiryDate: "2026-12-20"},
	}
	got := WatchlistWith(DefaultParams(), licences, asOf)
	if len(got) != len(licences) {
		t.Fatalf("Watchlist returned %d items, want %d", len(got), len(licences))
	}
	wantOrder := []string{"e", "b", "d", "f", "a", "c"}
	for i, want := range wantOrder {
		if got[i].Licence.ID != want {
			t.Errorf("position %d = %q (%s), want %q",
				i, got[i].Licence.ID, got[i].Licence.BusinessName, want)
		}
	}
	if got[len(got)-1].Renewal.Level != AlertUnknown {
		t.Errorf("the unreadable date should be graded Unknown, got %v", got[len(got)-1].Renewal.Level)
	}
	if !strings.Contains(got[len(got)-1].Renewal.Message, "could not be read") {
		t.Errorf("the unreadable date should say so: %q", got[len(got)-1].Renewal.Message)
	}
	if !got[0].Renewal.Overdue || !got[1].Renewal.Overdue {
		t.Error("the two expired licences should come first and be marked overdue")
	}
}

func TestWatchlistTieBreaksByName(t *testing.T) {
	asOf := day("2026-11-24")
	licences := []Licence{
		{ID: "z", BusinessName: "Zephyr Foods", Tier: TierStateLicence, ExpiryDate: "2026-12-01"},
		{ID: "a", BusinessName: "Amrit Foods", Tier: TierStateLicence, ExpiryDate: "2026-12-01"},
		{ID: "m", BusinessName: "Madhu Foods", Tier: TierStateLicence, ExpiryDate: "2026-12-01"},
	}
	got := Watchlist(licences, asOf)
	want := []string{"Amrit Foods", "Madhu Foods", "Zephyr Foods"}
	for i, name := range want {
		if got[i].Licence.BusinessName != name {
			t.Errorf("position %d = %q, want %q", i, got[i].Licence.BusinessName, name)
		}
	}
}

func TestWatchlistWithNoLicences(t *testing.T) {
	got := Watchlist(nil, day("2026-11-24"))
	if got == nil {
		t.Fatal("Watchlist(nil) = nil, want an empty list so it serialises as []")
	}
	if len(got) != 0 {
		t.Errorf("Watchlist(nil) has %d items, want none", len(got))
	}
}

func TestLicenceJSONRoundTrip(t *testing.T) {
	in := Licence{
		ID:            "l1",
		BusinessName:  "Ananda Dairy",
		LicenceNumber: "12345678901234",
		Tier:          TierCentralLicence,
		ExpiryDate:    "2029-02-28",
		Notes:         "Plant licence",
	}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("Marshal() = %v", err)
	}
	if !strings.Contains(string(b), `"tier":"central"`) {
		t.Errorf("stored form %s should write the tier as a readable slug", b)
	}
	var out Licence
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("Unmarshal() = %v", err)
	}
	if out != in {
		t.Errorf("round trip gave %+v, want %+v", out, in)
	}
}
