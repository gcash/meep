package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/gcash/bchd/bchrpc/pb"
	"github.com/gcash/bchd/chaincfg/chainhash"
	"github.com/gcash/bchd/txscript"
	"github.com/gcash/bchd/wire"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// Execute holds the options to the execute command.
type Execute struct {
	Transaction  string `short:"t" long:"tx" description:"the full transaction hex or BCH mainnet txid. If only a txid is provided the transaction will be looked up via the RPC server."`
	InputIndex   int    `short:"i" long:"idx" description:"the input index to debug"`
	InputAmount  int64  `short:"a" long:"amt" description:"the amount of the input (in satoshis) we're debugging. This can be omitted if the transaction is in the BCH blockchain as it will be looked up via the RPC server."`
	ScriptPubkey string `short:"s" long:"pkscript" description:"the input's scriptPubkey. This can be omitted if the transaction is in the BCH blockchain as it will be looked up via the RPC server."`
	RPCServer    string `long:"rpcserver" description:"A hostname:port for a gRPC API to use to fetch the transaction and scriptPubkey if not providing through the options."`
}

// Execute will run the Execute command. This executes the script, prints
// the result and exists.
func (x *Execute) Execute(_ []string) error {
	var (
		txBytes      []byte
		scriptPubkey []byte
		client       pb.BchrpcClient
		err          error
	)

	if txid, err := chainhash.NewHashFromStr(x.Transaction); err == nil {
		conn, err := grpc.Dial(x.RPCServer, grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, "")))
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

	if x.ScriptPubkey == "" {
		if client == nil {
			conn, err := grpc.Dial(x.RPCServer, grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, "")))
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
			conn, err := grpc.Dial(x.RPCServer, grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, "")))
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
			conn, err := grpc.Dial(x.RPCServer, grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, "")))
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

	flags := txscript.StandardVerifyFlags | txscript.ScriptAllowCashTokens
	vm, err := txscript.NewEngine(scriptPubkey, tx, x.InputIndex, flags, nil, nil, utxoCache, x.InputAmount)
	if err != nil {
		return err
	}
	if err := vm.Execute(); err != nil {
		return err
	}

	fmt.Println("Success!!!")

	return nil
}
