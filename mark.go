package jegan

import (
	"strings"

	"github.com/hymkor/jegan/internal/ansi"
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
