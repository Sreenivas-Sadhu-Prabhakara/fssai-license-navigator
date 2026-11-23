package fssai

import (
	"math"
	"strconv"
	"strings"
)

// FormatINR renders a rupee amount with the Indian digit grouping used on
// official forms: the last three digits, then groups of two, prefixed with the
// rupee sign. The amount is rounded to whole rupees.
//
//	1200000   -> "₹12,00,000"
//	200000000 -> "₹20,00,00,000"
func FormatINR(amount float64) string {
	if math.IsNaN(amount) || math.IsInf(amount, 0) {
		return "₹0"
	}
	n := int64(math.Round(amount))
	sign := ""
	if n < 0 {
		sign = "-"
		n = -n
	}
	return sign + "₹" + groupIndian(strconv.FormatInt(n, 10))
}

// groupIndian inserts commas into a string of digits: three at the end, then
// pairs.
func groupIndian(digits string) string {
	if len(digits) <= 3 {
		return digits
	}
	head, tail := digits[:len(digits)-3], digits[len(digits)-3:]
	var parts []string
	for len(head) > 2 {
		parts = append([]string{head[len(head)-2:]}, parts...)
		head = head[:len(head)-2]
	}
	if head != "" {
		parts = append([]string{head}, parts...)
	}
	return strings.Join(parts, ",") + "," + tail
}

// FormatNumber renders a plain quantity with Indian digit grouping, dropping a
// trailing ".0" so that capacities read as "2" rather than "2.0" while "1.5"
// survives.
//
//	2     -> "2"
//	1.5   -> "1.5"
//	50000 -> "50,000"
func FormatNumber(v float64) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return "0"
	}
	s := strconv.FormatFloat(v, 'f', -1, 64)
	sign := ""
	if strings.HasPrefix(s, "-") {
		sign, s = "-", s[1:]
	}
	whole, fraction, hasFraction := strings.Cut(s, ".")
	out := sign + groupIndian(whole)
	if hasFraction {
		out += "." + fraction
	}
	return out
}
