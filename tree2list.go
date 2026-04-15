package jegan

import (
	"container/list"

	"github.com/hymkor/jegan/internal/unjson"
)

func readPairs(prefix []byte, pairs []unjson.KeyValuePair, nest int) *list.List {
	L := list.New()
	L.PushBack(newElement(Mark('{'), nest, false, prefix))
	for _, pair := range pairs {
		sub := read(pair.Value, nest+1)

		front := sub.Front()
		orgF := ref(front)
		newF := &Pair{
			spaceKey:   pair.SpaceKey,
			key:        pair.Key,
			spaceColon: pair.SpaceColon,
			Element: Element{
				spaceValue: pair.Value.SpaceValue,
				value:      orgF.Value(),
				comma:      orgF.Comma(),
				nest:       nest + 1,
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
	L.PushBack(newElement(Mark('}'), nest, true, finalSpace))
	return L
}

func readObject(prefix []byte, object *unjson.Object, nest int) *list.List {
	if len(object.Pairs) <= 0 {
		L := list.New()
		L.PushBack(newElement(Mark('{'), nest, false, prefix))
		L.PushBack(newElement(Mark('}'), nest, true, object.Blank))
		return L
	}
	return readPairs(prefix, object.Pairs, nest)
}

func readElements(prefix []byte, elements []unjson.ArrayElement, nest int) *list.List {
	L := list.New()
	L.PushBack(newElement(Mark('['), nest, false, prefix))
	for _, element := range elements {
		sub := read(element.Entry, nest+1)
		ref(sub.Back()).SetSpaceCommaOrClose(element.PreComma)
		L.PushBackList(sub)
	}
	back := ref(L.Back())
	finalSpace := back.SpaceCommaOrClose()
	back.SetSpaceCommaOrClose(nil)
	back.SetComma(false)
	L.PushBack(newElement(Mark(']'), nest, true, finalSpace))
	return L
}

func readArray(prefix []byte, array *unjson.Array, nest int) *list.List {
	if len(array.Element) <= 0 {
		L := list.New()
		L.PushBack(newElement(Mark('['), nest, false, prefix))
		L.PushBack(newElement(Mark(']'), nest, true, array.Blank))
		return L
	}
	return readElements(prefix, array.Element, nest)
}

func read(t *unjson.Entry, nest int) *list.List {
	v := t.Value
	prefix := t.SpaceValue
	if x, ok := v.(*unjson.Object); ok {
		return readObject(prefix, x, nest)
	}
	if x, ok := v.(*unjson.Array); ok {
		return readArray(prefix, x, nest)
	}
	L := list.New()
	L.PushBack(newElement(v, nest, true, prefix))
	return L
}

func Read(v *unjson.Entry) *list.List {
	if v == nil {
		return nil
	}
	L := read(v, 0)
	if L != nil && L.Len() > 0 {
		ref(L.Back()).SetComma(false)
	}
	return L
}
