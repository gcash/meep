package main

import (
	"strings"
	"testing"

	"github.com/gcash/bchd/txscript"
	"github.com/gcash/bchd/wire"
)

// TestDescribeOpcode checks that every opcode the debugger can show has a
// description, so that opcodes added or renamed in bchd show up here.
func TestDescribeOpcode(t *testing.T) {
	shown := make(map[string]bool)
	for v := range 256 {
		op := byte(v)
		script := []byte{op}
		switch {
		case op >= txscript.OP_DATA_1 && op <= txscript.OP_DATA_75:
			script = append(script, make([]byte, op)...)
		case op == txscript.OP_PUSHDATA1:
			script = append(script, 0)
		case op == txscript.OP_PUSHDATA2:
			script = append(script, 0, 0)
		case op == txscript.OP_PUSHDATA4:
			script = append(script, 0, 0, 0, 0)
		}
		// Read the name the way the debugger does.
		pc := nextOp(t, newStepper(newTestEngine(t, wire.TxOut{PkScript: script})))
		name := strings.Fields(strings.Split(pc, ":")[2])[0]
		shown[name] = true
		if describeOpcode(name) == "" {
			t.Errorf("%s (0x%02x) has no description", name, op)
		}
	}
	for name := range opcodeDescriptions {
		if !shown[name] {
			t.Errorf("%s is described, but the debugger never shows that name", name)
		}
	}

	for name, want := range map[string]string{
		"OP_DATA_1":  "Push the next byte onto the stack.",
		"OP_DATA_20": "Push the next 20 bytes onto the stack.",
	} {
		if got := describeOpcode(name); got != want {
			t.Errorf("%s: got %q, want %q", name, got, want)
		}
	}
}
