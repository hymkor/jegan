package main

import (
	"bufio"
	"container/list"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/mattn/go-colorable"
	"github.com/mattn/go-runewidth"

	"github.com/nyaosorg/go-ttyadapter"
	"github.com/nyaosorg/go-ttyadapter/tty8pe"

	"github.com/hymkor/jegan/internal/ansi"
	"github.com/hymkor/jegan/internal/nonblock"
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

type ttyX struct {
	ttyadapter.Tty
	nonBlock *nonblock.NonBlock[any]
	work     func(any, error) bool
}

const updateInterval = 4

func newTtyX(
	tty ttyadapter.Tty,
	dataGetter func() (any, error),
	work func(any, error) bool) *ttyX {

	return &ttyX{
		Tty:      tty,
		nonBlock: nonblock.New(tty.GetKey, dataGetter),
		work:     work,
	}
}

func (t *ttyX) GetKey() (string, error) {
	return t.nonBlock.GetOr(t.work)
}

func (t *ttyX) Close() error {
	t.nonBlock.Close()
	return t.Tty.Close()
}

func main1(source io.Reader, title string) error {
	var tty ttyadapter.Tty
	lines := list.New()

	pager1 := &pager.Pager{
		Status: func(s *pager.Session) string {
			if title != "" {
				pos := 0
				if s != nil {
					pos = s.WinPos
				}
				return fmt.Sprintf(
					ansi.Reverse+"%s"+ansi.Inverse+" %d/%d ",
					title,
					pos,
					lines.Len())
			}
			return ""
		},
	}
	ttyout := colorable.NewColorableStdout()

	sc := bufio.NewScanner(source)
	const interval = 4
	displayUpdateTime := time.Now().Add(time.Second / interval)
	i := 0
	for {
		if !sc.Scan() {
			tty = &tty8pe.Tty{}
			break
		}
		obj := textElement(sc.Text())
		lines.PushBack(obj)
		i++
		if i >= 30 {
			tty = newTtyX(&tty8pe.Tty{},
				func() (any, error) {
					if sc.Scan() {
						return textElement(sc.Text()), nil
					}
					return nil, sc.Err()
				},
				func(obj any, err error) bool {
					if err != nil {
						return false
					}
					if obj != nil {
						lines.PushBack(obj)
					}
					if time.Now().After(displayUpdateTime) {
						status := pager1.Status(nil)
						if w, _, err := tty.Size(); err == nil {
							status = pager.Truncate(status, w)
						}
						fmt.Fprintf(ttyout, "\r%s"+ansi.EraseLine, status)
						displayUpdateTime = time.Now().Add(time.Second / interval)
					}
					return true
				},
			)
			break
		}
	}
	if err := sc.Err(); err != nil && !errors.Is(err, io.EOF) {
		return err
	}
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
