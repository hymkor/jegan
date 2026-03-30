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
	return runewidth.Truncate(b.String(), w-1, "")
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
		L.Back().Value.(*Element).comma = false
		L.PushBack(newElement("}", indent, true))
		return
	}
	if x, ok := v.([]any); ok {
		L.PushBack(newElement("[", indent, false))
		for _, value := range x {
			sub := read(value, indent+1)
			L.PushBackList(sub)
		}
		L.Back().Value.(*Element).comma = false
		L.PushBack(newElement("]", indent, false))
		return
	}
	L.PushBack(newElement(fmt.Sprintf("%#v", v), indent, true))
	return
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
	L := read(v, 0)

	pager1 := &pager.Pager{
		Status: func(_ *pager.Session, out io.Writer) error {
			if title != "" {
				fmt.Fprintf(out, "\x1B[7m%s\x1B[0m", title)
			}
			return nil
		},
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
