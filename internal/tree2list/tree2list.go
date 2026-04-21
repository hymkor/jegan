package tree2list

import (
	"github.com/hymkor/go-generics-list"

	"github.com/hymkor/jegan/internal/types"
	"github.com/hymkor/jegan/internal/unjson"
)

func readPairs(basePath *types.JsonPath, prefix []byte, pairs []unjson.KeyValuePair, nest int) *list.List[types.Line] {
	L := list.New[types.Line]()

	begin := types.NewItem(types.ObjStart, nest, false, prefix)
	begin.SetPath(basePath)
	L.PushBack(begin)

	for _, pair := range pairs {
		jp := &types.JsonPath{
			Parent: basePath,
			Text:   pair.Key,
		}
		sub := read(jp, pair.Value, nest+1)

		front := sub.Front()
		orgF := front.Value
		newF := &types.Pair{
			SpaceKey:   pair.SpaceKey,
			Key:        pair.Key,
			SpaceColon: pair.SpaceColon,
			Item:       *types.NewItem(orgF.Data(), nest+1, orgF.Comma(), pair.Value.SpaceValue),
		}
		newF.SetPath(orgF.Path())
		front.Value = newF

		sub.Back().Value.SetSpaceCommaOrClose(pair.SpaceCommaOrClose)
		L.PushBackList(sub)
	}
	back := L.Back().Value
	finalSpace := back.SpaceCommaOrClose()
	back.SetSpaceCommaOrClose(nil)
	back.SetComma(false)

	end := types.NewItem(types.ObjEnd, nest, true, finalSpace)
	end.SetPath(basePath)
	L.PushBack(end)

	return L
}

func readObject(basePath *types.JsonPath, prefix []byte, object *unjson.Object, nest int) *list.List[types.Line] {
	if len(object.Pairs) <= 0 {
		L := list.New[types.Line]()

		begin := types.NewItem(types.ObjStart, nest, false, prefix)
		begin.SetPath(basePath)
		L.PushBack(begin)

		end := types.NewItem(types.ObjEnd, nest, true, object.Blank)
		end.SetPath(basePath)
		L.PushBack(end)

		return L
	}
	return readPairs(basePath, prefix, object.Pairs, nest)
}

func readElements(basePath *types.JsonPath, prefix []byte, elements []unjson.ArrayElement, nest int) *list.List[types.Line] {
	L := list.New[types.Line]()

	begin := types.NewItem(types.ArrayStart, nest, false, prefix)
	begin.SetPath(basePath)
	L.PushBack(begin)

	for i, element := range elements {
		jp := &types.JsonPath{
			Parent: basePath,
			Index:  i,
		}
		sub := read(jp, element.Item, nest+1)
		sub.Back().Value.SetSpaceCommaOrClose(element.PreComma)
		L.PushBackList(sub)
	}
	back := L.Back().Value
	finalSpace := back.SpaceCommaOrClose()
	back.SetSpaceCommaOrClose(nil)
	back.SetComma(false)

	end := types.NewItem(types.ArrayEnd, nest, true, finalSpace)
	end.SetPath(basePath)
	L.PushBack(end)

	return L
}

func readArray(basePath *types.JsonPath, prefix []byte, array *unjson.Array, nest int) *list.List[types.Line] {
	if len(array.Element) <= 0 {
		L := list.New[types.Line]()

		begin := types.NewItem(types.ArrayStart, nest, false, prefix)
		begin.SetPath(basePath)
		L.PushBack(begin)

		end := types.NewItem(types.ArrayEnd, nest, true, array.Blank)
		end.SetPath(basePath)
		L.PushBack(end)

		return L
	}
	return readElements(basePath, prefix, array.Element, nest)
}

func read(basePath *types.JsonPath, t *unjson.Item, nest int) *list.List[types.Line] {
	v := t.Value
	prefix := t.SpaceValue
	if x, ok := v.(*unjson.Object); ok {
		return readObject(basePath, prefix, x, nest)
	}
	if x, ok := v.(*unjson.Array); ok {
		return readArray(basePath, prefix, x, nest)
	}
	L := list.New[types.Line]()

	e := types.NewItem(v, nest, true, prefix)
	e.SetPath(basePath)
	L.PushBack(e)
	return L
}

func Read(v *unjson.Item) *list.List[types.Line] {
	if v == nil {
		return nil
	}
	L := read(nil, v, 0)
	if L != nil && L.Len() > 0 {
		L.Back().Value.SetComma(false)
	}
	return L
}
