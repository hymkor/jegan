package main

import (
	"fmt"
	"io"
)

type scanner struct {
	r          io.RuneScanner
	line       int
	column     int
	lastLine   int
	lastColumn int
}

func newScanner(r io.RuneScanner) *scanner {
	return &scanner{
		r: r,
	}
}

func (sc *scanner) WrapError(e error) error {
	return fmt.Errorf("%d:%d %w",
		sc.lastLine+1,
		sc.lastColumn+1,
		e)
}

func (sc *scanner) ReadRune() (r rune, size int, err error) {
	r, size, err = sc.r.ReadRune()
	sc.lastLine = sc.line
	sc.lastColumn = sc.column
	if r == '\n' {
		sc.line++
		sc.column = 0
	} else {
		sc.column++
	}
	return
}

func (sc *scanner) UnreadRune() (err error) {
	err = sc.r.UnreadRune()
	if err != nil {
		return
	}
	sc.line = sc.lastLine
	sc.column = sc.lastColumn
	return
}
