package jegan

import (
	"encoding/json"
	"fmt"

	"github.com/hymkor/go-generics-list"

	"github.com/hymkor/jegan/internal/pager"
	"github.com/hymkor/jegan/internal/unjson"
)

type Session = pager.Session[Line]

type List = list.List[Line]

type Element = list.Element[Line]

func marshal[T any](data T) []byte {
	bin, err := json.Marshal(data)
	if err != nil {
		return []byte(fmt.Sprintf("%#v", data))
	}
	return bin
}

func unwrap(data any) any {
	if x, ok := data.(*modifiedLiteral); ok {
		data = x.Literal
	}
	if x, ok := data.(*unjson.Literal); ok {
		data = x.Data()
	}
	return data
}
