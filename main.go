package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/mattn/go-colorable"

	"github.com/nyaosorg/go-ttyadapter/tty8pe"
	"github.com/nyaosorg/go-windows-dbg"

	"github.com/hymkor/jegan/internal/pager"
)

func debug(v ...any) {
	if false {
		dbg.Println(v...)
	}
}

func main1(data []byte, title string) error {
	var v any
	err := json.Unmarshal(data, &v)
	if err != nil {
		return err
	}
	L := Read(v)
	app := newApplication(L)
	app.Title = title

	pager1 := &pager.Pager{
		Status: func(_ *pager.Session, out io.Writer) error {
			if title != "" {
				fmt.Fprintf(out, "\x1B[7m%s\x1B[0m\x1B[0K", title)
			}
			return nil
		},
		Handler: app.Handle,
	}
	ttyout := colorable.NewColorableStdout()
	return pager1.EventLoop(&tty8pe.Tty{}, L, ttyout)
}

func mains(args []string) error {
	if len(args) < 1 {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		return main1(data, "<STDIN>")
	}
	data, err := os.ReadFile(args[0])
	if err != nil {
		return err
	}
	return main1(data, args[0])
}

func main() {
	if err := mains(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
