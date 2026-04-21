package jegan

import (
	"io"
	"strings"
	"unicode"

	"github.com/hymkor/jegan/internal/ansi"
	"github.com/hymkor/jegan/internal/unjson"
)

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

func render(data any, b *strings.Builder) {
	type renderType interface {
		Render(*strings.Builder, func(any, *strings.Builder))
	}
	if r, ok := data.(renderType); ok {
		r.Render(b, render)
		return
	}
	if x, ok := data.(*unjson.RawBytes); ok {
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
	if x, ok := data.(*unjson.Literal); ok {
		data = x.Data()
		if _, ok := data.(string); ok {
			highlightString(x.Json(), ansi.Magenta, b)
			return
		}
		if _, ok := data.(float64); ok {
			b.Write(x.Json())
			return
		}
	} else {
		io.WriteString(b, ansi.Bold)
		defer io.WriteString(b, ansi.Thin)
	}
	if s, ok := data.(string); ok {
		highlightString(marshal(s), ansi.Magenta, b)
	} else if data == true {
		io.WriteString(b, ansi.Cyan+"true"+ansi.Default)
	} else if data == false {
		io.WriteString(b, ansi.Cyan+"false"+ansi.Default)
	} else if data == nil {
		io.WriteString(b, ansi.Cyan+"null"+ansi.Default)
	} else {
		b.Write(marshal(data))
	}
}

func isToBeContinued(p *Element) bool {
	if _, ok := p.Value.Data().(SkipDump); ok {
		return false
	}
	for {
		p = p.Next()
		if p == nil {
			return false
		}
		data := p.Value.Data()
		if objEnd.Equals(data) {
			return false
		}
		if arrayEnd.Equals(data) {
			return false
		}
		if _, ok := data.(SkipDump); !ok {
			return true
		}
	}
}

func Dump(L *List, w io.Writer) {
	for p := L.Front(); p != nil; p = p.Next() {
		if isToBeContinued(p) {
			p.Value.Dump(w)
		} else {
			p.Value.DumpWithoutComma(w)
		}
	}
}
