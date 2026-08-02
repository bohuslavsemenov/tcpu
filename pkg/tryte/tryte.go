package tryte

import (
	"fmt"
	"math"
	"strings"
)

const WordSize = 9

type Tryte [WordSize]Trit

func FromInt(n int) Tryte {
	maxValue := (int(math.Pow(3, float64(WordSize))) - 1) / 2
	minValue := -maxValue
	if n > maxValue || n < minValue {
		panic(fmt.Sprintf("Number %d cannot be representend with word length %d", n, WordSize))
	}

	var result Tryte
	for i := 0; i < WordSize && n != 0; i++ {
		rem := n % 3
		n /= 3

		switch rem {
		case 2:
			rem = -1
			n++
		case -2:
			rem = 1
			n--
		}

		result[i] = Trit(rem)
	}

	return result
}

func (t Tryte) ToInt() int {
	result := 0
	for i, v := range t {
		result += int(v) * int(math.Pow(3, float64(i)))
	}
	return result
}

func (t Tryte) String() string {
	var sb strings.Builder
	for i := WordSize - 1; i >= 0; i-- {
		sb.WriteString(t[i].String())
	}
	return sb.String()
}

func (t Tryte) IsNegative() bool {
	for i := WordSize - 1; i >= 0; i-- {
		if t[i] != O {
			return t[i] == F
		}
	}
	return false
}

func (t Tryte) Invert() Tryte {
	var result Tryte
	for i := range WordSize {
		result[i] = t[i].Invert()
	}
	return result
}

func (t Tryte) ShiftLeft(n int) (Tryte, bool) {
	var res Tryte
	if n <= 0 {
		return t, false
	}
	if n >= WordSize {
		return res, !t.IsZero()
	}

	overflow := false

	for i := WordSize - n; i < WordSize; i++ {
		if t[i] != O {
			overflow = true
			break
		}
	}

	for i := WordSize - 1; i >= n; i-- {
		res[i] = t[i-n]
	}

	return res, overflow
}

func (t Tryte) ShiftRight(n int) Tryte {
	var res Tryte
	if n <= 0 {
		return t
	}
	if n >= WordSize {
		return res
	}

	for i := 0; i < WordSize-n; i++ {
		res[i] = t[i+n]
	}
	return res
}

func (t Tryte) IsZero() bool {
	return t == Tryte{}
}
