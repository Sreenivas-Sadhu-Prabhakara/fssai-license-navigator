package fssai

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
)

// DateLayout is the date format used everywhere a date is stored or entered.
const DateLayout = "2006-01-02"

// AlertLevel grades how urgent a renewal is.
type AlertLevel int

// The alert levels, from "nothing to do" to "expired".
const (
	// AlertUnknown is used when an expiry date cannot be read.
	AlertUnknown AlertLevel = iota - 1
	AlertOK
	AlertNotice
	AlertWarning
	AlertCritical
	AlertOverdue
)

// Name returns the display name of the alert level.
func (l AlertLevel) Name() string {
	switch l {
	case AlertOK:
		return "In date"
	case AlertNotice:
		return "Notice"
	case AlertWarning:
		return "Warning"
	case AlertCritical:
		return "Critical"
	case AlertOverdue:
		return "Overdue"
	}
	return "Unknown"
}

func (l AlertLevel) String() string { return l.Name() }

// Slug returns a stable machine-readable key for the alert level.
func (l AlertLevel) Slug() string {
	switch l {
	case AlertOK:
		return "ok"
	case AlertNotice:
		return "notice"
	case AlertWarning:
		return "warning"
	case AlertCritical:
		return "critical"
	case AlertOverdue:
		return "overdue"
	}
	return "unknown"
}

// ErrUnknownAlertLevel reports an alert level slug that names no level.
var ErrUnknownAlertLevel = errors.New("unknown alert level")

// ParseAlertLevel reads an alert level from its slug.
func ParseAlertLevel(s string) (AlertLevel, error) {
	for _, l := range []AlertLevel{
		AlertUnknown, AlertOK, AlertNotice, AlertWarning, AlertCritical, AlertOverdue,
	} {
		if l.Slug() == strings.ToLower(strings.TrimSpace(s)) {
			return l, nil
		}
	}
	return AlertUnknown, fmt.Errorf("%w: %q", ErrUnknownAlertLevel, s)
}

// MarshalJSON writes the alert level as its slug.
func (l AlertLevel) MarshalJSON() ([]byte, error) { return json.Marshal(l.Slug()) }

// UnmarshalJSON reads an alert level written as a slug.
func (l *AlertLevel) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	parsed, err := ParseAlertLevel(s)
	if err != nil {
		return err
	}
	*l = parsed
	return nil
}

// ErrInvalidDate reports a date that is not a real calendar date in
// [DateLayout].
var ErrInvalidDate = errors.New("invalid date")

// ParseDate reads a YYYY-MM-DD date as a calendar date. It rejects dates that
// do not exist, so 2027-02-29 fails while 2028-02-29 is accepted.
func ParseDate(s string) (time.Time, error) {
	t, err := time.Parse(DateLayout, strings.TrimSpace(s))
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: %q is not a real date written as YYYY-MM-DD", ErrInvalidDate, s)
	}
	return t, nil
}

// civil strips a timestamp down to its calendar date at midnight UTC, so that
// day arithmetic is never shifted by a clock time or a daylight-saving change.
func civil(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// dayNumber counts whole days from the Unix epoch to a calendar date. civil
// guarantees an exact multiple of a day, so the division is exact for dates
// before the epoch too.
func dayNumber(t time.Time) int {
	return int(civil(t).Unix() / int64(24*time.Hour/time.Second))
}

// DaysBetween counts calendar days from one date to another, ignoring the time
// of day. It is negative when to is earlier than from. Month lengths and leap
// days are handled by the calendar, not by an approximation.
func DaysBetween(from, to time.Time) int {
	return dayNumber(to) - dayNumber(from)
}

// Renewal is how a single expiry date stands on a given day.
type Renewal struct {
	ExpiryDate    string     `json:"expiryDate"`
	AsOf          string     `json:"asOf"`
	DaysRemaining int        `json:"daysRemaining"`
	Level         AlertLevel `json:"level"`
	LevelName     string     `json:"levelName"`
	Overdue       bool       `json:"overdue"`
	// PenaltyINR is the indicative late fee already accrued on an expired
	// licence, and zero while the licence is still in date.
	PenaltyINR int    `json:"penaltyINR"`
	Message    string `json:"message"`
}

// RenewalStatus reports the days remaining and the alert level for an expiry
// date, using the published indicative thresholds. The licence counts as valid
// through the whole of its expiry date, so the day of expiry is nought days
// remaining and not yet overdue.
func RenewalStatus(expiry, asOf time.Time) Renewal {
	return RenewalStatusWith(DefaultParams(), expiry, asOf)
}

// RenewalStatusWith reports the days remaining and the alert level against a
// caller-supplied threshold set.
func RenewalStatusWith(p Params, expiry, asOf time.Time) Renewal {
	days := DaysBetween(asOf, expiry)
	r := Renewal{
		ExpiryDate:    civil(expiry).Format(DateLayout),
		AsOf:          civil(asOf).Format(DateLayout),
		DaysRemaining: days,
	}
	switch {
	case days < 0:
		r.Level, r.Overdue = AlertOverdue, true
		r.PenaltyINR = -days * p.LateRenewalPenaltyPerDayINR
		r.Message = fmt.Sprintf(
			"Expired %s ago. Operating on an expired licence is an offence, and a late renewal carries an indicative penalty of %s a day, about %s so far.",
			plural(-days, "day"), FormatINR(float64(p.LateRenewalPenaltyPerDayINR)), FormatINR(float64(r.PenaltyINR)))
	case days == 0:
		r.Level = AlertCritical
		r.Message = "Expires today. This is the last day the licence is valid."
	case days <= p.AlertCriticalDays:
		r.Level = AlertCritical
		r.Message = fmt.Sprintf("%s left. File the renewal today.", plural(days, "day"))
	case days <= p.AlertWarningDays:
		r.Level = AlertWarning
		r.Message = fmt.Sprintf("%s left. The renewal is expected to be filed before expiry, so file it now.", plural(days, "day"))
	case days <= p.AlertNoticeDays:
		r.Level = AlertNotice
		r.Message = fmt.Sprintf("%s left. Start gathering the renewal documents.", plural(days, "day"))
	default:
		r.Level = AlertOK
		r.Message = fmt.Sprintf("%s left. Nothing to do yet.", plural(days, "day"))
	}
	r.LevelName = r.Level.Name()
	return r
}

func plural(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", word)
	}
	return fmt.Sprintf("%d %ss", n, word)
}

// Licence is one saved registration or licence being watched for renewal.
type Licence struct {
	ID            string      `json:"id"`
	BusinessName  string      `json:"businessName"`
	LicenceNumber string      `json:"licenceNumber"`
	Tier          LicenceTier `json:"tier"`
	ExpiryDate    string      `json:"expiryDate"`
	Notes         string      `json:"notes"`
}

// ErrInvalidLicence reports a saved licence that cannot be watched.
var ErrInvalidLicence = errors.New("invalid licence")

// Validate checks the fields the watchdog depends on. The licence number is
// optional, but when it is given it has to be the fourteen digits FSSAI
// issues.
func (l Licence) Validate() error {
	if strings.TrimSpace(l.BusinessName) == "" {
		return fmt.Errorf("%w: a business name is needed", ErrInvalidLicence)
	}
	if _, err := ParseDate(l.ExpiryDate); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidLicence, err)
	}
	if l.Tier < TierBasicRegistration || l.Tier > TierCentralLicence {
		return fmt.Errorf("%w: %v", ErrInvalidLicence, ErrUnknownTier)
	}
	if n := NormalizeLicenceNumber(l.LicenceNumber); n != "" {
		if len(n) != LicenceNumberDigits {
			return fmt.Errorf("%w: an FSSAI number is %d digits, got %d",
				ErrInvalidLicence, LicenceNumberDigits, len(n))
		}
		for _, c := range n {
			if c < '0' || c > '9' {
				return fmt.Errorf("%w: an FSSAI number is all digits", ErrInvalidLicence)
			}
		}
	}
	return nil
}

// NormalizeLicenceNumber strips the spaces and hyphens people type into a
// licence number field.
func NormalizeLicenceNumber(s string) string {
	var b strings.Builder
	for _, c := range s {
		if c == ' ' || c == '-' || c == '\t' {
			continue
		}
		b.WriteRune(c)
	}
	return b.String()
}

// WatchItem pairs a saved licence with how its renewal stands today.
type WatchItem struct {
	Licence Licence `json:"licence"`
	Renewal Renewal `json:"renewal"`
}

// Watchlist grades every saved licence as of a given day and sorts the list by
// urgency, using the published indicative thresholds.
func Watchlist(ls []Licence, asOf time.Time) []WatchItem {
	return WatchlistWith(DefaultParams(), ls, asOf)
}

// WatchlistWith grades and sorts the saved licences against a caller-supplied
// threshold set. The most overdue licence comes first, then the soonest to
// expire; a licence whose expiry date cannot be read is listed last so it
// cannot quietly look safe.
func WatchlistWith(p Params, ls []Licence, asOf time.Time) []WatchItem {
	out := make([]WatchItem, 0, len(ls))
	for _, l := range ls {
		item := WatchItem{Licence: l}
		expiry, err := ParseDate(l.ExpiryDate)
		if err != nil {
			item.Renewal = Renewal{
				ExpiryDate: l.ExpiryDate,
				AsOf:       civil(asOf).Format(DateLayout),
				Level:      AlertUnknown,
				LevelName:  AlertUnknown.Name(),
				Message:    "The expiry date could not be read, so this licence is not being watched.",
			}
		} else {
			item.Renewal = RenewalStatusWith(p, expiry, asOf)
		}
		out = append(out, item)
	}
	slices.SortStableFunc(out, func(a, b WatchItem) int {
		au, bu := a.Renewal.Level == AlertUnknown, b.Renewal.Level == AlertUnknown
		if au != bu {
			if au {
				return 1
			}
			return -1
		}
		if c := cmp.Compare(a.Renewal.DaysRemaining, b.Renewal.DaysRemaining); c != 0 {
			return c
		}
		if c := strings.Compare(a.Licence.BusinessName, b.Licence.BusinessName); c != 0 {
			return c
		}
		return strings.Compare(a.Licence.ID, b.Licence.ID)
	})
	return out
}
