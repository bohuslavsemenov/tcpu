package cpu

import "github.com/bohuslavsemenov/tcpu/pkg/tryte"

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
