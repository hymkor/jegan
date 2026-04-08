package main

import (
	"bufio"
	"bytes"
	"container/list"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/nyaosorg/go-readline-ny"
	"github.com/nyaosorg/go-readline-ny/keys"
	"github.com/nyaosorg/go-readline-skk"
	"github.com/nyaosorg/go-ttyadapter"

	"github.com/hymkor/go-safewrite"
	"github.com/hymkor/go-safewrite/perm"
	"github.com/hymkor/jegan/internal/ansi"
	"github.com/hymkor/jegan/internal/pager"

	"github.com/hymkor/jegan/internal/unjson"
)

type Application struct {
	Name     string
	Trailing []byte

	list    *list.List
	cursor  *list.Element
	csrline int
	winline int
	message string
	dirty   bool
	indent  []byte
}

func (app *Application) Store(v *list.List) {
	if v == nil {
		return
	}
	if app.list == nil {
		app.list = v
	} else {
		app.list.PushBackList(v)
	}
}

func (app *Application) setCursor(c *list.Element) {
	ref(app.cursor).cursor = false
	app.cursor = c
	ref(app.cursor).cursor = true
}

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
	skkInit()
	cursorPosition := 65535
	if len(defaults) > 0 && strings.IndexByte(`"]}`, defaults[len(defaults)-1]) >= 0 {
		cursorPosition = readline.MojiCountInString(defaults) - 1
	} else if len(defaults) > 5 && strings.HasSuffix(defaults, ".json") {
		cursorPosition = readline.MojiCountInString(defaults) - 5
	}
	editor := &readline.Editor{
		Writer: session.TtyOut,
		PromptWriter: func(w io.Writer) (int, error) {
			return fmt.Fprintf(w, "\r%s "+ansi.EraseLine, prompt)
		},
		LineFeedWriter: func(readline.Result, io.Writer) (int, error) {
			return 0, nil
		},
		Cursor:  cursorPosition,
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
	result, err := editor.ReadLine(context.Background())
	io.WriteString(session.TtyOut, ansi.CursorOff)
	if err == readline.CtrlC {
		return "", errors.New("Canceled")
	}
	return result, err
}

func (app *Application) keyFuncReplace(
	session *pager.Session,
	input func(*pager.Session, any) []any) {

	element := ref(app.cursor)
	defaultv := element.value
	if _, ok := defaultv.(*unjson.RawBytes); ok {
		return
	}
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
			prev = func() bool { app.list.Remove(next); return true }
		} else if v == Mark('[') && nextv == Mark(']') {
			defaultv = []any{}
			prev = func() bool { app.list.Remove(next); return true }
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
		app.list.InsertAfter(
			newElement(values[1], element.nest, element.comma, prefix),
			app.cursor)
		element.value = values[0]
		element.comma = false
		app.dirty = true
	}
}

func (app *Application) inputFormat(session *pager.Session, defaultv any) []any {
	var defaults string
	if _, ok := defaultv.(struct{}); ok {
		defaults = ""
	} else if v, ok := defaultv.(interface{ Json() []byte }); ok {
		defaults = string(v.Json())
	} else {
		b, err := json.Marshal(defaultv)
		if err != nil {
			debug("(*Application) inputFormat: json.Marshal:", err.Error(), "for", defaultv)
		}
		defaults = string(b)
	}
	rawText, err := app.readLine(session, "New value:", defaults)
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

func (app *Application) inputTypeAndValue(session *pager.Session, defaultv any) []any {
	io.WriteString(session.TtyOut, ansi.CursorOn)
	defer io.WriteString(session.TtyOut, ansi.CursorOff)
	for {
		io.WriteString(session.TtyOut,
			"\r's':string, 'n':number, 'u':null, "+
				"'t':true, 'f':false, 'o':{}, 'a':[] ? "+ansi.EraseLine)
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
			text, err := app.readLine(session, "New string:", fmt.Sprint(defaultv))
			if err == nil {
				return []any{text}
			}
			session.TtyOut.Write([]byte{'\a'})
		case "n":
			text, err := app.readLine(session, "New number:", fmt.Sprint(defaultv))
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

func (app *Application) keyFuncInsert(session *pager.Session) {
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
			newPrefix = joinBytes(prefix, app.indent)
		} else {
			comma = true
			nest = nextElement.nest
			newPrefix = getPrefix(next)
		}
		values := app.inputFormat(session, struct{}{})
		switch len(values) {
		case 2: // [\n[\n],\n
			e1 := newElement(values[0], nest, false, newPrefix)
			e2 := newElement(values[1], nest, comma, nil)
			app.list.InsertBefore(e1, next)
			app.list.InsertBefore(e2, next)
			todo()
			app.nextLine(session)
			app.dirty = true
		case 1: // [\n value
			app.list.InsertBefore(
				newElement(values[0], nest, comma, newPrefix),
				next)
			todo()
			app.nextLine(session)
			app.dirty = true
		}
		return
	}
	if element := ref(app.cursor); element.value == Mark('{') {
		key, err := app.readLine(session, "Key: ", "")
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
			newPrefix = joinBytes(prefix, app.indent)
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
		values := app.inputFormat(session, struct{}{})
		switch len(values) {
		case 2: // { key:[]
			p1 := newPair(key, values[0], nest, false)
			p1.preKey = newPrefix
			e2 := newElement(values[1], nest, comma, nil)
			app.list.InsertBefore(p1, next)
			app.list.InsertBefore(e2, next)
			todo()
			app.nextLine(session)
			app.dirty = true
		case 1: // { key:value
			p1 := newPair(key, values[0], nest, comma)
			p1.preKey = newPrefix
			app.list.InsertBefore(p1, next)
			todo()
			app.nextLine(session)
			app.dirty = true
		}
		return
	}
	if isHashElement(app.cursor) {
		key, err := app.readLine(session, "Key: ", "")
		if err != nil {
			return
		}
		element := ref(app.cursor)
		if isDuplicated(app.cursor, element.nest, key) {
			app.message = fmt.Sprintf("\aduplicate key: %q", key)
			return
		}
		values := app.inputFormat(session, struct{}{})
		switch len(values) {
		case 2: // key:[],
			p1 := newPair(key, values[0], element.nest, false)
			p1.preKey = prefix
			e2 := newElement(values[1], element.nest, element.comma, nil)
			app.list.InsertAfter(e2, app.cursor)
			app.list.InsertAfter(p1, app.cursor)
			app.nextLine(session)
			app.dirty = true
		case 1: // key:value,
			p := newPair(key, values[0], element.nest, element.comma)
			p.preKey = prefix
			app.list.InsertAfter(p, app.cursor)
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
		values := app.inputFormat(session, struct{}{})
		switch len(values) {
		case 2: // [ \n ],
			e1 := newElement(values[0], element.nest, false, prefix)
			e2 := newElement(values[1], element.nest, element.comma, nil)
			app.list.InsertAfter(e2, app.cursor)
			app.list.InsertAfter(e1, app.cursor)
			app.nextLine(session)
			app.dirty = true
		case 1: // value,
			e := newElement(values[0], element.nest, element.comma, prefix)
			app.list.InsertAfter(e, app.cursor)
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
		app.list.Remove(app.cursor)
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
		app.list.Remove(app.cursor)
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

func (app *Application) keyFuncRemove(session *pager.Session) {
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
	app.list.Remove(next)
	app.removeCursor(session)
	ref(app.cursor).comma = comma
	app.dirty = true
}

func (app *Application) keyFuncSave(session *pager.Session) bool {
	fname, err := app.readLine(session, "Write to:", app.Name)
	if err != nil {
		app.message = err.Error()
		return false
	}
	fd, err := safewrite.Open(fname, func(info *safewrite.Info) bool {
		io.WriteString(session.TtyOut, ansi.CursorOn)
		defer io.WriteString(session.TtyOut, ansi.CursorOff)
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
	Dump(app.list, fd)
	fd.Write(app.Trailing)
	if err := fd.Close(); err != nil {
		app.message = err.Error()
		return false
	}
	perm.Track(fd)
	app.Name = fname
	app.dirty = false
	return true
}

func (app *Application) keyFuncQuit(session *pager.Session) bool {
	if !app.dirty {
		return false // fallback to pager's quit
	}
	io.WriteString(session.TtyOut, ansi.CursorOn)
	defer io.WriteString(session.TtyOut, ansi.CursorOff)

	io.WriteString(session.TtyOut, "\rQuit: Save changes ? ['y': save, 'n': quit without saving, other: cancel]"+ansi.EraseLine)
	key, err := session.GetKey()
	if err != nil {
		app.message = err.Error()
		return true
	}
	if key == "y" || key == "Y" {
		// save and quit
		if !app.keyFuncSave(session) {
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
	app.setCursor(c)
	app.csrline++
	for app.csrline-app.winline >= session.Height {
		session.Window = session.Window.Next()
		app.winline++
	}
}

func (app *Application) handle(session *pager.Session, key string) (bool, error) {
	switch key {
	default:
		return false, nil
	case "j", keys.Down, keys.CtrlN:
		app.nextLine(session)
	case "k", keys.Up, keys.CtrlP:
		if c := app.cursor.Prev(); c != nil {
			app.setCursor(c)
			app.csrline--
			for app.csrline < app.winline {
				session.Window = session.Window.Prev()
				app.winline--
			}
		}
	case "<":
		app.setCursor(app.list.Front())
		session.Front()
		app.winline = 0
		app.csrline = 0
	case ">":
		app.setCursor(app.list.Back())
		n := session.Back()
		app.csrline = app.list.Len() - 1
		app.winline = app.list.Len() - 1 - n
	case " ", "b", keys.CtrlC, keys.CtrlG:
	case "r":
		app.keyFuncReplace(session, app.inputFormat)
	case "R":
		app.keyFuncReplace(session, app.inputTypeAndValue)
	case "o":
		app.keyFuncInsert(session)
	case "d":
		app.keyFuncRemove(session)
	case "w":
		app.keyFuncSave(session)
	case "q":
		return app.keyFuncQuit(session), nil
	}
	return true, nil
}

func (app *Application) status(session *pager.Session) (rv string) {
	var mark rune
	if app.dirty {
		mark = '*'
	} else {
		mark = ' '
	}
	if app.message != "" {
		rv = fmt.Sprintf(ansi.Bold+"%s"+ansi.Thin+"%c"+ansi.EraseLine, app.message, mark)
		app.message = ""
	} else if app.Name != "" {
		rv = fmt.Sprintf(ansi.Reverse+"%s"+ansi.Inverse+"%c"+ansi.EraseLine, app.Name, mark)
	} else {
		rv = fmt.Sprintf(ansi.Bold+"Jegan %s-%s-%s"+ansi.Thin+ansi.EraseLine,
			version, runtime.GOOS, runtime.GOARCH)
	}
	return
}

func (app *Application) EventLoop(tty ttyadapter.Tty, ttyout io.Writer) error {
	if app.list == nil {
		app.list = list.New()
	}
	if app.list.Len() <= 0 {
		app.list.PushBack(newElement(Mark('{'), 0, false, nil))
		app.list.PushBack(newElement(Mark('}'), 0, false, nil))
	}
	if app.cursor == nil {
		app.cursor = app.list.Front()
		ref(app.cursor).cursor = true
	}
	if sample := app.list.Front().Next(); sample != nil {
		prefix := getPrefix(sample)
		if pos := bytes.IndexByte(prefix, '\n'); pos >= 0 {
			app.indent = prefix[pos+1:]
		}
	}
	pager1 := &pager.Pager{
		Status:  app.status,
		Handler: app.handle,
	}
	return pager1.EventLoop(tty, app.list, ttyout)
}

func (app *Application) Close() error {
	return perm.RestoreAll()
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
				return nil
			}
			if name == "" {
				return fmt.Errorf("<STDIN>:%w", err)
			}
			return fmt.Errorf("%s:%w", name, err)
		}
		app.Store(Read(v))
	}
}
