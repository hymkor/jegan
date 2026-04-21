package jegan

import (
	"encoding/json"
	"fmt"

	"github.com/hymkor/go-generics-list"

	"github.com/hymkor/jegan/internal/pager"
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

type Unwraper interface {
	Unwrap() any
}

func unwrap(data any) any {
	for {
		v, ok := data.(Unwraper)
		if !ok {
			return data
		}
		data = v.Unwrap()
	}
	return data
}
