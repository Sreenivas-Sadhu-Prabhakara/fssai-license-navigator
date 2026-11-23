package fssai

import (
	"math"
	"testing"
)

func TestFormatINR(t *testing.T) {
	tests := []struct {
		name   string
		amount float64
		want   string
	}{
		{"zero", 0, "₹0"},
		{"under a thousand", 999, "₹999"},
		{"basic registration fee", 100, "₹100"},
		{"exactly a thousand", 1000, "₹1,000"},
		{"ten thousand", 10000, "₹10,000"},
		{"one lakh groups in pairs", 100000, "₹1,00,000"},
		{"petty ceiling", 1200000, "₹12,00,000"},
		{"one crore", 10000000, "₹1,00,00,000"},
		{"state licence ceiling", 200000000, "₹20,00,00,000"},
		{"very large", 1234567890, "₹1,23,45,67,890"},
		{"negative", -5000, "-₹5,000"},
		{"rounds up", 1500.6, "₹1,501"},
		{"rounds down", 1500.4, "₹1,500"},
		{"not a number", math.NaN(), "₹0"},
		{"infinite", math.Inf(1), "₹0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatINR(tt.amount); got != tt.want {
				t.Errorf("FormatINR(%v) = %q, want %q", tt.amount, got, tt.want)
			}
		})
	}
}

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		name string
		v    float64
		want string
	}{
		{"zero", 0, "0"},
		{"whole tonnes drop the decimal", 2, "2"},
		{"half tonne keeps it", 1.5, "1.5"},
		{"hundred kilos", 100, "100"},
		{"fifty thousand litres groups", 50000, "50,000"},
		{"two kilos as tonnes", 0.002, "0.002"},
		{"not a number", math.NaN(), "0"},
		{"infinite", math.Inf(-1), "0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatNumber(tt.v); got != tt.want {
				t.Errorf("FormatNumber(%v) = %q, want %q", tt.v, got, tt.want)
			}
		})
	}
}
