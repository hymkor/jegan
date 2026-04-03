package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

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

func main1(data []byte, name string) error {
	var v any
	err := json.Unmarshal(data, &v)
	if err != nil {
		return err
	}
	app := newApplication(Read(v))
	defer app.Close()
	app.Name = name
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
