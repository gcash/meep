package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	tm "github.com/buger/goterm"
	"github.com/gcash/bchd/bchrpc/pb"
	"github.com/gcash/bchd/chaincfg"
	"github.com/gcash/bchd/chaincfg/chainhash"
	"github.com/gcash/bchd/txscript"
	"github.com/gcash/bchd/wire"
	term "github.com/nsf/termbox-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"strings"
	"time"
)

// Debug holds the options to the debug command.
type Debug struct {
	Transaction  string `short:"t" long:"tx" description:"the full transaction hex or BCH mainnet txid. If only a txid is provided the transaction will be looked up via the RPC server."`
	InputIndex   int    `short:"i" long:"idx" description:"the input index to debug"`
	InputAmount  int64  `short:"a" long:"amt" description:"the amount of the input (in satoshis) we're debugging. This can be omitted if the transaction is in the BCH blockchain as it will be looked up via the RPC server."`
	ScriptPubkey string `short:"s" long:"pkscript" description:"the input's scriptPubkey. This can be omitted if the transaction is in the BCH blockchain as it will be looked up via the RPC server."`
	RPCServer    string `long:"rpcserver" description:"A hostname:port for a gRPC API to use to fetch the transaction and scriptPubkey if not providing through the options."`
}

// Execute will run the Debug command. This drops into the terminal debugger and allows
// us to step forward and backwards.
func (x *Debug) Execute(args []string) error {
	var (
		txBytes      []byte
		scriptPubkey []byte
		client       pb.BchrpcClient
		err          error
		done         bool
		fail         bool
	)

	err = term.Init()
	if err != nil {
		panic(err)
	}

	defer term.Close()

	if txid, err := chainhash.NewHashFromStr(x.Transaction); err == nil {
		conn, err := grpc.Dial(x.RPCServer, grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, "")))
		if err != nil {
			return err
		}

		client = pb.NewBchrpcClient(conn)

		ctx, _ := context.WithTimeout(context.Background(), time.Second*10)
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

	if x.ScriptPubkey == "" {
		if client == nil {
			conn, err := grpc.Dial(x.RPCServer, grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, "")))
			if err != nil {
				return err
			}

			client = pb.NewBchrpcClient(conn)
		}

		ctx, _ := context.WithTimeout(context.Background(), time.Second*10)
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
			conn, err := grpc.Dial(x.RPCServer, grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, "")))
			if err != nil {
				return err
			}

			client = pb.NewBchrpcClient(conn)
		}

		ctx, _ := context.WithTimeout(context.Background(), time.Second*10)
		resp, err := client.GetTransaction(ctx, &pb.GetTransactionRequest{
			Hash: tx.TxIn[x.InputIndex].PreviousOutPoint.Hash[:],
		})
		if err != nil {
			return err
		}
		x.InputAmount = resp.Transaction.Outputs[tx.TxIn[x.InputIndex].PreviousOutPoint.Index].Value
	}

	vm, err := txscript.NewEngine(scriptPubkey, tx, x.InputIndex, txscript.StandardVerifyFlags, nil, nil, x.InputAmount)
	if err != nil {
		return err
	}
	vm.SetDebugMode()

	scriptClass, _, _, err := txscript.ExtractPkScriptAddrs(scriptPubkey, &chaincfg.MainNetParams)
	if err != nil {
		return err
	}
	isP2SH := scriptClass == txscript.ScriptHashTy

	scriptSig := tx.TxIn[x.InputIndex].SignatureScript

	disassembledScriptSig, err := txscript.DisasmString(scriptSig)
	if err != nil {
		return err
	}
	disassembledScriptPubkey, err := txscript.DisasmString(scriptPubkey)
	if err != nil {
		return err
	}

scriptLoop:
	tm.Println(tm.Background(tm.Color(tm.Bold(fmt.Sprintf("%s%s", "Debugger", strings.Repeat(" ", tm.Width()-8))), tm.BLACK), tm.GREEN))
	tm.Flush()
	var (
		dissasm              string
		scriptIdx, offsetIdx int
	)
	if !done && !fail {
		dissasm, err = vm.DisasmPC()
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
	}

	splitDisassembledScriptSig := strings.Split(disassembledScriptSig, " ")
	tm.Println(tm.Background(tm.Color(tm.Bold("ScriptSig"), tm.WHITE), tm.BLUE))
	for i, op := range splitDisassembledScriptSig {
		if scriptIdx == 0 && offsetIdx == i && !done {
			tm.Printf(tm.Background(tm.Color(tm.Bold("%s"), tm.BLACK), tm.WHITE), op)
			tm.Printf(" ")
		} else {
			tm.Printf("%s ", op)
		}
	}
	tm.Printf("\n\n")

	splitDisassembledScriptPubkey := strings.Split(disassembledScriptPubkey, " ")
	tm.Println(tm.Background(tm.Color(tm.Bold("ScriptPubkey"), tm.WHITE), tm.BLUE))
	for i, op := range splitDisassembledScriptPubkey {
		if scriptIdx == 1 && offsetIdx == i {
			tm.Printf(tm.Background(tm.Color(tm.Bold("%s"), tm.BLACK), tm.WHITE), op)
			tm.Printf(" ")
		} else {
			unexecuted := false
			for _, unex := range vm.GetUnexecutedIndexes() {
				if unex[0] == 1 && unex[1] == i {
					tm.Printf(tm.Background(tm.Color(tm.Bold("%s"), tm.RED), tm.BLACK), op)
					tm.Printf(" ")
					unexecuted = true
				}
			}
			if !unexecuted {
				tm.Printf("%s ", op)
			}
		}
	}
	tm.Printf("\n\n")

	if isP2SH {
		redeemScript, err := txscript.ExtractRedeemScript(scriptSig)
		if err != nil {
			return err
		}
		disassembledRedeemScript, err := txscript.DisasmString(redeemScript)
		if err != nil {
			return err
		}
		splitDisassembledRedeemScript := strings.Split(disassembledRedeemScript, " ")
		tm.Println(tm.Background(tm.Color(tm.Bold("RedeemScript"), tm.WHITE), tm.BLUE))
		for i, op := range splitDisassembledRedeemScript {
			if scriptIdx == 2 && offsetIdx == i {
				tm.Printf(tm.Background(tm.Color(tm.Bold("%s"), tm.BLACK), tm.WHITE), op)
				tm.Printf(" ")
			} else {
				unexecuted := false
				for _, unex := range vm.GetUnexecutedIndexes() {
					if unex[0] == 2 && unex[1] == i {
						tm.Printf(tm.Background(tm.Color(tm.Bold("%s"), tm.RED), tm.BLACK), op)
						tm.Printf(" ")
						unexecuted = true
					}
				}
				if !unexecuted {
					tm.Printf("%s ", op)
				}
			}
		}
		tm.Printf("\n\n")
	}

	fmt.Println()

	tm.Println(tm.Background(tm.Color(tm.Bold("Next Instruction"), tm.WHITE), tm.CYAN))
	tm.Printf("%s\n\n", dissasm)

	tm.Println(tm.Background(tm.Color(tm.Bold("Stack"), tm.WHITE), tm.MAGENTA))
	var box *tm.Box
	if done {
		box = tm.NewBox(100|tm.PCT, 3, 0)
		fmt.Fprintf(box, "%s\n", "Success!!!")
	} else if fail {
		box = tm.NewBox(100|tm.PCT, 3, 0)
		fmt.Fprintf(box, "%s %s\n", "Fail :(", err)
	} else {
		box = tm.NewBox(100|tm.PCT, len(vm.GetStack())+2, 0)
		stack := vm.GetStack()
		for i := len(stack) - 1; i >= 0; i-- {
			fmt.Fprintf(box, "%s\n", hex.EncodeToString(stack[i]))
		}
	}

	tm.Println(box.String())

	tm.Printf("%s%s%s%s%s%s\n", "F3", tm.Background(tm.Color(tm.Bold("Step Back"), tm.WHITE), tm.CYAN), "F4", tm.Background(tm.Color(tm.Bold("Step Forward"), tm.WHITE), tm.CYAN), "ESC", tm.Background(tm.Color(tm.Bold("Quit"), tm.WHITE), tm.CYAN))
	tm.Flush()

	for {
		switch ev := term.PollEvent(); ev.Type {
		case term.EventKey:
			switch ev.Key {
			case term.KeyEsc:
				return nil
			case term.KeyF4:
				if done {
					return nil
				}
				term.Sync()
				done, err = vm.Step()
				if err != nil {
					done = true
					fail = true
				}
				goto scriptLoop
			case term.KeyF3:
				term.Sync()
				err := vm.StepBack()
				if err != nil {
					return err
				}
				goto scriptLoop
				break
			}
		}
	}

	return nil
}
