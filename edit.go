package jegan

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/atotto/clipboard"

	"github.com/hymkor/go-generics-list"

	"github.com/hymkor/jegan/internal/ansi"
	"github.com/hymkor/jegan/internal/unjson"
)

func backup(v any, backup any) any {
	if m, ok := v.(*modifiedLiteral); ok {
		m.backup = backup
		return m
	}
	json, err := json.Marshal(v)
	if err != nil {
		json = []byte(fmt.Sprint(v))
	}
	return &modifiedLiteral{
		Literal: unjson.NewLiteral(v, json),
		backup:  backup,
	}
}

func (app *Application) keyFuncReplace(
	session *Session,
	input func(*Session, any) ([]any, error)) error {

	element := app.cursor.Value
	defaultv := element.Data()
	if _, ok := defaultv.(*unjson.RawBytes); ok {
		return nil
	}
	prev := func() bool { return false }

	if v, ok := unwrap(element.Data()).(Mark); ok {
		next := app.cursor.Next()
		if next == nil {
			return nil
		}
		nextv, ok := unwrap(next.Value.Data()).(Mark)
		if !ok {
			return nil
		}
		if objStart.Equals(v) && objEnd.Equals(nextv) {
			defaultv = map[string]any{}
			prev = func() bool { app.list.Remove(next); return true }
		} else if arrayStart.Equals(v) && arrayEnd.Equals(nextv) {
			defaultv = []any{}
			prev = func() bool { app.list.Remove(next); return true }
		} else {
			return nil
		}
	}
	newData, err := input(session, defaultv)
	if err != nil {
		return err
	}
	switch len(newData) {
	case 1:
		if prev() || element.Data() != newData[0] {
			app.dirty = true
			newData[0] = backup(newData[0], element.Data())
		}
		element.SetData(newData[0])
	case 2:
		prefix := app.cursor.Value.LeadingSpace()
		prev()
		app.list.InsertAfter(
			newItem(newData[1], element.Nest(), element.Comma(), prefix),
			app.cursor)
		element.SetData(backup(newData[0], element.Data()))
		element.SetComma(false)
		app.dirty = true
	}
	return nil
}

type tombstone struct {
	first any
	rest  *list.List[Line]
}

func (r *tombstone) Json() []byte {
	return []byte{}
}

func (t *tombstone) Render(b *strings.Builder, _ func(any, *strings.Builder)) {
	b.WriteString(ansi.Red)
	b.WriteString("<DEL>")
	b.WriteString(ansi.Default)
}

type modifiedLiteral struct {
	*unjson.Literal
	backup any
}

func (m *modifiedLiteral) Render(b *strings.Builder, render func(any, *strings.Builder)) {
	io.WriteString(b, ansi.Bold)
	render(m.Literal, b)
	io.WriteString(b, ansi.Thin)
}

func newModifiedLiteral(v any, j string) *modifiedLiteral {
	return &modifiedLiteral{Literal: unjson.NewLiteral(v, []byte(j))}
}

func makeDefaultFormat(v any) string {
	if _, ok := v.(struct{}); ok {
		return ""
	}
	if v, ok := v.(interface{ Json() []byte }); ok {
		return string(v.Json())
	}
	bin, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%q", v)
	}
	return string(bin)
}

func inputToAny(rawText string) ([]any, error) {
	normText := strings.TrimSpace(rawText)

	if len(normText) >= 2 && normText[0] == '"' && normText[len(normText)-1] == '"' {
		var s string
		err := json.Unmarshal([]byte(rawText), &s)
		if err == nil {
			return []any{newModifiedLiteral(s, normText)}, nil
		}
	}
	if number, err := strconv.ParseFloat(normText, 64); err == nil {
		return []any{newModifiedLiteral(number, normText)}, nil
	}
	if strings.EqualFold(normText, "null") {
		return []any{nil}, nil
	}
	if strings.EqualFold(normText, "true") {
		return []any{true}, nil
	}
	if strings.EqualFold(normText, "false") {
		return []any{false}, nil
	}
	if normText == "{}" {
		return []any{objStart, objEnd}, nil
	}
	if normText == "[]" {
		return []any{arrayStart, arrayEnd}, nil
	}
	return []any{rawText}, nil
}

func (app *Application) inputFormat(session *Session, defaultv any) ([]any, error) {
	defaults := makeDefaultFormat(defaultv)
	text, err := app.readLineElement(session, "New value:", defaults)
	if err != nil {
		return nil, err
	}
	return inputToAny(text)
}

func (app *Application) inputTypeAndValue(session *Session, defaultv any) ([]any, error) {
	io.WriteString(session.TtyOut, ansi.CursorOn)
	defer io.WriteString(session.TtyOut, ansi.CursorOff)
	for {
		io.WriteString(session.TtyOut,
			"\r's':string, 'n':number, 'u':null, "+
				"'t':true, 'f':false, 'o':{}, 'a':[] ? "+ansi.EraseLine)
		key, err := session.GetKey()
		if err != nil {
			return nil, err
		}
		switch key {
		case "\a":
			return nil, errCanceled
		case "s":
			text, err := app.readLineString(session, "New string:", fmt.Sprint(defaultv))
			if err != nil {
				return nil, err
			}
			return []any{text}, nil
		case "n":
			text, err := app.readLine(session, "New number:", fmt.Sprint(defaultv))
			if err != nil {
				return nil, err
			}
			newValue, err := strconv.ParseFloat(text, 64)
			if err == nil {
				return []any{newValue}, nil
			}
			session.TtyOut.Write([]byte{'\a'})
		case "u":
			return []any{nil}, nil
		case "t":
			return []any{true}, nil
		case "f":
			return []any{false}, nil
		case "o":
			return []any{objStart, objEnd}, nil
		case "a":
			return []any{arrayStart, arrayEnd}, nil
		default:
			session.TtyOut.Write([]byte{'\a'})
		}
	}
}

func isDuplicated(cursor *list.Element[Line], nest int, key string) bool {
	for p := cursor; p != nil; p = p.Prev() {
		i := p.Value.Nest()
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
		i := p.Value.Nest()
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

func findPairBefore(p *list.Element[Line]) *Pair {
	for ; p != nil; p = p.Prev() {
		if pair, ok := p.Value.(*Pair); ok {
			return pair
		}
	}
	return nil
}

func findSameLevelPairBefore(p *list.Element[Line]) (*Pair, bool) {
	if pair, ok := p.Value.(*Pair); ok {
		return pair, true
	}
	nest := p.Value.Nest()
	for {
		p = p.Prev()
		if p == nil {
			return nil, false
		}
		element := p.Value
		i := element.Nest()
		if i == nest {
			if pair, ok := p.Value.(*Pair); ok {
				return pair, true
			}
		} else if i < nest {
			if element.Data() != objStart {
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

func reflectIndex(p *list.Element[Line], nest int, plusminus int) {
	for ; p != nil; p = p.Next() {
		r := p.Value
		if n := r.Nest(); n < nest {
			return
		} else if n > nest {
			continue
		}
		if path := r.Path(); path != nil {
			if v := r.Data(); v != objEnd && v != arrayEnd {
				path.index += plusminus
			}
		}
	}
}

func (app *Application) keyFuncInsert(session *Session) error {
	space := app.cursor.Value.LeadingSpace()
	currentNest := app.cursor.Value.Nest()
	if e := app.cursor.Value; arrayStart.Equals(e.Data()) {
		next := app.cursor.Next()
		nextElement := next.Value
		var comma bool
		var nest int
		var newPrefix []byte
		todo := func() {}
		if arrayEnd.Equals(nextElement.Data()) {
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
		newData, err := app.inputFormat(session, struct{}{})
		if err != nil {
			return err
		}
		switch len(newData) {
		case 2: // [\n[\n],\n
			reflectIndex(app.cursor.Next(), currentNest+1, +1)
			e1 := newItem(newData[0], nest, false, newPrefix)
			e1.SetPath(app.cursor.Value.Path().ChildIndex(0))
			e2 := newItem(newData[1], nest, comma, nil)
			e2.SetPath(e1.Path())
			app.list.InsertBefore(e1, next)
			app.list.InsertBefore(e2, next)
			todo()
			app.nextLine(session)
			app.dirty = true
		case 1: // [\n value
			reflectIndex(app.cursor.Next(), currentNest+1, +1)
			e1 := newItem(newData[0], nest, comma, newPrefix)
			e1.SetPath(app.cursor.Value.Path().ChildIndex(0))
			app.list.InsertBefore(e1, next)
			todo()
			app.nextLine(session)
			app.dirty = true
		}
		return nil
	}
	if e := app.cursor.Value; objStart.Equals(e.Data()) {
		key, err := app.readLineString(session, "Key:", "")
		if err != nil {
			return err
		}
		sample := findPairBefore(app.cursor)
		next := app.cursor.Next()
		nextElement := next.Value
		var comma bool
		var nest int
		var newPrefix []byte
		todo := func() {}
		if objEnd.Equals(nextElement.Data()) {
			comma = false
			outerPrefix := nextElement.LeadingSpace() // space before }
			if len(outerPrefix) == 0 {
				outerPrefix = space
				todo = func() { next.Value.SetLeadingSpace(space) }
			}
			nest = nextElement.Nest() + 1
			newPrefix = joinBytes(outerPrefix, app.indent)
		} else {
			if isDuplicated(next, nextElement.Nest(), key) {
				return fmt.Errorf("duplicate key: %q", key)
			}
			comma = true
			nest = nextElement.Nest()
			newPrefix = nextElement.LeadingSpace()
		}
		newData, err := app.inputFormat(session, struct{}{})
		if err != nil {
			return err
		}
		switch len(newData) {
		case 2: // { key:[]
			p1 := &Pair{
				spaceKey: newPrefix,
				key:      key,
				Item: Item{
					data:  newData[0],
					nest:  nest,
					comma: false,
				},
			}
			if sample != nil {
				p1.spaceColon = sample.spaceColon
				p1.spaceValue = sample.spaceValue
			}
			e2 := newItem(newData[1], nest, comma, nil)
			app.list.InsertBefore(p1, next)
			app.list.InsertBefore(e2, next)
			todo()
			app.nextLine(session)
			app.dirty = true
		case 1: // { key:value
			p1 := &Pair{
				spaceKey: newPrefix,
				key:      key,
				Item: Item{
					data:  newData[0],
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
		return nil
	}
	if sample, ok := findSameLevelPairBefore(app.cursor); ok {
		key, err := app.readLineString(session, "Key:", "")
		if err != nil {
			return err
		}
		element := app.cursor.Value
		if isDuplicated(app.cursor, element.Nest(), key) {
			return fmt.Errorf("duplicate key: %q", key)
		}
		newData, err := app.inputFormat(session, struct{}{})
		if err != nil {
			return err
		}
		switch len(newData) {
		case 2: // key:[],
			p1 := &Pair{
				spaceKey: space,
				key:      key,
				Item: Item{
					data:  newData[0],
					nest:  element.Nest(),
					comma: false,
				},
			}
			if sample != nil {
				p1.spaceColon = sample.spaceColon
				p1.spaceValue = sample.spaceValue
			}
			e2 := newItem(newData[1], element.Nest(), element.Comma(), nil)
			app.list.InsertAfter(e2, app.cursor)
			app.list.InsertAfter(p1, app.cursor)
			app.nextLine(session)
			app.dirty = true
		case 1: // key:value,
			p := &Pair{
				spaceKey: space,
				key:      key,
				Item: Item{
					data:  newData[0],
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
		return nil
	}
	if element, ok := app.cursor.Value.(*Item); ok {
		nest := app.cursor.Value.Nest()
		if nest < 0 {
			return nil
		}
		newData, err := app.inputFormat(session, struct{}{})
		if err != nil {
			return nil
		}
		switch len(newData) {
		case 2: // [ \n ],
			reflectIndex(app.cursor.Next(), currentNest, +1)
			e1 := newItem(newData[0], element.Nest(), false, space)
			e2 := newItem(newData[1], element.Nest(), element.Comma(), nil)
			j := app.cursor.Value.Path()
			e1.SetPath(j.parent.ChildIndex(j.index + 1))
			e2.SetPath(e1.Path())
			app.list.InsertAfter(e2, app.cursor)
			app.list.InsertAfter(e1, app.cursor)
			app.nextLine(session)
			app.dirty = true
		case 1: // value,
			reflectIndex(app.cursor.Next(), currentNest, +1)
			e := newItem(newData[0], element.Nest(), element.Comma(), space)
			j := app.cursor.Value.Path()
			e.SetPath(j.parent.ChildIndex(j.index + 1))
			app.list.InsertAfter(e, app.cursor)
			app.nextLine(session)
			app.dirty = true
		}
		element.SetComma(true)
	}
	return nil
}

func (app *Application) collapse(p *list.Element[Line], nest int, end Mark) (*list.List[Line], bool, error) {
	kill := list.New[Line]()
	for {
		if p == nil {
			return nil, false, errors.New("Unexpected state: missing element after '{' or '['")
		}
		n := p.Value
		kill.PushBack(n)
		q := p.Next()
		app.list.Remove(p)
		if n.Nest() == nest && end.Equals(n.Data()) {
			return kill, n.Comma(), nil
		}
		p = q
	}
}

func (app *Application) expand(at *list.Element[Line], lines *list.List[Line]) {
	for p := lines.Back(); p != nil; p = p.Prev() {
		app.list.InsertAfter(p.Value, at)
	}
}

func (app *Application) keyFuncRemove(session *Session) error {
	element := app.cursor.Value
	data := element.Data()
	if _, ok := data.(*tombstone); ok {
		return nil
	}
	var end Mark
	if v := unwrap(data); v == objStart {
		end = objEnd
	} else if v == arrayStart {
		end = arrayEnd
	} else {
		if _, ok := v.(Mark); !ok {
			element.SetData(&tombstone{first: data})
		}
		return nil
	}
	nest := element.Nest()
	if nest == 0 {
		return errors.New("Cannot delete top-level object or array")
	}
	kill, comma, err := app.collapse(app.cursor.Next(), nest, end)
	if err != nil {
		return err
	}
	element.SetData(&tombstone{
		first: element.Data(),
		rest:  kill,
	})

	element.SetComma(comma)
	return nil
}

func (app *Application) keyFuncCopy(session *Session) error {
	r := app.cursor.Value
	var buffer strings.Builder
	r.Path().Dump(&buffer)

	data := r.Data()
	if _, ok := unwrap(data).(Mark); !ok {
		buffer.WriteString(" = ")
		if f, ok := data.(interface{ Json() []byte }); ok {
			buffer.Write(f.Json())
		} else {
			bin, err := json.Marshal(data)
			if err != nil {
				fmt.Fprint(&buffer, data)
			} else {
				buffer.Write(bin)
			}
		}
	}
	s := buffer.String()
	app.message = s
	clipboard.WriteAll(s)
	return nil
}

func (app *Application) keyFuncUndo(session *Session) error {
	r := app.cursor.Value
	if d, ok := r.Data().(*tombstone); ok {
		r.SetData(d.first)
		if d.rest.Len() > 0 {
			r.SetComma(false)
			app.expand(app.cursor, d.rest)
		}
		return nil
	}
	m, ok := r.Data().(*modifiedLiteral)
	if !ok || m.backup == nil {
		return nil
	}
	if objStart.Equals(m.Literal) {
		next := app.cursor.Next()
		if next == nil || !objEnd.Equals(next.Value.Data()) {
			return errors.New("not empty object")
		}
		app.list.Remove(next)
	} else if arrayStart.Equals(m.Literal) {
		next := app.cursor.Next()
		if next == nil || !arrayEnd.Equals(next.Value.Data()) {
			return errors.New("not empty array")
		}
		app.list.Remove(next)
	}
	r.SetData(m.backup)
	return nil
}

type collapsed struct {
	name  string
	first any
	rest  *list.List[Line]
}

func (c *collapsed) Render(b *strings.Builder, _ func(any, *strings.Builder)) {
	b.WriteString(ansi.Red)
	b.WriteString(c.name)
	b.WriteString(ansi.Default)
}

func (c *collapsed) Json() []byte {
	var b bytes.Buffer
	fmt.Fprint(&b, c.first)
	p := c.rest.Front()
	for p != nil {
		next := p.Next()
		if isToBeContinued(p) && next != nil {
			p.Value.Dump(&b)
		} else {
			p.Value.DumpWithoutComma(&b)
		}
		p = next
	}
	return b.Bytes()
}

func (app *Application) keyFuncCollapseExpand(session *Session) error {
	element := app.cursor.Value
	data := element.Data()
	var end Mark
	var name string
	if date := unwrap(data); date == objStart {
		end = objEnd
		name = "{..}"
	} else if date == arrayStart {
		end = arrayEnd
		name = "[..]"
	} else if c, ok := date.(*collapsed); ok {
		app.expand(app.cursor, c.rest)
		r := app.cursor.Value
		r.SetData(c.first)
		r.SetComma(false)
		return nil
	} else {
		return nil
	}
	nest := element.Nest()
	if nest == 0 {
		return nil
	}
	kill, comma, err := app.collapse(app.cursor.Next(), nest, end)
	if err != nil {
		return err
	}
	element.SetData(&collapsed{
		name:  name,
		first: data,
		rest:  kill,
	})

	element.SetComma(comma)
	return nil
}
