package main

import (
	"container/list"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/mattn/go-runewidth"

	"github.com/nyaosorg/go-readline-ny"
	"github.com/nyaosorg/go-readline-ny/keys"
	"github.com/nyaosorg/go-ttyadapter"

	"github.com/hymkor/go-safewrite"
	"github.com/hymkor/go-safewrite/perm"
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

func (app *Application) replaceTypeAndValue(
	session *pager.Session,
	input func(*pager.Session, any) []any) {

	element := ref(app.cursor)
	if _, ok := element.value.(Mark); ok {
		return
	}
	values := input(session, element.value)
	switch len(values) {
	case 1:
		element.value = values[0]
	case 2:
		app.L.InsertAfter(
			newElement(values[1], element.indent, element.comma),
			app.cursor)
		element.value = values[0]
		element.comma = false
	}
}

func (app *Application) readNewValue(session *pager.Session, defaultv any) []any {
	var defaults string
	if v, ok := defaultv.(Mark); ok {
		defaults = string(rune(v))
	} else {
		b, err := json.Marshal(defaultv)
		if err != nil {
			debug("(*Application) readNewValue: json.Marshal:", err.Error(), "for", defaultv)
		}
		defaults = string(b)
	}
	rawText, err := app.ReadLine(session, "New value:", defaults)
	if err != nil {
		app.message = err.Error()
		return nil
	}
	normText := strings.TrimSpace(rawText)

	if len(normText) >= 2 && normText[0] == '"' && normText[len(normText)-1] == '"' {
		var s string
		err := json.Unmarshal([]byte(rawText), &s)
		if err == nil {
			return []any{s}
		}
	}
	if number, err := strconv.ParseFloat(normText, 64); err == nil {
		return []any{number}
	}
	if strings.EqualFold(normText, "null") {
		return []any{nil}
	}
	if strings.EqualFold(normText, "true") {
		return []any{true}
	}
	if strings.EqualFold(normText, "false") {
		return []any{false}
	}
	if normText == "{}" {
		return []any{Mark('{'), Mark('}')}
	}
	if normText == "[]" {
		return []any{Mark('['), Mark(']')}
	}
	return []any{rawText}
}

func (app *Application) readNewValue2(session *pager.Session, defaultv any) []any {
	for {
		fmt.Fprint(session.TtyOut,
			"\r'1':String, '2':Number, '3':null, "+
				"'4':true, '5':false, '6':{}, '7':[] ? \x1B[0K")
		key, err := session.GetKey()
		if err != nil {
			app.message = err.Error()
			return nil
		}
		switch key {
		case "\a":
			app.message = "Canceled"
			return nil
		case "1":
			text, err := app.ReadLine(session, "New string:", fmt.Sprint(defaultv))
			if err == nil {
				return []any{text}
			}
			session.TtyOut.Write([]byte{'\a'})
		case "2":
			text, err := app.ReadLine(session, "New number:", fmt.Sprint(defaultv))
			if err == nil {
				newValue, err := strconv.ParseFloat(text, 64)
				if err == nil {
					return []any{newValue}
				}
			}
			session.TtyOut.Write([]byte{'\a'})
		case "3":
			return []any{nil}
		case "4":
			return []any{true}
		case "5":
			return []any{false}
		case "6":
			return []any{Mark('{'), Mark('}')}
		case "7":
			return []any{Mark('['), Mark(']')}
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
		values := app.readNewValue(session, "")
		switch len(values) {
		case 2: // [\n[\n],\n
			app.L.InsertBefore(
				newElement(values[0], indent, false),
				next)
			app.L.InsertBefore(
				newElement(values[1], indent, comma),
				next)
		case 1: // [\n value
			app.L.InsertBefore(
				newElement(values[0], indent, comma),
				next)
		}
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
				app.message = fmt.Sprintf("\aduplicate key: %q", key)
				return
			}
			comma = true
			indent = element.indent
		}
		values := app.readNewValue(session, "")
		switch len(values) {
		case 2: // { key:[]
			app.L.InsertBefore(
				newPair(key, values[0], indent, false),
				next)
			app.L.InsertBefore(
				newElement(values[1], indent, comma),
				next)
		case 1: // { key:value
			app.L.InsertBefore(
				newPair(key, values[0], indent, comma),
				next)
		}
		return
	}
	if isHashElement(app.cursor) {
		key, err := app.ReadLine(session, "Key: ", "")
		if err != nil {
			return
		}
		element := ref(app.cursor)
		if isDuplicated(app.cursor, element.indent, key) {
			app.message = fmt.Sprintf("\aduplicate key: %q", key)
			return
		}
		values := app.readNewValue(session, "")
		switch len(values) {
		case 2: // key:[],
			app.L.InsertAfter(
				newElement(values[1], element.indent, element.comma),
				app.cursor)
			app.L.InsertAfter(
				newPair(key, values[0], element.indent, false),
				app.cursor)
		case 1: // key:value,
			app.L.InsertAfter(
				newPair(key, values[0], element.indent, element.comma),
				app.cursor)
		}
		element.comma = true
		return
	}
	if element, ok := app.cursor.Value.(*Element); ok {
		index := getIndex(app.cursor)
		if index < 0 {
			return
		}
		values := app.readNewValue(session, "")
		switch len(values) {
		case 2: // [ \n ],
			app.L.InsertAfter(
				newElement(values[1], element.indent, element.comma),
				app.cursor)
			app.L.InsertAfter(
				newElement(values[0], element.indent, false),
				app.cursor)
		case 1: // value,
			app.L.InsertAfter(
				newElement(values[0], element.indent, element.comma),
				app.cursor)
		}
		element.comma = true
	}
}

func (app *Application) removeCursor(session *pager.Session) {
	comma := ref(app.cursor).comma
	if next := app.cursor.Next(); next != nil {
		ref(app.cursor).cursor = false
		app.L.Remove(app.cursor)
		app.cursor = next
		ref(app.cursor).cursor = true
		if !comma {
			if p := app.cursor.Prev(); p != nil {
				ref(p).comma = false
			}
		}
	} else if prev := app.cursor.Prev(); prev != nil {
		ref(app.cursor).cursor = false
		app.L.Remove(app.cursor)
		app.cursor = prev
		ref(app.cursor).cursor = true
		app.csrline--
		if app.csrline < app.winline {
			session.Window = app.cursor
			app.winline = app.csrline
		}
		if !comma {
			ref(prev).comma = false
		}
	}
}

func (app *Application) removeLine(session *pager.Session) {
	element := ref(app.cursor)
	mark, ok := element.value.(Mark)
	if !ok {
		app.removeCursor(session)
		return
	}
	if mark != Mark('{') && mark != Mark('[') {
		return
	}
	if element.indent == 0 {
		return
	}
	next := app.cursor.Next()
	if next == nil {
		return
	}
	n := ref(next)
	if n.value != Mark(']') && n.value != Mark('}') {
		return
	}
	comma := n.comma
	app.L.Remove(next)
	app.removeCursor(session)
	ref(app.cursor).comma = comma
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
	case " ", "b", keys.CtrlC, keys.CtrlG:
	case "r":
		app.replaceTypeAndValue(session, app.readNewValue)
	case "R":
		app.replaceTypeAndValue(session, app.readNewValue2)
	case "o":
		app.insertNewValue(session)
	case "d":
		app.removeLine(session)
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
		perm.Track(fd)
		app.Title = fname
	}
	return true, nil
}

func (app *Application) Status(session *pager.Session, out io.Writer) error {
	if app.message != "" {
		fmt.Fprintf(out, "\x1B[1m%s\x1B[0m\x1B[0K",
			runewidth.Truncate(app.message, session.Width-1, ""))
		app.message = ""
	} else if app.Title != "" {
		fmt.Fprintf(out, "\x1B[7m%s\x1B[0m\x1B[0K",
			runewidth.Truncate(app.Title, session.Width-1, ""))
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

func (app *Application) Close() error {
	return perm.RestoreAll()
}
