package jegan

import (
	"container/list"
	"errors"
	"fmt"
	"strings"

	"github.com/hymkor/jegan/internal/pager"
	"github.com/hymkor/jegan/internal/unjson"
)

func unwrap(v any) any {
	if vv, ok := v.(*modifiedLiteral); ok {
		v = vv.Literal
	}
	if vv, ok := v.(*unjson.Literal); ok {
		v = vv.Value()
	}
	return v
}

func newCompare(v any) (func(key string, value any) bool, bool) {
	v = unwrap(v)
	if s, ok := v.(string); ok {
		s = strings.ToLower(s)
		return func(key string, value any) bool {
			key = strings.ToLower(key)
			if strings.Contains(key, s) {
				return true
			}
			other := unwrap(value)
			if o, ok := other.(string); ok {
				o = strings.ToLower(o)
				return strings.Contains(o, s)
			}
			return false
		}, true
	}
	if n, ok := v.(float64); ok {
		return func(_ string, other any) bool {
			other = unwrap(other)
			if o, ok := other.(float64); ok {
				return n == o
			}
			return false
		}, true
	}
	if v == nil {
		return func(_ string, other any) bool {
			other = unwrap(other)
			return other == nil
		}, true
	}
	if v == true {
		return func(_ string, other any) bool {
			other = unwrap(other)
			if o, ok := other.(bool); ok {
				return o == true
			}
			return false
		}, true
	}
	if v == false {
		return func(_ string, other any) bool {
			other = unwrap(other)
			if o, ok := other.(bool); ok {
				return o == false
			}
			return false
		}, true
	}
	return func(string, any) bool { return false }, false
}

func (app *Application) keyFuncSearch(session *pager.Session, revert bool) error {

	prompt := "Search:"
	if revert {
		prompt = "Search (backward):"
	}
	text, err := app.readLineElement(session, prompt, "")
	if err != nil {
		return err
	}
	targets, err := inputToAny(text)
	if err != nil {
		return err
	}
	if len(targets) != 1 {
		return errors.New("can not search not single value")
	}
	compare, ok := newCompare(targets[0])
	if !ok {
		return fmt.Errorf("can not search: %v", targets[0])
	}
	if revert {
		app.search = func() error {
			app.message = fmt.Sprintf("Prev: %v", targets[0])
			return app.searchBackward(session, compare)
		}
		app.revert = func() error {
			app.message = fmt.Sprintf("Next: %v", targets[0])
			return app.searchForward(session, compare)
		}
		return app.searchBackward(session, compare)
	}
	app.search = func() error {
		app.message = fmt.Sprintf("Next: %v", targets[0])
		return app.searchForward(session, compare)
	}
	app.revert = func() error {
		app.message = fmt.Sprintf("Prev: %v", targets[0])
		return app.searchBackward(session, compare)
	}
	return app.searchForward(session, compare)
}

func compareKeyAndValue(p *list.Element, compare func(string, any) bool) bool {
	if pair, ok := p.Value.(*Pair); ok {
		return compare(pair.key, pair.value)
	}
	return compare("", ref(p).Value())
}

func (app *Application) searchForward(
	session *pager.Session,
	compare func(key string, value any) bool) error {

	_cursor := app.cursor
	_csrPos := app.csrline

	for {
		_cursor = _cursor.Next()
		if _cursor == nil {
			return nil
		}
		_csrPos++

		if compareKeyAndValue(_cursor, compare) {
			app.setCursor(_cursor)
			app.csrline = _csrPos
			for _csrPos-session.WinPos >= session.Height {
				session.Window = session.Window.Next()
				session.WinPos++
			}
			return nil
		}
	}
	return nil
}

func (app *Application) searchBackward(
	session *pager.Session,
	compare func(string, any) bool) error {

	_cursor := app.cursor
	_csrPos := app.csrline

	for {
		_cursor = _cursor.Prev()
		if _cursor == nil {
			return nil
		}
		_csrPos--

		if compareKeyAndValue(_cursor, compare) {
			app.setCursor(_cursor)
			app.csrline = _csrPos
			for _csrPos < session.WinPos {
				session.Window = _cursor
				session.WinPos = _csrPos
			}
			return nil
		}
	}
	return nil
}
