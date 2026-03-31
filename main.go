package main

import (
	"container/list"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/mattn/go-colorable"
	"github.com/mattn/go-runewidth"

	"github.com/nyaosorg/go-readline-ny"
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

type Mark rune

func (m Mark) String() string {
	return string(rune(m))
}

func (m Mark) GoString() string {
	return string(rune(m))
}

type Element struct {
	value  any
	indent int
	comma  bool
	cursor bool
	setter func(any)
}

func (e *Element) Display(w int) string {
	var b strings.Builder
	for i := 0; i < e.indent; i++ {
		b.WriteString("  ")
	}
	fmt.Fprintf(&b, "%#v", e.value)
	if e.comma {
		b.WriteByte(',')
	}
	line := runewidth.Truncate(b.String(), w-1, "")
	if e.cursor {
		line = "\x1B[4m" + runewidth.FillRight(line, w-1) + "\x1B[24m"
	}
	return line
}

func (e *Element) CanSetValue() bool {
	return e.setter != nil
}

func (e *Element) SetValue(value any) bool {
	debug("(*Element) SetValue:", value)
	if e.setter == nil {
		return false
	}
	e.value = value
	e.setter(value)
	return true
}

func (e *Element) Value() any {
	return e.value
}

func (e *Element) SetComma(value bool) {
	e.comma = value
}

func (e *Element) SetCursor(value bool) {
	e.cursor = value
}

type Pair struct {
	key string
	Element
}

func (pair *Pair) Display(w int) string {
	var b strings.Builder
	for i := 0; i < pair.indent; i++ {
		b.WriteString("  ")
	}
	fmt.Fprintf(&b, "%q: %#v", pair.key, pair.value)
	if pair.comma {
		b.WriteByte(',')
	}
	line := runewidth.Truncate(b.String(), w-1, "")
	if pair.cursor {
		line = "\x1B[4m" + runewidth.FillRight(line, w-1) + "\x1B[24m"
	}
	return line
}

func setComma(e *list.Element, value bool) {
	e.Value.(interface{ SetComma(bool) }).SetComma(value)
}

func setCursor(e *list.Element, value bool) {
	e.Value.(interface{ SetCursor(bool) }).SetCursor(value)
}

func setValue(e *list.Element, value any) bool {
	debug("setValue:", value)
	v, ok := e.Value.(interface{ SetValue(any) bool })
	return ok && v.SetValue(value)
}

func getValue(e *list.Element) (v any, ok bool) {
	obj, ok := e.Value.(interface{ Value() any })
	if ok {
		v = obj.Value()
	}
	return
}

func canSetValue(e *list.Element) bool {
	v, ok := e.Value.(interface{ CanSetValue() bool })
	return ok && v.CanSetValue()
}

func newElement(v any, i int, comma bool, setter func(any)) *Element {
	return &Element{value: v, indent: i, comma: comma, setter: setter}
}

func newPair(k string, v any, i int, comma bool, setter func(any)) *Pair {
	return &Pair{
		key:     k,
		Element: Element{value: v, indent: i, comma: comma, setter: setter},
	}
}

func canLet(v any) bool {
	if _, ok := v.(map[string]any); ok {
		return false
	}
	if _, ok := v.([]any); ok {
		return false
	}
	return true
}

func read(v any, indent int, setter func(any)) (L *list.List) {
	L = list.New()
	if x, ok := v.(map[string]any); ok {
		L.PushBack(newElement(Mark('{'), indent, false, nil))
		for key, val := range x {
			var setter1 func(any)
			if canLet(val) {
				key1 := key
				setter1 = func(v any) {
					debug("read: setter:", key, v)
					x[key1] = v
				}
			}
			sub := read(val, indent+1, setter1)
			first := sub.Remove(sub.Front()).(*Element)
			n := newPair(key, first.value, indent+1, first.comma, setter1)
			L.PushBack(n)
			L.PushBackList(sub)
		}
		setComma(L.Back(), false)
		L.PushBack(newElement(Mark('}'), indent, true, nil))
		return
	}
	if x, ok := v.([]any); ok {
		L.PushBack(newElement(Mark('['), indent, false, setter))
		for i, value := range x {
			var setter1 func(any)
			if canLet(value) {
				i1 := i
				setter1 = func(v any) {
					debug("read: setter:", i1, v)
					x[i1] = v
				}
			}
			sub := read(value, indent+1, setter1)
			L.PushBackList(sub)
		}
		setComma(L.Back(), false)
		L.PushBack(newElement(Mark(']'), indent, true, nil))
		return
	}
	L.PushBack(newElement(v, indent, true, setter))
	return
}

func Read(v any, indent int, setter func(any)) (L *list.List) {
	L = read(v, indent, setter)
	setComma(L.Back(), false)
	return L
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
	result, err := editor.ReadLine(context.Background())
	io.WriteString(session.TtyOut, "\x1B[?25l")
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
