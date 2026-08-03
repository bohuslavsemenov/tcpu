package assembler

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bohuslavsemenov/tcpu/pkg/cpu"
	"github.com/bohuslavsemenov/tcpu/pkg/tryte"
)

type parsedLine struct {
	originalLine string
	mnemonic     string
	args         []string
	address      int
	size         int
}

// Assembler handles assembly to bytecode parsing and translation.
type Assembler struct {
	labels         map[string]int
	currentAddress int
}

func NewAssembler() *Assembler {
	return &Assembler{
		labels:         make(map[string]int),
		currentAddress: 0,
	}
}

// Assemble translates assembly source code into a slice of Trytes.
func Assemble(source string) ([]tryte.Tryte, error) {
	a := NewAssembler()
	return a.Assemble(source)
}

// Assemble translates assembly source code into a slice of Trytes.
func (a *Assembler) Assemble(source string) ([]tryte.Tryte, error) {
	lines := strings.Split(source, "\n")

	program, err := a.pass1(lines)
	if err != nil {
		return nil, err
	}

	return a.pass2(program)
}

// pass1 parses structure, counts instruction sizes, and records label addresses.
func (a *Assembler) pass1(lines []string) ([]parsedLine, error) {
	var program []parsedLine

	for lineNum, line := range lines {
		cleanLine := line
		// Strip comments
		if idx := strings.IndexAny(line, ";#"); idx != -1 {
			cleanLine = line[:idx]
		}
		cleanLine = strings.TrimSpace(cleanLine)
		if cleanLine == "" {
			continue
		}

		// Handle labels
		if idx := strings.Index(cleanLine, ":"); idx != -1 {
			labelName := strings.TrimSpace(cleanLine[:idx])
			if labelName == "" {
				return nil, fmt.Errorf("line %d: empty label name", lineNum+1)
			}
			a.labels[strings.ToLower(labelName)] = a.currentAddress
			cleanLine = strings.TrimSpace(cleanLine[idx+1:])
			if cleanLine == "" {
				continue
			}
		}

		// Parse mnemonic and arguments
		parts := strings.Fields(cleanLine)
		if len(parts) == 0 {
			continue
		}

		mnemonic := strings.ToUpper(parts[0])
		var args []string

		if len(parts) > 1 {
			argString := strings.Join(parts[1:], " ")
			for _, arg := range strings.Split(argString, ",") {
				arg = strings.TrimSpace(arg)
				if arg != "" {
					args = append(args, arg)
				}
			}
		}

		size := a.instructionSize(mnemonic)

		program = append(program, parsedLine{
			originalLine: line,
			mnemonic:     mnemonic,
			args:         args,
			address:      a.currentAddress,
			size:         size,
		})

		a.currentAddress += size
	}

	return program, nil
}

// pass2 translates parsed lines into Tryte bytecode.
func (a *Assembler) pass2(program []parsedLine) ([]tryte.Tryte, error) {
	var bytecode []tryte.Tryte

	for i, pl := range program {
		insts, err := a.assembleLine(pl, i+1)
		if err != nil {
			return nil, err
		}
		bytecode = append(bytecode, insts...)
	}

	return bytecode, nil
}

// instructionSize returns the memory size (in trytes) of the given instruction.
func (a *Assembler) instructionSize(mnemonic string) int {
	if mnemonic == "LDI" {
		return 2
	}
	return 1
}

// assembleLine compiles a single parsed line into one or two trytes.
func (a *Assembler) assembleLine(pl parsedLine, lineNum int) ([]tryte.Tryte, error) {
	switch pl.mnemonic {
	case "HALT":
		if len(pl.args) != 0 {
			return nil, fmt.Errorf("line %d: HALT expects 0 arguments, got %d", lineNum, len(pl.args))
		}
		return []tryte.Tryte{a.encode(cpu.OpHalt, 0, 0)}, nil

	case "RET":
		if len(pl.args) != 0 {
			return nil, fmt.Errorf("line %d: RET expects 0 arguments, got %d", lineNum, len(pl.args))
		}
		return []tryte.Tryte{a.encode(cpu.OpRet, 0, 0)}, nil

	case "ADD", "SUB", "MUL", "CMP", "LD", "AND", "OR", "SHL", "SHR", "DIV", "MOD":
		if len(pl.args) != 2 {
			return nil, fmt.Errorf("line %d: %s expects 2 arguments, got %d", lineNum, pl.mnemonic, len(pl.args))
		}
		src, err := a.parseReg(pl.args[0])
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNum, err)
		}
		dst, err := a.parseReg(pl.args[1])
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNum, err)
		}

		op := a.opForMnemonic(pl.mnemonic)
		return []tryte.Tryte{a.encode(op, src, dst)}, nil

	case "ST":
		if len(pl.args) != 2 {
			return nil, fmt.Errorf("line %d: ST expects 2 arguments, got %d", lineNum, len(pl.args))
		}
		valReg, err := a.parseReg(pl.args[0])
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNum, err)
		}
		addrReg, err := a.parseReg(pl.args[1])
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNum, err)
		}
		return []tryte.Tryte{a.encode(cpu.OpSt, addrReg, valReg)}, nil

	case "LDI":
		if len(pl.args) != 2 {
			return nil, fmt.Errorf("line %d: LDI expects 2 arguments, got %d", lineNum, len(pl.args))
		}
		dst, err := a.parseReg(pl.args[1])
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNum, err)
		}

		// First word: instruction header
		inst := a.encode(cpu.OpLdi, 0, dst)

		// Second word: immediate value or label address
		immStr := pl.args[0]
		var val int
		if labelAddr, isLabel := a.labels[strings.ToLower(immStr)]; isLabel {
			val = labelAddr
		} else {
			parsedVal, err := strconv.Atoi(immStr)
			if err != nil {
				return nil, fmt.Errorf("line %d: invalid immediate value/label: %s", lineNum, immStr)
			}
			val = parsedVal
		}

		return []tryte.Tryte{inst, tryte.FromInt(val)}, nil

	case "JMP", "JEQ", "JNE", "JLT", "JGT", "CALL":
		if len(pl.args) != 1 {
			return nil, fmt.Errorf("line %d: %s expects 1 argument, got %d", lineNum, pl.mnemonic, len(pl.args))
		}
		src, err := a.parseReg(pl.args[0])
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNum, err)
		}

		op := a.opForMnemonic(pl.mnemonic)
		return []tryte.Tryte{a.encode(op, src, 0)}, nil

	default:
		return nil, fmt.Errorf("line %d: unknown instruction mnemonic: %s", lineNum, pl.mnemonic)
	}
}

// opForMnemonic maps a mnemonic to its corresponding opcode.
func (a *Assembler) opForMnemonic(mnemonic string) int {
	switch mnemonic {
	case "ADD":
		return cpu.OpAdd
	case "SUB":
		return cpu.OpSub
	case "MUL":
		return cpu.OpMul
	case "CMP":
		return cpu.OpCmp
	case "LD":
		return cpu.OpLd
	case "JMP":
		return cpu.OpJmp
	case "JEQ":
		return cpu.OpJeq
	case "JNE":
		return cpu.OpJne
	case "JLT":
		return cpu.OpJlt
	case "JGT":
		return cpu.OpJgt
	case "CALL":
		return cpu.OpCall
	case "RET":
		return cpu.OpRet
	case "AND":
		return cpu.OpAnd
	case "OR":
		return cpu.OpOr
	case "SHL":
		return cpu.OpShl
	case "SHR":
		return cpu.OpShr
	case "DIV":
		return cpu.OpDiv
	case "MOD":
		return cpu.OpMod
	default:
		return 0
	}
}

// parseReg parses register names to register indices.
func (a *Assembler) parseReg(s string) (int, error) {
	s = strings.ToUpper(s)
	if !strings.HasPrefix(s, "R") || len(s) < 2 {
		return 0, fmt.Errorf("invalid register name: %s", s)
	}
	idx, err := strconv.Atoi(s[1:])
	if err != nil || idx < 0 || idx > 8 {
		return 0, fmt.Errorf("invalid register index: %s (must be R0-R8)", s)
	}
	return idx, nil
}

// encode packs the instruction opcode, source, and destination registers using ternary base-3 math.
func (a *Assembler) encode(op, src, dst int) tryte.Tryte {
	return tryte.FromInt(src + dst*27 + op*729)
}
