package jegan

import (
	"github.com/hymkor/go-generics-list"

	"github.com/hymkor/jegan/internal/pager"

	types "github.com/hymkor/jegan/internal/types"
)

type JsonPath = types.JsonPath

type Mark = types.Mark

type Line = types.Line

type Pair = types.Pair

type Session = pager.Session[Line]

type List = list.List[Line]

type Element = list.Element[Line]
