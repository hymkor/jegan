package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

func mains(args []string) error {
	app := &Application{Name: strings.Join(args, "+")}
	defer app.Close()

	if len(args) <= 0 {
		if !isatty.IsTerminal(uintptr(os.Stdin.Fd())) {
			app.Load(os.Stdin, "")
		}
	} else {
		for _, arg := range args {
			if arg == "-" {
				app.Load(os.Stdin, "")
				continue
			}
			fnames, err := filepath.Glob(arg)
			if err != nil || len(fnames) <= 0 {
				fnames = []string{arg}
			}
			for _, fn := range fnames {
				fd, err := os.Open(fn)
				if err != nil {
					return err
				}
				if err := app.Load(fd, fn); err != nil {
					return err
				}
				if err := fd.Close(); err != nil {
					return err
				}
			}
		}
	}
	disable := colorable.EnableColorsStdout(nil)
	if disable != nil {
		defer disable()
	}
	ttyout := colorable.NewColorableStdout()
	return app.EventLoop(&tty8pe.Tty{}, ttyout)
}

func main() {
	if err := mains(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
