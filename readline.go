package jegan

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/nyaosorg/go-readline-ny"
	"github.com/nyaosorg/go-readline-ny/keys"
	"github.com/nyaosorg/go-readline-skk"

	"github.com/hymkor/jegan/internal/ansi"
	"github.com/hymkor/jegan/internal/pager"
)

var errCanceled = errors.New("Canceled")

var skkInitOnce sync.Once

func skkInit() {
	skkInitOnce.Do(func() {
		env := os.Getenv("GOREADLINESKK")
		if env != "" {
			_, err := skk.Config{
				MiniBuffer: skk.MiniBufferOnCurrentLine{},
			}.SetupWithString(env)
			if err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
			}
		}
	})
}

func (app *Application) readLine(session *pager.Session, prompt, defaults string) (string, error) {
	nop := func(*readline.Editor) {}
	return app.readLineOpt(session, prompt, defaults, nop)
}

func (app *Application) readLinePath(session *pager.Session, prompt, defaults string) (string, error) {
	opt := func(editor *readline.Editor) {
		if len(defaults) > 5 && strings.HasSuffix(defaults, ".json") {
			editor.Cursor = readline.MojiCountInString(defaults) - 5
		}
	}
	return app.readLineOpt(session, prompt, defaults, opt)
}

func (app *Application) readLineElement(session *pager.Session, prompt, defaults string) (string, error) {
	opt := func(editor *readline.Editor) {
		if len(defaults) > 0 && strings.IndexByte(`"]}`, defaults[len(defaults)-1]) >= 0 {
			editor.Cursor = readline.MojiCountInString(defaults) - 1
		}
	}
	return app.readLineOpt(session, prompt, defaults, opt)
}

func (app *Application) readLineString(session *pager.Session, prompt, defaults string) (string, error) {
	opt := func(e *readline.Editor) {
		e.OnAfterRender = func(B *readline.Buffer, availWidth int) {
			if availWidth >= 1 {
				B.Out.Write([]byte{'"', '\b'})
			}
		}
		e.PromptWriter = func(w io.Writer) (int, error) {
			return fmt.Fprintf(w, "\r%s \"%s", prompt, ansi.EraseLine)
		}
	}
	return app.readLineOpt(session, "", defaults, opt)
}

func (app *Application) readLineOpt(session *pager.Session, prompt, defaults string, opt func(*readline.Editor)) (string, error) {
	if ap, ok := app.ttyIn.(*autoPilot); ok {
		return ap.next()
	}
	skkInit()
	editor := &readline.Editor{
		Writer: session.TtyOut,
		PromptWriter: func(w io.Writer) (int, error) {
			return fmt.Fprintf(w, "\r%s "+ansi.EraseLine, prompt)
		},
		LineFeedWriter: func(readline.Result, io.Writer) (int, error) {
			return 0, nil
		},
		Cursor:  65535,
		Default: defaults,
		Highlight: []readline.Highlight{
			skk.WhiteMarkerHighlight,
			skk.BlackMarkerHighlight,
		},
		ResetColor:   "\x1B[0m",
		DefaultColor: "\x1B[0m",
	}
	editor.BindKey(keys.CtrlG, readline.CmdInterrupt)
	editor.BindKey(keys.Escape+keys.CtrlG, readline.CmdInterrupt)
	editor.BindKey(keys.CtrlL, readline.CmdRepaintLine)
	opt(editor)
	result, err := editor.ReadLine(context.Background())
	io.WriteString(session.TtyOut, ansi.CursorOff)
	if err == readline.CtrlC {
		return "", errCanceled
	}
	return result, err
}

func askYesNo(session *pager.Session, message string) (bool, error) {
	io.WriteString(session.TtyOut, ansi.CursorOn)
	defer io.WriteString(session.TtyOut, ansi.CursorOff)

	session.TtyOut.Write([]byte{'\r'})
	io.WriteString(session.TtyOut, message)
	io.WriteString(session.TtyOut, ansi.EraseLine)

	ans, err := session.GetKey()
	if err != nil {
		return false, err
	}
	fmt.Fprintf(session.TtyOut, " %q", ans)
	switch ans {
	case "y", "Y":
		return true, nil
	case "n", "N":
		return false, nil
	default:
		return false, errors.New("canceled")
	}
}
