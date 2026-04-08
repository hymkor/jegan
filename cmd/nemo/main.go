package main

import (
	"bufio"
	"container/list"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/mattn/go-colorable"
	"github.com/mattn/go-runewidth"

	"github.com/nyaosorg/go-ttyadapter/tty8pe"

	"github.com/hymkor/jegan/internal/ansi"
	"github.com/hymkor/jegan/internal/pager"
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

	pager1 := &pager.Pager{
		Status: func(s *pager.Session) string {
			var b strings.Builder
			if title != "" {
				b.WriteString(ansi.Reverse)
				b.WriteString(title)
				b.WriteString(ansi.Inverse)
			}
			L := lines.Len()
			b.Write([]byte{' '})
			start := s.WinPos
			end := s.WinPos + s.Pager.Height - 1
			if end > L {
				end = L
			}
			fmt.Fprintf(&b, "%d-%d", start+1, end+1)
			b.Write([]byte{'/'})
			fmt.Fprintf(&b, "%d", lines.Len())
			b.WriteString(ansi.EraseLine)
			return b.String()
		},
	}

	session := &pager.Session{
		Pager:  pager1,
		List:   lines,
		TtyOut: colorable.NewColorableStdout(),
	}

	sc := bufio.NewScanner(source)
	const interval = 4
	displayUpdateTime := time.Now().Add(time.Second / interval)

	dataGetter := func() (any, error) {
		if sc.Scan() {
			return textElement(sc.Text()), nil
		}
		if err := sc.Err(); err != nil {
			return nil, err
		}
		return nil, io.EOF
	}

	dataStore := func(obj any, err error) bool {
		if err != nil {
			return false
		}
		if obj != nil {
			lines.PushBack(obj)
		}
		if time.Now().After(displayUpdateTime) {
			session.UpdateStatus()
			displayUpdateTime = time.Now().Add(time.Second / interval)
		}
		return true
	}
	return eventLoop(session, &tty8pe.Tty{}, dataGetter, dataStore)
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
