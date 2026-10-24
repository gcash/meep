package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"crypto/tls"
	tm "github.com/buger/goterm"
	"github.com/gcash/bchd/bchrpc/pb"
	"github.com/gcash/bchd/chaincfg"
	"github.com/gcash/bchd/chaincfg/chainhash"
	"github.com/gcash/bchd/txscript"
	"github.com/gcash/bchd/wire"
	term "github.com/nsf/termbox-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// Debug holds the options to the debug command.
type Debug struct {
	Transaction      string `short:"t" long:"tx" description:"the full transaction hex or BCH mainnet txid. If only a txid is provided the transaction will be looked up via the RPC server."`
	InputIndex       int    `short:"i" long:"idx" description:"the input index to debug"`
	InputAmount      int64  `short:"a" long:"amt" description:"the amount of the input (in satoshis) we're debugging. This can be omitted if the transaction is in the BCH blockchain as it will be looked up via the RPC server."`
	ScriptPubkey     string `short:"s" long:"pkscript" description:"the input's scriptPubkey. This can be omitted if the transaction is in the BCH blockchain as it will be looked up via the RPC server."`
	RPCServer        string `long:"rpcserver" description:"A hostname:port for a gRPC API to use to fetch the transaction and scriptPubkey if not providing through the options."`
	RPCAllowInsecure bool   `long:"rpcallowinsecure" description:"Allow insecure TLS (skip certificate validation)."`
}

func dial(x *Debug) (*grpc.ClientConn, error) {
	if x.RPCAllowInsecure {
		tlsCfg := &tls.Config{
			InsecureSkipVerify: true,
		}
		creds := credentials.NewTLS(tlsCfg)
		return grpc.NewClient(x.RPCServer, grpc.WithTransportCredentials(creds))
	}
	return grpc.NewClient(x.RPCServer, grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, "")))
}

// redeemScriptOps disassembles the redeem script that a P2SH scriptSig pushes,
// one entry per instruction. If the scriptSig pushes nothing, it returns a
// note saying so instead, and the engine reports the failure when it runs.
func redeemScriptOps(scriptSig []byte) []string {
	redeemScript, err := txscript.ExtractRedeemScript(scriptSig)
	if err != nil {
		return []string{fmt.Sprintf("(missing: %v)", err)}
	}
	// A malformed redeem script disassembles as far as it parses, followed by
	// [error]. The engine fails when it tries to run it.
	disassembled, _ := txscript.DisasmString(redeemScript)
	return strings.Split(disassembled, " ")
}

// Execute will run the Debug command. This drops into the terminal debugger and allows
// us to step forward and backwards.
func (x *Debug) Execute(_ []string) error {
	var (
		txBytes      []byte
		scriptPubkey []byte
		client       pb.BchrpcClient
		err          error
	)
	if tm.Width() <= 0 || tm.Height() <= 0 {
		return errNoTerminal
	}
	restore := enableVirtualTerminal()
	defer restore()

	err = term.Init()
	if err != nil {
		panic(err)
	}

	defer term.Close()

	if txid, err := chainhash.NewHashFromStr(x.Transaction); err == nil {
		conn, err := dial(x)
		if err != nil {
			return err
		}

		client = pb.NewBchrpcClient(conn)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		resp, err := client.GetRawTransaction(ctx, &pb.GetRawTransactionRequest{
			Hash: txid[:],
		})
		if err != nil {
			return err
		}
		txBytes = resp.Transaction
	} else {
		txBytes, err = hex.DecodeString(x.Transaction)
		if err != nil {
			return err
		}
	}

	tx := &wire.MsgTx{}
	if err := tx.BchDecode(bytes.NewReader(txBytes), wire.ProtocolVersion, wire.BaseEncoding); err != nil {
		return err
	}

	if len(tx.TxIn) == 0 {
		return errors.New("transaction has no inputs")
	}

	if x.InputIndex < 0 || x.InputIndex >= len(tx.TxIn) {
		return fmt.Errorf("input index %d is out of range: the transaction's last input is %d", x.InputIndex, len(tx.TxIn)-1)
	}

	if x.ScriptPubkey == "" {
		if client == nil {
			conn, err := dial(x)
			if err != nil {
				return err
			}

			client = pb.NewBchrpcClient(conn)
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		resp, err := client.GetTransaction(ctx, &pb.GetTransactionRequest{
			Hash: tx.TxIn[x.InputIndex].PreviousOutPoint.Hash[:],
		})
		if err != nil {
			return err
		}
		scriptPubkey = resp.Transaction.Outputs[tx.TxIn[x.InputIndex].PreviousOutPoint.Index].PubkeyScript
	} else {
		scriptPubkey, err = hex.DecodeString(x.ScriptPubkey)
		if err != nil {
			return err
		}
	}

	if x.InputAmount == 0 {
		if client == nil {
			conn, err := dial(x)
			if err != nil {
				return err
			}

			client = pb.NewBchrpcClient(conn)
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		resp, err := client.GetTransaction(ctx, &pb.GetTransactionRequest{
			Hash: tx.TxIn[x.InputIndex].PreviousOutPoint.Hash[:],
		})
		if err != nil {
			return err
		}
		x.InputAmount = resp.Transaction.Outputs[tx.TxIn[x.InputIndex].PreviousOutPoint.Index].Value
	}

	utxoCache := txscript.NewUtxoCache()
	for i, txIn := range tx.TxIn {
		if client == nil {
			conn, err := dial(x)
			if err != nil {
				return err
			}

			client = pb.NewBchrpcClient(conn)
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		resp, err := client.GetTransaction(ctx, &pb.GetTransactionRequest{
			Hash: txIn.PreviousOutPoint.Hash[:],
		})
		if err != nil {
			return err
		}
		output := resp.Transaction.Outputs[txIn.PreviousOutPoint.Index]

		txOut := wire.TxOut{}
		txOut.Value = output.Value
		txOut.PkScript = output.PubkeyScript
		txOut.TokenData = wire.TokenData{}
		if output.CashToken != nil {
			copy(txOut.TokenData.CategoryID[:], output.CashToken.CategoryId)
			txOut.TokenData.BitField = output.CashToken.Bitfield[0]
			txOut.TokenData.Amount = output.CashToken.Amount
			txOut.TokenData.Commitment = output.CashToken.Commitment
		}
		utxoCache.AddEntry(i, txOut)
	}

	vm, err := txscript.NewEngine(scriptPubkey, tx, x.InputIndex, engineFlags, nil, nil, utxoCache, x.InputAmount)
	if err != nil {
		return err
	}

	scriptClass, _, _, err := txscript.ExtractPkScriptAddrs(scriptPubkey, &chaincfg.MainNetParams)
	if err != nil {
		return err
	}
	isP2SH := (scriptClass == txscript.ScriptHashTy) || (scriptClass == txscript.ScriptHash32Ty)

	scriptSig := tx.TxIn[x.InputIndex].SignatureScript

	disassembledScriptSig, err := txscript.DisasmString(scriptSig)
	if err != nil {
		return err
	}
	disassembledScriptPubkey, err := txscript.DisasmString(scriptPubkey)
	if err != nil {
		return err
	}

	var (
		splitDisassembledRedeemScript []string
		unExecutedOpcodes             [][]int
	)
	splitDisassembledScriptSig := strings.Split(disassembledScriptSig, " ")
	splitDisassembledScriptPubkey := strings.Split(disassembledScriptPubkey, " ")
	if isP2SH {
		splitDisassembledRedeemScript = redeemScriptOps(scriptSig)
	}

	st := newStepper(vm)

	var fastForward = false
scriptLoop:
	tm.Clear()
	tm.MoveCursor(1, 1)
	_, _ = tm.Println(tm.Background(tm.Color(tm.Bold(fmt.Sprintf("%s%s", "Debugger", strings.Repeat(" ", max(tm.Width()-8, 0)))), tm.BLACK), tm.GREEN))
	_, _ = tm.Println()
	var (
		dissasm              string
		scriptIdx, offsetIdx int
	)
	if !st.done {
		dissasm, err = st.vm.DisasmPC()
		if err != nil {
			return err
		}
		s := strings.Split(dissasm, ":")
		scriptIdxBytes, err := hex.DecodeString(s[0])
		if err != nil {
			return err
		}
		scriptIdx = int(binary.BigEndian.Uint32(append([]byte{0x00, 0x00, 0x00}, scriptIdxBytes...)))
		offsetIdxBytes, err := hex.DecodeString(s[1])
		if err != nil {
			return err
		}
		offsetIdx = int(binary.BigEndian.Uint32(append([]byte{0x00, 0x00}, offsetIdxBytes...)))

		if !st.vm.IsBranchExecuting() && !strings.Contains(s[2], "OP_ENDIF") {
			contains := false
			for _, op := range unExecutedOpcodes {
				if op[0] == scriptIdx && op[1] == offsetIdx {
					contains = true
					break
				}
			}
			if !contains {
				unExecutedOpcodes = append(unExecutedOpcodes, []int{scriptIdx, offsetIdx})
			}
		}
	}

	_, _ = tm.Println(tm.Background(tm.Color(tm.Bold("ScriptSig"), tm.WHITE), tm.BLUE))
	for i, op := range splitDisassembledScriptSig {
		if scriptIdx == 0 && offsetIdx == i && !st.done {
			_, _ = tm.Printf(tm.Background(tm.Color(tm.Bold("%s"), tm.BLACK), tm.WHITE), op)
			_, _ = tm.Printf(" ")
		} else {
			_, _ = tm.Printf("%s ", op)
		}
	}
	_, _ = tm.Printf("\n\n")

	_, _ = tm.Println(tm.Background(tm.Color(tm.Bold("ScriptPubkey"), tm.WHITE), tm.BLUE))
	for i, op := range splitDisassembledScriptPubkey {
		if scriptIdx == 1 && offsetIdx == i {
			_, _ = tm.Printf(tm.Background(tm.Color(tm.Bold("%s"), tm.BLACK), tm.WHITE), op)
			_, _ = tm.Printf(" ")
		} else {
			unexecuted := false
			for _, unex := range unExecutedOpcodes {
				if unex[0] == 1 && unex[1] == i {
					_, _ = tm.Printf(tm.Background(tm.Color(tm.Bold("%s"), tm.RED), tm.BLACK), op)
					_, _ = tm.Printf(" ")
					unexecuted = true
				}
			}
			if !unexecuted {
				_, _ = tm.Printf("%s ", op)
			}
		}
	}
	_, _ = tm.Printf("\n\n")

	if isP2SH {
		_, _ = tm.Println(tm.Background(tm.Color(tm.Bold("RedeemScript"), tm.WHITE), tm.BLUE))
		for i, op := range splitDisassembledRedeemScript {
			if scriptIdx == 2 && offsetIdx == i {
				_, _ = tm.Printf(tm.Background(tm.Color(tm.Bold("%s"), tm.BLACK), tm.WHITE), op)
				_, _ = tm.Printf(" ")
			} else {
				unexecuted := false
				for _, unex := range unExecutedOpcodes {
					if unex[0] == 2 && unex[1] == i && scriptIdx >= unex[0] && offsetIdx >= unex[1] {
						_, _ = tm.Printf(tm.Background(tm.Color(tm.Bold("%s"), tm.RED), tm.BLACK), op)
						_, _ = tm.Printf(" ")
						unexecuted = true
					}
				}
				if !unexecuted {
					_, _ = tm.Printf("%s ", op)
				}
			}
		}
		_, _ = tm.Printf("\n\n")
	}

	_, _ = tm.Println(tm.Background(tm.Color(tm.Bold("Next Instruction"), tm.WHITE), tm.CYAN))
	_, _ = tm.Printf("%s\n\n", dissasm)

	_, _ = tm.Println(tm.Background(tm.Color(tm.Bold("Stack"), tm.WHITE), tm.MAGENTA))
	var box *tm.Box
	if st.done && st.err == nil {
		box = tm.NewBox(100|tm.PCT, 3, 0)
		_, _ = fmt.Fprintf(box, "%s\n", "Success!!!")
	} else if st.done {
		box = tm.NewBox(100|tm.PCT, 3, 0)
		_, _ = fmt.Fprintf(box, "%s %s\n", "Fail :(", st.err)
	} else {
		box = tm.NewBox(100|tm.PCT, len(st.vm.GetStack())+2, 0)
		stack := st.vm.GetStack()
		for i := len(stack) - 1; i >= 0; i-- {
			_, _ = fmt.Fprintf(box, "%s\n", hex.EncodeToString(stack[i]))
		}
	}

	_, _ = tm.Println(box.String())

	altstack := st.vm.GetAltStack()
	if len(altstack) > 0 && !st.done {
		_, _ = tm.Println(tm.Background(tm.Color(tm.Bold("Alt Stack"), tm.WHITE), tm.MAGENTA))
		box := tm.NewBox(100|tm.PCT, len(altstack)+2, 0)
		for i := len(altstack) - 1; i >= 0; i-- {
			_, _ = fmt.Fprintf(box, "%s\n", hex.EncodeToString(altstack[i]))
		}
		_, _ = tm.Println(box.String())
	}

	footer := "F3/Left/p" + tm.Background(tm.Color(tm.Bold("Step Back"), tm.WHITE), tm.CYAN)
	if !st.done {
		footer += "F4/Right/n" + tm.Background(tm.Color(tm.Bold("Step Forward"), tm.WHITE), tm.CYAN)
	}
	footer += "ESC/q" + tm.Background(tm.Color(tm.Bold("Quit"), tm.WHITE), tm.CYAN)
	_, _ = tm.Println(footer)
	flush()

	if fastForward {
		st.next()
		fastForward = !st.done && !st.vm.IsBranchExecuting()
		goto scriptLoop
	}

	for {
		switch ev := term.PollEvent(); ev.Type {
		case term.EventKey:
			switch {
			case ev.Key == term.KeyEsc || ev.Ch == 'q':
				return nil
			case ev.Key == term.KeyF4 || ev.Key == term.KeyArrowRight || ev.Ch == 'n':
				if st.done {
					// Stay on the result. Stepping back and quitting still work.
					continue
				}
				st.next()
				fastForward = !st.done && !st.vm.IsBranchExecuting()
				goto scriptLoop
			case ev.Key == term.KeyF3 || ev.Key == term.KeyArrowLeft || ev.Ch == 'p':
				st.back()
				goto scriptLoop
			}
		}
	}
}
