package assembler

import (
	"testing"

	"github.com/bohuslavsemenov/tcpu/pkg/cpu"
	"github.com/bohuslavsemenov/tcpu/pkg/tryte"
)

// Helper to construct a 9-trit instruction from (opcode, src, dst) values using base-3 arithmetic
func makeInst(opVal, srcVal, dstVal int) tryte.Tryte {
	return tryte.FromInt(srcVal + dstVal*27 + opVal*729)
}

func TestAssemble_Valid(t *testing.T) {
	source := `
		; This is a comment
		LDI 3, R0
		LDI 5, R1 # inline comment
		ADD R1, R0

		# Let's test a label jump
		LOOP:
		LDI 1, R2
		ADD R2, R0
		CMP R0, R3
		LDI LOOP, R4
		JLT R4
		HALT
	`

	bytecode, err := Assemble(source)
	if err != nil {
		t.Fatalf("Assemble failed: %v", err)
	}

	// Calculate expected instruction size
	// LDI 3, R0 -> 2 words
	// LDI 5, R1 -> 2 words
	// ADD R1, R0 -> 1 word
	// LOOP: LDI 1, R2 -> 2 words (LOOP is at address 5)
	// ADD R2, R0 -> 1 word
	// CMP R0, R3 -> 1 word
	// LDI LOOP, R4 -> 2 words
	// JLT R4 -> 1 word (referencing R4)
	// HALT -> 1 word
	// Total size = 2 + 2 + 1 + 2 + 1 + 1 + 2 + 1 + 1 = 13 words.
	expectedSize := 13
	if len(bytecode) != expectedSize {
		t.Errorf("expected bytecode size %d, got %d", expectedSize, len(bytecode))
	}

	// Verify target address of LOOP is resolved as 5
	// LDI LOOP, R4 is at index 9 (inst at 9, immediate value at 10).
	// Index 10 should contain tryte representing 5.
	loopAddrTryte := bytecode[10]
	if loopAddrTryte.ToInt() != 5 {
		t.Errorf("expected LOOP label to resolve to 5, got %d", loopAddrTryte.ToInt())
	}

	// JLT R4 is at index 11, referencing R4
	// JLT is OpJlt (7), arg src is R4 (register index 4).
	// So makeInst(cpu.OpJlt, 4, 0).
	// Let's verify bytecode at index 11.
	expectedJlt := makeInst(cpu.OpJlt, 4, 0)
	if bytecode[11] != expectedJlt {
		t.Errorf("expected instruction at index 11 to be JLT R4, got %v", bytecode[11])
	}
}

func TestAssemble_Errors(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{"invalid register name", "LDI 3, R9"},
		{"invalid register syntax", "ADD R1, X0"},
		{"wrong argument count", "ADD R1"},
		{"unknown mnemonic", "FOO R1, R2"},
		{"empty label name", ": LDI 3, R0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Assemble(tt.source)
			if err == nil {
				t.Errorf("expected error for %q, got nil", tt.name)
			}
		})
	}
}

func TestAssemble_Subroutines(t *testing.T) {
	source := `
		CALL R0
		RET
	`
	bytecode, err := Assemble(source)
	if err != nil {
		t.Fatalf("Assemble failed: %v", err)
	}

	if len(bytecode) != 2 {
		t.Fatalf("expected bytecode size 2, got %d", len(bytecode))
	}

	expectedCall := makeInst(cpu.OpCall, 0, 0)
	if bytecode[0] != expectedCall {
		t.Errorf("expected instruction 0 to be CALL R0, got %v", bytecode[0])
	}

	expectedRet := makeInst(cpu.OpRet, 0, 0)
	if bytecode[1] != expectedRet {
		t.Errorf("expected instruction 1 to be RET, got %v", bytecode[1])
	}
}
