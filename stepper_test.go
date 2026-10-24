package main

import (
	"errors"
	"strings"
	"testing"

	"github.com/gcash/bchd/txscript"
	"github.com/gcash/bchd/wire"
)

// newTestEngine returns an engine for the only input of a transaction, which
// spends prevOut. The input's scriptSig is empty, so only prevOut's script
// runs.
func newTestEngine(t *testing.T, prevOut wire.TxOut) *txscript.Engine {
	t.Helper()
	tx := wire.NewMsgTx(2)
	tx.AddTxIn(wire.NewTxIn(&wire.OutPoint{}, nil))
	utxoCache := txscript.NewUtxoCache()
	utxoCache.AddEntry(0, prevOut)
	vm, err := txscript.NewEngine(prevOut.PkScript, tx, 0, engineFlags, nil, nil, utxoCache, prevOut.Value)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	return vm
}

// runToEnd steps until execution finishes and returns the number of steps.
func runToEnd(t *testing.T, st *stepper) int {
	t.Helper()
	for steps := 1; steps <= 1000; steps++ {
		st.next()
		if st.done {
			return steps
		}
	}
	t.Fatal("script did not finish")
	return 0
}

// nextOp returns the disassembly of the instruction that runs next.
func nextOp(t *testing.T, st *stepper) string {
	t.Helper()
	pc, err := st.vm.DisasmPC()
	if err != nil {
		t.Fatalf("DisasmPC: %v", err)
	}
	return pc
}

// TestStepperStaysOnResult checks that stepping forward at the end keeps the
// result (issue #18), and that stepping back returns to the final instruction.
func TestStepperStaysOnResult(t *testing.T) {
	script := []byte{txscript.OP_1, txscript.OP_1, txscript.OP_ADD, txscript.OP_2, txscript.OP_EQUAL}
	st := newStepper(newTestEngine(t, wire.TxOut{Value: 1000, PkScript: script}))
	runToEnd(t, st)
	if st.err != nil {
		t.Fatalf("script failed: %v", st.err)
	}

	saved := len(st.history)
	st.next()
	if !st.done || st.err != nil || len(st.history) != saved {
		t.Fatalf("next after the end changed the result: done %v, err %v", st.done, st.err)
	}

	st.back()
	if st.done || st.err != nil {
		t.Fatalf("back did not clear the result: done %v, err %v", st.done, st.err)
	}
	if pc := nextOp(t, st); !strings.HasSuffix(pc, "OP_EQUAL") {
		t.Fatalf("back went to %q, want the final OP_EQUAL", pc)
	}
	st.next()
	if !st.done || st.err != nil {
		t.Fatalf("rerunning the final instruction: done %v, err %v", st.done, st.err)
	}
}

// TestStepperKeepsFirstError checks that a failure stays on screen when
// stepping forward again, and that stepping back clears it. Step reports some
// failures as finished and others as not, so both kinds are covered.
func TestStepperKeepsFirstError(t *testing.T) {
	tests := []struct {
		name   string
		script []byte
		code   txscript.ErrorCode
		failOp string
	}{
		{
			name:   "failed opcode",
			script: []byte{txscript.OP_0, txscript.OP_VERIFY},
			code:   txscript.ErrVerify,
			failOp: "OP_VERIFY",
		},
		{
			name:   "unbalanced conditional",
			script: []byte{txscript.OP_1, txscript.OP_IF, txscript.OP_1},
			code:   txscript.ErrUnbalancedConditional,
			failOp: "OP_1",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			st := newStepper(newTestEngine(t, wire.TxOut{Value: 1000, PkScript: test.script}))
			runToEnd(t, st)
			var scriptErr txscript.Error
			if !errors.As(st.err, &scriptErr) || scriptErr.ErrorCode != test.code {
				t.Fatalf("got error %v, want %v", st.err, test.code)
			}

			first := st.err
			st.next()
			if st.err != first {
				t.Fatalf("error changed from %v to %v", first, st.err)
			}

			st.back()
			if st.done || st.err != nil {
				t.Fatalf("back did not clear the failure: done %v, err %v", st.done, st.err)
			}
			if pc := nextOp(t, st); !strings.HasSuffix(pc, test.failOp) {
				t.Fatalf("back went to %q, want the failing %s", pc, test.failOp)
			}
		})
	}
}

// TestStepperLoop checks that a loop can take more steps than the script has
// instructions.
func TestStepperLoop(t *testing.T) {
	// Count from 0 to 5.
	script := []byte{
		txscript.OP_0,
		txscript.OP_BEGIN,
		txscript.OP_1ADD, txscript.OP_DUP, txscript.OP_5, txscript.OP_NUMEQUAL,
		txscript.OP_UNTIL,
		txscript.OP_5, txscript.OP_NUMEQUAL,
	}
	st := newStepper(newTestEngine(t, wire.TxOut{Value: 1000, PkScript: script}))
	steps := runToEnd(t, st)
	if st.err != nil {
		t.Fatalf("script failed: %v", st.err)
	}
	if steps <= len(script) {
		t.Fatalf("took %d steps, want more than the %d instructions", steps, len(script))
	}
	if len(st.history) != steps {
		t.Fatalf("history has %d states, want %d", len(st.history), steps)
	}
}

// TestStepperBackKeepsUtxos checks that stepping back keeps the spent outputs,
// which OP_UTXOTOKENCATEGORY reads (issue #19).
func TestStepperBackKeepsUtxos(t *testing.T) {
	script := []byte{txscript.OP_0, txscript.OP_UTXOTOKENCATEGORY, txscript.OP_DROP, txscript.OP_1}
	prevOut := wire.TxOut{
		Value:    1000,
		PkScript: script,
		TokenData: wire.TokenData{
			CategoryID: [32]byte{1},
			Amount:     100,
			BitField:   wire.HAS_AMOUNT,
		},
	}
	st := newStepper(newTestEngine(t, prevOut))
	st.next()
	st.next()
	st.back()
	st.next()
	if st.err != nil {
		t.Fatalf("OP_UTXOTOKENCATEGORY after stepping back: %v", st.err)
	}
	if stack := st.vm.GetStack(); len(stack) != 1 || len(stack[0]) != 32 {
		t.Fatalf("stack after OP_UTXOTOKENCATEGORY is %x, want the 32 byte token category", stack)
	}
	runToEnd(t, st)
	if st.err != nil {
		t.Fatalf("script failed: %v", st.err)
	}
}
