package main

import (
	"container/list"
	"encoding/json"
	"io"
	"strings"
)

type Mark rune

func (m Mark) String() string {
	return string(rune(m))
}

func (m Mark) GoString() string {
	return string(rune(m))
}

type Element struct {
	value   any
	nest    int
	comma   bool
	cursor  bool
	prefix  []byte
	postfix []byte
}

func (e *Element) Dump(w io.Writer) {
	w.Write(e.prefix)
	if v, ok := e.value.(Mark); ok {
		w.Write([]byte{byte(v)})
	} else {
		b, err := json.Marshal(e.value)
		if err != nil {
			debug("(*Element) Dump: json.Marshal:", err.Error(), "for", e.value)
		}
		w.Write(b)
	}
	w.Write(e.postfix)
	if e.comma {
		w.Write([]byte{','})
	}
}

func highlightString(s string, color string, b *strings.Builder) {
	L := len(s) - 1
	if len(s) >= 2 && s[0] == '"' && s[L] == '"' {
		b.WriteByte('"')
		b.WriteString(color) // "\x1B[35m"
		b.WriteString(s[1:L])
		b.WriteString("\x1B[39m")
		b.WriteByte('"')
	} else {
		b.WriteString(s)
	}
}

func (e *Element) highlight(b *strings.Builder) {
	const (
		cyan   = "\x1B[36m"
		normal = "\x1B[39m"
	)
	v := e.value
	if m, ok := v.(Mark); ok {
		b.WriteString("\x1B[31m")
		b.WriteRune(rune(m))
		b.WriteString(normal)
	} else if s, ok := v.(string); ok {
		jsonBin, _ := json.Marshal(s)
		highlightString(string(jsonBin), "\x1B[35m", b)
	} else if v == true {
		io.WriteString(b, cyan+"true"+normal)
	} else if v == false {
		io.WriteString(b, cyan+"false"+normal)
	} else if v == nil {
		io.WriteString(b, cyan+"null"+normal)
	} else {
		bin, _ := json.Marshal(e.value)
		b.Write(bin)
		return
	}
	if e.comma {
		b.WriteByte(',')
	}
}

func (e *Element) Display(w int) string {
	var b strings.Builder
	if e.cursor {
		b.WriteString("\x1B[4m")
	}
	for i := 0; i < e.nest; i++ {
		b.WriteString("  ")
	}
	e.highlight(&b)
	if e.cursor {
		b.WriteString(strings.Repeat(" ", w))
		b.WriteString("\x1B[24m")
	}
	return b.String()
}

type Pair struct {
	key string
	Element
	preKey []byte
	preCol []byte
}

func (pair *Pair) Display(w int) string {
	var b strings.Builder
	if pair.cursor {
		b.WriteString("\x1B[4m")
	}
	for i := 0; i < pair.nest; i++ {
		b.WriteString("  ")
	}
	jsonBin, _ := json.Marshal(pair.key)
	highlightString(string(jsonBin), "\x1B[33m", &b)
	b.WriteString(": ")
	pair.Element.highlight(&b)
	if pair.cursor {
		b.WriteString(strings.Repeat(" ", w))
		b.WriteString("\x1B[24m")
	}
	return b.String()
}

func ref(e *list.Element) *Element {
	pair, ok := e.Value.(*Pair)
	if ok {
		return &pair.Element
	}
	return e.Value.(*Element)
}

func newElement(v any, i int, comma bool, prefix []byte) *Element {
	return &Element{
		value:  v,
		nest:   i,
		comma:  comma,
		prefix: prefix}
}

func newPair(k string, v any, i int, comma bool) *Pair {
	return &Pair{
		key: k,
		Element: Element{
			value: v,
			nest:  i,
			comma: comma}}
}

func (p *Pair) Dump(w io.Writer) {
	w.Write(p.preKey)
	b, _ := json.Marshal(p.key)
	w.Write(b)
	w.Write(p.preCol)
	w.Write([]byte{':'})
	p.Element.Dump(w)
}

func read(v any, nest int) (L *list.List) {
	var prefix []byte
	if t, ok := v.(*token); ok {
		v = t.value
		prefix = t.prefix
	}
	L = list.New()
	if x, ok := v.(object); ok {
		v = []keyValuePair(x)
	}
	if x, ok := v.([]keyValuePair); ok {
		L.PushBack(newElement(Mark('{'), nest, false, prefix))
		for _, kv := range x {
			key := kv.key
			val := kv.value
			sub := read(val, nest+1)
			first := sub.Remove(sub.Front()).(*Element)
			n := newPair(key, first.value, nest+1, first.comma)
			n.preKey = kv.preKey
			n.preCol = kv.preCol
			n.Element.prefix = kv.value.prefix
			L.PushBack(n)
			L.PushBackList(sub)
			if sub.Len() >= 1 {
				ref(sub.Back()).postfix = kv.last
			} else {
				n.Element.postfix = kv.last
			}
		}
		ref(L.Back()).comma = false
		L.PushBack(newElement(Mark('}'), nest, true, nil))
		return

	}
	if x, ok := v.(map[string]any); ok {
		L.PushBack(newElement(Mark('{'), nest, false, prefix))
		for key, val := range x {
			sub := read(val, nest+1)
			first := sub.Remove(sub.Front()).(*Element)
			n := newPair(key, first.value, nest+1, first.comma)
			L.PushBack(n)
			L.PushBackList(sub)
		}
		ref(L.Back()).comma = false
		L.PushBack(newElement(Mark('}'), nest, true, nil))
		return
	}
	if x, ok := v.(array); ok {
		v = []arrayElement(x)
	}
	if x, ok := v.([]arrayElement); ok {
		L.PushBack(newElement(Mark('['), nest, false, prefix))
		for _, v := range x {
			sub := read(v.token, nest+1)
			ref(sub.Back()).postfix = v.preComma
			L.PushBackList(sub)
		}
		ref(L.Back()).comma = false
		L.PushBack(newElement(Mark(']'), nest, true, nil))
		return

	}
	if x, ok := v.([]any); ok {
		L.PushBack(newElement(Mark('['), nest, false, prefix))
		for _, value := range x {
			sub := read(value, nest+1)
			L.PushBackList(sub)
		}
		ref(L.Back()).comma = false
		L.PushBack(newElement(Mark(']'), nest, true, nil))
		return
	}
	L.PushBack(newElement(v, nest, true, prefix))
	return
}

func Read(v any) (L *list.List) {
	L = read(v, 0)
	ref(L.Back()).comma = false
	return L
}

func Dump(L *list.List, format *Format, w io.Writer) {
	for p := L.Front(); p != nil; p = p.Next() {
		p.Value.(interface{ Dump(io.Writer) }).Dump(w)
	}
}
