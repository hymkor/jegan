package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mattn/go-colorable"
	"github.com/mattn/go-isatty"

	"github.com/nyaosorg/go-ttyadapter"
	"github.com/nyaosorg/go-ttyadapter/tty8pe"
	"github.com/nyaosorg/go-windows-dbg"
)

func debug(v ...any) {
	if false {
		dbg.Println(v...)
	}
}

var flagAuto = flag.String("auto", "", "auto pilot")

func mains(args []string) error {
	app := &Application{Name: strings.Join(args, "+")}
	defer app.Close()

	getTtyOut := colorable.NewColorableStdout

	if len(args) <= 0 {
		if !isatty.IsTerminal(uintptr(os.Stdin.Fd())) {
			if err := app.Load(os.Stdin, ""); err != nil {
				return err
			}
		}
		getTtyOut = colorable.NewColorableStderr
	} else {
		for _, arg := range args {
			if arg == "-" {
				if err := app.Load(os.Stdin, ""); err != nil {
					return err
				}
				getTtyOut = colorable.NewColorableStderr
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
	ttyout := getTtyOut()
	var ttyin ttyadapter.Tty
	if *flagAuto != "" {
		ttyin = &autoPilot{script: *flagAuto}
	} else {
		ttyin = &tty8pe.Tty{}
	}
	return app.EventLoop(ttyin, ttyout)
}

func main() {
	flag.Parse()
	if err := mains(flag.Args()); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
