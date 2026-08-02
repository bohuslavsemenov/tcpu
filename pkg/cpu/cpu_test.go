package cpu

import (
	"bytes"
	"strings"
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

func TestCPU_MemoryMappedIO(t *testing.T) {
	cpu := &CPU{}
	cpu.Reset()

	// Inject custom mocked Stdin and Stdout
	var stdoutBuf bytes.Buffer
	stdinReader := strings.NewReader("A") // ASCII 65

	cpu.Stdin = stdinReader
	cpu.Stdout = &stdoutBuf

	// Set registers to address values
	cpu.R[1] = tryte.FromInt(9001) // stdin address
	cpu.R[2] = tryte.FromInt(9000) // stdout address
	cpu.R[3] = tryte.FromInt(1)    // constant 1 to add

	// Program instructions:
	// LD R1, R0    ; Load character from stdin (R1) into R0
	_ = cpu.WriteMem(tryte.FromInt(0), MakeInst(OpLd, 1, 0))
	// ADD R3, R0   ; R0 = R0 + R3 (65 + 1 = 66 -> ASCII 'B')
	_ = cpu.WriteMem(tryte.FromInt(1), MakeInst(OpAdd, 3, 0))
	// ST R0, R2    ; Write character from R0 to stdout (R2)
	_ = cpu.WriteMem(tryte.FromInt(2), MakeInst(OpSt, 2, 0))
	// HALT
	_ = cpu.WriteMem(tryte.FromInt(3), MakeInst(OpHalt, 0, 0))

	// Step 1: execute LD R1, R0
	_, err := cpu.Step()
	if err != nil {
		t.Fatalf("LD error: %v", err)
	}
	if cpu.R[0].ToInt() != 65 {
		t.Errorf("expected R[0] to be 65, got %d", cpu.R[0].ToInt())
	}

	// Step 2: execute ADD R3, R0
	_, err = cpu.Step()
	if err != nil {
		t.Fatalf("ADD error: %v", err)
	}
	if cpu.R[0].ToInt() != 66 {
		t.Errorf("expected R[0] to be 66, got %d", cpu.R[0].ToInt())
	}

	// Step 3: execute ST R0, R2
	_, err = cpu.Step()
	if err != nil {
		t.Fatalf("ST error: %v", err)
	}
	
	outputStr := stdoutBuf.String()
	if outputStr != "B" {
		t.Errorf("expected stdout to receive 'B', got %q", outputStr)
	}
}
