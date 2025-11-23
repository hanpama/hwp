package main

import (
	"flag"
	"fmt"
	"os"

	hwpcat "github.com/hanpama/hwp"
)

func main() {
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "Usage: %s <hwp-file>\n", os.Args[0])
		os.Exit(1)
	}

	filename := flag.Arg(0)

	file, err := os.Open(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	if err := hwpcat.Read(file, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}
}
