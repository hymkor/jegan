package jegan

import (
	"github.com/hymkor/jegan/types"
)

func (app *Application) keyFuncUpperGroupHead(session *Session, target, here types.Mark) error {
	p := app.cursor
	csrline := app.csrline
	nest := p.Value.Nest()
	if types.Unwrap(p.Value.Data()) != here {
		nest--
	}
	for {
		p = p.Prev()
		csrline--

		if p == nil {
			break
		}
		if p.Value.Nest() == nest {
			if types.Unwrap(p.Value.Data()) == target {
				app.setCursor(p)
				app.csrline = csrline
				if csrline < session.WinPos {
					session.Window = p
					session.WinPos = csrline
				}
				return nil
			}
			nest--
			if nest < 0 {
				break
			}
		}
	}
	app.message = "Not found: " + string(target)
	return nil
}

func (app *Application) keyFuncUpperGroupTail(session *Session, target, here types.Mark) error {
	p := app.cursor
	csrline := app.csrline
	nest := p.Value.Nest()
	if types.Unwrap(p.Value.Data()) != here {
		nest--
	}
	for {
		p = p.Next()
		csrline++

		if p == nil {
			break
		}
		if p.Value.Nest() == nest {
			if types.Unwrap(p.Value.Data()) == target {
				app.setCursor(p)
				app.csrline = csrline
				if csrline >= session.WinPos+session.ContentHeight {
					session.Window = p
					session.WinPos = csrline
					for i := 0; i < session.ContentHeight-1; i++ {
						p = session.Window.Prev()
						if p == nil {
							break
						}
						session.Window = p
						session.WinPos--
					}
				}
				return nil
			}
			nest--
			if nest < 0 {
				break
			}
		}
	}
	app.message = "Not found: " + string(target)
	return nil
}
