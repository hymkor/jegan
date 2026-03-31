package main

import (
	"container/list"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/mattn/go-colorable"

	"github.com/nyaosorg/go-readline-ny"
	"github.com/nyaosorg/go-readline-ny/keys"
	"github.com/nyaosorg/go-ttyadapter/tty8pe"
	"github.com/nyaosorg/go-windows-dbg"

	"github.com/hymkor/go-safewrite"
	"github.com/hymkor/jegan/internal/pager"
)

func debug(v ...any) {
	if false {
		dbg.Println(v...)
	}
}

type Application struct {
	L       *list.List
	cursor  *list.Element
	csrline int
	winline int
	Root    any
	Title   string
}

func newApplication(L *list.List) *Application {
	cursor := L.Front()
	setCursor(cursor, true)

	return &Application{
		L:      L,
		cursor: cursor,
	}
}

func (app *Application) SetCursor(c *list.Element) {
	setCursor(app.cursor, false)
	app.cursor = c
	setCursor(app.cursor, true)
}

func (app *Application) ReadLine(session *pager.Session, prompt, defaults string) (string, error) {
	editor := &readline.Editor{
		Writer: session.TtyOut,
		PromptWriter: func(w io.Writer) (int, error) {
			return fmt.Fprintf(w, "\r%s \x1B[0K", prompt)
		},
		LineFeedWriter: func(readline.Result, io.Writer) (int, error) {
			return 0, nil
		},
		Cursor:  65535,
		Default: defaults,
	}
	editor.BindKey(keys.CtrlG, readline.CmdInterrupt)
	editor.BindKey(keys.Escape+keys.CtrlG, readline.CmdInterrupt)
	result, err := editor.ReadLine(context.Background())
	io.WriteString(session.TtyOut, "\x1B[?25l")
	if err == readline.CtrlC {
		return "", errors.New("Canceled")
	}
	return result, err
}

func (app *Application) readNewValue(session *pager.Session, prompt string) (string, bool) {
	if !canSetValue(app.cursor) {
		session.TtyOut.Write([]byte{'\a'})
		return "", false
	}
	value, _ := getValue(app.cursor)
	result, err := app.ReadLine(session, prompt, fmt.Sprint(value))

	return result, err == nil
}

func (app *Application) Handle(session *pager.Session, key string) (bool, error) {
	switch key {
	default:
		return false, nil
	case "j", "\x1B[B":
		if c := app.cursor.Next(); c != nil {
			app.SetCursor(c)
			app.csrline++
			for app.csrline-app.winline >= session.Height {
				session.Window = session.Window.Next()
				app.winline++
			}
		}
	case "k", "\x1B[A":
		if c := app.cursor.Prev(); c != nil {
			app.SetCursor(c)
			app.csrline--
			for app.csrline < app.winline {
				session.Window = session.Window.Prev()
				app.winline--
			}
		}
	case "<":
		app.SetCursor(app.L.Front())
		session.Front()
		app.winline = 0
		app.csrline = 0
	case ">":
		app.SetCursor(app.L.Back())
		n := session.Back()
		app.csrline = app.L.Len() - 1
		app.winline = app.L.Len() - 1 - n
	case " ", "b":
		return true, nil
	case "r":
		text, ok := app.readNewValue(session, "New string:")
		if !ok || !setValue(app.cursor, text) {
			session.TtyOut.Write([]byte{'\a'})
		}
		return true, nil
	case "f":
		text, ok := app.readNewValue(session, "New number:")
		if ok {
			newValue, err := strconv.ParseFloat(text, 64)
			if err != nil || !setValue(app.cursor, newValue) {
				session.TtyOut.Write([]byte{'\a'})
			}
		}
		return true, nil
	case "w":
		fname, err := app.ReadLine(session, "Write to:", app.Title)
		if err != nil {
			fmt.Fprintf(session.TtyOut, "\r%s\x1B[0K", err.Error())
			break
		}
		data, err := json.Marshal(app.Root)
		if err != nil {
			fmt.Fprintf(session.TtyOut, "\r%s\x1B[0K", err.Error())
			break
		}
		fd, err := safewrite.Open(fname, func(info *safewrite.Info) bool {
			for {
				if info.ReadOnly() {
					fmt.Fprintf(session.TtyOut, "\rOverwrite READONLY file %q ? ", info.Name)
				} else {
					fmt.Fprintf(session.TtyOut, "\rOverwrite file %q ? ", info.Name)
				}
				ans, err := session.GetKey()
				if err != nil {
					return false
				}
				if strings.EqualFold(ans, "y") {
					return true
				}
				if strings.EqualFold(ans, "n") {
					return false
				}
			}
		})
		if err != nil {
			fmt.Fprintf(session.TtyOut, "\r%s\x1B[0K", err.Error())
			break
		}
		fd.Write(data)
		if err := fd.Close(); err != nil {
			dbg.Println(err.Error())
			fmt.Fprintf(session.TtyOut, "\r%s\x1B[0K", err.Error())
			break
		}
		if err := safewrite.RestorePerm(fd); err != nil {
			dbg.Println(err.Error())
			fmt.Fprintf(session.TtyOut, "\r%s\x1B[0K", err.Error())
		}
	}
	return true, nil
}

func main1(data []byte, title string) error {
	var v any
	err := json.Unmarshal(data, &v)
	if err != nil {
		return err
	}
	L := Read(v, 0, func(x any) { v = x })
	app := newApplication(L)
	app.Root = v
	app.Title = title

	pager1 := &pager.Pager{
		Status: func(_ *pager.Session, out io.Writer) error {
			if title != "" {
				fmt.Fprintf(out, "\x1B[7m%s\x1B[0m\x1B[0K", title)
			}
			return nil
		},
		Handler: app.Handle,
	}
	ttyout := colorable.NewColorableStdout()
	return pager1.EventLoop(&tty8pe.Tty{}, L, ttyout)
}

func mains(args []string) error {
	if len(args) < 1 {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		return main1(data, "<STDIN>")
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
