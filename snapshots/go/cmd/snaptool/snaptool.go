// Command snaptool interacts with snapshots for Bazel projects.
// See "snaptool --help" for more information.
package main

import (
	"fmt"
	"log"
	"os"

	flag "github.com/spf13/pflag"
)

type command int

const (
	digestCmd command = iota
	collectCmd
	diffCmd
	getCmd
	pushCmd
	tagCmd
	helpCmd
)

var commandFromName = map[string]command{
	"collect": collectCmd,
	"diff":    diffCmd,
	"digest":  digestCmd,
	"get":     getCmd,
	"push":    pushCmd,
	"tag":     tagCmd,
	"help":    helpCmd,
}

func main() {
	log.SetPrefix("snaptool: ")
	log.SetFlags(0) // don't print timestamps

	if err := run(os.Args[1:]); err != nil && err != flag.ErrHelp {
		log.Fatal(err)
	}
}

func run(args []string) error {
	cmd := helpCmd
	if len(args) == 1 && (args[0] == "-h" || args[0] == "-help" || args[0] == "--help") {
		cmd = helpCmd
	} else if len(args) > 0 {
		c, ok := commandFromName[args[0]]
		if ok {
			cmd = c
			args = args[1:]
		}
	}

	switch cmd {
	case collectCmd:
		return runCollect(args)
	case diffCmd:
		return runDiff(args)
	case digestCmd:
		return runDigest(args)
	case getCmd:
		return runGet(args)
	case pushCmd:
		return runPush(args)
	case tagCmd:
		return runTag(args)
	case helpCmd:
		fallthrough
	default:
		return runHelp()
	}
}

func runHelp() error {
	fmt.Fprint(os.Stderr, `usage: snaptool <command> [args...]
Snaptool is a tool for creating and interacting with Snapshots for Bazel.
These snapshots are summaries of the outputs of a set of Bazel targets, and
can be used to check whether a target has changed.
Snaptool may be run with one of the commands below.
	digest - Create a digest of a set of files.
	collect - Create a snapshot from all tracker files.
	diff - Compute difference between two snapshots.
	get - Get a snapshot from the remote bucket.
	push - Push a snapshot to the remote bucket.
	tag - Tag a remote snapshot.
	help - Show this message
For usage information for a specific command, run the command with the -h flag.
For example:
	snaptool digest -h
Snaptool is under active development, and its interface may change without
notice.
`)
	return flag.ErrHelp
}
