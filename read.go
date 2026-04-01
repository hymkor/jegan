package main

import (
	"container/list"
	"fmt"
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
	setter func(any)
	parent any
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

func newElement(v any, i int, comma bool, setter func(any), parent any) *Element {
	return &Element{
		value:  v,
		indent: i,
		comma:  comma,
		setter: setter,
		parent: parent}
}

func newPair(k string, v any, i int, comma bool, setter func(any), parent any) *Pair {
	return &Pair{
		key: k,
		Element: Element{
			value:  v,
			indent: i,
			comma:  comma,
			setter: setter,
			parent: parent}}
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

func read(v any, indent int, setter func(any), parent any) (L *list.List) {
	L = list.New()
	if x, ok := v.(map[string]any); ok {
		L.PushBack(newElement(Mark('{'), indent, false, nil, parent))
		for key, val := range x {
			var setter1 func(any)
			if canLet(val) {
				key1 := key
				setter1 = func(v any) {
					debug("read: setter:", key, v)
					x[key1] = v
				}
			}
			sub := read(val, indent+1, setter1, x)
			first := sub.Remove(sub.Front()).(*Element)
			n := newPair(key, first.value, indent+1, first.comma, setter1, x)
			L.PushBack(n)
			L.PushBackList(sub)
		}
		setComma(L.Back(), false)
		L.PushBack(newElement(Mark('}'), indent, true, nil, parent))
		return
	}
	if x, ok := v.([]any); ok {
		L.PushBack(newElement(Mark('['), indent, false, setter, parent))
		for i, value := range x {
			var setter1 func(any)
			if canLet(value) {
				i1 := i
				setter1 = func(v any) {
					debug("read: setter:", i1, v)
					x[i1] = v
				}
			}
			sub := read(value, indent+1, setter1, x)
			L.PushBackList(sub)
		}
		setComma(L.Back(), false)
		L.PushBack(newElement(Mark(']'), indent, true, nil, parent))
		return
	}
	L.PushBack(newElement(v, indent, true, setter, parent))
	return
}

func Read(v any, indent int, setter func(any)) (L *list.List) {
	L = read(v, indent, setter, nil)
	setComma(L.Back(), false)
	return L
}
