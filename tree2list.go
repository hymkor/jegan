package jegan

import (
	"github.com/hymkor/go-generics-list"

	"github.com/hymkor/jegan/internal/unjson"
)

func readPairs(basePath *JsonPath, prefix []byte, pairs []unjson.KeyValuePair, nest int) *list.List[Line] {
	L := list.New[Line]()

	begin := newElement(objStart, nest, false, prefix)
	begin.SetPath(basePath)
	L.PushBack(begin)

	for _, pair := range pairs {
		jp := &JsonPath{
			parent: basePath,
			text:   pair.Key,
		}
		sub := read(jp, pair.Value, nest+1)

		front := sub.Front()
		orgF := ref(front)
		newF := &Pair{
			spaceKey:   pair.SpaceKey,
			key:        pair.Key,
			spaceColon: pair.SpaceColon,
			Item: Item{
				spaceValue: pair.Value.SpaceValue,
				data:       orgF.Data(),
				comma:      orgF.Comma(),
				nest:       nest + 1,
				path:       orgF.Path(),
			},
		}
		front.Value = newF

		ref(sub.Back()).SetSpaceCommaOrClose(pair.SpaceCommaOrClose)
		L.PushBackList(sub)
	}
	back := ref(L.Back())
	finalSpace := back.SpaceCommaOrClose()
	back.SetSpaceCommaOrClose(nil)
	back.SetComma(false)

	end := newElement(objEnd, nest, true, finalSpace)
	end.SetPath(basePath)
	L.PushBack(end)

	return L
}

func readObject(basePath *JsonPath, prefix []byte, object *unjson.Object, nest int) *list.List[Line] {
	if len(object.Pairs) <= 0 {
		L := list.New[Line]()

		begin := newElement(objStart, nest, false, prefix)
		begin.SetPath(basePath)
		L.PushBack(begin)

		end := newElement(objEnd, nest, true, object.Blank)
		end.SetPath(basePath)
		L.PushBack(end)

		return L
	}
	return readPairs(basePath, prefix, object.Pairs, nest)
}

func readElements(basePath *JsonPath, prefix []byte, elements []unjson.ArrayElement, nest int) *list.List[Line] {
	L := list.New[Line]()

	begin := newElement(arrayStart, nest, false, prefix)
	begin.SetPath(basePath)
	L.PushBack(begin)

	for i, element := range elements {
		jp := &JsonPath{
			parent: basePath,
			index:  i,
		}
		sub := read(jp, element.Item, nest+1)
		ref(sub.Back()).SetSpaceCommaOrClose(element.PreComma)
		L.PushBackList(sub)
	}
	back := ref(L.Back())
	finalSpace := back.SpaceCommaOrClose()
	back.SetSpaceCommaOrClose(nil)
	back.SetComma(false)

	end := newElement(arrayEnd, nest, true, finalSpace)
	end.SetPath(basePath)
	L.PushBack(end)

	return L
}

func readArray(basePath *JsonPath, prefix []byte, array *unjson.Array, nest int) *list.List[Line] {
	if len(array.Element) <= 0 {
		L := list.New[Line]()

		begin := newElement(arrayStart, nest, false, prefix)
		begin.SetPath(basePath)
		L.PushBack(begin)

		end := newElement(arrayEnd, nest, true, array.Blank)
		end.SetPath(basePath)
		L.PushBack(end)

		return L
	}
	return readElements(basePath, prefix, array.Element, nest)
}

func read(basePath *JsonPath, t *unjson.Item, nest int) *list.List[Line] {
	v := t.Value
	prefix := t.SpaceValue
	if x, ok := v.(*unjson.Object); ok {
		return readObject(basePath, prefix, x, nest)
	}
	if x, ok := v.(*unjson.Array); ok {
		return readArray(basePath, prefix, x, nest)
	}
	L := list.New[Line]()

	e := newElement(v, nest, true, prefix)
	e.path = basePath
	L.PushBack(e)
	return L
}

func Read(v *unjson.Item) *list.List[Line] {
	if v == nil {
		return nil
	}
	L := read(nil, v, 0)
	if L != nil && L.Len() > 0 {
		ref(L.Back()).SetComma(false)
	}
	return L
}
