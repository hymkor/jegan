package pager

import (
	"fmt"
	"io"
	"strings"

	"github.com/mattn/go-runewidth"

	"github.com/nyaosorg/go-readline-ny/keys"
	"github.com/nyaosorg/go-ttyadapter"

	"github.com/hymkor/go-generics-list"
	"github.com/hymkor/jegan/internal/ansi"
)

type EventResult int

const (
	Handled EventResult = iota
	PassToPager
	QuitApp
)

type Displayer interface {
	Display(width int) string
}

type Pager[T Displayer] struct {
	cache   []string
	Width   int
	Height  int
	Handler func(*Session[T], string) (EventResult, error)
	Status  func(*Session[T]) string
	offset  int
}

func Truncate(s string, width int) string {
	w := 0
	ansi := false
	overflow := false
	var b strings.Builder
	for _, c := range s {
		if !ansi {
			if c == '\x1B' {
				ansi = true
			} else {
				w += runewidth.RuneWidth(c)
				if w >= width {
					overflow = true
				}
			}
		}
		if !overflow || ansi {
			b.WriteRune(c)
		}
		if ansi && (('A' <= c && c <= 'Z') || ('a' <= c && c <= 'z')) {
			ansi = false
		}
	}
	return b.String()
}

func trimLeft(line string, offset int) string {
	if offset == 0 {
		return line
	}
	var buffer strings.Builder
	escape := false
	w := 0
	for i, c := range line {
		if c == '\x1B' {
			escape = true
		}
		if w >= offset {
			buffer.WriteString(line[i:])
			break
		}
		if escape {
			buffer.WriteRune(c)
		} else {
			w += runewidth.RuneWidth(c)
		}
		if ('A' <= c && c <= 'Z') || ('a' <= c && c <= 'z') {
			escape = false
		}
	}
	return buffer.String()
}

func (pager *Pager[T]) show(fetch func(int) (string, bool), out io.Writer) func() {
	i := 0
	for i < pager.Height {
		line, ok := fetch(pager.Width)
		if !ok {
			for ; i < len(pager.cache) && i < pager.Height; i++ {
				io.WriteString(out, ansi.EraseLine+"\n")
				pager.cache[i] = ""
			}
			break
		}
		line = trimLeft(line, pager.offset)
		if i >= len(pager.cache) || pager.cache[i] != line {
			io.WriteString(out, Truncate(line, pager.Width))
			io.WriteString(out, ansi.EraseLine)
		}
		out.Write([]byte{'\n'})
		if i < len(pager.cache) {
			pager.cache[i] = line
		} else {
			pager.cache = append(pager.cache, line)
		}
		i++
	}
	return func() {
		fmt.Fprintf(out, "\x1B[%dF", i)
	}
}

type Session[T Displayer] struct {
	*Pager[T]
	List   *list.List[T]
	Window *list.Element[T]
	WinPos int
	tail   *list.Element[T]
	TtyOut io.Writer
	GetKey func() (string, error)
}

func (session *Session[T]) UpdateStatus() {
	if session.Status != nil {
		line := session.Status(session)
		session.TtyOut.Write([]byte{'\r'})
		io.WriteString(session.TtyOut,
			Truncate(trimLeft(line, session.offset), session.Width))
		io.WriteString(session.TtyOut, ansi.EraseLine)
	}
}

func (session *Session[T]) MoveFront() {
	session.Window = session.List.Front()
	session.WinPos = 0
}

func (session *Session[T]) rollup() (i int) {
	for i < session.Height-1 {
		w := session.Window.Prev()
		if w == nil {
			return
		}
		session.Window = w
		session.WinPos--
		i++
	}
	return
}

func (session *Session[T]) MoveBack() int {
	session.Window = session.List.Back()
	session.WinPos = session.List.Len() - 1
	return session.rollup()
}

func (session *Session[T]) MoveNextPage() {
	for i := 0; i < session.Height && session.tail != nil; i++ {
		session.Window = session.Window.Next()
		session.WinPos++
		session.tail = session.tail.Next()
	}
}

func (session *Session[T]) MovePrevPage() {
	if w := session.Window.Prev(); w != nil {
		session.Window = w
		session.WinPos--
		session.rollup()
	}
}

func (session *Session[T]) MoveNextLine() {
	if session.tail != nil {
		if w := session.Window.Next(); w != nil {
			session.Window = w
			session.WinPos++
		}
	}
}

func (session *Session[T]) MovePrevLine() {
	if w := session.Window.Prev(); w != nil {
		session.Window = w
		session.WinPos--
	}
}

func (session *Session[T]) EventLoop() error {
	session.Window = session.List.Front()

	io.WriteString(session.TtyOut, ansi.CursorOff)
	defer io.WriteString(session.TtyOut, ansi.CursorOn+"\n")

	for {
		session.tail = session.Window
		rewind := session.show(func(width int) (line string, ok bool) {
			if session.tail != nil {
				line, ok = session.tail.Value.Display(session.offset+session.Width), true
				session.tail = session.tail.Next()
			}
			return
		}, session.TtyOut)

		session.UpdateStatus()

		key, err := session.GetKey()
		if err != nil {
			return err
		}
		if session.Handler != nil {
			if result, err := session.Handler(session, key); err != nil {
				return err
			} else if result == Handled {
				rewind()
				continue
			} else if result == QuitApp {
				return nil
			}
		}
		switch key {
		case "<":
			session.MoveFront()
		case ">":
			session.MoveBack()
		case " ":
			session.MoveNextPage()
		case "b":
			session.MovePrevPage()
		case "j", keys.Down, keys.CtrlN:
			session.MoveNextLine()
		case "k", keys.Up, keys.CtrlP:
			session.MovePrevLine()
		case "q", keys.CtrlC, keys.CtrlG:
			return nil
		case "l", keys.Right, keys.CtrlF:
			session.offset++
		case "h", keys.Left, keys.CtrlB:
			if session.offset > 0 {
				session.offset--
			}
		case "0", "^":
			session.offset = 0
		}
		rewind()
	}
	return nil
}

func (pager *Pager[T]) EventLoop(tty ttyadapter.Tty, L *list.List[T], ttyout io.Writer) error {
	if err := tty.Open(nil); err != nil {
		return err
	}
	defer tty.Close()

	width, height, err := tty.Size()
	if err != nil {
		return err
	}
	pager.Width = width
	pager.Height = height - 1

	session := &Session[T]{
		Pager:  pager,
		List:   L,
		GetKey: tty.GetKey,
		TtyOut: ttyout,
	}
	return session.EventLoop()
}
