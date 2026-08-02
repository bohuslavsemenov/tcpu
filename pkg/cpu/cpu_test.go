package cpu

import (
	"testing"

	"github.com/bohuslavsemenov/tcpu/pkg/tryte"
)

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
