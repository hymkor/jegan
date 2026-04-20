package unjson

import (
	"fmt"
)

type Literal struct {
	data any
	json []byte
}

func NewLiteral(v any, j []byte) *Literal {
	return &Literal{data: v, json: j}
}

func (L *Literal) Data() any       { return L.data }
func (L *Literal) Json() []byte    { return L.json }
func (L *Literal) String() string  { return fmt.Sprint(L.data) }
func (L Literal) GoString() string { return fmt.Sprint(string(L.json)) }

type RawBytes struct {
	json []byte
}

func (R *RawBytes) Data() any       { return string(R.json) }
func (R *RawBytes) Json() []byte    { return R.json }
func (R *RawBytes) String() string  { return string(R.json) }
func (R RawBytes) GoString() string { return string(R.json) }
