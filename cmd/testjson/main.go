package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

var (
	flagJavaScript = flag.Bool("js", false, "JavaScript-style assignments")
	flagDebug      = flag.Bool("debug", false, "for debug")
)

func readBody(r io.Reader) (data []byte, err error) {
	if *flagJavaScript {
		br := bufio.NewReader(r)
		data, err = br.ReadSlice('=')
		if err != nil {
			return
		}
		if *flagDebug {
			fmt.Fprintf(os.Stderr, "skip: %q\n", data)
		}
		r = br
	}
	data, err = io.ReadAll(r)
	return
}

func check(r io.Reader, name string) error {
	data, err := readBody(r)
	if err != nil && !errors.Is(err, io.EOF) {
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
	flag.Parse()
	if err := mains(flag.Args()); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
