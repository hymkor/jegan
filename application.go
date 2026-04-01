package main

import (
	"container/list"
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/nyaosorg/go-readline-ny"
	"github.com/nyaosorg/go-readline-ny/keys"
	"github.com/nyaosorg/go-ttyadapter"

	"github.com/hymkor/go-safewrite"
	"github.com/hymkor/jegan/internal/pager"
)

type Application struct {
	L       *list.List
	cursor  *list.Element
	csrline int
	winline int
	Title   string
	message string
}

func newApplication(L *list.List) *Application {
	cursor := L.Front()
	ref(cursor).cursor = true

	return &Application{
		L:      L,
		cursor: cursor,
	}
}

func (app *Application) SetCursor(c *list.Element) {
	ref(app.cursor).cursor = false
	app.cursor = c
	ref(app.cursor).cursor = true
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

func (app *Application) replaceValueOnly(session *pager.Session) {
	element := ref(app.cursor)
	if number, ok := element.value.(float64); ok {
		text, err := app.ReadLine(session, "New number:", fmt.Sprint(number))
		if err == nil {
			newValue, err := strconv.ParseFloat(text, 64)
			if err == nil {
				element.value = newValue
				return
			}
		}
		app.message = err.Error()
	} else if text, ok := element.value.(string); ok {
		text, err := app.ReadLine(session, "New string:", fmt.Sprint(text))
		if err == nil {
			element.value = text
			return
		}
		app.message = err.Error()
	}
	session.TtyOut.Write([]byte{'\a'})
}

func (app *Application) replaceTypeAndValue(session *pager.Session) {
	element := ref(app.cursor)
	newValue, ok := app.readNewValue(session, fmt.Sprint(element.value))
	if ok {
		element.value = newValue
	}
}

func (app *Application) readNewValue(session *pager.Session, defaults string) (any, bool) {

	for {
		fmt.Fprint(session.TtyOut,
			"\r'1':String, '2':Number, '3':null, '4':true, '5':false ? \x1B[0K")
		key, err := session.GetKey()
		if err != nil {
			app.message = err.Error()
			return nil, false
		}
		switch key {
		case "\a":
			app.message = "Canceled"
			return nil, false
		case "1":
			text, err := app.ReadLine(session, "New string:", defaults)
			if err == nil {
				return text, true
			}
			session.TtyOut.Write([]byte{'\a'})
		case "2":
			text, err := app.ReadLine(session, "New number:", defaults)
			if err == nil {
				newValue, err := strconv.ParseFloat(text, 64)
				if err == nil {
					return newValue, true
				}
			}
			session.TtyOut.Write([]byte{'\a'})
		case "3":
			return nil, true
		case "4":
			return true, true
		case "5":
			return false, true
		default:
			session.TtyOut.Write([]byte{'\a'})
		}
	}
}

func getIndex(cursor *list.Element) (index int) {
	indent := ref(cursor).indent
	for {
		cursor = cursor.Prev()
		if cursor == nil {
			return -1
		}
		i := ref(cursor).indent
		if i < indent {
			return
		}
		index++
	}
}

func isDuplicated(cursor *list.Element, indent int, key string) bool {
	for p := cursor; p != nil; p = p.Prev() {
		i := ref(p).indent
		if i < indent {
			break
		}
		if i == indent {
			q, ok := p.Value.(*Pair)
			if ok && q.key == key {
				return true
			}
		}
	}
	for p := cursor.Next(); p != nil; p = p.Next() {
		i := ref(p).indent
		if i < indent {
			break
		}
		if i == indent {
			q, ok := p.Value.(*Pair)
			if ok && q.key == key {
				return true
			}
		}
	}
	return false
}

func isHashElement(p *list.Element) bool {
	if _, ok := p.Value.(*Pair); ok {
		return true
	}
	indent := ref(p).indent
	for {
		p = p.Prev()
		if p == nil {
			return false
		}
		element := ref(p)
		i := element.indent
		if i == indent {
			if _, ok := p.Value.(*Pair); ok {
				return true
			}
		} else if i < indent {
			return element.value == Mark('{')
		}
	}
}

func (app *Application) insertNewValue(session *pager.Session) {
	if element := ref(app.cursor); element.value == Mark('[') {
		next := app.cursor.Next()
		element = ref(next)
		var comma bool
		var indent int
		if element.value == Mark(']') {
			comma = false
			indent = element.indent + 1
		} else {
			comma = true
			indent = element.indent
		}
		value, ok := app.readNewValue(session, "")
		if !ok {
			return
		}
		app.L.InsertBefore(
			newElement(value, indent, comma),
			next)
		return
	}
	if element := ref(app.cursor); element.value == Mark('{') {
		key, err := app.ReadLine(session, "Key: ", "")
		if err != nil {
			return
		}
		next := app.cursor.Next()
		element = ref(next)
		var comma bool
		var indent int
		if element.value == Mark('}') {
			comma = false
			indent = element.indent + 1
		} else {
			if isDuplicated(next, element.indent, key) {
				session.TtyOut.Write([]byte{'\a'})
				return
			}
			comma = true
			indent = element.indent
		}
		value, ok := app.readNewValue(session, "")
		if !ok {
			return
		}
		app.L.InsertBefore(
			newPair(key, value, indent, comma),
			next)
		return
	}
	if isHashElement(app.cursor) {
		key, err := app.ReadLine(session, "Key: ", "")
		if err != nil {
			return
		}
		element := ref(app.cursor)
		if isDuplicated(app.cursor, element.indent, key) {
			session.TtyOut.Write([]byte{'\a'})
			return
		}
		value, ok := app.readNewValue(session, "")
		if !ok {
			return
		}
		app.L.InsertAfter(
			newPair(key, value, element.indent, element.comma),
			app.cursor)
		element.comma = true
		return
	}
	if element, ok := app.cursor.Value.(*Element); ok {
		index := getIndex(app.cursor)
		if index < 0 {
			return
		}
		value, ok := app.readNewValue(session, "")
		if !ok {
			return
		}
		app.L.InsertAfter(
			newElement(value, element.indent, element.comma),
			app.cursor)
		element.comma = true
	}
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
	case "r":
		app.replaceValueOnly(session)
	case "R":
		app.replaceTypeAndValue(session)
	case "o":
		app.insertNewValue(session)
	case "w":
		fname, err := app.ReadLine(session, "Write to:", app.Title)
		if err != nil {
			app.message = err.Error()
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
			app.message = err.Error()
			break
		}
		Dump(app.L, fd)
		if err := fd.Close(); err != nil {
			app.message = err.Error()
			break
		}
		if err := safewrite.RestorePerm(fd); err != nil {
			app.message = err.Error()
		}
	}
	return true, nil
}

func (app *Application) Status(_ *pager.Session, out io.Writer) error {
	if app.message != "" {
		fmt.Fprintf(out, "\x1B[1m%s\x1B[0m\x1B[0K", app.message)
		app.message = ""
		return nil
	}
	if app.Title != "" {
		fmt.Fprintf(out, "\x1B[7m%s\x1B[0m\x1B[0K", app.Title)
	}
	return nil
}

func (app *Application) EventLoop(tty ttyadapter.Tty, ttyout io.Writer) error {
	pager1 := &pager.Pager{
		Status:  app.Status,
		Handler: app.Handle,
	}
	return pager1.EventLoop(tty, app.L, ttyout)
}
