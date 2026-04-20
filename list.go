package jegan

import (
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/hymkor/go-generics-list"
	"github.com/hymkor/jegan/internal/ansi"
	"github.com/hymkor/jegan/internal/unjson"
)

type JsonPath struct {
	parent *JsonPath
	text   string
	index  int
}

func (j *JsonPath) ChildIndex(i int) *JsonPath {
	return &JsonPath{
		parent: j,
		index:  i,
	}
}

func (j *JsonPath) ChildKey(key string) *JsonPath {
	return &JsonPath{
		parent: j,
		text:   key,
		index:  -1,
	}
}

var rxSymbol = regexp.MustCompile("^[_A-Za-z][_A-Za-z0-9]*$")

func (j *JsonPath) Dump(w io.Writer) {
	if j == nil {
		return
	}
	if j.parent != nil {
		j.parent.Dump(w)
	}
	if j.text != "" {
		if rxSymbol.MatchString(j.text) {
			fmt.Fprintf(w, ".%s", j.text)
		} else {
			fmt.Fprintf(w, ".%q", j.text)
		}
	} else {
		if j.parent == nil {
			w.Write([]byte{'.'})
		}
		fmt.Fprintf(w, "[%d]", j.index)
	}
}

type Line interface {
	LeadingSpace() []byte
	SetLeadingSpace(v []byte)
	Data() any
	SetData(any)
	SpaceCommaOrClose() []byte
	SetSpaceCommaOrClose([]byte)
	Comma() bool
	SetComma(bool)

	Nest() int
	SetCursor(bool)

	Path() *JsonPath
	SetPath(*JsonPath)

	Display(int) string
	Dump(w io.Writer)
	DumpWithoutComma(w io.Writer)
}

func ref(p *list.Element[Line]) Line {
	return p.Value
}

type Item struct {
	spaceValue        []byte
	data              any
	spaceCommaOrClose []byte
	comma             bool

	nest   int
	cursor bool
	path   *JsonPath
}

func (e *Item) LeadingSpace() []byte          { return e.spaceValue }
func (e *Item) SetLeadingSpace(v []byte)      { e.spaceValue = v }
func (e *Item) Data() any                     { return e.data }
func (e *Item) SetData(v any)                 { e.data = v }
func (e *Item) SpaceCommaOrClose() []byte     { return e.spaceCommaOrClose }
func (e *Item) SetSpaceCommaOrClose(v []byte) { e.spaceCommaOrClose = v }
func (e *Item) Comma() bool                   { return e.comma }
func (e *Item) SetComma(v bool)               { e.comma = v }
func (e *Item) Nest() int                     { return e.nest }
func (e *Item) SetCursor(v bool)              { e.cursor = v }
func (e *Item) Path() *JsonPath               { return e.path }
func (e *Item) SetPath(v *JsonPath)           { e.path = v }

func (e *Item) Dump(w io.Writer) {
	e.DumpWithoutComma(w)
	if e.comma {
		w.Write([]byte{','})
	}
}

func (e *Item) DumpWithoutComma(w io.Writer) {
	if _, ok := e.data.(*tombstone); ok {
		return
	}
	w.Write(e.spaceValue)
	if v, ok := e.data.(interface{ Json() []byte }); ok {
		w.Write(v.Json())
	} else {
		b, err := json.Marshal(e.data)
		if err != nil {
			fmt.Fprint(w, e.data)
		} else {
			w.Write(b)
		}
	}
	w.Write(e.spaceCommaOrClose)
}

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

func (m Mark) Equals(v any) bool {
	v = unwrap(v)
	return v == m
}

func (m Mark) Render(b *strings.Builder, _ func(any, *strings.Builder)) {
	b.WriteString(ansi.Red)
	b.WriteRune(rune(m))
	b.WriteString(ansi.Default)
}

const (
	objStart   = Mark('{')
	objEnd     = Mark('}')
	arrayStart = Mark('[')
	arrayEnd   = Mark(']')
)

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

func (e *Item) highlight(b *strings.Builder) {
	e.highlightWithoutComma(b)
	if e.comma {
		b.WriteByte(',')
	}
}

func (e *Item) highlightWithoutComma(b *strings.Builder) {
	render(e.data, b)
}

func render(v any, b *strings.Builder) {
	type renderType interface {
		Render(*strings.Builder, func(any, *strings.Builder))
	}
	if r, ok := v.(renderType); ok {
		r.Render(b, render)
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
	if x, ok := v.(*unjson.Literal); ok {
		v = x.Data()
		if _, ok := v.(string); ok {
			highlightString(x.Json(), ansi.Magenta, b)
			return
		}
		if _, ok := v.(float64); ok {
			b.Write(x.Json())
			return
		}
	} else {
		io.WriteString(b, ansi.Bold)
		defer io.WriteString(b, ansi.Thin)
	}
	if s, ok := v.(string); ok {
		jsonBin, err := json.Marshal(s)
		if err != nil {
			debug("(*Entry) highlight", s, err.Error())
			jsonBin = []byte(strconv.Quote(s))
		}
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

func (e *Item) Display(w int) string {
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
	spaceKey   []byte
	key        string
	spaceColon []byte
	Item
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
	jsonBin, err := json.Marshal(pair.key)
	if err != nil {
		debug("(*Pair) Display", pair.key, err.Error())
		jsonBin = []byte(strconv.Quote(pair.key))
	}
	highlightString(jsonBin, ansi.Yellow, &b)
	b.WriteString(": ")
	pair.Item.highlight(&b)
	if pair.cursor {
		b.WriteString(strings.Repeat(" ", w))
		b.WriteString(ansi.NoUnderLine)
	}
	return b.String()
}

func newElement(v any, i int, comma bool, prefix []byte) *Item {
	return &Item{
		spaceValue: prefix,
		data:       v,
		comma:      comma,
		nest:       i,
	}
}

func (p *Pair) Dump(w io.Writer) {
	p.dumpKey(w)
	p.Item.Dump(w)
}

func (p *Pair) DumpWithoutComma(w io.Writer) {
	if _, ok := p.Data().(*tombstone); ok {
		return
	}
	p.dumpKey(w)
	p.Item.DumpWithoutComma(w)
}

func (p *Pair) dumpKey(w io.Writer) {
	if _, ok := p.Item.data.(*tombstone); ok {
		return
	}
	w.Write(p.spaceKey)
	b, err := json.Marshal(p.key)
	if err != nil {
		debug("(*Pair) Dump", p.key, err.Error())
		b = []byte(strconv.Quote(p.key))
	}
	w.Write(b)
	w.Write(p.spaceColon)
	w.Write([]byte{':'})
}

func isToBeContinued(p *list.Element[Line]) bool {
	if _, ok := ref(p).Data().(*tombstone); ok {
		return false
	}
	for {
		p = p.Next()
		if p == nil {
			return false
		}
		v := ref(p).Data()
		if objEnd.Equals(v) {
			return false
		}
		if arrayEnd.Equals(v) {
			return false
		}
		if _, ok := v.(*tombstone); !ok {
			return true
		}
	}
}

func Dump(L *list.List[Line], w io.Writer) {
	for p := L.Front(); p != nil; p = p.Next() {
		if isToBeContinued(p) {
			ref(p).Dump(w)
		} else {
			ref(p).DumpWithoutComma(w)
		}
	}
}
