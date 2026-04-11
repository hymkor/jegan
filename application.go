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
	opt(editor)
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
	rawText, err := app.readLineElement(session, "New value:", defaults)
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
			text, err := app.readLineString(session, "New string:", fmt.Sprint(defaultv))
			if err != nil {
				session.TtyOut.Write([]byte{'\a'})
				return nil
			}
			return []any{text}
		case "n":
			text, err := app.readLine(session, "New number:", fmt.Sprint(defaultv))
			if err != nil {
				session.TtyOut.Write([]byte{'\a'})
				return nil
			}
			newValue, err := strconv.ParseFloat(text, 64)
			if err == nil {
				return []any{newValue}
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
		key, err := app.readLineString(session, "Key:", "")
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
		key, err := app.readLineString(session, "Key:", "")
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

func (app *Application) removeCursorAndNext() {
	prev := app.cursor.Prev()
	next := app.cursor.Next()
	if next == nil {
		return
	}
	newCurrent := next.Next()
	if newCurrent == nil {
		app.message = "Internal error: no valid cursor position after deletion"
		return
	}
	comma := ref(next).comma

	app.list.Remove(app.cursor)
	app.list.Remove(next)

	app.cursor = newCurrent
	ref(newCurrent).cursor = true
	if prev != nil {
		m, ok := ref(prev).value.(Mark)
		if !ok || (m != Mark('{') && m != Mark('[')) {
			ref(prev).comma = comma
		}
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
		app.message = "Cannot delete top-level object or array"
		return
	}
	next := app.cursor.Next()
	if next == nil {
		app.message = "Unexpected state: missing element after '{' or '['"
		return
	}
	n := ref(next)
	if n.value != Mark(']') && n.value != Mark('}') {
		app.message = "Cannot delete non-empty object or array"
		return
	}
	app.removeCursorAndNext()
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
		Dump(app.list, os.Stdout)
		os.Stdout.Write(app.Trailing)
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
	Dump(app.list, fd)
	fd.Write(app.Trailing)
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

func (app *Application) handle(session *pager.Session, key string) (pager.EventResult, error) {
	switch key {
	default:
		return pager.PassToPager, nil
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
	return pager.Handled, nil
}

func (app *Application) status(session *pager.Session) (rv string) {
	if app.message != "" {
		rv = fmt.Sprintf(ansi.Bold+"%s"+ansi.Thin+ansi.EraseLine, app.message)
		app.message = ""
	} else if app.Name != "" {
		var mark rune
		if app.dirty {
			mark = '*'
		} else {
			mark = ' '
		}
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
