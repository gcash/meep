package main

import (
	"fmt"
	"github.com/jessevdk/go-flags"
	"os"
)

const defaultRPCServer = "bchd.greyh.at:8335"

var (
	opts   Opts
	parser = flags.NewParser(&opts, flags.Default)
)

func main() {
	parser.AddCommand("debug",
		"Enter Debugger",
		"Enter the script debugging mode",
		&Debug{
			RPCServer: defaultRPCServer,
		})
	parser.AddCommand("execute",
		"Execute a script",
		"Execute a script, print the result and exit.",
		&Execute{
			RPCServer: defaultRPCServer,
		})
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Println(Version())
		return
	}
	if _, err := parser.Parse(); err != nil {
		os.Exit(1)
	}
}

// Opts holds the top level options which contains the version that we
// can print by using meep -v.
type Opts struct {
	Version bool `short:"v" long:"version" description:"Print the version number and exit"`
}
