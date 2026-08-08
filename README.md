# TCPU - Balanced Ternary CPU Emulator & Assembler

**TCPU** is a complete balanced ternary computer architecture, assembler, and software suite implemented in Go. Unlike standard binary computers that operate on bits (`0` and `1`), TCPU operates on **trits** (`-1`, `0`, and `+1`) and **trytes** (9-trit words).

TCPU features a full 9-tryte instruction set architecture (ISA), memory-mapped I/O (MMIO) for text output, raw terminal input, video framebuffer rendering, persistent virtual disk storage, hardware interrupts, process memory relocation, and software system calls.

---

## Architecture Overview

### Balanced Ternary Fundamentals

* **Trit (`tryte.Trit`)**: The fundamental unit of information with three states:
  * `F` (False / -1)
  * `O` (Zero / 0)
  * `T` (True / +1)
* **Tryte (`tryte.Tryte`)**: The primary word size consisting of **9 Trits**.
  * **Range**: $-9841$ to $+9841$ (representing $3^9 = 19,683$ distinct values).
* **Address Space**: $19,683$ memory words indexed from address `-9841` to `+9841`.

### Registers

TCPU contains 9 general-purpose registers and several specialized control registers:

* **`R0` – `R8`**: 9 General-purpose 1-Tryte registers.
* **`PC`**: Program Counter.
* **`Compare`**: Trit comparison flag (`F` for $<$, `O` for $=$, `T` for $>$).
* **`Carry`**: Boolean arithmetic carry flag.
* **`SPC` / `SCompare` / `SCarry`**: Saved state registers for hardware interrupts.
* **`SYS_PC` / `SYS_Compare` / `SYS_Carry`**: Saved state registers for software system calls (`SYS`).
* **`IVR`**: Hardware Interrupt Vector Register.
* **`SIVR`**: System Call Vector Register.
* **`Base`**: Memory relocation register for process isolation.
* **`IntStatus`**: Interrupt status flags.

---

## Memory Map & I/O (MMIO)

| Address Range / Value | Name | Description |
|---|---|---|
| `2000` – `2143` | `AddrVRAMStart` – `AddrVRAMEnd` | Video RAM framebuffer (144 trytes, 12×12 display grid) |
| `9000` | `AddrStdout` | Character output stream (ASCII print) |
| `9001` | `AddrStdin` | Character input stream / Keyboard queue |
| `9002` | `AddrVideoRefresh` | Triggers rendering of the VRAM framebuffer |
| `9003` | `AddrIVR` | Hardware Interrupt Vector Register |
| `9004` | `AddrDiskSector` | Virtual disk sector selector |
| `9005` | `AddrDiskCommand` | Virtual disk command trigger (`-1` = Write, `1` = Read) |
| `9006` – `9086` | `AddrDiskBufferStart` – `AddrDiskBufferEnd` | Virtual disk sector buffer (81 trytes) |
| `9087` | `AddrSIVR` | System Interrupt / Syscall Vector Register |
| `9088` | `AddrBase` | Process base relocation address |
| `9089` | `AddrSPC` | Saved PC register |
| `9090` | `AddrSysPC` | Saved System PC register |
| `9091` | `AddrIntStatus` | Interrupt status register |

---

## Instruction Set Architecture (ISA)

TCPU instructions are encoded in 1 to 3 trytes.

| Opcode | Mnemonic | Arguments | Description |
|---|---|---|---|
| `0` | `HALT` | None | Stop CPU execution |
| `1` | `ADD` | `dst, src` | `R[dst] = R[dst] + R[src]` |
| `-1` | `SUB` | `dst, src` | `R[dst] = R[dst] - R[src]` |
| `2` | `MUL` | `dst, src` | `R[dst] = R[dst] * R[src]` |
| `-2` | `CMP` | `r1, r2` | Compare `R[r1]` with `R[r2]`, set `Compare` trit (`F`, `O`, `T`) |
| `3` | `LD` | `dst, src` | `R[dst] = Memory[R[src]]` |
| `-3` | `ST` | `src, dst` | `Memory[R[src]] = R[dst]` |
| `4` | `LDI` | `dst, imm` | `R[dst] = immediate_value` |
| `5` | `JMP` | `target` | Jump unconditionally to address |
| `6` | `JEQ` | `target` | Jump if `Compare == O` (equal) |
| `-6` | `JNE` | `target` | Jump if `Compare != O` (not equal) |
| `7` | `JLT` | `target` | Jump if `Compare == F` (less than) |
| `-7` | `JGT` | `target` | Jump if `Compare == T` (greater than) |
| `8` | `CALL` | `target` | Push return address and jump to subroutine |
| `-8` | `RET` | None | Return from subroutine |
| `9` | `AND` | `dst, src` | Bitwise/tritwise AND |
| `-9` | `OR` | `dst, src` | Bitwise/tritwise OR |
| `10` | `SHL` | `dst, count` | Shift left |
| `-10` | `SHR` | `dst, count` | Shift right |
| `11` | `DIV` | `dst, src` | `R[dst] = R[dst] / R[src]` |
| `-11` | `MOD` | `dst, src` | `R[dst] = R[dst] % R[src]` |
| `12` | `RTE` | None | Return from interrupt or exception handler |
| `-12` | `SYS` | None | Trigger software interrupt / system call |

---

## Quick Start

### Prerequisites

* [Go](https://golang.org/) 1.22 or higher.

### Running a Program

Run an assembly (`.tasm`) file using `cmd/tcpu`:

```bash
# Run Hello World
go run ./cmd/tcpu programs/hello.tasm

# Run Snake game (requires -r flag for raw terminal input)
go run ./cmd/tcpu -r programs/snake.tasm

# Run with step-by-step CPU execution debugging (-d flag)
go run ./cmd/tcpu -d programs/factorial.tasm
```

### Command Line Flags

* `-r`: Enable raw terminal mode and non-blocking input for interactive programs and games.
* `-d`: Enable step-by-step CPU debug logging output to `stderr`.

---

## Included Assembly Programs (`programs/`)

* **`hello.tasm`**: Hello World string output demonstration.
* **`snake.tasm`**: Interactive Snake game rendered on the 12×12 text VRAM display.
* **`pong.tasm`**: Real-time Pong game using keyboard input and VRAM rendering.
* **`clock.tasm`**: Real-time clock powered by 10Hz hardware timer interrupts.
* **`disk.tasm`**: Persistent virtual disk read/write demonstration.
* **`factorial.tasm`**: Factorial calculation algorithm.
* **`fibonacci.tasm`**: Fibonacci series generator.
* **`syscall.tasm`**: Software system call (`SYS`) example.
* **`key_interrupt.tasm`**: Asynchronous keyboard hardware interrupt handler.
* **`reloc.tasm`**: Process memory relocation demonstration.
* **`echo.tasm`**: Interactive echo program.
* **`multiply.tasm`** / **`divmod.tasm`**: Multiplication and division math utilities.
* **`sum.tasm`** / **`logic.tasm`** / **`shift.tasm`**: Arithmetic, bitwise/tritwise logic, and shift operations.
* **`subroutine.tasm`**: Function call and return stack demonstration.

---

## Running Tests

Run all unit tests across the codebase:

```bash
go test ./...
```

---

## Package Structure

```
.
├── cmd/
│   └── tcpu/           # CLI entry point to run .tasm files on the TCPU emulator
├── pkg/
│   ├── tryte/          # Balanced ternary Trit and Tryte data types & arithmetic
│   ├── cpu/            # CPU emulator logic, registers, MMIO, and instruction decoder
│   └── assembler/      # TASM assembly parser, lexer, and bytecode generator
├── programs/           # TASM assembly sample applications
└── README.md
```
