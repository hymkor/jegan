package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"unicode"

	"github.com/mattn/go-colorable"
	"github.com/mattn/go-isatty"

	"github.com/nyaosorg/go-ttyadapter/tty8pe"
	"github.com/nyaosorg/go-windows-dbg"
)

func debug(v ...any) {
	if false {
		dbg.Println(v...)
	}
}

func getFormat(source []byte) (indent, newline []byte) {
	pos := bytes.IndexByte(source, '\n')
	if pos < 0 {
		return []byte{}, []byte{}
	}
	if pos > 1 && source[pos-1] == '\r' {
		newline = []byte{'\r', '\n'}
	} else {
		newline = []byte{'\n'}
	}
	indent = []byte{}
	for {
		pos++
		if pos >= len(source) || !unicode.IsSpace(rune(source[pos])) {
			return
		}
		indent = append(indent, source[pos])
	}
}

func main1(data []byte, name string) error {
	// var v any
	// err := json.Unmarshal(data, &v)
	v, err := unmarshal(data)
	if err != nil {
		if name != "" {
			err = fmt.Errorf("%s:%w", name, err)
		} else {
			err = fmt.Errorf("<STDIN>:%w", err)
		}
		return err
	}
	app := newApplication(Read(v))
	defer app.Close()
	app.Name = name
	app.indent, app.newline = getFormat(data)
	ttyout := colorable.NewColorableStdout()
	return app.EventLoop(&tty8pe.Tty{}, ttyout)
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
