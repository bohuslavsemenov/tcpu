package cpu

import (
	"testing"

	"github.com/bohuslavsemenov/tcpu/pkg/tryte"
)

// Helper to construct a 9-trit instruction from (opcode, src, dst) values using base-3 arithmetic
func MakeInst(opVal, srcVal, dstVal int) tryte.Tryte {
	return tryte.FromInt(srcVal + dstVal*27 + opVal*729)
}

func TestCPU_ResetAndMemory(t *testing.T) {
	cpu := &CPU{}
	cpu.Reset()

	for i := range cpu.R {
		if cpu.R[i].ToInt() != 0 {
			t.Errorf("expected R[%d] to be 0, got %d", i, cpu.R[i].ToInt())
		}
	}

	// Test basic memory read/write
	addr := tryte.FromInt(500)
	val := tryte.FromInt(123)
	err := cpu.WriteMem(addr, val)
	if err != nil {
		t.Fatalf("WriteMem error: %v", err)
	}

	readVal, err := cpu.ReadMem(addr)
	if err != nil {
		t.Fatalf("ReadMem error: %v", err)
	}

	if readVal.ToInt() != 123 {
		t.Errorf("expected read value to be 123, got %d", readVal.ToInt())
	}
}

func TestCPU_JumpsAndCompare(t *testing.T) {
	cpu := &CPU{}
	cpu.Reset()

	// Load 50 into R[4], 50 into R[5], 60 into R[6]
	cpu.R[4] = tryte.FromInt(50)
	cpu.R[5] = tryte.FromInt(50)
	cpu.R[6] = tryte.FromInt(60)

	// Set register target jump addresses
	cpu.R[7] = tryte.FromInt(100) // Jump destination

	// CMP R[4], R[5] (50 == 50 -> Compare flag = O)
	_ = cpu.WriteMem(tryte.FromInt(0), MakeInst(OpCmp, 5, 4))
	// JEQ R[7] (should jump to 100)
	_ = cpu.WriteMem(tryte.FromInt(1), MakeInst(OpJeq, 7, 0))

	// Step 1: execute CMP
	_, _ = cpu.Step()
	if cpu.Compare != tryte.O {
		t.Errorf("expected Compare flag to be O (equal), got %v", cpu.Compare)
	}

	// Step 2: execute JEQ (should branch to PC=100)
	_, _ = cpu.Step()
	if cpu.PC.ToInt() != 100 {
		t.Errorf("expected PC to jump to 100, got %d", cpu.PC.ToInt())
	}

	// Reset PC to 10
	cpu.PC = tryte.FromInt(10)
	// CMP R[4], R[6] (50 < 60 -> Compare flag = F)
	_ = cpu.WriteMem(tryte.FromInt(10), MakeInst(OpCmp, 6, 4))
	// JLT R[7] (should jump to 100)
	_ = cpu.WriteMem(tryte.FromInt(11), MakeInst(OpJlt, 7, 0))

	// Step CMP
	_, _ = cpu.Step()
	if cpu.Compare != tryte.F {
		t.Errorf("expected Compare flag to be F (less), got %v", cpu.Compare)
	}

	// Step JLT
	_, _ = cpu.Step()
	if cpu.PC.ToInt() != 100 {
		t.Errorf("expected PC to jump to 100, got %d", cpu.PC.ToInt())
	}
}

func TestCPU_StoreAndLoad(t *testing.T) {
	cpu := &CPU{}
	cpu.Reset()

	// We want to store the value of R[0] (which is 99) at address 500
	cpu.R[0] = tryte.FromInt(99)
	cpu.R[1] = tryte.FromInt(500) // Address register

	// ST: store R[0] to address in R[1]
	_ = cpu.WriteMem(tryte.FromInt(0), MakeInst(OpSt, 1, 0))
	// LD: load address in R[1] to R[2]
	_ = cpu.WriteMem(tryte.FromInt(1), MakeInst(OpLd, 1, 2))
	// HALT
	_ = cpu.WriteMem(tryte.FromInt(2), MakeInst(OpHalt, 0, 0))

	// ST
	_, _ = cpu.Step()
	val, _ := cpu.ReadMem(cpu.R[1])
	if val.ToInt() != 99 {
		t.Errorf("expected value at address 500 to be 99, got %d", val.ToInt())
	}

	// LD
	_, _ = cpu.Step()
	if cpu.R[2].ToInt() != 99 {
		t.Errorf("expected R[2] to load 99, got %d", cpu.R[2].ToInt())
	}
}
