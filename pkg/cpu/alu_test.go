package cpu

import (
	"fmt"
	"testing"

	"github.com/bohuslavsemenov/tcpu/pkg/tryte"
)

func TestAdd(t *testing.T) {
	tests := []struct {
		a        tryte.Tryte
		b        tryte.Tryte
		expected tryte.Tryte
		carry    bool
	}{
		{
			a:        tryte.FromInt(0),
			b:        tryte.FromInt(0),
			expected: tryte.FromInt(0),
			carry:    false,
		},
		{
			a:        tryte.FromInt(1),
			b:        tryte.FromInt(1),
			expected: tryte.FromInt(2),
			carry:    false,
		},
		{
			a:        tryte.FromInt(1),
			b:        tryte.FromInt(-1),
			expected: tryte.FromInt(0),
			carry:    false,
		},
		{
			a:        tryte.FromInt(9841),
			b:        tryte.FromInt(9841),
			expected: tryte.FromInt(-1),
			carry:    true,
		},
		{
			a:        tryte.FromInt(-9841),
			b:        tryte.FromInt(-9841),
			expected: tryte.FromInt(1),
			carry:    true,
		},
	}

	var alu ALU

	for _, tt := range tests {
		t.Run(fmt.Sprintf("(a: %v, b: %v)", tt.a, tt.b), func(t *testing.T) {
			got, carry := alu.Add(tt.a, tt.b)

			if got != tt.expected {
				t.Errorf("Add(%v, %v) mismatch:\n  got:  %v %t\n  want: %v %t",
					tt.a, tt.b, got, carry, tt.expected, tt.carry)
			}
		})
	}
}

func TestSub(t *testing.T) {
	tests := []struct {
		a        tryte.Tryte
		b        tryte.Tryte
		expected tryte.Tryte
		carry    bool
	}{
		{
			a:        tryte.FromInt(0),
			b:        tryte.FromInt(0),
			expected: tryte.FromInt(0),
			carry:    false,
		},
		{
			a:        tryte.FromInt(1),
			b:        tryte.FromInt(1),
			expected: tryte.FromInt(0),
			carry:    false,
		},
		{
			a:        tryte.FromInt(-1),
			b:        tryte.FromInt(-1),
			expected: tryte.FromInt(0),
			carry:    false,
		},
		{
			a:        tryte.FromInt(1),
			b:        tryte.FromInt(-1),
			expected: tryte.FromInt(2),
			carry:    false,
		},
		{
			a:        tryte.FromInt(-9841),
			b:        tryte.FromInt(9841),
			expected: tryte.FromInt(1),
			carry:    true,
		},
	}

	var alu ALU

	for _, tt := range tests {
		t.Run(fmt.Sprintf("(a: %v, b: %v)", tt.a, tt.b), func(t *testing.T) {
			got, carry := alu.Sub(tt.a, tt.b)

			if got != tt.expected {
				t.Errorf("Sub(%v, %v) mismatch:\n  got:  %v %t\n  want: %v %t",
					tt.a, tt.b, got, carry, tt.expected, tt.carry)
			}
		})
	}
}

func TestCompare(t *testing.T) {
	tests := []struct {
		a        tryte.Tryte
		b        tryte.Tryte
		expected tryte.Trit
	}{
		{
			a:        tryte.FromInt(0),
			b:        tryte.FromInt(0),
			expected: tryte.O,
		},
		{
			a:        tryte.FromInt(1),
			b:        tryte.FromInt(0),
			expected: tryte.T,
		},
		{
			a:        tryte.FromInt(-1),
			b:        tryte.FromInt(0),
			expected: tryte.F,
		},
		{
			a:        tryte.FromInt(-4),
			b:        tryte.FromInt(1),
			expected: tryte.F,
		},
	}

	var alu ALU

	for _, tt := range tests {
		t.Run(fmt.Sprintf("(a: %v, b: %v)", tt.a, tt.b), func(t *testing.T) {
			got := alu.Compare(tt.a, tt.b)

			if got != tt.expected {
				t.Errorf("Compare(%v, %v) mismatch:\n  got:  %s\n  want: %s",
					tt.a, tt.b, got, tt.expected)
			}
		})
	}
}

func TestMultiply(t *testing.T) {
	var alu ALU

	tests := []struct {
		a        tryte.Tryte
		b        tryte.Tryte
		expected tryte.Tryte
		overflow bool
	}{
		{tryte.FromInt(0), tryte.FromInt(0), tryte.FromInt(0), false},
		{tryte.FromInt(1), tryte.FromInt(1), tryte.FromInt(1), false},
		{tryte.FromInt(1), tryte.FromInt(-1), tryte.FromInt(-1), false},
		{tryte.FromInt(-1), tryte.FromInt(-1), tryte.FromInt(1), false},
		{tryte.FromInt(3), tryte.FromInt(5), tryte.FromInt(15), false},
		{tryte.FromInt(3), tryte.FromInt(-5), tryte.FromInt(-15), false},
		{tryte.FromInt(-3), tryte.FromInt(-5), tryte.FromInt(15), false},
		{tryte.FromInt(9841), tryte.FromInt(0), tryte.FromInt(0), false},
		{tryte.FromInt(9841), tryte.FromInt(1), tryte.FromInt(9841), false},
		{tryte.FromInt(9841), tryte.FromInt(-1), tryte.FromInt(-9841), false},
		{tryte.FromInt(-9841), tryte.FromInt(1), tryte.FromInt(-9841), false},
		{tryte.FromInt(-9841), tryte.FromInt(-1), tryte.FromInt(9841), false},
		{tryte.FromInt(100), tryte.FromInt(98), tryte.FromInt(9800), false},
		{tryte.FromInt(100), tryte.FromInt(-98), tryte.FromInt(-9800), false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d * %d", tt.a.ToInt(), tt.b.ToInt()), func(t *testing.T) {
			got, overflow := alu.Multiply(tt.a, tt.b)
			if got != tt.expected {
				t.Errorf("Multiply(%d, %d) = %d (expected %d)", tt.a.ToInt(), tt.b.ToInt(), got.ToInt(), tt.expected.ToInt())
			}
			if overflow != tt.overflow {
				t.Errorf("Multiply(%d, %d) overflow = %t (expected %t)", tt.a.ToInt(), tt.b.ToInt(), overflow, tt.overflow)
			}
		})
	}

	// Dynamic test cases where we multiply various combinations within non-overflow limits
	for a := -90; a <= 90; a += 10 {
		for b := -90; b <= 90; b += 10 {
			expectedVal := a * b
			expectedTryte := tryte.FromInt(expectedVal)
			t.Run(fmt.Sprintf("dynamic: %d * %d", a, b), func(t *testing.T) {
				got, overflow := alu.Multiply(tryte.FromInt(a), tryte.FromInt(b))
				if got != expectedTryte {
					t.Errorf("Multiply(%d, %d) = %d (expected %d)", a, b, got.ToInt(), expectedVal)
				}
				if overflow {
					t.Errorf("Multiply(%d, %d) reported unexpected overflow", a, b)
				}
			})
		}
	}

	// Test cases that should overflow
	overflowTests := []struct {
		a int
		b int
	}{
		{9841, 2},
		{-9841, 2},
		{9841, -2},
		{-9841, -2},
		{200, 50},
		{-200, 50},
		{200, -50},
		{-200, -50},
	}
	for _, tt := range overflowTests {
		t.Run(fmt.Sprintf("overflow: %d * %d", tt.a, tt.b), func(t *testing.T) {
			_, overflow := alu.Multiply(tryte.FromInt(tt.a), tryte.FromInt(tt.b))
			if !overflow {
				t.Errorf("Multiply(%d, %d) expected overflow but got none", tt.a, tt.b)
			}
		})
	}
}

func TestAnd(t *testing.T) {
	var alu ALU

	a := tryte.Tryte{tryte.T, tryte.O, tryte.F, tryte.T, tryte.O, tryte.F, tryte.T, tryte.O, tryte.F}
	b := tryte.Tryte{tryte.T, tryte.T, tryte.T, tryte.O, tryte.O, tryte.O, tryte.F, tryte.F, tryte.F}
	expected := tryte.Tryte{tryte.T, tryte.O, tryte.F, tryte.O, tryte.O, tryte.F, tryte.F, tryte.F, tryte.F}

	got := alu.And(a, b)
	if got != expected {
		t.Errorf("And(%v, %v) = %v, expected %v", a, b, got, expected)
	}
}

func TestOr(t *testing.T) {
	var alu ALU

	a := tryte.Tryte{tryte.T, tryte.O, tryte.F, tryte.T, tryte.O, tryte.F, tryte.T, tryte.O, tryte.F}
	b := tryte.Tryte{tryte.T, tryte.T, tryte.T, tryte.O, tryte.O, tryte.O, tryte.F, tryte.F, tryte.F}
	expected := tryte.Tryte{tryte.T, tryte.T, tryte.T, tryte.T, tryte.O, tryte.O, tryte.T, tryte.O, tryte.F}

	got := alu.Or(a, b)
	if got != expected {
		t.Errorf("Or(%v, %v) = %v, expected %v", a, b, got, expected)
	}
}

func TestShift(t *testing.T) {
	var alu ALU

	// Test ShiftLeft: 5 shifted left by 2 should be 5 * 9 = 45
	val := tryte.FromInt(5)
	shifted, overflow := alu.ShiftLeft(val, 2)
	if shifted.ToInt() != 45 {
		t.Errorf("ShiftLeft(5, 2) = %d, expected 45", shifted.ToInt())
	}
	if overflow {
		t.Errorf("ShiftLeft(5, 2) reported unexpected overflow")
	}

	// Test ShiftRight: 45 shifted right by 2 should be 45 / 9 = 5
	val2 := tryte.FromInt(45)
	shifted2 := alu.ShiftRight(val2, 2)
	if shifted2.ToInt() != 5 {
		t.Errorf("ShiftRight(45, 2) = %d, expected 5", shifted2.ToInt())
	}
}
