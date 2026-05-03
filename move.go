package jegan

import (
	"errors"
	"io"

	"github.com/nyaosorg/go-readline-ny"

	"github.com/hymkor/jegan/types"
)

func (app *Application) keyFuncMoveTo(session *Session) error {
	location, err := app.readLineOpt(session, "JSON Path:", "", func(*readline.Editor) {})
	if err != nil {
		return err
	}
	jsonpath, err := types.ParseJson(location)
	if err != nil {
		return err
	}
	p, n := jsonpath.Search(app.list.Front())
	if p == nil {
		p = app.list.Back()
		n = app.list.Len() - 1

		err = app.completeLoading(session)
		if err != nil && !errors.Is(err, io.EOF) {
			return err
		}
		q, m := jsonpath.Search(p)
		if q == nil {
			app.message = jsonpath.String() + ": not found"
			return nil
		}
		n += m
		p = q
	}
	app.setCursor(p)
	app.csrline = n
	session.Window = p
	session.WinPos = n
	for i := session.ContentHeight / 2; i > 0; i-- {
		p := session.Window.Prev()
		if p == nil {
			break
		}
		session.Window = p
		session.WinPos--
	}
	return nil
}
