package jegan

import (
	"bytes"
	"container/list"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/atotto/clipboard"

	"github.com/hymkor/jegan/internal/ansi"
	"github.com/hymkor/jegan/internal/pager"
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
	session *pager.Session,
	input func(*pager.Session, any) ([]any, error)) error {

	element := ref(app.cursor)
	defaultv := element.Value()
	if _, ok := defaultv.(*unjson.RawBytes); ok {
		return nil
	}
	prev := func() bool { return false }

	if v, ok := unwrap(element.Value()).(Mark); ok {
		next := app.cursor.Next()
		if next == nil {
			return nil
		}
		nextv, ok := unwrap(ref(next).Value()).(Mark)
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
	values, err := input(session, defaultv)
	if err != nil {
		return err
	}
	switch len(values) {
	case 1:
		if prev() || element.Value() != values[0] {
			app.dirty = true
			values[0] = backup(values[0], element.Value())
		}
		element.SetValue(values[0])
	case 2:
		prefix := ref(app.cursor).LeadingSpace()
		prev()
		app.list.InsertAfter(
			newElement(values[1], element.Nest(), element.Comma(), prefix),
			app.cursor)
		element.SetValue(backup(values[0], element.Value()))
		element.SetComma(false)
		app.dirty = true
	}
	return nil
}

type tombstone struct {
	first any
	rest  []Line
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

func (app *Application) inputFormat(session *pager.Session, defaultv any) ([]any, error) {
	defaults := makeDefaultFormat(defaultv)
	text, err := app.readLineElement(session, "New value:", defaults)
	if err != nil {
		return nil, err
	}
	return inputToAny(text)
}

func (app *Application) inputTypeAndValue(session *pager.Session, defaultv any) ([]any, error) {
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

func isDuplicated(cursor *list.Element, nest int, key string) bool {
	for p := cursor; p != nil; p = p.Prev() {
		i := ref(p).Nest()
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
		i := ref(p).Nest()
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
	nest := ref(p).Nest()
	for {
		p = p.Prev()
		if p == nil {
			return nil, false
		}
		element := ref(p)
		i := element.Nest()
		if i == nest {
			if pair, ok := p.Value.(*Pair); ok {
				return pair, true
			}
		} else if i < nest {
			if element.Value() != objStart {
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

func reflectIndex(p *list.Element, nest int, plusminus int) {
	for ; p != nil; p = p.Next() {
		r := ref(p)
		if n := r.Nest(); n < nest {
			return
		} else if n > nest {
			continue
		}
		if path := r.Path(); path != nil {
			if v := r.Value(); v != objEnd && v != arrayEnd {
				path.index += plusminus
			}
		}
	}
}

func (app *Application) keyFuncInsert(session *pager.Session) error {
	space := ref(app.cursor).LeadingSpace()
	currentNest := ref(app.cursor).Nest()
	if e := ref(app.cursor); arrayStart.Equals(e.Value()) {
		next := app.cursor.Next()
		nextElement := ref(next)
		var comma bool
		var nest int
		var newPrefix []byte
		todo := func() {}
		if arrayEnd.Equals(nextElement.Value()) {
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
		values, err := app.inputFormat(session, struct{}{})
		if err != nil {
			return err
		}
		switch len(values) {
		case 2: // [\n[\n],\n
			reflectIndex(app.cursor.Next(), currentNest+1, +1)
			e1 := newElement(values[0], nest, false, newPrefix)
			e1.SetPath(ref(app.cursor).Path().ChildIndex(0))
			e2 := newElement(values[1], nest, comma, nil)
			e2.SetPath(e1.Path())
			app.list.InsertBefore(e1, next)
			app.list.InsertBefore(e2, next)
			todo()
			app.nextLine(session)
			app.dirty = true
		case 1: // [\n value
			reflectIndex(app.cursor.Next(), currentNest+1, +1)
			e1 := newElement(values[0], nest, comma, newPrefix)
			e1.SetPath(ref(app.cursor).Path().ChildIndex(0))
			app.list.InsertBefore(e1, next)
			todo()
			app.nextLine(session)
			app.dirty = true
		}
		return nil
	}
	if e := ref(app.cursor); objStart.Equals(e.Value()) {
		key, err := app.readLineString(session, "Key:", "")
		if err != nil {
			return err
		}
		sample := findPairBefore(app.cursor)
		next := app.cursor.Next()
		nextElement := ref(next)
		var comma bool
		var nest int
		var newPrefix []byte
		todo := func() {}
		if objEnd.Equals(nextElement.Value()) {
			comma = false
			outerPrefix := nextElement.LeadingSpace() // space before }
			if len(outerPrefix) == 0 {
				outerPrefix = space
				todo = func() { ref(next).SetLeadingSpace(space) }
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
		values, err := app.inputFormat(session, struct{}{})
		if err != nil {
			return err
		}
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
		return nil
	}
	if sample, ok := findSameLevelPairBefore(app.cursor); ok {
		key, err := app.readLineString(session, "Key:", "")
		if err != nil {
			return err
		}
		element := ref(app.cursor)
		if isDuplicated(app.cursor, element.Nest(), key) {
			return fmt.Errorf("duplicate key: %q", key)
		}
		values, err := app.inputFormat(session, struct{}{})
		if err != nil {
			return err
		}
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
		return nil
	}
	if element, ok := app.cursor.Value.(*Element); ok {
		nest := ref(app.cursor).Nest()
		if nest < 0 {
			return nil
		}
		values, err := app.inputFormat(session, struct{}{})
		if err != nil {
			return nil
		}
		switch len(values) {
		case 2: // [ \n ],
			reflectIndex(app.cursor.Next(), currentNest, +1)
			e1 := newElement(values[0], element.Nest(), false, space)
			e2 := newElement(values[1], element.Nest(), element.Comma(), nil)
			j := ref(app.cursor).Path()
			e1.SetPath(j.parent.ChildIndex(j.index + 1))
			e2.SetPath(e1.Path())
			app.list.InsertAfter(e2, app.cursor)
			app.list.InsertAfter(e1, app.cursor)
			app.nextLine(session)
			app.dirty = true
		case 1: // value,
			reflectIndex(app.cursor.Next(), currentNest, +1)
			e := newElement(values[0], element.Nest(), element.Comma(), space)
			j := ref(app.cursor).Path()
			e.SetPath(j.parent.ChildIndex(j.index + 1))
			app.list.InsertAfter(e, app.cursor)
			app.nextLine(session)
			app.dirty = true
		}
		element.SetComma(true)
	}
	return nil
}

func (app *Application) collapse(p *list.Element, nest int, end Mark) ([]Line, bool, error) {
	kill := []Line{}
	for {
		if p == nil {
			return nil, false, errors.New("Unexpected state: missing element after '{' or '['")
		}
		n := ref(p)
		kill = append(kill, n)
		q := p.Next()
		app.list.Remove(p)
		if n.Nest() == nest && end.Equals(n.Value()) {
			return kill, n.Comma(), nil
		}
		p = q
	}
}

func (app *Application) expand(p *list.Element, lines []Line) {
	for i := len(lines) - 1; i >= 0; i-- {
		app.list.InsertAfter(lines[i], app.cursor)
	}
}

func (app *Application) keyFuncRemove(session *pager.Session) error {
	element := ref(app.cursor)
	value := element.Value()
	if _, ok := value.(*tombstone); ok {
		return nil
	}
	var end Mark
	if v := unwrap(value); v == objStart {
		end = objEnd
	} else if v == arrayStart {
		end = arrayEnd
	} else {
		if _, ok := v.(Mark); !ok {
			element.SetValue(&tombstone{first: value})
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
	element.SetValue(&tombstone{
		first: element.Value(),
		rest:  kill,
	})
	element.SetComma(comma)
	return nil
}

func (app *Application) keyFuncCopy(session *pager.Session) error {
	r := ref(app.cursor)
	var buffer strings.Builder
	r.Path().Dump(&buffer)

	v := r.Value()
	if _, ok := unwrap(v).(Mark); !ok {
		buffer.WriteString(" = ")
		if f, ok := v.(interface{ Json() []byte }); ok {
			buffer.Write(f.Json())
		} else {
			bin, err := json.Marshal(v)
			if err != nil {
				fmt.Fprint(&buffer, v)
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

func (app *Application) keyFuncUndo(session *pager.Session) error {
	r := ref(app.cursor)
	if d, ok := r.Value().(*tombstone); ok {
		r.SetValue(d.first)
		if len(d.rest) > 0 {
			r.SetComma(false)
			app.expand(app.cursor, d.rest)
		}
		return nil
	}
	m, ok := r.Value().(*modifiedLiteral)
	if !ok || m.backup == nil {
		return nil
	}
	if objStart.Equals(m.Literal) {
		next := app.cursor.Next()
		if next == nil || !objEnd.Equals(ref(next).Value()) {
			return errors.New("not empty object")
		}
		app.list.Remove(next)
	} else if arrayStart.Equals(m.Literal) {
		next := app.cursor.Next()
		if next == nil || !arrayEnd.Equals(ref(next).Value()) {
			return errors.New("not empty array")
		}
		app.list.Remove(next)
	}
	r.SetValue(m.backup)
	return nil
}

type collapsed struct {
	name  string
	first any
	rest  []Line
}

func (c *collapsed) Render(b *strings.Builder, _ func(any, *strings.Builder)) {
	b.WriteString(ansi.Red)
	b.WriteString(c.name)
	b.WriteString(ansi.Default)
}

func (c *collapsed) Json() []byte {
	var b bytes.Buffer
	fmt.Fprint(&b, c.first)
	for i := 0; i < len(c.rest)-1; i++ {
		c.rest[i].Dump(&b)
	}
	c.rest[len(c.rest)-1].DumpWithoutComma(&b)
	return b.Bytes()
}

func (app *Application) keyFuncCollapseExpand(session *pager.Session) error {
	element := ref(app.cursor)
	value := element.Value()
	var end Mark
	var name string
	if v := unwrap(value); v == objStart {
		end = objEnd
		name = "{..}"
	} else if v == arrayStart {
		end = arrayEnd
		name = "[..]"
	} else if c, ok := v.(*collapsed); ok {
		app.expand(app.cursor.Next(), c.rest)
		r := ref(app.cursor)
		r.SetValue(c.first)
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
	element.SetValue(&collapsed{
		name:  name,
		first: value,
		rest:  kill,
	})
	element.SetComma(comma)
	return nil
}
