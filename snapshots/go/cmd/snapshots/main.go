package main

import (
	"log"
	"os"
)

func main() {
	log.SetPrefix("snapshots: ")
	log.SetFlags(0) // don't print timestamps

	err := Execute(os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}
}
