package jegan

import (
	"io"
	"strings"

	"github.com/hymkor/jegan/internal/ansi"
)

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
		w.Write(marshal(e.data))
	}
	w.Write(e.spaceCommaOrClose)
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

func newItem(v any, i int, comma bool, prefix []byte) *Item {
	return &Item{
		spaceValue: prefix,
		data:       v,
		comma:      comma,
		nest:       i,
	}
}
