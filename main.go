package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"unicode"

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

type Format struct {
	indent []byte
}

func getFormat(source []byte) *Format {
	format := &Format{}
	pos := bytes.IndexByte(source, '\n')
	if pos < 0 {
		return format
	}
	format.indent = []byte{}
	for {
		pos++
		if pos >= len(source) || !unicode.IsSpace(rune(source[pos])) {
			return format
		}
		format.indent = append(format.indent, source[pos])
	}
}

func main1(data []byte, name string) error {
	app := newApplication()
	app.Name = name
	app.format = getFormat(data)
	defer app.Close()

	br := bytes.NewReader(data)
	for {
		v, err := unjson.Unmarshal(br)
		if err != nil {
			var e *unjson.ErrTrailingData
			if errors.As(err, &e) {
				err = e.Err
				app.trailing = e.Trailing
			}
			if errors.Is(err, io.EOF) {
				app.Store(Read(v))
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
			return main1([]byte{'{', '}'}, "")
		}
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		return main1(data, "")
	}
	data, err := os.ReadFile(args[0])
	if err != nil {
		return err
	}
	return main1(data, args[0])
}

func main() {
	if err := mains(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
