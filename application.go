package main

import (
	"container/list"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"runtime"
	"strconv"
	"strings"

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
	Name    string
	message string
	dirty   bool
	format  *Format
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
	cursorPosition := 65535
	if len(defaults) > 0 && strings.IndexByte(`"]}`, defaults[len(defaults)-1]) >= 0 {
		cursorPosition = readline.MojiCountInString(defaults) - 1
	} else if len(defaults) > 5 && strings.HasSuffix(defaults, ".json") {
		cursorPosition = readline.MojiCountInString(defaults) - 5
	}
	editor := &readline.Editor{
		Writer: session.TtyOut,
		PromptWriter: func(w io.Writer) (int, error) {
			return fmt.Fprintf(w, "\r%s \x1B[0K", prompt)
		},
		LineFeedWriter: func(readline.Result, io.Writer) (int, error) {
			return 0, nil
		},
		Cursor:  cursorPosition,
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
	defaultv := element.value
	prev := func() bool { return false }

	if v, ok := element.value.(Mark); ok {
		next := app.cursor.Next()
		if next == nil {
			return
		}
		nextv, ok := ref(next).value.(Mark)
		if !ok {
			return
		}
		if v == Mark('{') && nextv == Mark('}') {
			defaultv = map[string]any{}
			prev = func() bool { app.L.Remove(next); return true }
		} else if v == Mark('[') && nextv == Mark(']') {
			defaultv = []any{}
			prev = func() bool { app.L.Remove(next); return true }
		} else {
			return
		}
	}
	values := input(session, defaultv)
	switch len(values) {
	case 1:
		if prev() || element.value != values[0] {
			app.dirty = true
		}
		element.value = values[0]
	case 2:
		prefix := getPrefix(app.cursor)
		prev()
		app.L.InsertAfter(
			newElement(values[1], element.nest, element.comma, prefix),
			app.cursor)
		element.value = values[0]
		element.comma = false
		app.dirty = true
	}
}

func (app *Application) readNewValue(session *pager.Session, defaultv any) []any {
	var defaults string
	if _, ok := defaultv.(struct{}); ok {
		defaults = ""
	} else if v, ok := defaultv.(Mark); ok {
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
			"\r's':string, 'n':number, 'u':null, "+
				"'t':true, 'f':false, 'o':{}, 'a':[] ? \x1B[0K")
		key, err := session.GetKey()
		if err != nil {
			app.message = err.Error()
			return nil
		}
		switch key {
		case "\a":
			app.message = "Canceled"
			return nil
		case "s":
			text, err := app.ReadLine(session, "New string:", fmt.Sprint(defaultv))
			if err == nil {
				return []any{text}
			}
			session.TtyOut.Write([]byte{'\a'})
		case "n":
			text, err := app.ReadLine(session, "New number:", fmt.Sprint(defaultv))
			if err == nil {
				newValue, err := strconv.ParseFloat(text, 64)
				if err == nil {
					return []any{newValue}
				}
			}
			session.TtyOut.Write([]byte{'\a'})
		case "u":
			return []any{nil}
		case "t":
			return []any{true}
		case "f":
			return []any{false}
		case "o":
			return []any{Mark('{'), Mark('}')}
		case "a":
			return []any{Mark('['), Mark(']')}
		default:
			session.TtyOut.Write([]byte{'\a'})
		}
	}
}

func getIndex(cursor *list.Element) (index int) {
	nest := ref(cursor).nest
	for {
		cursor = cursor.Prev()
		if cursor == nil {
			return -1
		}
		i := ref(cursor).nest
		if i < nest {
			return
		}
		index++
	}
}

func isDuplicated(cursor *list.Element, nest int, key string) bool {
	for p := cursor; p != nil; p = p.Prev() {
		i := ref(p).nest
		if i < nest {
			break
		}
		if i == nest {
			q, ok := p.Value.(*Pair)
			if ok && q.key == key {
				return true
			}
		}
	}
	for p := cursor.Next(); p != nil; p = p.Next() {
		i := ref(p).nest
		if i < nest {
			break
		}
		if i == nest {
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
	nest := ref(p).nest
	for {
		p = p.Prev()
		if p == nil {
			return false
		}
		element := ref(p)
		i := element.nest
		if i == nest {
			if _, ok := p.Value.(*Pair); ok {
				return true
			}
		} else if i < nest {
			return element.value == Mark('{')
		}
	}
}

func getPrefix(p *list.Element) []byte {
	if pair, ok := p.Value.(*Pair); ok {
		return pair.preKey
	}
	return ref(p).prefix
}

func setPrefix(p *list.Element, prefix []byte) {
	if pair, ok := p.Value.(*Pair); ok {
		pair.preKey = prefix
	}
	ref(p).prefix = prefix
}

func joinBytes(args ...[]byte) []byte {
	b := []byte{}
	for _, b1 := range args {
		b = append(b, b1...)
	}
	return b
}

func (app *Application) insertNewValue(session *pager.Session) {
	prefix := getPrefix(app.cursor)
	if element := ref(app.cursor); element.value == Mark('[') {
		next := app.cursor.Next()
		nextElement := ref(next)
		var comma bool
		var nest int
		var newPrefix []byte
		todo := func() {}
		if nextElement.value == Mark(']') {
			comma = false
			todo = func() { setPrefix(next, prefix) }
			nest = nextElement.nest + 1
			newPrefix = joinBytes(prefix, app.format.indent)
		} else {
			comma = true
			nest = nextElement.nest
			newPrefix = getPrefix(next)
		}
		values := app.readNewValue(session, struct{}{})
		switch len(values) {
		case 2: // [\n[\n],\n
			e1 := newElement(values[0], nest, false, newPrefix)
			e2 := newElement(values[1], nest, comma, nil)
			app.L.InsertBefore(e1, next)
			app.L.InsertBefore(e2, next)
			todo()
			app.nextLine(session)
			app.dirty = true
		case 1: // [\n value
			app.L.InsertBefore(
				newElement(values[0], nest, comma, newPrefix),
				next)
			todo()
			app.nextLine(session)
			app.dirty = true
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
		var nest int
		var newPrefix []byte
		todo := func() {}
		if element.value == Mark('}') {
			comma = false
			nest = element.nest + 1
			newPrefix = joinBytes(prefix, app.format.indent)
			todo = func() { setPrefix(next, prefix) }
		} else {
			if isDuplicated(next, element.nest, key) {
				app.message = fmt.Sprintf("\aduplicate key: %q", key)
				return
			}
			comma = true
			nest = element.nest
			newPrefix = getPrefix(next)
		}
		values := app.readNewValue(session, struct{}{})
		switch len(values) {
		case 2: // { key:[]
			p1 := newPair(key, values[0], nest, false)
			p1.preKey = newPrefix
			e2 := newElement(values[1], nest, comma, nil)
			app.L.InsertBefore(p1, next)
			app.L.InsertBefore(e2, next)
			todo()
			app.nextLine(session)
			app.dirty = true
		case 1: // { key:value
			p1 := newPair(key, values[0], nest, comma)
			p1.preKey = newPrefix
			app.L.InsertBefore(p1, next)
			todo()
			app.nextLine(session)
			app.dirty = true
		}
		return
	}
	if isHashElement(app.cursor) {
		key, err := app.ReadLine(session, "Key: ", "")
		if err != nil {
			return
		}
		element := ref(app.cursor)
		if isDuplicated(app.cursor, element.nest, key) {
			app.message = fmt.Sprintf("\aduplicate key: %q", key)
			return
		}
		values := app.readNewValue(session, struct{}{})
		switch len(values) {
		case 2: // key:[],
			p1 := newPair(key, values[0], element.nest, false)
			p1.preKey = prefix
			e2 := newElement(values[1], element.nest, element.comma, nil)
			app.L.InsertAfter(e2, app.cursor)
			app.L.InsertAfter(p1, app.cursor)
			app.nextLine(session)
			app.dirty = true
		case 1: // key:value,
			p := newPair(key, values[0], element.nest, element.comma)
			p.preKey = prefix
			app.L.InsertAfter(p, app.cursor)
			app.nextLine(session)
			app.dirty = true
		}
		element.comma = true
		return
	}
	if element, ok := app.cursor.Value.(*Element); ok {
		index := getIndex(app.cursor)
		if index < 0 {
			return
		}
		values := app.readNewValue(session, struct{}{})
		switch len(values) {
		case 2: // [ \n ],
			e1 := newElement(values[0], element.nest, false, prefix)
			e2 := newElement(values[1], element.nest, element.comma, nil)
			app.L.InsertAfter(e2, app.cursor)
			app.L.InsertAfter(e1, app.cursor)
			app.nextLine(session)
			app.dirty = true
		case 1: // value,
			e := newElement(values[0], element.nest, element.comma, prefix)
			app.L.InsertAfter(e, app.cursor)
			app.nextLine(session)
			app.dirty = true
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
		app.dirty = true
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
		app.dirty = true
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
	if element.nest == 0 {
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
	app.dirty = true
}

func (app *Application) save(session *pager.Session) bool {
	fname, err := app.ReadLine(session, "Write to:", app.Name)
	if err != nil {
		app.message = err.Error()
		return false
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
		return false
	}
	Dump(app.L, app.format, fd)
	if err := fd.Close(); err != nil {
		app.message = err.Error()
		return false
	}
	perm.Track(fd)
	app.Name = fname
	app.dirty = false
	return true
}

func (app *Application) quit(session *pager.Session) bool {
	if !app.dirty {
		return false // fallback to pager's quit
	}
	fmt.Fprint(session.TtyOut, "\rQuit: Save changes ? ['y': save, 'n': quit without saving, other: cancel]\x1B[0K")
	key, err := session.GetKey()
	if err != nil {
		app.message = err.Error()
		return true
	}
	if key == "y" || key == "Y" {
		// save and quit
		if !app.save(session) {
			return true
		}
		return false
	} else if key == "n" || key == "N" {
		// does not save, but quit
		return false // fallback to pager's quit
	} else {
		// cancel
		return true
	}
}

func (app *Application) nextLine(session *pager.Session) {
	c := app.cursor.Next()
	if c == nil {
		return
	}
	app.SetCursor(c)
	app.csrline++
	for app.csrline-app.winline >= session.Height {
		session.Window = session.Window.Next()
		app.winline++
	}
}

func (app *Application) Handle(session *pager.Session, key string) (bool, error) {
	switch key {
	default:
		return false, nil
	case "j", "\x1B[B":
		app.nextLine(session)
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
		app.save(session)
	case "q":
		return app.quit(session), nil
	}
	return true, nil
}

func (app *Application) Status(session *pager.Session) (rv string) {
	var mark rune
	if app.dirty {
		mark = '*'
	} else {
		mark = ' '
	}
	if app.message != "" {
		rv = fmt.Sprintf("\x1B[1m%s\x1B[22m%c\x1B[0K", app.message, mark)
		app.message = ""
	} else if app.Name != "" {
		rv = fmt.Sprintf("\x1B[7m%s\x1B[27m%c\x1B[0K", app.Name, mark)
	} else {
		rv = fmt.Sprintf("\x1B[1mJegan %s-%s-%s\x1B[22m\x1B[0K",
			version, runtime.GOOS, runtime.GOARCH)
	}
	return
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
