package jegan

import (
	"errors"
	"fmt"
	"strings"

	"github.com/hymkor/jegan/types"
)

func newCompare(v any) (func(key string, data any) bool, bool) {
	v = types.Unwrap(v)
	if s, ok := v.(string); ok {
		s = strings.ToLower(s)
		return func(key string, data any) bool {
			key = strings.ToLower(key)
			if strings.Contains(key, s) {
				return true
			}
			other := types.Unwrap(data)
			if o, ok := other.(string); ok {
				o = strings.ToLower(o)
				return strings.Contains(o, s)
			}
			return false
		}, true
	}
	if n, ok := v.(float64); ok {
		return func(_ string, other any) bool {
			other = types.Unwrap(other)
			if o, ok := other.(float64); ok {
				return n == o
			}
			return false
		}, true
	}
	if v == nil {
		return func(_ string, other any) bool {
			other = types.Unwrap(other)
			return other == nil
		}, true
	}
	if v == true {
		return func(_ string, other any) bool {
			other = types.Unwrap(other)
			if o, ok := other.(bool); ok {
				return o == true
			}
			return false
		}, true
	}
	if v == false {
		return func(_ string, other any) bool {
			other = types.Unwrap(other)
			if o, ok := other.(bool); ok {
				return o == false
			}
			return false
		}, true
	}
	return func(string, any) bool { return false }, false
}

func (app *Application) keyFuncSearch(session *Session, revert bool) error {

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

	match := func(p *Element) scanResult {
		var result bool
		if pair, ok := p.Value.(*types.Pair); ok {
			result = compare(pair.Key, pair.Data())
		} else {
			result = compare("", p.Value.Data())
		}
		if result {
			return scanMatch
		}
		return scanContinue
	}

	if revert {
		app.search = func() error {
			app.message = fmt.Sprintf("Prev: %v", targets[0])
			return app.searchBackward(session, match)
		}
		app.revert = func() error {
			app.message = fmt.Sprintf("Next: %v", targets[0])
			return app.searchForward(session, match)
		}
		return app.searchBackward(session, match)
	}
	app.search = func() error {
		app.message = fmt.Sprintf("Next: %v", targets[0])
		return app.searchForward(session, match)
	}
	app.revert = func() error {
		app.message = fmt.Sprintf("Prev: %v", targets[0])
		return app.searchBackward(session, match)
	}
	return app.searchForward(session, match)
}

func (app *Application) searchForward(session *Session, match func(*Element) scanResult) error {
	app.scanForwardUntil(session, match)
	return nil
}

func (app *Application) searchBackward(session *Session, match func(*Element) scanResult) error {
	app.scanBackwardUntil(session, match)
	return nil
}

type scanResult int

const (
	scanContinue scanResult = iota
	scanMatch
	scanStop
)

func (app *Application) scanForwardUntil(
	session *Session,
	match func(*Element) scanResult) bool {

	_cursor := app.cursor
	_csrPos := app.csrline

	for {
		_cursor = _cursor.Next()
		if _cursor == nil {
			return false
		}
		_csrPos++

		result := match(_cursor)
		if result == scanMatch {
			app.setCursor(_cursor)
			app.csrline = _csrPos
			for _csrPos-session.WinPos >= session.ContentHeight {
				session.Window = session.Window.Next()
				session.WinPos++
			}
			return true
		}
		if result == scanStop {
			return false
		}
	}
}

func (app *Application) scanBackwardUntil(
	session *Session,
	match func(*Element) scanResult) bool {

	_cursor := app.cursor
	_csrPos := app.csrline

	for {
		_cursor = _cursor.Prev()
		if _cursor == nil {
			return false
		}
		_csrPos--

		result := match(_cursor)
		if result == scanMatch {
			app.setCursor(_cursor)
			app.csrline = _csrPos
			for _csrPos < session.WinPos {
				session.Window = _cursor
				session.WinPos = _csrPos
			}
			return true
		}
		if result == scanStop {
			return false
		}
	}
}

func matchMark(nest int, target types.Mark) func(*Element) scanResult {
	return func(p *Element) scanResult {
		if p.Value.Nest() == nest {
			if types.Unwrap(p.Value.Data()) == target {
				return scanMatch
			}
			nest--
			if nest < 0 {
				return scanStop
			}
		}
		return scanContinue
	}
}

func (app *Application) keyFuncUpperGroupHead(session *Session, target, here types.Mark) error {
	nest := app.cursor.Value.Nest()
	if types.Unwrap(app.cursor.Value.Data()) != here {
		nest--
	}
	found := app.scanBackwardUntil(session, matchMark(nest, target))
	if !found {
		app.message = "Not found: " + string(target)
	}
	return nil
}

func (app *Application) keyFuncUpperGroupTail(session *Session, target, here types.Mark) error {
	nest := app.cursor.Value.Nest()
	if types.Unwrap(app.cursor.Value.Data()) != here {
		nest--
	}
	found := app.scanForwardUntil(session, matchMark(nest, target))
	if !found {
		app.message = "Not found: " + string(target)
	}
	return nil
}
