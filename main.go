package jegan

import (
	"io"
	"os"
	"path/filepath"

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

func parseArgs(args []string, load func(io.Reader, string) error) (useStdin bool, names []string, err error) {
	if len(args) <= 0 {
		useStdin = true
		if !isatty.IsTerminal(uintptr(os.Stdin.Fd())) {
			err = load(os.Stdin, "")
		}
		return
	}
	for _, arg := range args {
		if arg == "-" {
			useStdin = true
			if err = load(os.Stdin, ""); err != nil {
				return
			}
			continue
		}
		var fnames []string
		fnames, err = filepath.Glob(arg)
		if err != nil || len(fnames) <= 0 {
			fnames = []string{arg}
		}
		for _, fn := range fnames {
			var fd *os.File
			fd, err = os.Open(fn)
			if err != nil {
				return
			}
			if err = load(fd, fn); err != nil {
				fd.Close()
				return
			}
			if err = fd.Close(); err != nil {
				return
			}
			names = append(names, fn)
		}
	}
	return
}

type Config struct {
	Auto string
}

func (c *Config) Run(args []string) error {
	app := &Application{}
	defer app.Close()

	disable := colorable.EnableColorsStdout(nil)
	if disable != nil {
		defer disable()
	}

	var ttyIn ttyadapter.Tty
	if c.Auto != "" {
		ttyIn = &autoPilot{script: c.Auto}
	} else {
		ttyIn = &tty8pe.Tty{}
	}

	useStdin, names, err := parseArgs(args, app.Load)
	if err != nil {
		return err
	}
	if len(names) == 1 {
		app.Name = names[0]
	}

	var ttyOut io.Writer
	if useStdin {
		ttyOut = colorable.NewColorableStderr()
	} else {
		ttyOut = colorable.NewColorableStdout()
	}

	return app.EventLoop(ttyIn, ttyOut)
}
