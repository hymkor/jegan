package jegan

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/mattn/go-isatty"

	"github.com/nyaosorg/go-inline-animation"

	"github.com/hymkor/go-generics-list"
	"github.com/hymkor/go-safewrite"
	"github.com/hymkor/go-safewrite/perm"

	"github.com/hymkor/jegan/internal/ansi"
	"github.com/hymkor/jegan/internal/decode"
	"github.com/hymkor/jegan/internal/pager"
	"github.com/hymkor/jegan/internal/types"
	// "github.com/hymkor/jegan/internal/tree2list"
	// "github.com/hymkor/jegan/internal/unjson"
)

func expandArgs(args []string) (useStdin bool, names []string) {
	if len(args) <= 0 {
		useStdin = true
		if !isatty.IsTerminal(uintptr(os.Stdin.Fd())) {
			names = []string{""}
		}
		return
	}
	for _, arg := range args {
		if arg == "-" {
			useStdin = true
			names = append(names, "")
			continue
		}
		var fnames []string
		fnames, err := filepath.Glob(arg)
		if err != nil || len(fnames) <= 0 {
			names = append(names, arg)
		} else {
			names = append(names, fnames...)
		}
	}
	return
}

func loadEach(names []string, load func(io.Reader, string) error) error {
	for _, fname := range names {
		if fname == "" {
			if err := load(os.Stdin, ""); err != nil {
				return err
			}
			continue
		}
		fd, err := os.Open(fname)
		if err != nil {
			return err
		}
		err1 := load(fd, fname)
		err2 := fd.Close()
		if err1 != nil {
			return err1
		}
		if err2 != nil {
			return err2
		}
	}
	return nil
}

func (app *Application) keyFuncSave(session *Session) error {
	return app.writeFile(session)
}

func (app *Application) writeFile(session *Session) error {
	fname, err := app.readLinePath(session, "Write to:", app.Name)
	if err != nil {
		return err
	}
	if fname == "" || fname == "-" {
		session.TtyOut.Write([]byte{' '})
		end := animation.Dots.Progress(session.TtyOut)
		defer end()

		types.Dump(app.list, os.Stdout)
		app.dirty = false
		return nil
	}
	var callBackErr error
	fd, err := safewrite.Open(fname, func(info *safewrite.Info) bool {
		var format string
		if info.ReadOnly() {
			format = "Overwrite READONLY file %q ?"
		} else {
			format = "Overwrite file %q ?"
		}
		ans, err := askYesNo(session, fmt.Sprintf(format, info.Name))
		if err != nil {
			callBackErr = err
			return false
		}
		return ans
	})
	if err != nil {
		return err
	}
	if callBackErr != nil {
		return callBackErr
	}
	session.TtyOut.Write([]byte{' '})
	end := animation.Dots.Progress(session.TtyOut)
	defer end()

	types.Dump(app.list, fd)
	if err := fd.Close(); err != nil {
		return err
	}
	perm.Track(fd)
	app.Name = fname
	app.dirty = false
	return nil
}

func (app *Application) keyFuncQuit(session *Session) (pager.EventResult, error) {
	if !app.dirty {
		return pager.QuitApp, nil
	}
	io.WriteString(session.TtyOut, ansi.CursorOn)
	defer io.WriteString(session.TtyOut, ansi.CursorOff)

	yesSave, err := askYesNo(session, "Quit: Save changes ? ['y': save, 'n': quit without saving, other: cancel]")
	if err != nil {
		return pager.Handled, err // err includes cancel
	}
	if yesSave {
		if err := app.writeFile(session); err != nil {
			return pager.Handled, err
		}
	}
	return pager.QuitApp, nil
}

func (app *Application) load(r io.Reader, name string, store func(types.Line) error) error {
	br, ok := r.(io.RuneScanner)
	if !ok {
		br = bufio.NewReader(r)
	}
	for {
		err := decode.Unmarshal(br, store)
		if err != nil {
			if errors.Is(err, io.EOF) {
				if err == io.EOF {
					return nil
				}
			}
			if name == "" {
				return fmt.Errorf("<STDIN>:%w", err)
			}
			return fmt.Errorf("%s:%w", name, err)
		}
	}
}

func (app *Application) Load(r io.Reader, name string) error {
	if app.list == nil {
		app.list = list.New[types.Line]()
	}
	return app.load(r, name, func(line types.Line) error {
		app.list.PushBack(line)
		return nil
	})
}
