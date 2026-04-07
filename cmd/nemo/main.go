package main

import (
	"bufio"
	"container/list"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

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
	tty := &tty8pe.Tty{}
	lines := list.New()
	sc := bufio.NewScanner(source)
	for sc.Scan() {
		obj := textElement(sc.Text())
		lines.PushBack(obj)
	}
	if err := sc.Err(); err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	pager1 := &pager.Pager{
		Status: func(_ *pager.Session) string {
			if title != "" {
				return fmt.Sprintf(ansi.Reverse+"%s"+ansi.Inverse, title)
			}
			return ""
		},
	}
	ttyout := colorable.NewColorableStdout()
	return pager1.EventLoop(tty, lines, ttyout)
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
