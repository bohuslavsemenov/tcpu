package tryte

type Trit int8

const (
	F Trit = -1
	O Trit = 0
	T Trit = 1
)

func (t Trit) Invert() Trit {
	return -t
}

func (t Trit) String() string {
	switch t {
	case F:
		return "F"
	case T:
		return "T"
	default:
		return "O"
	}
}
