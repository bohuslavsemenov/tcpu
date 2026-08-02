package tryte

import (
	"strconv"
	"testing"
)

func TestFromInt(t *testing.T) {
	tests := []struct {
		number   int
		expected Tryte
	}{
		{number: 0, expected: Tryte{O, O, O, O, O, O, O, O, O}},
		{number: 1, expected: Tryte{T, O, O, O, O, O, O, O, O}},
		{number: 5, expected: Tryte{F, F, T, O, O, O, O, O, O}},
		{number: -5, expected: Tryte{T, T, F, O, O, O, O, O, O}},
		{number: 9841, expected: Tryte{T, T, T, T, T, T, T, T, T}},
		{number: -9841, expected: Tryte{F, F, F, F, F, F, F, F, F}},
	}

	for _, tt := range tests {
		t.Run(strconv.Itoa(tt.number), func(t *testing.T) {
			got := FromInt(tt.number)

			if got != tt.expected {
				t.Errorf("FromInt(%d) array mismatch:\n  got:  %v (%s)\n  want: %v (%s)",
					tt.number, got, got.String(), tt.expected, tt.expected.String())
			}
		})
	}
}

func TestFromInt_PanicOutOfBounds(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("FromInt should have panicked for out-of-bounds input")
		}
	}()

	// THIS line MUST be called after the defer statement
	FromInt(9842)
}

func TestToInt(t *testing.T) {
	tests := []struct {
		tryte    Tryte
		expected int
	}{
		{tryte: Tryte{O, O, O, O, O, O, O, O, O}, expected: 0},
		{tryte: Tryte{T, O, O, O, O, O, O, O, O}, expected: 1},
		{tryte: Tryte{F, F, T, O, O, O, O, O, O}, expected: 5},
		{tryte: Tryte{T, T, F, O, O, O, O, O, O}, expected: -5},
		{tryte: Tryte{T, T, T, T, T, T, T, T, T}, expected: 9841},
		{tryte: Tryte{F, F, F, F, F, F, F, F, F}, expected: -9841},
	}

	for _, tt := range tests {
		t.Run(tt.tryte.String(), func(t *testing.T) {
			got := tt.tryte.ToInt()
			if got != tt.expected {
				t.Errorf("ToInt() mismatch:\n  got:  %d\n  want: %d",
					got, tt.expected)
			}
		})
	}
}

func TestInvert(t *testing.T) {
	tests := []struct {
		tryte    Tryte
		expected Tryte
	}{
		{tryte: Tryte{O, O, O, O, O, O, O, O, O}, expected: Tryte{O, O, O, O, O, O, O, O, O}},
		{tryte: Tryte{T, O, O, O, O, O, O, O, O}, expected: Tryte{F, O, O, O, O, O, O, O, O}},
		{tryte: Tryte{T, T, T, T, T, T, T, T, T}, expected: Tryte{F, F, F, F, F, F, F, F, F}},
		{tryte: Tryte{F, F, F, F, F, F, F, F, F}, expected: Tryte{T, T, T, T, T, T, T, T, T}},
	}

	for _, tt := range tests {
		t.Run(tt.tryte.String(), func(t *testing.T) {
			got := tt.tryte.Invert()
			if got != tt.expected {
				t.Errorf("ToInt() mismatch:\n  got:  %v\n  want: %v",
					got, tt.expected)
			}
		})
	}
}

func TestIsNegative(t *testing.T) {
	tests := []struct {
		tryte    Tryte
		expected bool
	}{
		{tryte: Tryte{O, O, O, O, O, O, O, O, O}, expected: false},
		{tryte: Tryte{T, O, O, O, O, O, O, O, O}, expected: false},
		{tryte: Tryte{F, O, O, O, O, O, O, O, O}, expected: true},
		{tryte: Tryte{T, T, T, T, T, T, T, T, T}, expected: false},
		{tryte: Tryte{F, F, F, F, F, F, F, F, F}, expected: true},
	}

	for _, tt := range tests {
		t.Run(tt.tryte.String(), func(t *testing.T) {
			got := tt.tryte.IsNegative()
			if got != tt.expected {
				t.Errorf("ToInt() mismatch:\n  got:  %t\n  want: %t",
					got, tt.expected)
			}
		})
	}
}

func TestShiftLeft(t *testing.T) {
	tests := []struct {
		name         string
		tryte        Tryte
		n            int
		expected     Tryte
		wantOverflow bool
	}{
		// Basic shifts & zero shifts
		{
			name:         "Zero tryte shift 0",
			tryte:        Tryte{O, O, O, O, O, O, O, O, O},
			n:            0,
			expected:     Tryte{O, O, O, O, O, O, O, O, O},
			wantOverflow: false,
		},
		{
			name:         "Single LSB trit shift 0",
			tryte:        Tryte{T, O, O, O, O, O, O, O, O},
			n:            0,
			expected:     Tryte{T, O, O, O, O, O, O, O, O},
			wantOverflow: false,
		},
		{
			name:         "Single LSB trit shift 1",
			tryte:        Tryte{T, O, O, O, O, O, O, O, O},
			n:            1,
			expected:     Tryte{O, T, O, O, O, O, O, O, O},
			wantOverflow: false,
		},
		{
			name:         "Alternating trits shift 5",
			tryte:        Tryte{T, F, T, F, T, F, T, F, T},
			n:            5,
			expected:     Tryte{O, O, O, O, O, T, F, T, F},
			wantOverflow: true, // Non-zero trits at index 4, 6, 8 were shifted out
		},

		// Boundary & Overflow Cases
		{
			name:         "Shift max by WordSize-1 (no overflow)",
			tryte:        Tryte{T, O, O, O, O, O, O, O, O},
			n:            8,
			expected:     Tryte{O, O, O, O, O, O, O, O, T},
			wantOverflow: false,
		},
		{
			name:         "MSB non-zero shift 1 (overflow)",
			tryte:        Tryte{O, O, O, O, O, O, O, O, F},
			n:            1,
			expected:     Tryte{O, O, O, O, O, O, O, O, O},
			wantOverflow: true,
		},
		{
			name:         "Shift past WordSize (n >= 9)",
			tryte:        Tryte{T, T, T, T, T, T, T, T, T},
			n:            9,
			expected:     Tryte{O, O, O, O, O, O, O, O, O},
			wantOverflow: true,
		},
		{
			name:         "Zero tryte shift past WordSize (no overflow)",
			tryte:        Tryte{O, O, O, O, O, O, O, O, O},
			n:            10,
			expected:     Tryte{O, O, O, O, O, O, O, O, O},
			wantOverflow: false,
		},
		{
			name:         "Negative shift count",
			tryte:        Tryte{T, F, O, O, O, O, O, O, O},
			n:            -1,
			expected:     Tryte{T, F, O, O, O, O, O, O, O},
			wantOverflow: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, overflow := tt.tryte.ShiftLeft(tt.n)
			if got != tt.expected || overflow != tt.wantOverflow {
				t.Errorf("ShiftLeft(%d) mismatch:\n  got:  %v (overflow: %v)\n  want: %v (overflow: %v)",
					tt.n, got, overflow, tt.expected, tt.wantOverflow)
			}
		})
	}
}

func TestShiftRight(t *testing.T) {
	tests := []struct {
		name     string
		tryte    Tryte
		n        int
		expected Tryte
	}{
		// Basic shifts & zero shifts
		{
			name:     "Zero tryte shift 0",
			tryte:    Tryte{O, O, O, O, O, O, O, O, O},
			n:        0,
			expected: Tryte{O, O, O, O, O, O, O, O, O},
		},
		{
			name:     "Single MSB trit shift 0",
			tryte:    Tryte{O, O, O, O, O, O, O, O, T},
			n:        0,
			expected: Tryte{O, O, O, O, O, O, O, O, T},
		},
		{
			name:     "Single MSB trit shift 1",
			tryte:    Tryte{O, O, O, O, O, O, O, O, T},
			n:        1,
			expected: Tryte{O, O, O, O, O, O, O, T, O},
		},
		{
			name:     "Alternating trits shift 4",
			tryte:    Tryte{T, F, T, F, T, F, T, F, T},
			n:        4,
			expected: Tryte{T, F, T, F, T, O, O, O, O},
		},

		// Boundary & Limit Cases
		{
			name:     "MSB shift by WordSize-1 to LSB",
			tryte:    Tryte{O, O, O, O, O, O, O, O, F},
			n:        8,
			expected: Tryte{F, O, O, O, O, O, O, O, O},
		},
		{
			name:     "LSB non-zero shift 1 (discarded)",
			tryte:    Tryte{T, O, O, O, O, O, O, O, O},
			n:        1,
			expected: Tryte{O, O, O, O, O, O, O, O, O},
		},
		{
			name:     "Shift equal to WordSize (n == 9)",
			tryte:    Tryte{T, T, T, T, T, T, T, T, T},
			n:        9,
			expected: Tryte{O, O, O, O, O, O, O, O, O},
		},
		{
			name:     "Shift greater than WordSize (n > 9)",
			tryte:    Tryte{T, F, T, F, T, F, T, F, T},
			n:        15,
			expected: Tryte{O, O, O, O, O, O, O, O, O},
		},
		{
			name:     "Negative shift count",
			tryte:    Tryte{O, O, O, O, O, O, O, F, T},
			n:        -2,
			expected: Tryte{O, O, O, O, O, O, O, F, T},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.tryte.ShiftRight(tt.n)
			if got != tt.expected {
				t.Errorf("ShiftRight(%d) mismatch:\n  got:  %v\n  want: %v",
					tt.n, got, tt.expected)
			}
		})
	}
}
