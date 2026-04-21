package auto

import (
	"fmt"
	"io"
	"strings"
	"unicode/utf8"
)

type Pilot struct {
	script string
}

func New(s string) *Pilot {
	return &Pilot{script: s}
}

func (ap *Pilot) Open(_ func(w, h int)) error {
	return nil
}

func (ap *Pilot) Size() (int, int, error) {
	return 80, 25, nil
}

func (ap *Pilot) Next() (string, error) {
	if ap.script == "" {
		return "", io.EOF
	}
	var command string
	command, ap.script, _ = strings.Cut(ap.script, "|")
	return command, nil
}

func (ap *Pilot) ReadLine(io.Writer, string, string) (string, error) {
	return ap.Next()
}

func (ap *Pilot) GetKey() (string, error) {
	key, err := ap.Next()
	if err != nil || len(key) <= 1 || key[0] == '\x1B' {
		return key, err
	}
	if utf8.RuneCountInString(key) != 1 {
		return key, fmt.Errorf("%#v: too long string for getkey", key)
	}
	return key, nil
}

func (ap *Pilot) Close() error {
	return nil
}
