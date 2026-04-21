package jegan

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/hymkor/go-generics-list"

	"github.com/hymkor/jegan/internal/ansi"
)

type collapsed struct {
	name  string
	first any
	rest  *List
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

func (app *Application) collapse(p *Element, nest int, end Mark) (*List, bool, error) {
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

func (app *Application) expand(at *Element, lines *List) {
	for p := lines.Back(); p != nil; p = p.Prev() {
		app.list.InsertAfter(p.Value, at)
	}
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

type tombstone struct {
	first any
	rest  *List
}

func (r *tombstone) SkipDump() {}

func (r *tombstone) Json() []byte {
	return []byte{}
}

func (t *tombstone) Render(b *strings.Builder, _ func(any, *strings.Builder)) {
	b.WriteString(ansi.Red)
	b.WriteString("<DEL>")
	b.WriteString(ansi.Default)
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
