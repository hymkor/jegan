package jegan

import (
	"container/list"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"unicode"

	"github.com/hymkor/jegan/internal/ansi"
	"github.com/hymkor/jegan/internal/unjson"
)

type Mark rune

func (m Mark) String() string {
	return string(rune(m))
}

func (m Mark) GoString() string {
	return string(rune(m))
}

func (m Mark) Json() []byte {
	return []byte{byte(m)}
}

type Node interface {
	LeadingSpace() []byte
	SetLeadingSpace(v []byte)
	Nest() int
	Comma() bool
	SetComma(bool)
	Value() any
	SetValue(any)
	SpaceCommaOrClose() []byte
	SetCursor(bool)
	SetSpaceCommaOrClose([]byte)
	Display(int) string
	Dump(w io.Writer)
}

func node(p *list.Element) Node {
	return p.Value.(Node)
}

type Element struct {
	value             any
	nest              int
	comma             bool
	cursor            bool
	spaceValue        []byte
	spaceCommaOrClose []byte
}

func (e *Element) LeadingSpace() []byte      { return e.spaceValue }
func (e *Element) SetLeadingSpace(v []byte)  { e.spaceValue = v }
func (e *Element) Nest() int                 { return e.nest }
func (e *Element) Comma() bool               { return e.comma }
func (e *Element) SetComma(v bool)           { e.comma = v }
func (e *Element) Value() any                { return e.value }
func (e *Element) SetValue(v any)            { e.value = v }
func (e *Element) SpaceCommaOrClose() []byte { return e.spaceCommaOrClose }
func (e *Element) SetCursor(v bool)          { e.cursor = v }
func (e *Element) SetSpaceCommaOrClose(v []byte) {
	e.spaceCommaOrClose = v
}

func (e *Element) Dump(w io.Writer) {
	w.Write(e.spaceValue)
	if v, ok := e.value.(interface{ Json() []byte }); ok {
		w.Write(v.Json())
	} else {
		b, err := json.Marshal(e.value)
		if err != nil {
			fmt.Fprint(w, e.value)
		} else {
			w.Write(b)
		}
	}
	w.Write(e.spaceCommaOrClose)
	if e.comma {
		w.Write([]byte{','})
	}
}

func highlightString(s []byte, color string, b *strings.Builder) {
	L := len(s) - 1
	if len(s) >= 2 && s[0] == '"' && s[L] == '"' {
		b.WriteByte('"')
		b.WriteString(color)
		b.Write(s[1:L])
		b.WriteString(ansi.Default)
		b.WriteByte('"')
	} else {
		b.Write(s)
	}
}

func (e *Element) highlight(b *strings.Builder) {
	if e.comma {
		defer b.WriteByte(',')
	}
	v := e.value
	if m, ok := v.(Mark); ok {
		b.WriteString(ansi.Red)
		b.WriteRune(rune(m))
		b.WriteString(ansi.Default)
		return
	}
	if x, ok := v.(*unjson.RawBytes); ok {
		b.WriteString(ansi.Red)
		escape := false
		for _, v := range x.String() {
			if escape {
				if ('a' <= v && v <= 'z') || ('A' <= v && v <= 'Z') {
					escape = false
				}
				continue
			}
			if v == '\x1B' {
				escape = true
				continue
			}
			if unicode.IsSpace(v) {
				b.Write([]byte{' '})
			} else {
				b.WriteRune(v)
			}
		}
		b.WriteString(ansi.Default)
		return
	}
	if x, ok := v.(*modifiedLiteral); ok {
		io.WriteString(b, ansi.Bold)
		defer io.WriteString(b, ansi.Thin)
		v = x.Literal
	}
	if x, ok := v.(*unjson.Literal); ok {
		value := x.Value()
		if _, ok := value.(string); ok {
			highlightString(x.Json(), ansi.Magenta, b)
			return
		}
		if _, ok := value.(float64); ok {
			b.Write(x.Json())
			return
		}
		v = x.Value()
	} else {
		io.WriteString(b, ansi.Bold)
		defer io.WriteString(b, ansi.Thin)
	}
	if s, ok := v.(string); ok {
		jsonBin, _ := json.Marshal(s)
		highlightString(jsonBin, ansi.Magenta, b)
	} else if v == true {
		io.WriteString(b, ansi.Cyan+"true"+ansi.Default)
	} else if v == false {
		io.WriteString(b, ansi.Cyan+"false"+ansi.Default)
	} else if v == nil {
		io.WriteString(b, ansi.Cyan+"null"+ansi.Default)
	} else {
		bin, err := json.Marshal(v)
		if err != nil {
			fmt.Fprint(b, v)
		} else {
			b.Write(bin)
		}
	}
}

func (e *Element) Display(w int) string {
	var b strings.Builder
	if e.cursor {
		b.WriteString(ansi.UnderLine)
	}
	for i := 0; i < e.nest; i++ {
		b.WriteString("  ")
	}
	e.highlight(&b)
	if e.cursor {
		b.WriteString(strings.Repeat(" ", w))
		b.WriteString(ansi.NoUnderLine)
	}
	return b.String()
}

type Pair struct {
	key string
	Element
	spaceKey   []byte
	spaceColon []byte
}

func (p *Pair) LeadingSpace() []byte     { return p.spaceKey }
func (p *Pair) SetLeadingSpace(v []byte) { p.spaceValue = v }

func (pair *Pair) Display(w int) string {
	var b strings.Builder
	if pair.cursor {
		b.WriteString(ansi.UnderLine)
	}
	for i := 0; i < pair.nest; i++ {
		b.WriteString("  ")
	}
	jsonBin, _ := json.Marshal(pair.key)
	highlightString(jsonBin, ansi.Yellow, &b)
	b.WriteString(": ")
	pair.Element.highlight(&b)
	if pair.cursor {
		b.WriteString(strings.Repeat(" ", w))
		b.WriteString(ansi.NoUnderLine)
	}
	return b.String()
}

func newElement(v any, i int, comma bool, prefix []byte) *Element {
	return &Element{
		value:      v,
		nest:       i,
		comma:      comma,
		spaceValue: prefix}
}

func (p *Pair) Dump(w io.Writer) {
	w.Write(p.spaceKey)
	b, _ := json.Marshal(p.key)
	w.Write(b)
	w.Write(p.spaceColon)
	w.Write([]byte{':'})
	p.Element.Dump(w)
}

func read(t *unjson.Entry, nest int) (L *list.List) {
	v := t.Value
	prefix := t.SpaceValue
	L = list.New()
	if x, ok := v.(*unjson.Object); ok {
		if len(x.Pairs) <= 0 {
			L.PushBack(newElement(Mark('{'), nest, false, prefix))
			L.PushBack(newElement(Mark('}'), nest, true, x.Blank))
			return
		}
		v = []unjson.KeyValuePair(x.Pairs)
	} else if x, ok := v.(*unjson.Array); ok {
		if len(x.Element) <= 0 {
			L.PushBack(newElement(Mark('['), nest, false, prefix))
			L.PushBack(newElement(Mark(']'), nest, true, x.Blank))
			return
		}
		v = []unjson.ArrayElement(x.Element)
	}
	if x, ok := v.([]unjson.KeyValuePair); ok {
		L.PushBack(newElement(Mark('{'), nest, false, prefix))
		for _, kv := range x {
			sub := read(kv.Value, nest+1)
			first := sub.Remove(sub.Front()).(*Element)
			n := &Pair{
				spaceKey:   kv.SpaceKey,
				key:        kv.Key,
				spaceColon: kv.SpaceColon,
				Element: Element{
					spaceValue: kv.Value.SpaceValue,
					value:      first.value,
					nest:       nest + 1,
					comma:      first.comma,
				},
			}
			L.PushBack(n)
			L.PushBackList(sub)
			if sub.Len() > 0 {
				node(sub.Back()).SetSpaceCommaOrClose(kv.SpaceCommaOrClose)
			} else {
				n.SetSpaceCommaOrClose(kv.SpaceCommaOrClose)
			}
		}
		back := node(L.Back())
		finalSpace := back.SpaceCommaOrClose()
		back.SetSpaceCommaOrClose(nil)
		back.SetComma(false)
		L.PushBack(newElement(Mark('}'), nest, true, finalSpace))
		return
	}
	if x, ok := v.([]unjson.ArrayElement); ok {
		L.PushBack(newElement(Mark('['), nest, false, prefix))
		for _, v := range x {
			sub := read(v.Entry, nest+1)
			node(sub.Back()).SetSpaceCommaOrClose(v.PreComma)
			L.PushBackList(sub)
		}
		back := node(L.Back())
		finalSpace := back.SpaceCommaOrClose()
		back.SetSpaceCommaOrClose(nil)
		back.SetComma(false)
		L.PushBack(newElement(Mark(']'), nest, true, finalSpace))
		return

	}
	L.PushBack(newElement(v, nest, true, prefix))
	return
}

func Read(v *unjson.Entry) (L *list.List) {
	if v == nil {
		return nil
	}
	L = read(v, 0)
	if L != nil && L.Len() > 0 {
		node(L.Back()).SetComma(false)
	}
	return L
}

func Dump(L *list.List, w io.Writer) {
	for p := L.Front(); p != nil; p = p.Next() {
		node(p).Dump(w)
	}
}
