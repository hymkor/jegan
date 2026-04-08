package pager

import (
	"container/list"
	"fmt"
	"io"
	"strings"

	"github.com/mattn/go-runewidth"

	"github.com/nyaosorg/go-readline-ny/keys"
	"github.com/nyaosorg/go-ttyadapter"

	"github.com/hymkor/jegan/internal/ansi"
)

type Pager struct {
	cache   []string
	Width   int
	Height  int
	Handler func(*Session, string) (bool, error)
	Status  func(*Session) string
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

func (pager *Pager) Show(fetch func(int) (string, bool), out io.Writer) func() {
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

type Session struct {
	*Pager
	List   *list.List
	Window *list.Element
	tail   *list.Element
	TtyOut io.Writer
	GetKey func() (string, error)
}

func (session *Session) Front() {
	session.Window = session.List.Front()
}

func (session *Session) rollup() (i int) {
	for i < session.Height-1 {
		w := session.Window.Prev()
		if w == nil {
			return
		}
		session.Window = w
		i++
	}
	return
}

func (session *Session) Back() int {
	session.Window = session.List.Back()
	return session.rollup()
}

func (session *Session) NextPage() {
	for i := 0; i < session.Height && session.tail != nil; i++ {
		session.Window = session.Window.Next()
		session.tail = session.tail.Next()
	}
}

func (session *Session) PrevPage() {
	if w := session.Window.Prev(); w != nil {
		session.Window = w
		session.rollup()
	}
}

func (session *Session) Next() {
	if session.tail != nil {
		if w := session.Window.Next(); w != nil {
			session.Window = w
		}
	}
}

func (session *Session) Prev() {
	if w := session.Window.Prev(); w != nil {
		session.Window = w
	}
}

type Displayer interface {
	Display(width int) string
}

func (pager *Pager) eventLoop(getkey func() (string, error), L *list.List, ttyout io.Writer) error {
	session := &Session{
		Pager:  pager,
		Window: L.Front(),
		List:   L,
		GetKey: getkey,
		TtyOut: ttyout,
	}
	io.WriteString(ttyout, ansi.CursorOff)
	defer io.WriteString(ttyout, ansi.CursorOn+"\n")

	for {
		session.tail = session.Window
		rewind := pager.Show(func(width int) (line string, ok bool) {
			if session.tail != nil {
				if obj, okk := session.tail.Value.(Displayer); okk {
					line, ok = obj.Display(pager.Width), true
				} else {
					line, ok = session.tail.Value.(string)
				}
				session.tail = session.tail.Next()
			}
			return
		}, ttyout)
		if pager.Status != nil {
			s := pager.Status(session)
			s = Truncate(s, pager.Width)
			io.WriteString(ttyout, s)
		}
		key, err := getkey()
		if err != nil {
			return err
		}
		if pager.Handler != nil {
			if ok, err := pager.Handler(session, key); err != nil {
				return err
			} else if ok {
				rewind()
				continue
			}
		}
		switch key {
		case "<":
			session.Front()
		case ">":
			session.Back()
		case " ":
			session.NextPage()
		case "b":
			session.PrevPage()
		case "j", keys.Down, keys.CtrlN:
			session.Next()
		case "k", keys.Up, keys.CtrlP:
			session.Prev()
		case "q", keys.CtrlC, keys.CtrlG:
			return nil
		case "l", keys.Right, keys.CtrlF:
			pager.offset++
		case "h", keys.Left, keys.CtrlB:
			if pager.offset > 0 {
				pager.offset--
			}
		default:
		}
		rewind()
	}
	return nil
}

func (pager *Pager) EventLoop(tty ttyadapter.Tty, L *list.List, ttyout io.Writer) error {
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
	return pager.eventLoop(tty.GetKey, L, ttyout)
}
