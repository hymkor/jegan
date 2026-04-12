package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/hymkor/jegan"
)

func main() {
	cfg := &jegan.Config{}
	flag.StringVar(&cfg.Auto, "auto", "", "automate key and readline inputs (e.g. \"w|-|q\")")
	flag.Parse()
	if err := cfg.Run(flag.Args()); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
