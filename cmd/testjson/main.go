package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func check(r io.Reader, name string) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	var v any
	err = json.Unmarshal(data, &v)
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	fmt.Println("OK:", name)
	return nil
}

func mains(args []string) error {
	if len(args) <= 0 {
		return check(os.Stdin, "<STDIN>")
	}
	for _, arg := range args {
		if arg == "-" {
			if err := check(os.Stdin, "<STDIN>"); err != nil {
				return err
			}
			continue
		}
		fnames, err := filepath.Glob(arg)
		if err != nil {
			fnames = []string{arg}
		}
		for _, fname1 := range fnames {
			fd, err := os.Open(fname1)
			if err != nil {
				return err
			}
			if err = check(fd, fname1); err != nil {
				fmt.Println("NG:", err.Error())
			}
			err1 := fd.Close()
			if err1 != nil {
				return err1
			}
		}
	}
	return nil
}

func main() {
	if err := mains(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
