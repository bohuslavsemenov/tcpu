package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/bohuslavsemenov/tcpu/pkg/assembler"
	"github.com/bohuslavsemenov/tcpu/pkg/cpu"
	"github.com/bohuslavsemenov/tcpu/pkg/tryte"
)

var rawState string

func makeRaw() {
	// On macOS, -f /dev/stdin forces stty to query/modify stdin explicitly
	cmd := exec.Command("stty", "-f", "/dev/stdin", "-g")
	cmd.Stdin = os.Stdin
	out, err := cmd.Output()
	if err == nil {
		rawState = strings.TrimSpace(string(out))
		rawCmd := exec.Command("stty", "-f", "/dev/stdin", "raw", "-echo")
		rawCmd.Stdin = os.Stdin
		_ = rawCmd.Run()
	}
}

func restore() {
	if rawState != "" {
		cmd := exec.Command("stty", "-f", "/dev/stdin", rawState)
		cmd.Stdin = os.Stdin
		_ = cmd.Run()
	}
}

func fatalf(format string, v ...interface{}) {
	restore()
	log.Fatalf(format, v...)
}

type NonBlockingConsoleReader struct {
	stdinChan chan byte
}

func NewNonBlockingConsoleReader() *NonBlockingConsoleReader {
	r := &NonBlockingConsoleReader{
		stdinChan: make(chan byte, 100),
	}
	go func() {
		var buf [1]byte
		for {
			n, err := os.Stdin.Read(buf[:])
			if err != nil || n == 0 {
				close(r.stdinChan)
				break
			}
			r.stdinChan <- buf[0]
		}
	}()
	return r
}

func (r *NonBlockingConsoleReader) Read(p []byte) (int, error) {
	// Sleep for 150ms game tick
	time.Sleep(150 * time.Millisecond)

	// Read last key from channel
	var lastKey byte = 0
	hasKeys := false
	for {
		select {
		case b, ok := <-r.stdinChan:
			if !ok {
				if hasKeys {
					p[0] = lastKey
					return 1, nil
				}
				p[0] = 0
				return 1, nil
			}
			lastKey = b
			hasKeys = true
		default:
			if hasKeys {
				p[0] = lastKey
				return 1, nil
			}
			p[0] = 0
			return 1, nil
		}
	}
}

func main() {
	debugFlag := flag.Bool("d", false, "enable step-by-step CPU debug logging to stderr")
	rawFlag := flag.Bool("r", false, "enable raw terminal mode and non-blocking input for games (e.g., Snake)")
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		fmt.Printf("Usage: %s [-d] [-r] <path-to-assembly-file>\n", os.Args[0])
		os.Exit(1)
	}

	filePath := args[0]
	sourceBytes, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatalf("failed to read assembly file: %v", err)
	}

	tcpu := &cpu.CPU{}
	tcpu.Reset()

	// Auto-enable raw mode for snake, or if raw flag is specified
	useRaw := *rawFlag || strings.Contains(strings.ToLower(filePath), "snake")

	if useRaw {
		makeRaw()
		defer restore()
		tcpu.Stdin = NewNonBlockingConsoleReader()
	} else {
		tcpu.Stdin = os.Stdin
	}

	bytecode, err := assembler.Assemble(string(sourceBytes))
	if err != nil {
		fatalf("failed to assemble source: %v", err)
	}

	// Write program to CPU Memory starting at address 0
	for i, val := range bytecode {
		err := tcpu.WriteMem(tryte.FromInt(i), val)
		if err != nil {
			fatalf("failed to write instruction to memory: %v", err)
		}
	}

	stepCount := 0
	for {
		stepCount++
		currentPC := tcpu.PC.ToInt()

		keepRunning, err := tcpu.Step()
		if err != nil {
			fatalf("Execution error at Step %d (PC: %d): %v", stepCount, currentPC, err)
		}

		if *debugFlag {
			fmt.Fprintf(os.Stderr, "Step %2d | PC: %3d -> %3d | R0: %3d | R1: %3d | R2: %3d | R3: %3d | R4: %3d | R5: %3d\n",
				stepCount, currentPC, tcpu.PC.ToInt(),
				tcpu.R[0].ToInt(), tcpu.R[1].ToInt(), tcpu.R[2].ToInt(),
				tcpu.R[3].ToInt(), tcpu.R[4].ToInt(), tcpu.R[5].ToInt())
		}

		if !keepRunning {
			if *debugFlag {
				fmt.Fprintln(os.Stderr, "-----------------------------------------------------")
				fmt.Fprintln(os.Stderr, "CPU halted.")
				fmt.Fprintln(os.Stderr, "Final Register State:")
				for i, r := range tcpu.R {
					fmt.Fprintf(os.Stderr, "R%d: %5d\n", i, r.ToInt())
				}
				fmt.Fprintf(os.Stderr, "PC: %5d\n", tcpu.PC.ToInt())
			}
			break
		}
	}
}
