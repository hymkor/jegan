package jegan

import (
	"bytes"
	"container/list"
	"fmt"
	"io"
	"runtime"

	"github.com/nyaosorg/go-readline-ny/keys"
	"github.com/nyaosorg/go-ttyadapter"

	"github.com/hymkor/go-safewrite/perm"

	"github.com/hymkor/jegan/internal/ansi"
	"github.com/hymkor/jegan/internal/pager"
)

type Application struct {
	Name string

	list    *list.List
	cursor  *list.Element
	csrline int
	message string
	dirty   bool
	indent  []byte
	ttyin   ttyadapter.Tty
}

func (app *Application) Store(v *list.List) {
	if v == nil {
		return
	}
	if app.list == nil {
		app.list = v
	} else {
		app.list.PushBackList(v)
	}
}

func (app *Application) setCursor(c *list.Element) {
	ref(app.cursor).SetCursor(false)
	app.cursor = c
	ref(app.cursor).SetCursor(true)
}

func (app *Application) nextLine(session *pager.Session) {
	c := app.cursor.Next()
	if c == nil {
		return
	}
	app.setCursor(c)
	app.csrline++
	for app.csrline-session.WinPos >= session.Height {
		session.MoveNextLine()
	}
}

func (app *Application) keyFuncNextPage(session *pager.Session) {
	ref(app.cursor).SetCursor(false)
	defer func() {
		ref(app.cursor).SetCursor(true)
	}()

	for i := 0; i < session.Height; i++ {
		c := app.cursor.Next()
		if c == nil {
			break
		}
		app.cursor = c
		app.csrline++
		session.MoveNextLine()
	}
}

func (app *Application) keyFuncPrevPage(session *pager.Session) {
	ref(app.cursor).SetCursor(false)
	defer func() {
		ref(app.cursor).SetCursor(true)
	}()

	session.MovePrevPage()
	for i := 0; i < session.Height; i++ {
		c := app.cursor.Prev()
		if c == nil {
			break
		}
		app.cursor = c
		app.csrline--
	}
}

func (app *Application) handle(session *pager.Session, key string) (pager.EventResult, error) {
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
	case keys.CtrlC, keys.CtrlG:
	case "r":
		app.keyFuncReplace(session, app.inputFormat)
	case "R":
		app.keyFuncReplace(session, app.inputTypeAndValue)
	case "o":
		app.keyFuncInsert(session)
	case "d":
		app.keyFuncRemove(session)
	case "w":
		app.keyFuncSave(session)
	case "q":
		return app.keyFuncQuit(session), nil
	}
	return pager.Handled, nil
}

func (app *Application) status(session *pager.Session) (rv string) {
	if app.message != "" {
		rv = fmt.Sprintf(ansi.Bold+"%s"+ansi.Thin+ansi.EraseLine, app.message)
		app.message = ""
	} else if app.Name != "" {
		var mark rune
		if app.dirty {
			mark = '*'
		} else {
			mark = ' '
		}
		rv = fmt.Sprintf(ansi.Reverse+"%s"+ansi.Inverse+"%c"+ansi.EraseLine, app.Name, mark)
	} else {
		rv = fmt.Sprintf(ansi.Bold+"Jegan %s-%s-%s"+ansi.Thin+ansi.EraseLine,
			version, runtime.GOOS, runtime.GOARCH)
	}
	return
}

func (app *Application) EventLoop(tty ttyadapter.Tty, ttyout io.Writer) error {
	app.ttyin = tty
	if app.list == nil {
		app.list = list.New()
	}
	if app.list.Len() <= 0 {
		app.list.PushBack(newElement(Mark('{'), 0, false, nil))
		app.list.PushBack(newElement(Mark('}'), 0, false, nil))
	}
	if app.cursor == nil {
		app.cursor = app.list.Front()
		ref(app.cursor).SetCursor(true)
	}
	if sample := app.list.Front().Next(); sample != nil {
		prefix := ref(sample).LeadingSpace()
		if pos := bytes.IndexByte(prefix, '\n'); pos >= 0 {
			app.indent = prefix[pos+1:]
		}
	}
	pager1 := &pager.Pager{
		Status:  app.status,
		Handler: app.handle,
	}
	return pager1.EventLoop(tty, app.list, ttyout)
}

func (app *Application) Close() error {
	return perm.RestoreAll()
}
