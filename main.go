package jegan

import (
	"io"

	"github.com/mattn/go-colorable"

	"github.com/nyaosorg/go-ttyadapter"
	"github.com/nyaosorg/go-ttyadapter/tty8pe"

	"github.com/hymkor/jegan/internal/auto"
)

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
		ttyIn = auto.New(c.Auto)
	} else {
		ttyIn = &tty8pe.Tty{}
	}

	useStdin, names := expandArgs(args)
	app.Name = names[0]

	var ttyOut io.Writer
	if useStdin {
		ttyOut = colorable.NewColorableStderr()
	} else {
		ttyOut = colorable.NewColorableStdout()
	}

	if err := loadEach(names, app.Load); err != nil {
		return err
	}

	return app.EventLoop(ttyIn, ttyOut)
}
