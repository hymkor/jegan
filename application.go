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
	"github.com/nyaosorg/go-windows-dbg"

	"github.com/hymkor/go-safewrite"
	"github.com/hymkor/jegan/internal/pager"
)

type Application struct {
	L       *list.List
	cursor  *list.Element
	csrline int
	winline int
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

func (app *Application) replaceValueOnly(session *pager.Session) {
	value, _ := getValue(app.cursor)
	if number, ok := value.(float64); ok {
		text, err := app.ReadLine(session, "New number:", fmt.Sprint(number))
		if err == nil {
			newValue, err := strconv.ParseFloat(text, 64)
			if err == nil && setValue(app.cursor, newValue) {
				return
			}
		}
	} else if text, ok := value.(string); ok {
		text, err := app.ReadLine(session, "New string:", fmt.Sprint(text))
		if err == nil && setValue(app.cursor, text) {
			return
		}
	}
	session.TtyOut.Write([]byte{'\a'})
}

func (app *Application) replaceTypeAndValue(session *pager.Session) {
	original, _ := getValue(app.cursor)
	newValue, ok := app.readNewValue(session, fmt.Sprint(original))
	if ok {
		setValue(app.cursor, newValue)
	}
}

func (app *Application) readNewValue(session *pager.Session, defaults string) (any, bool) {

	for {
		fmt.Fprint(session.TtyOut,
			"\r'1':String, '2':Number, '3':null, '4':true, '5':false ? \x1B[0K")
		key, err := session.GetKey()
		if err != nil {
			fmt.Fprintf(session.TtyOut, "\r%s\x1B[0K", err.Error())
			return nil, false
		}
		switch key {
		case "\a":
			fmt.Fprint(session.TtyOut, "\rCanceled\x1B[0K")
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
	indent := cursor.Value.(interface{ Indent() int }).Indent()
	for {
		cursor = cursor.Prev()
		if cursor == nil {
			return -1
		}
		i := cursor.Value.(interface{ Indent() int }).Indent()
		if i < indent {
			return
		}
		index++
	}
}

func isDuplicated(cursor *list.Element, indent int, key string) bool {
	for p := cursor; p != nil; p = p.Prev() {
		i := p.Value.(interface{ Indent() int }).Indent()
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
		i := p.Value.(interface{ Indent() int }).Indent()
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

func (app *Application) insertNewValue(session *pager.Session) {
	if pair, ok := app.cursor.Value.(*Pair); ok {
		key, err := app.ReadLine(session, "Key: ", "")
		if err != nil {
			return
		}
		if isDuplicated(app.cursor, pair.Element.indent, key) {
			session.TtyOut.Write([]byte{'\a'})
			return
		}
		value, ok := app.readNewValue(session, "")
		if !ok {
			return
		}
		app.L.InsertAfter(
			newPair(key,
				value,
				pair.Element.indent,
				pair.Element.comma),
			app.cursor)
		pair.Element.comma = true
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
			newElement(
				value,
				element.indent,
				element.comma),
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
		Dump(app.L, fd)
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
