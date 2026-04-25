package jegan

import (
	"context"
	"errors"
	"io"

	"github.com/mattn/go-colorable"

	"github.com/nyaosorg/go-ttyadapter"
	"github.com/nyaosorg/go-ttyadapter/tty8pe"

	"github.com/hymkor/go-generics-list"

	"github.com/hymkor/jegan/internal/auto"
	"github.com/hymkor/jegan/internal/nonblockpush"
	"github.com/hymkor/jegan/internal/ttyhook"
	"github.com/hymkor/jegan/internal/types"
)

type Config struct {
	Auto string
}

func (c *Config) Run(args []string) error {
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
	var ttyOut io.Writer
	if useStdin {
		ttyOut = colorable.NewColorableStderr()
	} else {
		ttyOut = colorable.NewColorableStdout()
	}
	return Start(ttyIn, names, ttyOut)
}

func Start(ttyIn ttyadapter.Tty, names []string, ttyOut io.Writer) error {
	keyWorker := nonblockpush.New[types.Line](ttyIn.GetKey)
	defer keyWorker.Close()

	app := &Application{
		list:       list.New[types.Line](),
		dataStream: keyWorker.DataStream(),
	}
	if len(names) == 1 {
		app.Name = names[0]
	}
	defer app.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		loadEach(names, func(r io.Reader, name string) error {
			return app.load(r, name, func(line types.Line) error {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case app.dataStream <- nonblockpush.NewDataStream[types.Line](line, nil):
					return nil
				}
			})
		})
		close(app.dataStream)
	}()

	newTtyIn := ttyhook.New(ttyIn, func(_ func() (string, error)) (string, error) {
		return keyWorker.GetOr(func(data types.Line, err error) bool {
			if err != nil && !errors.Is(err, io.EOF) {
				return false
			}
			if data != nil {
				app.list.PushBack(data)
			}
			return err == nil
		})
	})
	return app.EventLoop(newTtyIn, ttyOut)
}
