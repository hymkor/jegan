package jegan

import (
	"strings"

	"github.com/nyaosorg/go-ttyadapter/tty8pe"

	"github.com/hymkor/go-generics-list"
	"github.com/hymkor/nemo/pager"
)

var help = `Jegan - A Terminal JSON Editor

- 'F1' : Show this help screen (press 'q' to close)

- 'j', '↓', 'Ctrl-N' : Move to the next item
- 'k', '↑', 'Ctrl-P' : Move to the previous item
- 'l', '→', 'Ctrl-F' : Scroll the view to the right
- 'h', '←', 'Ctrl-B' : Scroll the view to the left
- '0', '^' : Reset horizontal scroll (jump to column 0)
- 'Space', 'PageDown' : Move to the next page of items
- 'b', 'PageUp'       : Move to the previous page of items
- '<' : Move to the first item
- '>' : Move to the last item
- '/' : Search forward
- '?' : Search backward
- 'n' : Repeat search in the same direction
- 'N' : Repeat search in the opposite direction
- '@' : Jump to the item specified by a JSON path
- 'z' : Toggle collapse/expand
- 'o' : Insert a new item below the cursor.
  - For object items, enter both key and value.
  - For array items, enter only the value.
  - The key is used as entered (no quotes required).
  - The value is interpreted as follows:
    - '"..."' → string (escape sequences are interpreted)
    - Input that can be parsed as a number → number
    - 'null' → null
    - 'true' / 'false' → boolean
    - '{}' → empty object
    - '[]' → empty array
    - Otherwise → string (used as-is)
  - 'Ctrl+G' cancels the current input
  - Empty input is treated as an empty string ('""').
  - Duplicate keys in objects are not allowed.
- 'r' : Modify the item at the cursor (same input method as 'o')
- 'R' : Modify the item at the cursor (explicitly specify the value type)
- 'd' : Delete the item at the cursor
- 'u' : UNDO
- 'Ctrl+C' : Copy the current path and value to the clipboard
- 'w' : Save to file
- 'q' : Quit`

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
		&tty8pe.Tty{},  // terminal input
		lines,          // data source
		session.TtyOut) // output

	session.ClearCache()

	return err
}
