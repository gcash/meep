# meep
Meep is a command line Bitcoin Cash script debugger

<img src="https://i.imgur.com/Xls1Km8.png">

### Install
```
go get github.com/gcash/meep
dep ensure
```

### Usage
```
Usage:
  meep [OPTIONS] debug [debug-OPTIONS]

Enter the script debugging mode

Application Options:
  -v, --version        Print the version number and exit

Help Options:
  -h, --help           Show this help message

[debug command options]
      -t, --tx=        the full transaction hex or BCH mainnet txid. If only a txid is provided the
                       transaction will be looked up via the RPC server.
      -i, --idx=       the input index to debug
      -a, --amt=       the amount of the input (in satoshis) we're debugging. This can be omitted if the
                       transaction is in the BCH blockchain as it will be looked up via the RPC server.
      -s, --pkscript=  the input's scriptPubkey. This can be omitted if the transaction is in the BCH
                       blockchain as it will be looked up via the RPC server.
          --rpcserver= A hostname:port for a gRPC API to use to fetch the transaction and scriptPubkey if
                       not providing through the options. (default: bchd.greyh.at:8335)

```

### Examples

```
// P2PKH
meep debug --tx=048e890c1931a8c3f908d5826943c47021c5bfebc6c8ef96684207c53cfa7ea3

// Tree signature
meep debug --tx=3332086562ccde663bec7928352098992b9e626bb8f1e95486d618219598c578

// Last will
meep debug --tx=b0f79eea9e05fe83e198efff4363e3e347d5efd77a22fbf39a09347162fb2560
```
