package jegan

import (
	"bytes"
	"fmt"
	"io"
	"runtime"
	"strings"

	"github.com/nyaosorg/go-readline-ny/keys"
	"github.com/nyaosorg/go-readline-ny/simplehistory"
	"github.com/nyaosorg/go-ttyadapter"

	"github.com/hymkor/go-generics-list"
	"github.com/hymkor/go-safewrite/perm"

	"github.com/hymkor/jegan/internal/ansi"
	"github.com/hymkor/jegan/internal/pager"
)

type Application struct {
	Name string

	list    *List
	cursor  *Element
	csrline int
	message string
	dirty   bool
	indent  []byte
	ttyIn   ttyadapter.Tty
	history *simplehistory.Container

	search func() error
	revert func() error
}

func (app *Application) Store(v *List) {
	if v == nil {
		return
	}
	if app.list == nil {
		app.list = v
	} else {
		app.list.PushBackList(v)
	}
}

func (app *Application) setCursor(c *Element) {
	app.cursor.Value.SetCursor(false)
	app.cursor = c
	app.cursor.Value.SetCursor(true)
}

func (app *Application) nextLine(session *Session) {
	c := app.cursor.Next()
	if c == nil {
		return
	}
	app.setCursor(c)
	app.csrline++
	for app.csrline-session.WinPos >= session.ContentHeight {
		session.MoveNextLine()
	}
}

func (app *Application) keyFuncNextPage(session *Session) {
	app.cursor.Value.SetCursor(false)
	defer func() {
		app.cursor.Value.SetCursor(true)
	}()

	for i := 0; i < session.ContentHeight; i++ {
		c := app.cursor.Next()
		if c == nil {
			break
		}
		app.cursor = c
		app.csrline++
		session.MoveNextLine()
	}
}

func (app *Application) keyFuncPrevPage(session *Session) {
	app.cursor.Value.SetCursor(false)
	defer func() {
		app.cursor.Value.SetCursor(true)
	}()

	session.MovePrevPage()
	for i := 0; i < session.ContentHeight; i++ {
		c := app.cursor.Prev()
		if c == nil {
			break
		}
		app.cursor = c
		app.csrline--
	}
}

func (app *Application) handle(session *Session, key string) (pager.EventResult, error) {
	result := pager.Handled
	var err error
	switch key {
	default:
		return pager.PassToPager, nil
	case "j", keys.Down, keys.CtrlN:
		app.nextLine(session)
	case "k", keys.Up, keys.CtrlP:
		if c := app.cursor.Prev(); c != nil {
			app.setCursor(c)
			app.csrline--
			for app.csrline < session.WinPos {
				session.MovePrevLine()
			}
		}
	case "/":
		err = app.keyFuncSearch(session, false)
	case "?":
		err = app.keyFuncSearch(session, true)
	case "n":
		if app.search != nil {
			err = app.search()
		}
	case "N":
		if app.revert != nil {
			err = app.revert()
		}
	case "<":
		app.setCursor(app.list.Front())
		app.csrline = 0
		session.MoveFront()
	case ">":
		app.setCursor(app.list.Back())
		app.csrline = app.list.Len() - 1
		session.MoveBack()
	case " ", keys.PageDown:
		app.keyFuncNextPage(session)
	case "b", keys.PageUp:
		app.keyFuncPrevPage(session)
	case keys.CtrlG:
	case "r":
		err = app.keyFuncReplace(session, app.inputFormat)
	case "R":
		err = app.keyFuncReplace(session, app.inputTypeAndValue)
	case "o":
		err = app.keyFuncInsert(session)
	case "d":
		err = app.keyFuncRemove(session)
	case keys.CtrlC:
		err = app.keyFuncCopy(session)
	case "u":
		err = app.keyFuncUndo(session)
	case "z":
		err = app.keyFuncCollapseExpand(session)
	case "w":
		err = app.keyFuncSave(session)
	case "q":
		result, err = app.keyFuncQuit(session)
	}
	if err != nil {
		app.message = err.Error()
		debug(app.message)
	}
	return result, nil
}

func (app *Application) status(session *Session) (text string) {
	if app.message != "" {
		text = fmt.Sprintf(ansi.Bold+"%s"+ansi.Thin+ansi.EraseLine, app.message)
		app.message = ""
	} else if app.Name != "" {
		var b strings.Builder

		b.WriteString(ansi.Reverse)
		b.WriteString(app.Name)
		b.WriteString(ansi.Inverse)
		if app.dirty {
			b.WriteString("* ")
		} else {
			b.WriteString("  ")
		}
		r := app.cursor.Value
		r.Path().Dump(&b)
		if p, ok := r.(*Pair); ok {
			if _, ok := unwrap(p.Item.data).(Mark); !ok {
				b.WriteString(" = ")
				p.Item.highlightWithoutComma(&b)
			}
		} else if e, ok := r.(*Item); ok {
			if _, ok := unwrap(e.data).(Mark); !ok {
				b.WriteString(" = ")
				e.highlightWithoutComma(&b)
			}
		}
		b.WriteString(ansi.EraseLine)

		text = b.String()
	} else {
		text = fmt.Sprintf(ansi.Bold+"Jegan %s-%s-%s"+ansi.Thin+ansi.EraseLine,
			version, runtime.GOOS, runtime.GOARCH)
	}
	return
}

func (app *Application) EventLoop(ttyIn ttyadapter.Tty, ttyOut io.Writer) error {
	app.ttyIn = ttyIn
	if app.list == nil {
		app.list = list.New[Line]()
	}
	if app.list.Len() <= 0 {
		app.list.PushBack(newItem(objStart, 0, false, nil))
		app.list.PushBack(newItem(objEnd, 0, false, nil))
	}
	if app.cursor == nil {
		app.cursor = app.list.Front()
		app.cursor.Value.SetCursor(true)
	}
	if sample := app.list.Front().Next(); sample != nil {
		prefix := sample.Value.LeadingSpace()
		if pos := bytes.IndexByte(prefix, '\n'); pos >= 0 {
			app.indent = prefix[pos+1:]
		}
	}
	pager1 := &pager.Pager[Line]{
		Status:  app.status,
		Handler: app.handle,
	}
	return pager1.EventLoop(ttyIn, app.list, ttyOut)
}

func (app *Application) Close() error {
	return perm.RestoreAll()
}
