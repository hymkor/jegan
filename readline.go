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
	"github.com/nyaosorg/go-readline-ny/completion"
	"github.com/nyaosorg/go-readline-ny/keys"
	"github.com/nyaosorg/go-readline-ny/simplehistory"
	"github.com/nyaosorg/go-readline-skk"
	"github.com/nyaosorg/go-ttyadapter/auto"

	"github.com/hymkor/jegan/internal/ansi"
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

func (app *Application) readLine(session *Session, prompt, defaults string) (string, error) {
	nop := func(*readline.Editor) {}
	return app.readLineOpt(session, prompt, defaults, nop)
}

func (app *Application) readLinePath(session *Session, prompt, defaults string) (string, error) {
	opt := func(editor *readline.Editor) {
		if len(defaults) > 5 && strings.HasSuffix(defaults, ".json") {
			editor.Cursor = readline.MojiCountInString(defaults) - 5
		}
		editor.BindKey(keys.CtrlI, &completion.CmdCompletion2{
			Candidates: completion.PathComplete,
		})
	}
	return app.readLineOpt(session, prompt, defaults, opt)
}

func (app *Application) readLineElement(session *Session, prompt, defaults string) (string, error) {
	opt := func(editor *readline.Editor) {
		if len(defaults) > 0 && strings.IndexByte(`"]}`, defaults[len(defaults)-1]) >= 0 {
			editor.Cursor = readline.MojiCountInString(defaults) - 1
		}
	}
	return app.readLineOpt(session, prompt, defaults, opt)
}

func (app *Application) readLineString(session *Session, prompt, defaults string) (string, error) {
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

func (app *Application) readLineOpt(session *Session, prompt, defaults string, opt func(*readline.Editor)) (string, error) {
	if ap, ok := app.ttyIn.(*auto.Pilot); ok {
		return ap.GetKey()
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
		PredictColor: [...]string{"\x1B[3;22;34m", "\x1B[23;39m"},
	}
	editor.BindKey(keys.CtrlG, readline.CmdInterrupt)
	editor.BindKey(keys.Escape+keys.CtrlG, readline.CmdInterrupt)
	editor.BindKey(keys.CtrlL, readline.CmdRepaintLine)
	opt(editor)
	updateHistory := func(string) {}
	if editor.History == nil {
		if app.history == nil {
			app.history = simplehistory.New()
			app.history.Add("null")
			app.history.Add("false")
			app.history.Add("true")
			app.history.Add("\"\"")
			app.history.Add("{}")
			app.history.Add("[]")
		}
		editor.History = app.history
		updateHistory = func(s string) { app.history.Add(s) }
	}
	result, err := editor.ReadLine(context.Background())
	io.WriteString(session.TtyOut, ansi.CursorOff)
	if err == readline.CtrlC {
		return "", errCanceled
	}
	updateHistory(result)
	return result, err
}

func askYesNo(session *Session, message string) (bool, error) {
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
