package cpu

import (
	"fmt"
	"io"

	"github.com/bohuslavsemenov/tcpu/pkg/tryte"
)

const MemorySize = 19683 // 3^9

type CPU struct {
	R       [9]tryte.Tryte
	PC      tryte.Tryte
	Compare tryte.Trit
	Carry   bool
	ALU     ALU
	Memory  [MemorySize]tryte.Tryte

	Stdin  io.Reader
	Stdout io.Writer
}

// Reset resets the CPU registers to their default state.
func (cpu *CPU) Reset() {
	for i := range cpu.R {
		cpu.R[i] = tryte.FromInt(0)
	}
	cpu.PC = tryte.FromInt(0)
	cpu.Compare = tryte.O
	cpu.Carry = false
}

// Map a balanced ternary Tryte address (from -9841 to 9841) to 0..19682
func (cpu *CPU) tryteToAddr(t tryte.Tryte) (int, error) {
	val := t.ToInt()
	addr := val + 9841
	if addr < 0 || addr >= MemorySize {
		return 0, fmt.Errorf("address out of bounds: %d (tryte value %d)", addr, val)
	}
	return addr, nil
}

// ReadMem reads a Tryte from memory at the given Tryte address.
func (cpu *CPU) ReadMem(addr tryte.Tryte) (tryte.Tryte, error) {
	idx, err := cpu.tryteToAddr(addr)
	if err != nil {
		return tryte.Tryte{}, err
	}
	return cpu.Memory[idx], nil
}

// WriteMem writes a Tryte to memory at the given Tryte address.
func (cpu *CPU) WriteMem(addr tryte.Tryte, val tryte.Tryte) error {
	idx, err := cpu.tryteToAddr(addr)
	if err != nil {
		return err
	}
	cpu.Memory[idx] = val
	return nil
}
