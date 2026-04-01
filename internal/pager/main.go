package pager

import (
	"container/list"
	"fmt"
	"io"

	"github.com/nyaosorg/go-ttyadapter"
)

type Pager struct {
	cache   []string
	Width   int
	Height  int
	Handler func(*Session, string) (bool, error)
	Status  func(*Session, io.Writer) error
}

func (pager *Pager) Show(fetch func(int) (string, bool), out io.Writer) func() {
	i := 0
	for i < pager.Height {
		line, ok := fetch(pager.Width)
		if !ok {
			for ; i < len(pager.cache) && i < pager.Height; i++ {
				io.WriteString(out, "\x1B[0K\n")
				pager.cache[i] = ""
			}
			break
		}
		if i >= len(pager.cache) || pager.cache[i] != line {
			io.WriteString(out, line)
			io.WriteString(out, "\x1B[0K")
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
	io.WriteString(ttyout, "\x1B[?25l")
	defer io.WriteString(ttyout, "\x1B[?25h\n")

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
			pager.Status(session, ttyout)
		}
		key, err := getkey()
		if err != nil {
			return err
		}
		if pager.Handler != nil {
			if ok, msg, err := pager.Handler(session, key); err != nil {
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
		case "j", "\x1B[B":
			session.Next()
		case "k", "\x1B[A":
			session.Prev()
		case "\x03", "\a", "q":
			return nil
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
