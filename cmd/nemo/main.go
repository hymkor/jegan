package main

import (
	"bufio"
	"container/list"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mattn/go-colorable"
	"github.com/mattn/go-runewidth"

	"github.com/nyaosorg/go-ttyadapter/tty8pe"

	"github.com/hymkor/jegan/internal/ansi"
	"github.com/hymkor/jegan/internal/asyncpager"
)

type textElement string

func (t textElement) Display(w int) string {
	s := string(t)
	for {
		i := strings.IndexByte(s, '\t')
		if i < 0 {
			return runewidth.Truncate(s, w-1, "")
		}
		s = s[:i] + "    "[i%4:] + s[i+1:]
	}
}

func main1(source io.Reader, title string) error {
	lines := list.New()

	pg := &asyncpager.Pager{
		Status: func(session *asyncpager.Session) string {
			var b strings.Builder
			if title != "" {
				b.WriteString(ansi.Reverse)
				b.WriteString(title)
				b.WriteString(ansi.Inverse)
			}
			L := lines.Len()
			start := session.WinPos
			end := session.WinPos + session.Pager.Height - 1
			if end+1 > L {
				end = L - 1
			}
			fmt.Fprintf(&b, " %d-%d / %d", start+1, end+1, L)
			return b.String()
		},
	}

	sc := bufio.NewScanner(source)

	getter := func() (asyncpager.Displayer, error) {
		if sc.Scan() {
			return textElement(sc.Text()), nil
		}
		if err := sc.Err(); err != nil {
			return nil, err
		}
		return nil, io.EOF
	}

	store := func(obj asyncpager.Displayer, err error) bool {
		if err != nil {
			return false
		}
		if obj != nil {
			lines.PushBack(obj)
		}
		return true
	}

	return pg.EventLoop(
		&tty8pe.Tty{},
		getter,
		store,
		lines,
		colorable.NewColorableStdout())
}

func mains(args []string) error {
	if len(args) < 1 {
		return main1(os.Stdin, "<STDIN>")
	}
	fd, err := os.Open(args[0])
	if err != nil {
		return err
	}
	defer fd.Close()
	return main1(fd, args[0])
}

func main() {
	if err := mains(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
