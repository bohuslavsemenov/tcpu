package cpu

import (
	"fmt"
	"io"
	"os"

	"github.com/bohuslavsemenov/tcpu/pkg/tryte"
)

const MemorySize = 19683 // 3^9

// Opcode constants
const (
	OpHalt = 0
	OpAdd  = 1
	OpSub  = -1
	OpMul  = 2
	OpCmp  = -2
	OpLd   = 3
	OpSt   = -3
	OpLdi  = 4
	OpJmp  = 5
	OpJeq  = 6
	OpJne  = -6
	OpJlt  = 7
	OpJgt  = -7
)

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

// Memory-mapped I/O addresses
const (
	AddrStdout = 9000
	AddrStdin  = 9001
)

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
	addrVal := addr.ToInt()
	if addrVal == AddrStdin {
		reader := cpu.Stdin
		if reader == nil {
			reader = os.Stdin
		}
		var buf [1]byte
		n, err := reader.Read(buf[:])
		if err != nil || n == 0 {
			return tryte.Tryte{}, fmt.Errorf("failed to read from stdin: %w", err)
		}
		return tryte.FromInt(int(buf[0])), nil
	}

	idx, err := cpu.tryteToAddr(addr)
	if err != nil {
		return tryte.Tryte{}, err
	}
	return cpu.Memory[idx], nil
}

// WriteMem writes a Tryte to memory at the given Tryte address.
func (cpu *CPU) WriteMem(addr tryte.Tryte, val tryte.Tryte) error {
	addrVal := addr.ToInt()
	if addrVal == AddrStdout {
		writer := cpu.Stdout
		if writer == nil {
			writer = os.Stdout
		}
		charByte := byte(val.ToInt())
		if charByte == 10 { // ASCII 10 is '\n'
			// In raw terminal mode, write carriage return '\r' before '\n' to prevent staircase effect
			_, err := writer.Write([]byte{13, 10})
			if err != nil {
				return fmt.Errorf("failed to write to stdout: %w", err)
			}
			return nil
		}
		_, err := writer.Write([]byte{charByte})
		if err != nil {
			return fmt.Errorf("failed to write to stdout: %w", err)
		}
		return nil
	}

	idx, err := cpu.tryteToAddr(addr)
	if err != nil {
		return err
	}
	cpu.Memory[idx] = val
	return nil
}

func (cpu *CPU) getRegIndex(t tryte.Tryte) int {
	val := int(t[0]) + int(t[1])*3 + int(t[2])*9
	return (val%9 + 9) % 9
}

// Step executes a single instruction cycle (Fetch, Decode, Execute).
// Returns (keepRunning, error).
func (cpu *CPU) Step() (bool, error) {
	// 1. Fetch
	inst, err := cpu.ReadMem(cpu.PC)
	if err != nil {
		return false, fmt.Errorf("fetch error: %w", err)
	}

	// Increment PC using ALU
	nextPC, _ := cpu.ALU.Add(cpu.PC, tryte.FromInt(1))
	cpu.PC = nextPC

	// 2. Decode
	// Symmetrical 3-3-3 trit extraction
	srcPart := tryte.Tryte{inst[0], inst[1], inst[2]}
	dstPart := tryte.Tryte{inst[3], inst[4], inst[5]}
	opPart  := tryte.Tryte{inst[6], inst[7], inst[8]}

	op := opPart.ToInt()
	srcIdx := cpu.getRegIndex(srcPart)
	dstIdx := cpu.getRegIndex(dstPart)

	// 3. Execute
	switch op {
	case OpHalt:
		return false, nil

	case OpAdd: // ADD: R[dst] = R[dst] + R[src]
		res, carry := cpu.ALU.Add(cpu.R[dstIdx], cpu.R[srcIdx])
		cpu.R[dstIdx] = res
		cpu.Carry = carry

	case OpSub: // SUB: R[dst] = R[dst] - R[src]
		res, carry := cpu.ALU.Sub(cpu.R[dstIdx], cpu.R[srcIdx])
		cpu.R[dstIdx] = res
		cpu.Carry = carry

	case OpMul: // MUL: R[dst] = R[dst] * R[src]
		res, overflow := cpu.ALU.Multiply(cpu.R[dstIdx], cpu.R[srcIdx])
		cpu.R[dstIdx] = res
		cpu.Carry = overflow

	case OpCmp: // CMP: compare R[dst] and R[src]
		cmpResult := cpu.ALU.Compare(cpu.R[dstIdx], cpu.R[srcIdx])
		cpu.Compare = cmpResult

	case OpLd: // LD: load from address stored in R[src] to R[dst]
		val, err := cpu.ReadMem(cpu.R[srcIdx])
		if err != nil {
			return false, fmt.Errorf("load memory error: %w", err)
		}
		cpu.R[dstIdx] = val

	case OpSt: // ST: store value in R[dst] to address stored in R[src]
		err := cpu.WriteMem(cpu.R[srcIdx], cpu.R[dstIdx])
		if err != nil {
			return false, fmt.Errorf("store memory error: %w", err)
		}

	case OpLdi: // LDI: load immediate value from next PC location
		val, err := cpu.ReadMem(cpu.PC)
		if err != nil {
			return false, fmt.Errorf("load immediate fetch error: %w", err)
		}
		cpu.R[dstIdx] = val
		// Increment PC again since we consumed the immediate
		nextPC, _ := cpu.ALU.Add(cpu.PC, tryte.FromInt(1))
		cpu.PC = nextPC

	case OpJmp: // JMP: PC = R[src]
		cpu.PC = cpu.R[srcIdx]

	case OpJeq: // JEQ: jump if Compare == O
		if cpu.Compare == tryte.O {
			cpu.PC = cpu.R[srcIdx]
		}

	case OpJne: // JNE: jump if Compare != O
		if cpu.Compare != tryte.O {
			cpu.PC = cpu.R[srcIdx]
		}

	case OpJlt: // JLT: jump if Compare == F (<)
		if cpu.Compare == tryte.F {
			cpu.PC = cpu.R[srcIdx]
		}

	case OpJgt: // JGT: jump if Compare == T (>)
		if cpu.Compare == tryte.T {
			cpu.PC = cpu.R[srcIdx]
		}

	default:
		return false, fmt.Errorf("unknown opcode: %d", op)
	}

	return true, nil
}
