package cpu

import (
	"bytes"
	"os"
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

func TestCPU_Subroutines(t *testing.T) {
	cpu := &CPU{}
	cpu.Reset()

	// Subroutine address is 100
	cpu.R[0] = tryte.FromInt(100)

	// Address 0: CALL R0 -> should jump to 100, and save PC=1 to R8
	_ = cpu.WriteMem(tryte.FromInt(0), MakeInst(OpCall, 0, 0))
	// Address 1: HALT
	_ = cpu.WriteMem(tryte.FromInt(1), MakeInst(OpHalt, 0, 0))

	// Address 100: RET -> should jump back to PC=1
	_ = cpu.WriteMem(tryte.FromInt(100), MakeInst(OpRet, 0, 0))

	// Step 1: execute CALL R0
	_, err := cpu.Step()
	if err != nil {
		t.Fatalf("CALL error: %v", err)
	}
	if cpu.PC.ToInt() != 100 {
		t.Errorf("expected PC to be 100, got %d", cpu.PC.ToInt())
	}
	if cpu.R[8].ToInt() != 1 {
		t.Errorf("expected R8 (LR) to be 1, got %d", cpu.R[8].ToInt())
	}

	// Step 2: execute RET (should return to PC=1)
	_, err = cpu.Step()
	if err != nil {
		t.Fatalf("RET error: %v", err)
	}
	if cpu.PC.ToInt() != 1 {
		t.Errorf("expected PC to return to 1, got %d", cpu.PC.ToInt())
	}

	// Step 3: execute HALT
	keepRunning, err := cpu.Step()
	if err != nil {
		t.Fatalf("HALT error: %v", err)
	}
	if keepRunning {
		t.Errorf("expected CPU to halt, but it is still running")
	}
}

func TestCPU_Framebuffer(t *testing.T) {
	cpu := &CPU{}
	cpu.Reset()

	var stdoutBuf bytes.Buffer
	cpu.Stdout = &stdoutBuf

	// Write 'A' (65) to row 0, col 0 (address 2000)
	err := cpu.WriteMem(tryte.FromInt(AddrVRAMStart), tryte.FromInt(65))
	if err != nil {
		t.Fatalf("WriteMem VRAM error: %v", err)
	}

	// Trigger refresh
	err = cpu.WriteMem(tryte.FromInt(AddrVideoRefresh), tryte.FromInt(1))
	if err != nil {
		t.Fatalf("WriteMem video refresh error: %v", err)
	}

	output := stdoutBuf.String()

	// Verify output has ANSI clear and border
	if !strings.Contains(output, "# # # # # # # # # # # # # #") {
		t.Errorf("expected output to contain border, got %q", output)
	}
	// Verify it contains the 'A' character
	if !strings.Contains(output, "A") {
		t.Errorf("expected output to contain 'A' pixel, got %q", output)
	}
}

func TestCPU_LogicalOperations(t *testing.T) {
	cpu := &CPU{}
	cpu.Reset()

	// Initial values: R0 = [T, O, F, T, O, F, T, O, F], R1 = [T, T, T, O, O, O, F, F, F]
	r0Val := tryte.Tryte{tryte.T, tryte.O, tryte.F, tryte.T, tryte.O, tryte.F, tryte.T, tryte.O, tryte.F}
	r1Val := tryte.Tryte{tryte.T, tryte.T, tryte.T, tryte.O, tryte.O, tryte.O, tryte.F, tryte.F, tryte.F}
	cpu.R[0] = r0Val
	cpu.R[1] = r1Val

	// Program:
	// AND R1, R0 -> R0 = R0 AND R1 (expected [T, O, F, O, O, F, F, F, F])
	_ = cpu.WriteMem(tryte.FromInt(0), MakeInst(OpAnd, 1, 0))
	// OR R1, R0 -> R0 = R0 OR R1 (expected [T, T, T, T, O, O, T, O, F] since R0 starts as And-result and we OR with R1)
	_ = cpu.WriteMem(tryte.FromInt(1), MakeInst(OpOr, 1, 0))

	// Step 1: execute AND
	_, err := cpu.Step()
	if err != nil {
		t.Fatalf("AND execution error: %v", err)
	}
	expectedAnd := tryte.Tryte{tryte.T, tryte.O, tryte.F, tryte.O, tryte.O, tryte.F, tryte.F, tryte.F, tryte.F}
	if cpu.R[0] != expectedAnd {
		t.Errorf("expected R0 after AND to be %v, got %v", expectedAnd, cpu.R[0])
	}

	// Step 2: execute OR
	_, err = cpu.Step()
	if err != nil {
		t.Fatalf("OR execution error: %v", err)
	}
	expectedOr := tryte.Tryte{tryte.T, tryte.T, tryte.T, tryte.O, tryte.O, tryte.O, tryte.F, tryte.F, tryte.F}
	if cpu.R[0] != expectedOr {
		t.Errorf("expected R0 after OR to be %v, got %v", expectedOr, cpu.R[0])
	}
}

func TestCPU_ShiftOperations(t *testing.T) {
	cpu := &CPU{}
	cpu.Reset()

	// Initial values: R0 = 5, R1 = 2 (shift amount)
	cpu.R[0] = tryte.FromInt(5)
	cpu.R[1] = tryte.FromInt(2)

	// Program:
	// SHL R1, R0 -> R0 = R0 << 2 (expected 5 * 9 = 45)
	_ = cpu.WriteMem(tryte.FromInt(0), MakeInst(OpShl, 1, 0))
	// SHR R1, R0 -> R0 = R0 >> 2 (expected 45 / 9 = 5)
	_ = cpu.WriteMem(tryte.FromInt(1), MakeInst(OpShr, 1, 0))

	// Step 1: execute SHL
	_, err := cpu.Step()
	if err != nil {
		t.Fatalf("SHL execution error: %v", err)
	}
	if cpu.R[0].ToInt() != 45 {
		t.Errorf("expected R0 after SHL to be 45, got %d", cpu.R[0].ToInt())
	}
	if cpu.Carry {
		t.Errorf("expected no carry after SHL, got true")
	}

	// Step 2: execute SHR
	_, err = cpu.Step()
	if err != nil {
		t.Fatalf("SHR execution error: %v", err)
	}
	if cpu.R[0].ToInt() != 5 {
		t.Errorf("expected R0 after SHR to be 5, got %d", cpu.R[0].ToInt())
	}
}

func TestCPU_DivModOperations(t *testing.T) {
	cpu := &CPU{}
	cpu.Reset()

	// Initial values: R0 = 5, R1 = 3
	cpu.R[0] = tryte.FromInt(5)
	cpu.R[1] = tryte.FromInt(3)

	// Program:
	// DIV R1, R0 -> R0 = R0 / R1 (expected 5 / 3 = 2)
	_ = cpu.WriteMem(tryte.FromInt(0), MakeInst(OpDiv, 1, 0))
	// Reset R0 back to 5 for Mod
	// MOD R1, R0 -> R0 = R0 % R1 (expected 5 % 3 = -1)
	_ = cpu.WriteMem(tryte.FromInt(1), MakeInst(OpMod, 1, 0))

	// Step 1: execute DIV
	_, err := cpu.Step()
	if err != nil {
		t.Fatalf("DIV execution error: %v", err)
	}
	if cpu.R[0].ToInt() != 2 {
		t.Errorf("expected R0 after DIV to be 2, got %d", cpu.R[0].ToInt())
	}

	// Reset R0 to 5
	cpu.R[0] = tryte.FromInt(5)

	// Step 2: execute MOD
	_, err = cpu.Step()
	if err != nil {
		t.Fatalf("MOD execution error: %v", err)
	}
	if cpu.R[0].ToInt() != -1 {
		t.Errorf("expected R0 after MOD to be -1, got %d", cpu.R[0].ToInt())
	}
}

func TestCPU_InterruptsAndRTE(t *testing.T) {
	cpu := &CPU{}
	cpu.Reset()

	// Set Interrupt Vector Register (IVR) to 100
	err := cpu.WriteMem(tryte.FromInt(AddrIVR), tryte.FromInt(100))
	if err != nil {
		t.Fatalf("failed to set IVR: %v", err)
	}

	// Address 0: NOP/HALT (will trigger interrupt at PC=1)
	_ = cpu.WriteMem(tryte.FromInt(0), MakeInst(OpHalt, 0, 0))

	// Address 100: RTE (return from interrupt)
	_ = cpu.WriteMem(tryte.FromInt(100), MakeInst(OpRte, 0, 0))

	// Step 1: execute address 0, PC becomes 1
	_, _ = cpu.Step()
	if cpu.PC.ToInt() != 1 {
		t.Fatalf("expected PC to be 1, got %d", cpu.PC.ToInt())
	}

	// Set a mock state to verify restoration
	cpu.Compare = tryte.T
	cpu.Carry = true

	// Trigger the interrupt!
	cpu.TriggerInterrupt()

	if !cpu.IntActive {
		t.Errorf("expected IntActive to be true")
	}
	if cpu.PC.ToInt() != 100 {
		t.Errorf("expected PC to jump to IVR 100, got %d", cpu.PC.ToInt())
	}
	if cpu.SPC.ToInt() != 1 {
		t.Errorf("expected SPC to save PC=1, got %d", cpu.SPC.ToInt())
	}
	if cpu.SCompare != tryte.T {
		t.Errorf("expected SCompare to save Compare=T")
	}
	if !cpu.SCarry {
		t.Errorf("expected SCarry to save Carry=true")
	}

	// Clear Compare/Carry in interrupt handler to verify they get restored by RTE
	cpu.Compare = tryte.O
	cpu.Carry = false

	// Step 2: execute RTE at PC=100
	_, err = cpu.Step()
	if err != nil {
		t.Fatalf("RTE execution error: %v", err)
	}

	if cpu.PC.ToInt() != 1 {
		t.Errorf("expected PC to return to 1, got %d", cpu.PC.ToInt())
	}
	if cpu.Compare != tryte.T {
		t.Errorf("expected Compare to restore to T, got %v", cpu.Compare)
	}
	if !cpu.Carry {
		t.Errorf("expected Carry to restore to true")
	}
	if cpu.IntActive {
		t.Errorf("expected IntActive to reset to false")
	}
}

func TestCPU_VirtualDisk(t *testing.T) {
	diskPath := "test_disk.img"
	defer os.Remove(diskPath)

	cpu := &CPU{
		DiskPath: diskPath,
	}
	cpu.Reset()

	// Write value 123 to index 0 of the disk buffer
	err := cpu.WriteMem(tryte.FromInt(AddrDiskBufferStart), tryte.FromInt(123))
	if err != nil {
		t.Fatalf("failed to write to disk buffer: %v", err)
	}

	// Set sector number to 5
	err = cpu.WriteMem(tryte.FromInt(AddrDiskSector), tryte.FromInt(5))
	if err != nil {
		t.Fatalf("failed to set sector: %v", err)
	}

	// Trigger Write command (-1)
	err = cpu.WriteMem(tryte.FromInt(AddrDiskCommand), tryte.FromInt(-1))
	if err != nil {
		t.Fatalf("failed to write command: %v", err)
	}

	// Clear the buffer in memory
	err = cpu.WriteMem(tryte.FromInt(AddrDiskBufferStart), tryte.FromInt(0))
	if err != nil {
		t.Fatalf("failed to clear disk buffer: %v", err)
	}

	// Trigger Read command (1)
	err = cpu.WriteMem(tryte.FromInt(AddrDiskCommand), tryte.FromInt(1))
	if err != nil {
		t.Fatalf("failed to read command: %v", err)
	}

	// Check that the value 123 was read back
	val, err := cpu.ReadMem(tryte.FromInt(AddrDiskBufferStart))
	if err != nil {
		t.Fatalf("failed to read memory: %v", err)
	}
	if val.ToInt() != 123 {
		t.Errorf("expected 123, got %d", val.ToInt())
	}
}

func TestCPU_Syscalls(t *testing.T) {
	cpu := &CPU{}
	cpu.Reset()

	// Set SIVR (syscall handler vector) to 200
	err := cpu.WriteMem(tryte.FromInt(AddrSIVR), tryte.FromInt(200))
	if err != nil {
		t.Fatalf("failed to set SIVR: %v", err)
	}

	// Set IVR (hardware timer handler vector) to 100
	err = cpu.WriteMem(tryte.FromInt(AddrIVR), tryte.FromInt(100))
	if err != nil {
		t.Fatalf("failed to set IVR: %v", err)
	}

	// Address 0: SYS (trigger syscall)
	_ = cpu.WriteMem(tryte.FromInt(0), MakeInst(OpSys, 0, 0))

	// Address 200: RTE (syscall return)
	_ = cpu.WriteMem(tryte.FromInt(200), MakeInst(OpRte, 0, 0))

	// Address 100: RTE (hardware interrupt return)
	_ = cpu.WriteMem(tryte.FromInt(100), MakeInst(OpRte, 0, 0))

	// Set a mock state to verify restoration
	cpu.Compare = tryte.T
	cpu.Carry = true

	// Step 1: execute SYS instruction at PC=0
	_, err = cpu.Step()
	if err != nil {
		t.Fatalf("SYS execution error: %v", err)
	}

	if !cpu.SYS_Active {
		t.Errorf("expected SYS_Active to be true")
	}
	if cpu.PC.ToInt() != 200 {
		t.Errorf("expected PC to jump to SIVR 200, got %d", cpu.PC.ToInt())
	}
	if cpu.SYS_PC.ToInt() != 1 { // PC gets incremented during step before jump
		t.Errorf("expected SYS_PC to save PC=1, got %d", cpu.SYS_PC.ToInt())
	}

	// Now we are inside Syscall handler. Trigger a nested hardware timer interrupt!
	cpu.TriggerInterrupt()

	if !cpu.IntActive {
		t.Errorf("expected IntActive to be true")
	}
	if cpu.PC.ToInt() != 100 {
		t.Errorf("expected PC to jump to IVR 100, got %d", cpu.PC.ToInt())
	}
	if cpu.SPC.ToInt() != 200 {
		t.Errorf("expected SPC to save PC=200, got %d", cpu.SPC.ToInt())
	}

	// Step 2: execute RTE at PC=100 (returns from hardware interrupt back to syscall handler)
	_, err = cpu.Step()
	if err != nil {
		t.Fatalf("RTE execution error: %v", err)
	}

	if cpu.PC.ToInt() != 200 {
		t.Errorf("expected PC to return to 200, got %d", cpu.PC.ToInt())
	}
	if cpu.IntActive {
		t.Errorf("expected IntActive to be false")
	}
	if !cpu.SYS_Active {
		t.Errorf("expected SYS_Active to remain true")
	}

	// Step 3: execute RTE at PC=200 (returns from system call back to user code)
	_, err = cpu.Step()
	if err != nil {
		t.Fatalf("RTE execution error: %v", err)
	}

	if cpu.PC.ToInt() != 1 {
		t.Errorf("expected PC to return to 1, got %d", cpu.PC.ToInt())
	}
	if cpu.SYS_Active {
		t.Errorf("expected SYS_Active to reset to false")
	}
	if cpu.Compare != tryte.T {
		t.Errorf("expected Compare to restore to T")
	}
	if !cpu.Carry {
		t.Errorf("expected Carry to restore to true")
	}
}

func TestCPU_BaseRelocation(t *testing.T) {
	cpu := &CPU{}
	cpu.Reset()

	// Set Base register to 1000
	err := cpu.WriteMem(tryte.FromInt(AddrBase), tryte.FromInt(1000))
	if err != nil {
		t.Fatalf("failed to set Base: %v", err)
	}

	// 1. Test relocation in USER MODE (SYS_Active = false, IntActive = false)
	// Write 123 to virtual address 5 (should go to physical address 1005)
	err = cpu.WriteMem(tryte.FromInt(5), tryte.FromInt(123))
	if err != nil {
		t.Fatalf("WriteMem error in user mode: %v", err)
	}

	// Read virtual address 5
	val, err := cpu.ReadMem(tryte.FromInt(5))
	if err != nil {
		t.Fatalf("ReadMem error in user mode: %v", err)
	}
	if val.ToInt() != 123 {
		t.Errorf("expected read from virtual address 5 to be 123, got %d", val.ToInt())
	}

	// Verify relocation effect on physical addresses
	// Temporarily bypass relocation by entering kernel mode (SYS_Active = true)
	cpu.SYS_Active = true

	// Read physical address 5 (should be 0)
	valPhys5, err := cpu.ReadMem(tryte.FromInt(5))
	if err != nil {
		t.Fatalf("ReadMem physical 5 error: %v", err)
	}
	if valPhys5.ToInt() != 0 {
		t.Errorf("expected physical address 5 to hold 0, got %d", valPhys5.ToInt())
	}

	// Read physical address 1005 (should hold 123)
	valPhys1005, err := cpu.ReadMem(tryte.FromInt(1005))
	if err != nil {
		t.Fatalf("ReadMem physical 1005 error: %v", err)
	}
	if valPhys1005.ToInt() != 123 {
		t.Errorf("expected physical address 1005 to hold 123, got %d", valPhys1005.ToInt())
	}

	// 2. Test relocation bypass in KERNEL MODE (SYS_Active = true)
	// Write 456 to address 5 (should go directly to physical address 5)
	err = cpu.WriteMem(tryte.FromInt(5), tryte.FromInt(456))
	if err != nil {
		t.Fatalf("WriteMem error in kernel mode: %v", err)
	}

	// Read address 5
	valKernel5, err := cpu.ReadMem(tryte.FromInt(5))
	if err != nil {
		t.Fatalf("ReadMem error in kernel mode: %v", err)
	}
	if valKernel5.ToInt() != 456 {
		t.Errorf("expected read in kernel mode from address 5 to be 456, got %d", valKernel5.ToInt())
	}

	// Verify physical address 1005 is still 123
	valPhys1005After, _ := cpu.ReadMem(tryte.FromInt(1005))
	if valPhys1005After.ToInt() != 123 {
		t.Errorf("expected physical address 1005 to remain 123, got %d", valPhys1005After.ToInt())
	}
}
