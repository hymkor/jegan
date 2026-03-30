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

func pager(source io.Reader, title string) error {
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
	pager := &Pager{
		Status: func(_ *Session, out io.Writer) error {
			if title != "" {
				fmt.Fprintf(out, "\x1B[7m%s\x1B[0m", title)
			}
			return nil
		},
	}
	ttyout := colorable.NewColorableStdout()
	return pager.EventLoop(tty, lines, ttyout)
}

func mains(args []string) error {
	if len(args) < 1 {
		return pager(os.Stdin, "<STDIN>")
	}
	fd, err := os.Open(args[0])
	if err != nil {
		return err
	}
	defer fd.Close()
	return pager(fd, args[0])
}

func main() {
	if err := mains(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
