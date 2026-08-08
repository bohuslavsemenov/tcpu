package cpu

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"time"

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
	OpCall = 8
	OpRet  = -8
	OpAnd  = 9
	OpOr   = -9
	OpShl  = 10
	OpShr  = -10
	OpDiv  = 11
	OpMod  = -11
	OpRte  = 12
	OpSys  = -12
)

type CPU struct {
	R           [9]tryte.Tryte
	PC          tryte.Tryte
	Compare     tryte.Trit
	Carry       bool
	ALU         ALU
	Memory      [MemorySize]tryte.Tryte
	SPC         tryte.Tryte
	SCompare    tryte.Trit
	SCarry      bool
	IVR         tryte.Tryte
	IntActive   bool
	DiskPath    string
	DiskSector  tryte.Tryte
	DiskStatus  tryte.Tryte
	SIVR        tryte.Tryte
	SYS_PC      tryte.Tryte
	SYS_Compare tryte.Trit
	SYS_Carry   bool
	SYS_Active             bool
	Base                   tryte.Tryte
	IntStatus              tryte.Tryte
	LastKey                byte
	InputQueue             chan byte
	KeyboardListenerActive bool

	Stdin  io.Reader
	Stdout io.Writer
}

// Memory-mapped I/O addresses
const (
	AddrStdout          = 9000
	AddrStdin           = 9001
	AddrVRAMStart       = 2000
	AddrVRAMEnd         = 2143
	AddrVideoRefresh    = 9002
	AddrIVR             = 9003
	AddrDiskSector      = 9004
	AddrDiskCommand     = 9005
	AddrDiskBufferStart = 9006
	AddrDiskBufferEnd   = 9086
	AddrSIVR            = 9087
	AddrBase            = 9088
	AddrSPC             = 9089
	AddrSysPC           = 9090
	AddrIntStatus       = 9091
)

// Reset resets the CPU registers to their default state.
func (cpu *CPU) Reset() {
	for i := range cpu.R {
		cpu.R[i] = tryte.FromInt(0)
	}
	cpu.PC = tryte.FromInt(0)
	cpu.Compare = tryte.O
	cpu.Carry = false
	cpu.SPC = tryte.FromInt(0)
	cpu.SCompare = tryte.O
	cpu.SCarry = false
	cpu.IVR = tryte.FromInt(0)
	cpu.IntActive = false
	cpu.DiskSector = tryte.FromInt(0)
	cpu.DiskStatus = tryte.FromInt(0)
	cpu.SIVR = tryte.FromInt(0)
	cpu.SYS_PC = tryte.FromInt(0)
	cpu.SYS_Compare = tryte.O
	cpu.SYS_Carry = false
	cpu.SYS_Active = false
	cpu.Base = tryte.FromInt(0)
	cpu.IntStatus = tryte.FromInt(0)
	cpu.LastKey = 0
	cpu.KeyboardListenerActive = false
	if cpu.InputQueue == nil {
		cpu.InputQueue = make(chan byte, 256)
	} else {
		// Drain queue
		for len(cpu.InputQueue) > 0 {
			<-cpu.InputQueue
		}
	}
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
	if addrVal == AddrStdout {
		return tryte.FromInt(0), nil
	}

	if addrVal == AddrStdin {
		if cpu.LastKey != 0 {
			val := tryte.FromInt(int(cpu.LastKey))
			cpu.LastKey = 0
			return val, nil
		}
		select {
		case char := <-cpu.InputQueue:
			return tryte.FromInt(int(char)), nil
		default:
		}
		if cpu.KeyboardListenerActive {
			if cpu.IVR.ToInt() == 0 {
				time.Sleep(100 * time.Millisecond)
			}
			return tryte.FromInt(0), nil
		}
		reader := cpu.Stdin
		if reader == nil {
			reader = os.Stdin
		}
		var buf [1]byte
		n, err := reader.Read(buf[:])
		if err != nil || n == 0 {
			return tryte.FromInt(0), nil
		}
		return tryte.FromInt(int(buf[0])), nil
	}

	if addrVal == AddrVideoRefresh {
		return tryte.FromInt(0), nil
	}

	if addrVal == AddrIVR {
		return cpu.IVR, nil
	}

	if addrVal == AddrDiskSector {
		return cpu.DiskSector, nil
	}

	if addrVal == AddrDiskCommand {
		return cpu.DiskStatus, nil
	}

	if addrVal >= AddrDiskBufferStart && addrVal <= AddrDiskBufferEnd {
		idx, err := cpu.tryteToAddr(addr)
		if err != nil {
			return tryte.Tryte{}, err
		}
		return cpu.Memory[idx], nil
	}

	if addrVal == AddrSIVR {
		return cpu.SIVR, nil
	}

	if addrVal == AddrBase {
		return cpu.Base, nil
	}

	if addrVal == AddrSPC {
		return cpu.SPC, nil
	}

	if addrVal == AddrSysPC {
		return cpu.SYS_PC, nil
	}

	if addrVal == AddrIntStatus {
		return cpu.IntStatus, nil
	}

	// Apply base relocation for user space addresses
	if addrVal < AddrVRAMStart && !cpu.IntActive && !cpu.SYS_Active {
		relVal := addrVal + cpu.Base.ToInt()
		if relVal < -9841 || relVal > 9841 {
			return tryte.Tryte{}, fmt.Errorf("relocated read address %d out of bounds", relVal)
		}
		addr = tryte.FromInt(relVal)
	}

	idx, err := cpu.tryteToAddr(addr)
	if err != nil {
		return tryte.Tryte{}, err
	}
	val := cpu.Memory[idx]
	return val, nil
}

// WriteMem writes a Tryte to memory at the given Tryte address.
func (cpu *CPU) WriteMem(addr, val tryte.Tryte) error {
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

	if addrVal == AddrVideoRefresh {
		err := cpu.renderFramebuffer()
		if err != nil {
			return fmt.Errorf("video refresh error: %w", err)
		}
		return nil
	}

	if addrVal == AddrIVR {
		cpu.IVR = val
		return nil
	}

	if addrVal == AddrSIVR {
		cpu.SIVR = val
		return nil
	}

	if addrVal == AddrBase {
		if val.ToInt() != 0 {
			fmt.Printf("[BASE WRITTEN BY PC %d! val=%d, R0=%d, R1=%d, R2=%d, R3=%d, R4=%d, R5=%d, R6=%d, R7=%d, R8=%d]\n", cpu.PC.ToInt(), val.ToInt(), cpu.R[0].ToInt(), cpu.R[1].ToInt(), cpu.R[2].ToInt(), cpu.R[3].ToInt(), cpu.R[4].ToInt(), cpu.R[5].ToInt(), cpu.R[6].ToInt(), cpu.R[7].ToInt(), cpu.R[8].ToInt())
		}
		cpu.Base = val
		return nil
	}

	if addrVal == AddrIntStatus {
		cpu.IntStatus = val
		return nil
	}

	if addrVal == AddrSPC {
		cpu.SPC = val
		return nil
	}

	if addrVal == AddrSysPC {
		cpu.SYS_PC = val
		return nil
	}

	if addrVal == AddrDiskSector {
		cpu.DiskSector = val
		return nil
	}

	if addrVal == AddrDiskCommand {
		err := cpu.executeDiskCommand(val.ToInt())
		if err != nil {
			return fmt.Errorf("disk command execution error: %w", err)
		}
		return nil
	}

	// Apply base relocation for user space addresses
	if addrVal < AddrVRAMStart && !cpu.IntActive && !cpu.SYS_Active {
		relVal := addrVal + cpu.Base.ToInt()
		if relVal < -9841 || relVal > 9841 {
			return fmt.Errorf("relocated write address %d out of bounds", relVal)
		}
		addr = tryte.FromInt(relVal)
	}

	idx, err := cpu.tryteToAddr(addr)
	if err != nil {
		return err
	}
	cpu.Memory[idx] = val
	return nil
}

func (cpu *CPU) renderFramebuffer() error {
	writer := cpu.Stdout
	if writer == nil {
		writer = os.Stdout
	}

	// 1. Clear Screen / Home Cursor
	// Send ANSI escape sequences: ESC [2J (clear screen) and ESC [H (home cursor)
	_, err := writer.Write([]byte{27, 91, 50, 74, 27, 91, 72})
	if err != nil {
		return fmt.Errorf("failed to write display controls to stdout: %w", err)
	}

	// 2. Draw Top Border (12 grid cols + 2 borders = 14 columns of '# # ')
	_, err = writer.Write([]byte("# # # # # # # # # # # # # # \r\n"))
	if err != nil {
		return fmt.Errorf("failed to write display border: %w", err)
	}

	// 3. Draw Grid
	for y := 0; y < 12; y++ {
		row := make([]byte, 0, 30)
		row = append(row, '#', ' ') // Left border
		for x := 0; x < 12; x++ {
			vramAddr := tryte.FromInt(AddrVRAMStart + y*12 + x)
			cellValTryte, err := cpu.ReadMem(vramAddr)
			if err != nil {
				return err
			}
			cellVal := cellValTryte.ToInt()
			if cellVal == 0 {
				row = append(row, '.', ' ') // Empty cell
			} else {
				row = append(row, byte(cellVal), ' ') // ASCII pixel character
			}
		}
		row = append(row, '#', '\r', '\n') // Right border and newline
		_, err = writer.Write(row)
		if err != nil {
			return fmt.Errorf("failed to write display row: %w", err)
		}
	}

	// 4. Draw Bottom Border
	_, err = writer.Write([]byte("# # # # # # # # # # # # # # \r\n"))
	if err != nil {
		return fmt.Errorf("failed to write display border: %w", err)
	}

	return nil
}

func (cpu *CPU) getRegIndex(t tryte.Tryte) int {
	val := int(t[0]) + int(t[1])*3 + int(t[2])*9
	return (val%9 + 9) % 9
}

// Step executes a single instruction cycle (Fetch, Decode, Execute).
// Returns (keepRunning, error).
func (cpu *CPU) Step() (bool, error) {
	// Check for pending keyboard interrupts
	if !cpu.IntActive && cpu.IVR.ToInt() != 0 {
		select {
		case char := <-cpu.InputQueue:
			cpu.IntActive = true
			cpu.IntStatus = tryte.FromInt(2) // Keyboard Interrupt Status = 2
			cpu.LastKey = char
			cpu.SPC = cpu.PC
			cpu.SCompare = cpu.Compare
			cpu.SCarry = cpu.Carry
			cpu.PC = cpu.IVR
		default:
		}
	}

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
		if cpu.R[srcIdx].ToInt() == AddrBase {
			fmt.Printf("[ST INSTRUCTION AT PC %d WRITING TO ADDRBASE (9088)! srcIdx=%d, dstIdx=%d, R[src]=%d, R[dst]=%d]\n", cpu.PC.ToInt(), srcIdx, dstIdx, cpu.R[srcIdx].ToInt(), cpu.R[dstIdx].ToInt())
		}
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

	case OpCall: // CALL: store return address in R8 and jump to R[src]
		cpu.R[8] = cpu.PC
		cpu.PC = cpu.R[srcIdx]

	case OpRet: // RET: return to address in R8
		cpu.PC = cpu.R[8]

	case OpAnd: // AND: R[dst] = R[dst] AND R[src]
		cpu.R[dstIdx] = cpu.ALU.And(cpu.R[dstIdx], cpu.R[srcIdx])

	case OpOr: // OR: R[dst] = R[dst] OR R[src]
		cpu.R[dstIdx] = cpu.ALU.Or(cpu.R[dstIdx], cpu.R[srcIdx])

	case OpShl: // SHL: R[dst] = R[dst] << R[src] (trits)
		res, overflow := cpu.ALU.ShiftLeft(cpu.R[dstIdx], cpu.R[srcIdx].ToInt())
		cpu.R[dstIdx] = res
		cpu.Carry = overflow

	case OpShr: // SHR: R[dst] = R[dst] >> R[src] (trits)
		cpu.R[dstIdx] = cpu.ALU.ShiftRight(cpu.R[dstIdx], cpu.R[srcIdx].ToInt())

	case OpDiv: // DIV: R[dst] = R[dst] / R[src]
		res, overflow, err := cpu.ALU.Div(cpu.R[dstIdx], cpu.R[srcIdx])
		if err != nil {
			return false, fmt.Errorf("division execution error: %w", err)
		}
		cpu.R[dstIdx] = res
		cpu.Carry = overflow

	case OpMod: // MOD: R[dst] = R[dst] % R[src]
		res, err := cpu.ALU.Mod(cpu.R[dstIdx], cpu.R[srcIdx])
		if err != nil {
			return false, fmt.Errorf("modulo execution error: %w", err)
		}
		cpu.R[dstIdx] = res

	case OpRte: // RTE: return from exception or system call
		if cpu.IntActive {
			cpu.PC = cpu.SPC
			cpu.Compare = cpu.SCompare
			cpu.Carry = cpu.SCarry
			cpu.IntActive = false
		} else if cpu.SYS_Active {
			cpu.PC = cpu.SYS_PC
			cpu.Compare = cpu.SYS_Compare
			cpu.Carry = cpu.SYS_Carry
			cpu.SYS_Active = false
		}

	case OpSys: // SYS: software interrupt / system call
		if cpu.SIVR.ToInt() != 0 && !cpu.SYS_Active {
			cpu.SYS_Active = true
			cpu.SYS_PC = cpu.PC
			cpu.SYS_Compare = cpu.Compare
			cpu.SYS_Carry = cpu.Carry
			cpu.PC = cpu.SIVR
		}

	default:
		return false, fmt.Errorf("unknown opcode: %d", op)
	}

	return true, nil
}

// TriggerInterrupt saves CPU state and jumps PC to the interrupt vector (IVR)
func (cpu *CPU) TriggerInterrupt() {
	if cpu.IVR.ToInt() != 0 && !cpu.IntActive {
		cpu.IntStatus = tryte.FromInt(1) // Timer Interrupt Status = 1
		cpu.IntActive = true
		cpu.SPC = cpu.PC
		cpu.SCompare = cpu.Compare
		cpu.SCarry = cpu.Carry
		cpu.PC = cpu.IVR
	}
}

// StartKeyboardListener begins monitoring stdin for asynchronous keyboard interrupts
func (cpu *CPU) StartKeyboardListener() {
	cpu.KeyboardListenerActive = true
	if cpu.InputQueue == nil {
		cpu.InputQueue = make(chan byte, 256)
	}
	go func() {
		var buf [1]byte
		for {
			reader := cpu.Stdin
			if reader == nil {
				reader = os.Stdin
			}
			n, err := reader.Read(buf[:])
			if err != nil || n == 0 {
				break
			}
			cpu.QueueKeyboardInput(buf[0])
		}
	}()
}

// QueueKeyboardInput adds a character to the queue
func (cpu *CPU) QueueKeyboardInput(char byte) {
	if cpu.InputQueue == nil {
		cpu.InputQueue = make(chan byte, 256)
	}
	select {
	case cpu.InputQueue <- char:
		// Succeeded
	default:
		// Queue full, drop
	}
}

// Disk command values
const (
	DiskCmdWrite = -1
	DiskCmdRead  = 1
)

// executeDiskCommand performs reads/writes from/to the virtual disk file
func (cpu *CPU) executeDiskCommand(cmd int) error {
	if cpu.DiskPath == "" {
		cpu.DiskPath = "disk.img"
	}

	// Open or create the file
	f, err := os.OpenFile(cpu.DiskPath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		cpu.DiskStatus = tryte.FromInt(-1)
		return err
	}
	defer f.Close()

	sectorNum := cpu.DiskSector.ToInt()
	if sectorNum < 0 {
		cpu.DiskStatus = tryte.FromInt(-1)
		return fmt.Errorf("invalid negative sector number: %d", sectorNum)
	}

	offset := int64(sectorNum) * 81 * 2
	_, err = f.Seek(offset, 0)
	if err != nil {
		cpu.DiskStatus = tryte.FromInt(-1)
		return err
	}

	if cmd == DiskCmdRead { // Read
		buf := make([]byte, 162)
		_, err := io.ReadFull(f, buf)
		if err != nil {
			// If EOF, just fill buffer with zeros
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				for i := 0; i < 81; i++ {
					idx, _ := cpu.tryteToAddr(tryte.FromInt(AddrDiskBufferStart + i))
					cpu.Memory[idx] = tryte.FromInt(0)
				}
				cpu.DiskStatus = tryte.FromInt(0)
				return nil
			}
			cpu.DiskStatus = tryte.FromInt(-1)
			return err
		}
		for i := 0; i < 81; i++ {
			val := int16(binary.BigEndian.Uint16(buf[i*2 : i*2+2]))
			idx, _ := cpu.tryteToAddr(tryte.FromInt(AddrDiskBufferStart + i))
			cpu.Memory[idx] = tryte.FromInt(int(val))
		}
		cpu.DiskStatus = tryte.FromInt(0)
	} else if cmd == DiskCmdWrite { // Write
		buf := make([]byte, 162)
		for i := 0; i < 81; i++ {
			idx, _ := cpu.tryteToAddr(tryte.FromInt(AddrDiskBufferStart + i))
			val := int16(cpu.Memory[idx].ToInt())
			binary.BigEndian.PutUint16(buf[i*2:i*2+2], uint16(val))
		}
		_, err = f.Write(buf)
		if err != nil {
			cpu.DiskStatus = tryte.FromInt(-1)
			return err
		}
		cpu.DiskStatus = tryte.FromInt(0)
	}
	return nil
}
