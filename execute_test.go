package main

import (
	"bytes"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/gcash/bchd/wire"
)

// TestExecuteInputIndexOutOfRange checks that an input index outside the
// transaction is an error rather than a crash.
func TestExecuteInputIndexOutOfRange(t *testing.T) {
	tx := wire.NewMsgTx(2)
	tx.AddTxIn(wire.NewTxIn(&wire.OutPoint{}, nil))
	var buf bytes.Buffer
	if err := tx.BchEncode(&buf, wire.ProtocolVersion, wire.BaseEncoding); err != nil {
		t.Fatal(err)
	}

	for _, idx := range []int{-1, 1} {
		cmd := &Execute{Transaction: hex.EncodeToString(buf.Bytes()), InputIndex: idx}
		if err := cmd.Execute(nil); err == nil || !strings.Contains(err.Error(), "out of range") {
			t.Errorf("input index %d: got error %v, want out of range", idx, err)
		}
	}
}
