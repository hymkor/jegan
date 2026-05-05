package jegan

import (
	_ "embed"
	"strings"

	"github.com/nyaosorg/go-ttyadapter/fav"

	"github.com/hymkor/go-generics-list"
	"github.com/hymkor/nemo/pager"
)

//go:embed help.md
var help string

// TextElement represents one line in the pager
type TextElement struct {
	Text string
}

// Display is called by pager to render each line
func (t TextElement) Display(screenWidth int) string {
	return t.Text
}

func (app *Application) keyFuncHelp(session *Session) error {
	session.TtyOut.Write([]byte{'\r'})

	// Create a linked list of lines
	lines := list.New[TextElement]()

	lines.PushBack(TextElement{Text: "Jegan - A Terminal JSON Editor"})
	lines.PushBack(TextElement{Text: ""})

	for _, line := range strings.Split(help, "\n") {
		lines.PushBack(TextElement{Text: line})
	}
	// Create pager
	pg := &pager.Pager[TextElement]{
		Status: func(session *pager.Session[TextElement]) string {
			return "\x1B[7mHELP\x1B[27m"
		},
	}
	// Run pager event loop
	err := pg.EventLoop(
		&fav.Tty{},     // terminal input
		lines,          // data source
		session.TtyOut) // output

	session.ClearCache()

	return err
}
