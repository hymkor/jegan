package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mattn/go-colorable"
	"github.com/mattn/go-isatty"

	"github.com/nyaosorg/go-ttyadapter/tty8pe"
	"github.com/nyaosorg/go-windows-dbg"

	"github.com/hymkor/jegan/internal/unjson"
)

func debug(v ...any) {
	if false {
		dbg.Println(v...)
	}
}

func main1(r io.Reader, name string) error {
	closer := func() error { return nil }
	if c, ok := r.(io.Closer); ok {
		closer = c.Close
	}
	br, ok := r.(io.RuneScanner)
	if !ok {
		br = bufio.NewReader(r)
	}
	app := &Application{Name: name}
	defer app.Close()

	for {
		v, err := unjson.Unmarshal(br)
		if err != nil {
			var e *unjson.ErrTrailingData
			if errors.As(err, &e) {
				err = e.Err
				app.Trailing = e.Trailing
			}
			if errors.Is(err, io.EOF) {
				app.Store(Read(v))
				if err := closer(); err != nil {
					return err
				}
				ttyout := colorable.NewColorableStdout()
				return app.EventLoop(&tty8pe.Tty{}, ttyout)
			}
			if name == "" {
				return fmt.Errorf("<STDIN>:%w", err)
			}
			return fmt.Errorf("%s:%w", name, err)
		}
		app.Store(Read(v))
	}
}

func mains(args []string) error {
	disable := colorable.EnableColorsStdout(nil)
	if disable != nil {
		defer disable()
	}
	if len(args) < 1 {
		if isatty.IsTerminal(uintptr(os.Stdin.Fd())) {
			return main1(strings.NewReader("{}"), "")
		}
		return main1(io.NopCloser(os.Stdin), "")
	}
	fd, err := os.Open(args[0])
	if err != nil {
		return err
	}
	return main1(fd, args[0])
}

func main() {
	if err := mains(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
