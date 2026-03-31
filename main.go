package main

import (
	"container/list"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mattn/go-colorable"
	"github.com/mattn/go-runewidth"

	"github.com/nyaosorg/go-ttyadapter/tty8pe"

	"github.com/hymkor/jegan/internal/pager"
)

type Element struct {
	value  any
	indent int
	comma  bool
	cursor bool
}

func (e *Element) Display(w int) string {
	var b strings.Builder
	for i := 0; i < e.indent; i++ {
		b.WriteString("  ")
	}
	fmt.Fprint(&b, e.value)
	if e.comma {
		b.WriteByte(',')
	}
	line := runewidth.Truncate(b.String(), w-1, "")
	if e.cursor {
		line = "\x1B[4m" + runewidth.FillRight(line, w-1) + "\x1B[24m"
	}
	return line
}

func (e *Element) SetComma(value bool) {
	e.comma = value
}

func setComma(e *list.Element, value bool) {
	e.Value.(interface{ SetComma(bool) }).SetComma(value)
}

func newElement(v any, i int, comma bool) *Element {
	return &Element{value: v, indent: i, comma: comma}
}

func read(v any, indent int) (L *list.List) {
	L = list.New()
	if x, ok := v.(map[string]any); ok {
		L.PushBack(newElement("{", indent, false))
		for key, val := range x {
			sub := read(val, indent+1)
			first := sub.Remove(sub.Front()).(*Element)
			n := newElement(fmt.Sprintf("%#v:%v", key, first.value), indent+1, first.comma)
			L.PushBack(n)
			L.PushBackList(sub)
		}
		setComma(L.Back(), false)
		L.PushBack(newElement("}", indent, true))
		return
	}
	if x, ok := v.([]any); ok {
		L.PushBack(newElement("[", indent, false))
		for _, value := range x {
			sub := read(value, indent+1)
			L.PushBackList(sub)
		}
		setComma(L.Back(), false)
		L.PushBack(newElement("]", indent, false))
		return
	}
	L.PushBack(newElement(fmt.Sprintf("%#v", v), indent, true))
	return
}

func Read(v any, indent int) (L *list.List) {
	L = read(v, indent)
	setComma(L.Back(), false)
	return L
}

type Application struct {
	L       *list.List
	cursor  *list.Element
	csrline int
	winline int
}

func newApplication(L *list.List) *Application {
	cursor := L.Front()
	cursor.Value.(*Element).cursor = true

	return &Application{
		L:      L,
		cursor: cursor,
	}
}

func (app *Application) SetCursor(c *list.Element) {
	app.cursor.Value.(*Element).cursor = false
	app.cursor = c
	app.cursor.Value.(*Element).cursor = true
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
	}
	return true, nil
}

func main1(source io.Reader, title string) error {
	data, err := io.ReadAll(source)
	if err != nil {
		return err
	}
	var v any
	err = json.Unmarshal(data, &v)
	if err != nil {
		return err
	}
	L := Read(v, 0)
	app := newApplication(L)

	pager1 := &pager.Pager{
		Status: func(_ *pager.Session, out io.Writer) error {
			if title != "" {
				fmt.Fprintf(out, "\x1B[7m%s\x1B[0m", title)
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
		return main1(os.Stdin, "<STDIN>")
	}
	fd, err := os.Open(args[0])
	if err != nil {
		return err
	}
	defer fd.Close()
	return main1(fd, args[0])
}

func main() {
	if err := mains(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
