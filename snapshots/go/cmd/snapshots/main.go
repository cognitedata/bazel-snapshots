package main

import (
	"log"
	"os"
)

func main() {
	log.SetPrefix("snapshots: ")
	log.SetFlags(0) // don't print timestamps

	Execute(os.Args[1:])
}
