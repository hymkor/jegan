package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

func main1(r io.Reader, name string) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	var v any
	err = json.Unmarshal(data, &v)
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	fmt.Println("OK")
	return nil
}

func mains(args []string) error {
	if len(args) > 0 {
		fd, err := os.Open(args[0])
		if err != nil {
			return err
		}
		defer fd.Close()
		return main1(fd, args[0])
	}
	return main1(os.Stdin, "<STDIN>")
}

func main() {
	if err := mains(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
