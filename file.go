package jegan

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/nyaosorg/go-inline-animation"

	"github.com/hymkor/go-safewrite"
	"github.com/hymkor/go-safewrite/perm"

	"github.com/hymkor/jegan/internal/ansi"
	"github.com/hymkor/jegan/internal/pager"
	"github.com/hymkor/jegan/internal/unjson"
)

func (app *Application) keyFuncSave(session *pager.Session) {
	err := app.writeFile(session)
	if err != nil {
		app.message = err.Error()
	}
}

func (app *Application) writeFile(session *pager.Session) error {
	fname, err := app.readLinePath(session, "Write to:", app.Name)
	if err != nil {
		return err
	}
	if fname == "" || fname == "-" {
		session.TtyOut.Write([]byte{' '})
		end := animation.Dots.Progress(session.TtyOut)
		defer end()

		Dump(app.list, os.Stdout)
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

	Dump(app.list, fd)
	if err := fd.Close(); err != nil {
		return err
	}
	perm.Track(fd)
	app.Name = fname
	app.dirty = false
	return nil
}

func (app *Application) keyFuncQuit(session *pager.Session) pager.EventResult {
	if !app.dirty {
		return pager.QuitApp
	}
	io.WriteString(session.TtyOut, ansi.CursorOn)
	defer io.WriteString(session.TtyOut, ansi.CursorOff)

	yesSave, err := askYesNo(session, "Quit: Save changes ? ['y': save, 'n': quit without saving, other: cancel]")
	if err != nil {
		app.message = err.Error() // err includes cancel
		return pager.Handled
	}
	if yesSave {
		if err := app.writeFile(session); err != nil {
			app.message = err.Error()
			return pager.Handled
		}
	}
	return pager.QuitApp
}
func (app *Application) Load(r io.Reader, name string) error {
	br, ok := r.(io.RuneScanner)
	if !ok {
		br = bufio.NewReader(r)
	}
	for {
		v, err := unjson.Unmarshal(br)
		if err != nil {
			if errors.Is(err, io.EOF) {
				app.Store(Read(v))
				if err == io.EOF {
					return nil
				}
			}
			if name == "" {
				return fmt.Errorf("<STDIN>:%w", err)
			}
			return fmt.Errorf("%s:%w", name, err)
		}
		app.Store(Read(v))
	}
}
