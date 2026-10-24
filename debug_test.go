package main

import (
	"reflect"
	"strings"
	"testing"

	"github.com/gcash/bchd/txscript"
)

// TestRedeemScriptOps checks that a P2SH scriptSig without a usable redeem
// script is shown rather than ending the debugger (issue #21).
func TestRedeemScriptOps(t *testing.T) {
	if got := redeemScriptOps(nil); len(got) != 1 || !strings.HasPrefix(got[0], "(missing: ") {
		t.Errorf("empty scriptSig: got %q, want a note that the redeem script is missing", got)
	}

	tests := []struct {
		name      string
		scriptSig []byte
		want      []string
	}{
		{
			name:      "redeem script",
			scriptSig: []byte{txscript.OP_DATA_3, txscript.OP_DUP, txscript.OP_DROP, txscript.OP_1},
			want:      []string{"OP_DUP", "OP_DROP", "1"},
		},
		{
			// The second instruction pushes 5 bytes, but only 1 is left.
			name:      "malformed redeem script",
			scriptSig: []byte{txscript.OP_DATA_3, txscript.OP_DUP, txscript.OP_DATA_5, 0xaa},
			want:      []string{"OP_DUP[error]"},
		},
	}
	for _, test := range tests {
		if got := redeemScriptOps(test.scriptSig); !reflect.DeepEqual(got, test.want) {
			t.Errorf("%s: got %q, want %q", test.name, got, test.want)
		}
	}
}
