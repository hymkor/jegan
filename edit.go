package jegan

import (
	"container/list"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/hymkor/jegan/internal/ansi"
	"github.com/hymkor/jegan/internal/pager"
	"github.com/hymkor/jegan/internal/unjson"
)

func (app *Application) keyFuncReplace(
	session *pager.Session,
	input func(*pager.Session, any) []any) {

	element := node(app.cursor)
	defaultv := element.Value()
	if _, ok := defaultv.(*unjson.RawBytes); ok {
		return
	}
	prev := func() bool { return false }

	if v, ok := element.Value().(Mark); ok {
		next := app.cursor.Next()
		if next == nil {
			return
		}
		nextv, ok := node(next).Value().(Mark)
		if !ok {
			return
		}
		if v == Mark('{') && nextv == Mark('}') {
			defaultv = map[string]any{}
			prev = func() bool { app.list.Remove(next); return true }
		} else if v == Mark('[') && nextv == Mark(']') {
			defaultv = []any{}
			prev = func() bool { app.list.Remove(next); return true }
		} else {
			return
		}
	}
	values := input(session, defaultv)
	switch len(values) {
	case 1:
		if prev() || element.Value() != values[0] {
			app.dirty = true
		}
		element.SetValue(values[0])
	case 2:
		prefix := node(app.cursor).LeadingSpace()
		prev()
		app.list.InsertAfter(
			newElement(values[1], element.Nest(), element.Comma(), prefix),
			app.cursor)
		element.SetValue(values[0])
		element.SetComma(false)
		app.dirty = true
	}
}

type modifiedLiteral struct {
	*unjson.Literal
}

func newModifiedLiteral(v any, j string) *modifiedLiteral {
	return &modifiedLiteral{Literal: unjson.NewLiteral(v, []byte(j))}
}

func (app *Application) inputFormat(session *pager.Session, defaultv any) []any {
	var defaults string
	if _, ok := defaultv.(struct{}); ok {
		defaults = ""
	} else if v, ok := defaultv.(interface{ Json() []byte }); ok {
		defaults = string(v.Json())
	} else {
		b, err := json.Marshal(defaultv)
		if err != nil {
			debug("(*Application) inputFormat: json.Marshal:", err.Error(), "for", defaultv)
		}
		defaults = string(b)
	}
	rawText, err := app.readLineElement(session, "New value:", defaults)
	if err != nil {
		app.message = err.Error()
		return nil
	}
	normText := strings.TrimSpace(rawText)

	if len(normText) >= 2 && normText[0] == '"' && normText[len(normText)-1] == '"' {
		var s string
		err := json.Unmarshal([]byte(rawText), &s)
		if err == nil {
			return []any{newModifiedLiteral(s, normText)}
		}
	}
	if number, err := strconv.ParseFloat(normText, 64); err == nil {
		return []any{newModifiedLiteral(number, normText)}
	}
	if strings.EqualFold(normText, "null") {
		return []any{nil}
	}
	if strings.EqualFold(normText, "true") {
		return []any{true}
	}
	if strings.EqualFold(normText, "false") {
		return []any{false}
	}
	if normText == "{}" {
		return []any{Mark('{'), Mark('}')}
	}
	if normText == "[]" {
		return []any{Mark('['), Mark(']')}
	}
	return []any{rawText}
}

func (app *Application) inputTypeAndValue(session *pager.Session, defaultv any) []any {
	io.WriteString(session.TtyOut, ansi.CursorOn)
	defer io.WriteString(session.TtyOut, ansi.CursorOff)
	for {
		io.WriteString(session.TtyOut,
			"\r's':string, 'n':number, 'u':null, "+
				"'t':true, 'f':false, 'o':{}, 'a':[] ? "+ansi.EraseLine)
		key, err := session.GetKey()
		if err != nil {
			app.message = err.Error()
			return nil
		}
		switch key {
		case "\a":
			app.message = "Canceled"
			return nil
		case "s":
			text, err := app.readLineString(session, "New string:", fmt.Sprint(defaultv))
			if err != nil {
				session.TtyOut.Write([]byte{'\a'})
				return nil
			}
			return []any{text}
		case "n":
			text, err := app.readLine(session, "New number:", fmt.Sprint(defaultv))
			if err != nil {
				session.TtyOut.Write([]byte{'\a'})
				return nil
			}
			newValue, err := strconv.ParseFloat(text, 64)
			if err == nil {
				return []any{newValue}
			}
			session.TtyOut.Write([]byte{'\a'})
		case "u":
			return []any{nil}
		case "t":
			return []any{true}
		case "f":
			return []any{false}
		case "o":
			return []any{Mark('{'), Mark('}')}
		case "a":
			return []any{Mark('['), Mark(']')}
		default:
			session.TtyOut.Write([]byte{'\a'})
		}
	}
}

func isDuplicated(cursor *list.Element, nest int, key string) bool {
	for p := cursor; p != nil; p = p.Prev() {
		i := node(p).Nest()
		if i < nest {
			break
		}
		if i == nest {
			q, ok := p.Value.(*Pair)
			if ok && q.key == key {
				return true
			}
		}
	}
	for p := cursor.Next(); p != nil; p = p.Next() {
		i := node(p).Nest()
		if i < nest {
			break
		}
		if i == nest {
			q, ok := p.Value.(*Pair)
			if ok && q.key == key {
				return true
			}
		}
	}
	return false
}

func findPairBefore(p *list.Element) *Pair {
	for ; p != nil; p = p.Prev() {
		if pair, ok := p.Value.(*Pair); ok {
			return pair
		}
	}
	return nil
}

func findSameLevelPairBefore(p *list.Element) (*Pair, bool) {
	if pair, ok := p.Value.(*Pair); ok {
		return pair, true
	}
	nest := node(p).Nest()
	for {
		p = p.Prev()
		if p == nil {
			return nil, false
		}
		element := node(p)
		i := element.Nest()
		if i == nest {
			if pair, ok := p.Value.(*Pair); ok {
				return pair, true
			}
		} else if i < nest {
			if element.Value() != Mark('{') {
				return nil, false
			}
			return findPairBefore(p), true
		}
	}
}

func joinBytes(args ...[]byte) []byte {
	b := []byte{}
	for _, b1 := range args {
		b = append(b, b1...)
	}
	return b
}

func (app *Application) keyFuncInsert(session *pager.Session) {
	space := node(app.cursor).LeadingSpace()
	if e := node(app.cursor); e.Value() == Mark('[') {
		next := app.cursor.Next()
		nextElement := node(next)
		var comma bool
		var nest int
		var newPrefix []byte
		todo := func() {}
		if nextElement.Value() == Mark(']') {
			comma = false
			outerPrefix := nextElement.LeadingSpace() // space before ]
			if len(outerPrefix) == 0 {
				outerPrefix = space // space before [
				todo = func() {
					nextElement.SetLeadingSpace(space)
				}
			}
			nest = nextElement.Nest() + 1
			newPrefix = joinBytes(outerPrefix, app.indent)
		} else {
			comma = true
			nest = nextElement.Nest()
			newPrefix = nextElement.LeadingSpace()
		}
		values := app.inputFormat(session, struct{}{})
		switch len(values) {
		case 2: // [\n[\n],\n
			e1 := newElement(values[0], nest, false, newPrefix)
			e2 := newElement(values[1], nest, comma, nil)
			app.list.InsertBefore(e1, next)
			app.list.InsertBefore(e2, next)
			todo()
			app.nextLine(session)
			app.dirty = true
		case 1: // [\n value
			app.list.InsertBefore(
				newElement(values[0], nest, comma, newPrefix),
				next)
			todo()
			app.nextLine(session)
			app.dirty = true
		}
		return
	}
	if e := node(app.cursor); e.Value() == Mark('{') {
		key, err := app.readLineString(session, "Key:", "")
		if err != nil {
			return
		}
		sample := findPairBefore(app.cursor)
		next := app.cursor.Next()
		nextElement := node(next)
		var comma bool
		var nest int
		var newPrefix []byte
		todo := func() {}
		if nextElement.Value() == Mark('}') {
			comma = false
			outerPrefix := nextElement.LeadingSpace() // space before }
			if len(outerPrefix) == 0 {
				outerPrefix = space
				todo = func() { node(next).SetLeadingSpace(space) }
			}
			nest = nextElement.Nest() + 1
			newPrefix = joinBytes(outerPrefix, app.indent)
		} else {
			if isDuplicated(next, nextElement.Nest(), key) {
				app.message = fmt.Sprintf("\aduplicate key: %q", key)
				return
			}
			comma = true
			nest = nextElement.Nest()
			newPrefix = nextElement.LeadingSpace()
		}
		values := app.inputFormat(session, struct{}{})
		switch len(values) {
		case 2: // { key:[]
			p1 := &Pair{
				spaceKey: newPrefix,
				key:      key,
				Element: Element{
					value: values[0],
					nest:  nest,
					comma: false,
				},
			}
			if sample != nil {
				p1.spaceColon = sample.spaceColon
				p1.spaceValue = sample.spaceValue
			}
			e2 := newElement(values[1], nest, comma, nil)
			app.list.InsertBefore(p1, next)
			app.list.InsertBefore(e2, next)
			todo()
			app.nextLine(session)
			app.dirty = true
		case 1: // { key:value
			p1 := &Pair{
				spaceKey: newPrefix,
				key:      key,
				Element: Element{
					value: values[0],
					nest:  nest,
					comma: comma,
				},
			}
			if sample != nil {
				p1.spaceColon = sample.spaceColon
				p1.spaceValue = sample.spaceValue
			}
			app.list.InsertBefore(p1, next)
			todo()
			app.nextLine(session)
			app.dirty = true
		}
		return
	}
	if sample, ok := findSameLevelPairBefore(app.cursor); ok {
		key, err := app.readLineString(session, "Key:", "")
		if err != nil {
			return
		}
		element := node(app.cursor)
		if isDuplicated(app.cursor, element.Nest(), key) {
			app.message = fmt.Sprintf("\aduplicate key: %q", key)
			return
		}
		values := app.inputFormat(session, struct{}{})
		switch len(values) {
		case 2: // key:[],
			p1 := &Pair{
				spaceKey: space,
				key:      key,
				Element: Element{
					value: values[0],
					nest:  element.Nest(),
					comma: false,
				},
			}
			if sample != nil {
				p1.spaceColon = sample.spaceColon
				p1.spaceValue = sample.spaceValue
			}
			e2 := newElement(values[1], element.Nest(), element.Comma(), nil)
			app.list.InsertAfter(e2, app.cursor)
			app.list.InsertAfter(p1, app.cursor)
			app.nextLine(session)
			app.dirty = true
		case 1: // key:value,
			p := &Pair{
				spaceKey: space,
				key:      key,
				Element: Element{
					value: values[0],
					nest:  element.Nest(),
					comma: element.Comma(),
				},
			}
			if sample != nil {
				p.spaceColon = sample.spaceColon
				p.spaceValue = sample.spaceValue
			}
			app.list.InsertAfter(p, app.cursor)
			app.nextLine(session)
			app.dirty = true
		}
		element.SetComma(true)
		return
	}
	if element, ok := app.cursor.Value.(*Element); ok {
		nest := node(app.cursor).Nest()
		if nest < 0 {
			return
		}
		values := app.inputFormat(session, struct{}{})
		switch len(values) {
		case 2: // [ \n ],
			e1 := newElement(values[0], element.Nest(), false, space)
			e2 := newElement(values[1], element.Nest(), element.Comma(), nil)
			app.list.InsertAfter(e2, app.cursor)
			app.list.InsertAfter(e1, app.cursor)
			app.nextLine(session)
			app.dirty = true
		case 1: // value,
			e := newElement(values[0], element.Nest(), element.Comma(), space)
			app.list.InsertAfter(e, app.cursor)
			app.nextLine(session)
			app.dirty = true
		}
		element.SetComma(true)
	}
}

func (app *Application) removeCursor(session *pager.Session) {
	comma := node(app.cursor).Comma()
	if next := app.cursor.Next(); next != nil {
		node(app.cursor).SetCursor(false)
		app.list.Remove(app.cursor)
		app.cursor = next
		node(app.cursor).SetCursor(true)
		if !comma {
			if p := app.cursor.Prev(); p != nil {
				node(p).SetComma(false)
			}
		}
		app.dirty = true
	} else if prev := app.cursor.Prev(); prev != nil {
		node(app.cursor).SetCursor(false)
		app.list.Remove(app.cursor)
		app.cursor = prev
		node(app.cursor).SetCursor(true)
		app.csrline--
		if app.csrline < app.winline {
			session.Window = app.cursor
			app.winline = app.csrline
		}
		if !comma {
			node(prev).SetComma(false)
		}
		app.dirty = true
	}
}

func (app *Application) removeCursorAndNext() {
	prev := app.cursor.Prev()
	next := app.cursor.Next()
	if next == nil {
		return
	}
	newCurrent := next.Next()
	if newCurrent == nil {
		app.message = "Internal error: no valid cursor position after deletion"
		return
	}
	comma := node(next).Comma()

	app.list.Remove(app.cursor)
	app.list.Remove(next)

	app.cursor = newCurrent
	node(newCurrent).SetCursor(true)
	if prev != nil {
		m, ok := node(prev).Value().(Mark)
		if !ok || (m != Mark('{') && m != Mark('[')) {
			node(prev).SetComma(comma)
		}
	}
}

func (app *Application) keyFuncRemove(session *pager.Session) {
	element := node(app.cursor)
	mark, ok := element.Value().(Mark)
	if !ok {
		app.removeCursor(session)
		return
	}
	if mark != Mark('{') && mark != Mark('[') {
		return
	}
	if element.Nest() == 0 {
		app.message = "Cannot delete top-level object or array"
		return
	}
	next := app.cursor.Next()
	if next == nil {
		app.message = "Unexpected state: missing element after '{' or '['"
		return
	}
	n := node(next)
	if n.Value() != Mark(']') && n.Value() != Mark('}') {
		app.message = "Cannot delete non-empty object or array"
		return
	}
	app.removeCursorAndNext()
}
