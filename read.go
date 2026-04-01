package main

import (
	"container/list"
	"fmt"
	"io"
	"strings"

	"github.com/mattn/go-runewidth"
)

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
}

func (e *Element) Dump(w io.Writer) {
	if e.value == nil {
		io.WriteString(w, "null")
	} else {
		fmt.Fprintf(w, "%#v", e.value)
	}
	if e.comma {
		w.Write([]byte{','})
	}
}

func (e *Element) Display(w int) string {
	var b strings.Builder
	for i := 0; i < e.indent; i++ {
		b.WriteString("  ")
	}
	e.Dump(&b)
	line := runewidth.Truncate(b.String(), w-1, "")
	if e.cursor {
		line = "\x1B[4m" + runewidth.FillRight(line, w-1) + "\x1B[24m"
	}
	return line
}

func (e *Element) SetValue(value any) bool {
	e.value = value
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

func (e *Element) Indent() int {
	return e.indent
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

func getElement(e *list.Element) (*Element, bool) {
	pair, ok := e.Value.(*Pair)
	if ok {
		return &pair.Element, true
	}
	v, ok := e.Value.(*Element)
	return v, ok
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

func newElement(v any, i int, comma bool) *Element {
	return &Element{
		value:  v,
		indent: i,
		comma:  comma}
}

func newPair(k string, v any, i int, comma bool) *Pair {
	return &Pair{
		key: k,
		Element: Element{
			value:  v,
			indent: i,
			comma:  comma}}
}

func (p *Pair) Dump(w io.Writer) {
	fmt.Fprintf(w, "%q:", p.key)
	p.Element.Dump(w)
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

func read(v any, indent int) (L *list.List) {
	L = list.New()
	if x, ok := v.(map[string]any); ok {
		L.PushBack(newElement(Mark('{'), indent, false))
		for key, val := range x {
			sub := read(val, indent+1)
			first := sub.Remove(sub.Front()).(*Element)
			n := newPair(key, first.value, indent+1, first.comma)
			L.PushBack(n)
			L.PushBackList(sub)
		}
		setComma(L.Back(), false)
		L.PushBack(newElement(Mark('}'), indent, true))
		return
	}
	if x, ok := v.([]any); ok {
		L.PushBack(newElement(Mark('['), indent, false))
		for _, value := range x {
			sub := read(value, indent+1)
			L.PushBackList(sub)
		}
		setComma(L.Back(), false)
		L.PushBack(newElement(Mark(']'), indent, true))
		return
	}
	L.PushBack(newElement(v, indent, true))
	return
}

func Read(v any) (L *list.List) {
	L = read(v, 0)
	setComma(L.Back(), false)
	return L
}

func Dump(L *list.List, w io.Writer) {
	for p := L.Front(); p != nil; p = p.Next() {
		p.Value.(interface{ Dump(io.Writer) }).Dump(w)
	}
}
