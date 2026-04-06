package unjson

import (
	"fmt"
)

type Literal struct {
	value any
	json  []byte
}

func (L *Literal) Value() any      { return L.value }
func (L *Literal) Json() []byte    { return L.json }
func (L *Literal) String() string  { return fmt.Sprint(L.value) }
func (L Literal) GoString() string { return fmt.Sprint(string(L.json)) }
