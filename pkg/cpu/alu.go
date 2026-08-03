package cpu

import (
	"fmt"

	"github.com/bohuslavsemenov/tcpu/pkg/tryte"
)

type ALU struct{}

func (alu *ALU) Add(a, b tryte.Tryte) (tryte.Tryte, bool) {
	var result tryte.Tryte
	carry := tryte.O
	for i := range tryte.WordSize {
		sum := int(a[i]) + int(b[i]) + int(carry)

		if sum > 1 {
			result[i] = tryte.Trit(sum - 3)
			carry = tryte.T
		} else if sum < -1 {
			result[i] = tryte.Trit(sum + 3)
			carry = tryte.F
		} else {
			result[i] = tryte.Trit(sum)
			carry = tryte.O
		}
	}

	return result, carry != tryte.O
}

func (alu *ALU) Sub(a, b tryte.Tryte) (tryte.Tryte, bool) {
	bNegative := b.Invert()
	return alu.Add(a, bNegative)
}

// Returns F if a < b, O if a == b, and T if a > b.
func (alu *ALU) Compare(a, b tryte.Tryte) tryte.Trit {
	for i := tryte.WordSize - 1; i >= 0; i-- {
		if a[i] != b[i] {
			if a[i] > b[i] {
				return tryte.T
			} else {
				return tryte.F
			}
		}
	}
	return tryte.O.Invert()
}

func (alu *ALU) Multiply(a, b tryte.Tryte) (tryte.Tryte, bool) {
	var accumulator tryte.Tryte
	overflow := false

	for i := 0; i < tryte.WordSize; i++ {
		bTrit := b[i]
		if bTrit == tryte.O {
			continue
		}

		shifted, shiftOverflow := a.ShiftLeft(i)
		if shiftOverflow {
			overflow = true
		}

		if bTrit == tryte.F {
			shifted = shifted.Invert()
		}

		var addOverflow bool
		accumulator, addOverflow = alu.Add(accumulator, shifted)
		if addOverflow {
			overflow = true
		}
	}

	return accumulator, overflow
}

// And performs a trit-wise minimum operation (Kleene/Lukasiewicz ternary AND)
func (alu *ALU) And(a, b tryte.Tryte) tryte.Tryte {
	var result tryte.Tryte
	for i := range tryte.WordSize {
		if a[i] < b[i] {
			result[i] = a[i]
		} else {
			result[i] = b[i]
		}
	}
	return result
}

// Or performs a trit-wise maximum operation (Kleene/Lukasiewicz ternary OR)
func (alu *ALU) Or(a, b tryte.Tryte) tryte.Tryte {
	var result tryte.Tryte
	for i := range tryte.WordSize {
		if a[i] > b[i] {
			result[i] = a[i]
		} else {
			result[i] = b[i]
		}
	}
	return result
}

// ShiftLeft shifts a Tryte left by n trits
func (alu *ALU) ShiftLeft(a tryte.Tryte, n int) (tryte.Tryte, bool) {
	return a.ShiftLeft(n)
}

// ShiftRight shifts a Tryte right by n trits
func (alu *ALU) ShiftRight(a tryte.Tryte, n int) tryte.Tryte {
	return a.ShiftRight(n)
}

// Div performs balanced ternary division: A = Q * B + R, where -|B|/2 <= R <= |B|/2
func (alu *ALU) Div(a, b tryte.Tryte) (tryte.Tryte, bool, error) {
	aVal := a.ToInt()
	bVal := b.ToInt()
	if bVal == 0 {
		return tryte.Tryte{}, false, fmt.Errorf("division by zero")
	}

	q := aVal / bVal
	r := aVal % bVal

	absB := bVal
	if absB < 0 {
		absB = -absB
	}
	halfB := absB / 2

	if r > halfB {
		if bVal > 0 {
			q++
		} else {
			q--
		}
	} else if r < -halfB {
		if bVal > 0 {
			q--
		} else {
			q++
		}
	}

	overflow := false
	if q > 9841 || q < -9841 {
		overflow = true
	}
	return tryte.FromInt(q), overflow, nil
}

// Mod performs balanced ternary modulo (remainder R from Division)
func (alu *ALU) Mod(a, b tryte.Tryte) (tryte.Tryte, error) {
	aVal := a.ToInt()
	bVal := b.ToInt()
	if bVal == 0 {
		return tryte.Tryte{}, fmt.Errorf("modulo by zero")
	}

	r := aVal % bVal

	absB := bVal
	if absB < 0 {
		absB = -absB
	}
	halfB := absB / 2

	if r > halfB {
		if bVal > 0 {
			r -= bVal
		} else {
			r += bVal
		}
	} else if r < -halfB {
		if bVal > 0 {
			r += bVal
		} else {
			r -= bVal
		}
	}

	return tryte.FromInt(r), nil
}
