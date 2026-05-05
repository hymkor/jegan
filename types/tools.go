package types

import (
	"encoding/json"
	"fmt"

	"github.com/hymkor/go-generics-list"

	"github.com/hymkor/nemo/pager"
)

type Session = pager.Session[Line]

type List = list.List[Line]

type Element = list.Element[Line]

func Marshal[T any](data T) []byte {
	bin, err := json.Marshal(data)
	if err != nil {
		return []byte(fmt.Sprintf("%#v", data))
	}
	return bin
}

func Unwrap(data any) any {
	type Unwraper interface {
		Unwrap() any
	}
	for {
		if v, ok := data.(interface{ Unwrap() any }); ok {
			data = v.Unwrap()
		} else if v, ok := data.(interface{ Data() any }); ok {
			data = v.Data()
		} else {
			return data
		}
	}
}
