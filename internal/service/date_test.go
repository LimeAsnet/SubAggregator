package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseMonthYear_Valid(t *testing.T) {
	tests := []struct {
		input    string
		wantYear int
		wantMon  time.Month
	}{
		{"07-2025", 2025, time.July},
		{"01-2024", 2024, time.January},
		{"12-2099", 2099, time.December},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseMonthYear(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.wantYear, got.Year())
			assert.Equal(t, tt.wantMon, got.Month())
			assert.Equal(t, 1, got.Day())
			assert.Equal(t, time.UTC, got.Location())
		})
	}
}

func TestParseMonthYear_Invalid(t *testing.T) {
	tests := []string{
		"",
		"2025-07",
		"7-2025",
		"13-2025",
		"00-2025",
		"ab-2025",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			_, err := ParseMonthYear(input)
			assert.ErrorIs(t, err, ErrInvalidDate)
		})
	}
}
